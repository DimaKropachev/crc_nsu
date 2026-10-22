package tests

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"career-consultations-backend/internal/models"
	"career-consultations-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPostgresConsultationLifecycle uses an isolated schema in TEST_DATABASE_URL.
// Point this variable at a disposable database; the test drops its schema on completion.
func TestPostgresConsultationLifecycle(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database to run integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	baseCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	base, err := pgxpool.NewWithConfig(ctx, baseCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	schema := "it_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := base.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = base.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`) }()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	migration, err := os.ReadFile("../migrations/000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range strings.Split(string(migration), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("apply migration statement %q: %v", statement, err)
		}
	}
	repo := &repository.Postgres{Pool: pool}

	var studentID, adminID, consultantID, slot1, slot2, slot3 uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO users(vk_id,first_name,role) VALUES(910000001,'Student','student') RETURNING id`).Scan(&studentID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users(vk_id,first_name,role) VALUES(910000002,'Consultant','admin') RETURNING id`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO consultant_profiles(user_id) VALUES($1) RETURNING id`, adminID).Scan(&consultantID); err != nil {
		t.Fatal(err)
	}
	baseStart := time.Now().Add(4 * time.Hour).UTC().Truncate(time.Minute)
	for i, target := range []*uuid.UUID{&slot1, &slot2, &slot3} {
		start := baseStart.Add(time.Duration(i) * time.Hour)
		if err := pool.QueryRow(ctx, `INSERT INTO consultation_slots(consultant_id,starts_at,ends_at) VALUES($1,$2,$3) RETURNING id`, consultantID, start, start.Add(30*time.Minute)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}

	// Concurrent booking of one slot must result in exactly one successful booking.
	type bookingResult struct {
		consultation models.Consultation
		err          error
	}
	var wg sync.WaitGroup
	results := make(chan bookingResult, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := repo.CreateConsultation(ctx, studentID, slot1, "concurrent booking", nil)
			results <- bookingResult{consultation: c, err: err}
		}()
	}
	wg.Wait()
	close(results)
	var booked models.Consultation
	successes, conflicts := 0, 0
	for result := range results {
		if result.err == nil {
			successes++
			booked = result.consultation
			continue
		}
		if errors.Is(result.err, repository.ErrSlotUnavailable) || errors.Is(result.err, repository.ErrConflict) {
			conflicts++
			continue
		}
		t.Errorf("unexpected booking error: %v", result.err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected exactly one booking and one conflict, got successes=%d conflicts=%d", successes, conflicts)
	}
	var historySlot uuid.UUID
	var historyStatus string
	if err := pool.QueryRow(ctx, `SELECT (new_value->>'slot_id')::uuid,new_value->>'status' FROM consultation_history WHERE consultation_id=$1 AND action='created'`, booked.ID).Scan(&historySlot, &historyStatus); err != nil {
		t.Fatalf("read booking history: %v", err)
	}
	if historySlot != slot1 || historyStatus != "pending" {
		t.Fatalf("unexpected booking history: slot=%s status=%q", historySlot, historyStatus)
	}

	// The first reschedule succeeds; the next one is rejected server-side.
	moved, err := repo.Reschedule(ctx, studentID, booked.ID, slot2)
	if err != nil {
		t.Fatalf("first reschedule failed: %v", err)
	}
	if moved.RescheduleCount != 1 || moved.SlotID != slot2 || moved.Status != "pending" {
		t.Fatalf("unexpected rescheduled consultation: %+v", moved)
	}
	if _, err := repo.Reschedule(ctx, studentID, booked.ID, slot3); !errors.Is(err, repository.ErrRescheduleLimit) {
		t.Fatalf("expected reschedule limit error, got %v", err)
	}

	// Cancellation releases a future slot, and the same slot can then be booked again.
	if err := repo.CancelConsultation(ctx, studentID, booked.ID); err != nil {
		t.Fatalf("cancel consultation: %v", err)
	}
	var oldSlotStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM consultation_slots WHERE id=$1`, slot2).Scan(&oldSlotStatus); err != nil {
		t.Fatal(err)
	}
	if oldSlotStatus != "available" {
		t.Fatalf("cancelled slot should be available, got %q", oldSlotStatus)
	}
	if _, err := repo.CreateConsultation(ctx, studentID, slot2, "book again after cancellation", nil); err != nil {
		t.Fatalf("slot was not released after cancellation: %v", err)
	}

	// Consultant status transitions must allow pending -> confirmed -> completed.
	last, err := repo.CreateConsultation(ctx, studentID, slot3, "status transition", nil)
	if err != nil {
		t.Fatalf("create consultation for status test: %v", err)
	}
	if err := repo.SetConsultationStatus(ctx, adminID, last.ID, "confirmed"); err != nil {
		t.Fatalf("confirm consultation: %v", err)
	}
	if err := repo.SetConsultationStatus(ctx, adminID, last.ID, "completed"); err != nil {
		t.Fatalf("complete consultation: %v", err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM consultations WHERE id=$1`, last.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "completed" {
		t.Fatalf("expected completed status, got %q", status)
	}

	// With no consent there are no sendable jobs, but maintenance must still commit.
	if _, err := pool.Exec(ctx, `UPDATE consultation_reminders SET status='sending',updated_at=now()-interval '6 minutes' WHERE consultation_id=$1`, last.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repo.ClaimDueReminder(ctx); err != nil || ok {
		t.Fatalf("expected no sendable reminder: ok=%v err=%v", ok, err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM consultation_reminders WHERE consultation_id=$1`, last.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "cancelled" {
		t.Fatalf("maintenance was rolled back: completed consultation reminder status=%q", status)
	}
}
