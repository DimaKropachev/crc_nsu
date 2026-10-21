package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"career-consultations-backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrForbidden       = errors.New("forbidden")
	ErrSlotUnavailable = errors.New("slot unavailable")
	ErrRescheduleLimit = errors.New("reschedule limit reached")
	ErrConsentRequired = errors.New("external AI consent required")
)

type Postgres struct{ Pool *pgxpool.Pool }

func New(ctx context.Context, databaseURL string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 15
	cfg.MinConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Postgres{Pool: pool}, nil
}

func (r *Postgres) UpsertVKUser(ctx context.Context, vkID int64, first, last, avatar string, isAdmin bool) (models.User, error) {
	role := "student"
	if isAdmin {
		role = "admin"
	}
	var user models.User
	err := r.Pool.QueryRow(ctx, `INSERT INTO users(vk_id, first_name, last_name, avatar_url, role, last_login_at)
VALUES($1,$2,$3,NULLIF($4,''),$5,now())
ON CONFLICT(vk_id) DO UPDATE SET first_name=EXCLUDED.first_name,last_name=EXCLUDED.last_name,avatar_url=EXCLUDED.avatar_url,
role=CASE WHEN users.role='admin' OR EXCLUDED.role='admin' THEN 'admin' ELSE 'student' END,updated_at=now(),last_login_at=now()
RETURNING id,vk_id,first_name,last_name,avatar_url,role,created_at,updated_at,last_login_at,notification_consent,notification_consent_at`, vkID, first, last, avatar, role).Scan(&user.ID, &user.VKID, &user.FirstName, &user.LastName, &user.AvatarURL, &user.Role, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt, &user.NotificationConsent, &user.NotificationConsentAt)
	if err != nil {
		return user, err
	}
	if user.Role == "admin" {
		_, err = r.Pool.Exec(ctx, `INSERT INTO consultant_profiles(user_id) VALUES($1) ON CONFLICT(user_id) DO NOTHING`, user.ID)
		if err != nil {
			return user, err
		}
	}
	return user, nil
}
func (r *Postgres) GetUser(ctx context.Context, id uuid.UUID) (models.User, error) {
	var u models.User
	err := r.Pool.QueryRow(ctx, `SELECT id,vk_id,first_name,last_name,avatar_url,role,created_at,updated_at,last_login_at,notification_consent,notification_consent_at FROM users WHERE id=$1`, id).Scan(&u.ID, &u.VKID, &u.FirstName, &u.LastName, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &u.NotificationConsent, &u.NotificationConsentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}
func (r *Postgres) GetConsultantID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.Pool.QueryRow(ctx, `SELECT id FROM consultant_profiles WHERE user_id=$1 AND is_active=true`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return id, ErrNotFound
	}
	return id, err
}
func (r *Postgres) ListConsultants(ctx context.Context) ([]models.Consultant, error) {
	rows, err := r.Pool.Query(ctx, `SELECT cp.id,cp.user_id,u.first_name,u.last_name,u.avatar_url,cp.specialization,cp.description,cp.is_active FROM consultant_profiles cp JOIN users u ON u.id=cp.user_id WHERE cp.is_active=true ORDER BY u.last_name,u.first_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Consultant, 0)
	for rows.Next() {
		var c models.Consultant
		if err := rows.Scan(&c.ID, &c.UserID, &c.FirstName, &c.LastName, &c.AvatarURL, &c.Specialization, &c.Description, &c.IsActive); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *Postgres) GetConsultant(ctx context.Context, id uuid.UUID) (models.Consultant, error) {
	var c models.Consultant
	err := r.Pool.QueryRow(ctx, `SELECT cp.id,cp.user_id,u.first_name,u.last_name,u.avatar_url,cp.specialization,cp.description,cp.is_active FROM consultant_profiles cp JOIN users u ON u.id=cp.user_id WHERE cp.id=$1 AND cp.is_active=true`, id).Scan(&c.ID, &c.UserID, &c.FirstName, &c.LastName, &c.AvatarURL, &c.Specialization, &c.Description, &c.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}
func (r *Postgres) ListSlots(ctx context.Context, consultantID uuid.UUID, from, to time.Time) ([]models.Slot, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id,consultant_id,starts_at,ends_at,status,created_at,updated_at FROM consultation_slots WHERE consultant_id=$1 AND status='available' AND starts_at >= $2 AND starts_at < $3 AND starts_at > now() ORDER BY starts_at`, consultantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Slot, 0)
	for rows.Next() {
		var s models.Slot
		if err := rows.Scan(&s.ID, &s.ConsultantID, &s.StartsAt, &s.EndsAt, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *Postgres) ListAdminSlots(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]models.Slot, error) {
	cid, err := r.GetConsultantID(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.Pool.Query(ctx, `SELECT id,consultant_id,starts_at,ends_at,status,created_at,updated_at FROM consultation_slots WHERE consultant_id=$1 AND starts_at >= $2 AND starts_at < $3 ORDER BY starts_at`, cid, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Slot, 0)
	for rows.Next() {
		var s models.Slot
		if err := rows.Scan(&s.ID, &s.ConsultantID, &s.StartsAt, &s.EndsAt, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *Postgres) CreateSlot(ctx context.Context, userID uuid.UUID, start, end time.Time) (models.Slot, error) {
	cid, err := r.GetConsultantID(ctx, userID)
	if err != nil {
		return models.Slot{}, err
	}
	var s models.Slot
	err = r.Pool.QueryRow(ctx, `INSERT INTO consultation_slots(consultant_id,starts_at,ends_at) VALUES($1,$2,$3) RETURNING id,consultant_id,starts_at,ends_at,status,created_at,updated_at`, cid, start, end).Scan(&s.ID, &s.ConsultantID, &s.StartsAt, &s.EndsAt, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if isConstraint(err) {
		return s, ErrConflict
	}
	return s, err
}
func (r *Postgres) UpdateSlot(ctx context.Context, userID, slotID uuid.UUID, start, end time.Time) (models.Slot, error) {
	cid, err := r.GetConsultantID(ctx, userID)
	if err != nil {
		return models.Slot{}, err
	}
	var s models.Slot
	err = r.Pool.QueryRow(ctx, `UPDATE consultation_slots SET starts_at=$3,ends_at=$4,updated_at=now() WHERE id=$1 AND consultant_id=$2 AND status='available' AND starts_at>now() RETURNING id,consultant_id,starts_at,ends_at,status,created_at,updated_at`, slotID, cid, start, end).Scan(&s.ID, &s.ConsultantID, &s.StartsAt, &s.EndsAt, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	if isConstraint(err) {
		return s, ErrConflict
	}
	return s, err
}
func (r *Postgres) BlockSlot(ctx context.Context, userID, slotID uuid.UUID) error {
	cid, err := r.GetConsultantID(ctx, userID)
	if err != nil {
		return err
	}
	tag, err := r.Pool.Exec(ctx, `UPDATE consultation_slots SET status='blocked',updated_at=now() WHERE id=$1 AND consultant_id=$2 AND status='available' AND starts_at>now()`, slotID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Postgres) CreateConsultation(ctx context.Context, studentID, slotID uuid.UUID, problem string, fileID *uuid.UUID) (models.Consultation, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return models.Consultation{}, err
	}
	defer tx.Rollback(ctx)
	var consultantID uuid.UUID
	var slotStatus string
	var starts, end time.Time
	var active bool
	err = tx.QueryRow(ctx, `SELECT s.consultant_id,s.status,s.starts_at,s.ends_at,cp.is_active FROM consultation_slots s JOIN consultant_profiles cp ON cp.id=s.consultant_id WHERE s.id=$1 FOR UPDATE OF s`, slotID).Scan(&consultantID, &slotStatus, &starts, &end, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Consultation{}, ErrNotFound
	}
	if err != nil {
		return models.Consultation{}, err
	}
	if slotStatus != "available" || !active || !starts.After(time.Now()) {
		return models.Consultation{}, ErrSlotUnavailable
	}
	if fileID != nil {
		var owner uuid.UUID
		err = tx.QueryRow(ctx, `SELECT owner_user_id FROM uploaded_files WHERE id=$1`, *fileID).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Consultation{}, ErrNotFound
		}
		if err != nil {
			return models.Consultation{}, err
		}
		if owner != studentID {
			return models.Consultation{}, ErrForbidden
		}
	}
	_, err = tx.Exec(ctx, `UPDATE consultation_slots SET status='booked',updated_at=now() WHERE id=$1 AND status='available'`, slotID)
	if err != nil {
		return models.Consultation{}, err
	}
	var c models.Consultation
	err = tx.QueryRow(ctx, `INSERT INTO consultations(student_id,consultant_id,slot_id,problem_description,resume_file_id) VALUES($1,$2,$3,$4,$5) RETURNING id,student_id,consultant_id,slot_id,problem_description,resume_file_id,status,reschedule_count,created_at,updated_at`, studentID, consultantID, slotID, problem, fileID).Scan(&c.ID, &c.StudentID, &c.ConsultantID, &c.SlotID, &c.ProblemDescription, &c.ResumeFileID, &c.Status, &c.RescheduleCount, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO consultation_history(consultation_id,actor_user_id,action,new_value) VALUES($1,$2,'created',jsonb_build_object('slot_id',$3,'status','pending'))`, c.ID, studentID, slotID)
	if err != nil {
		return c, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO consultation_reminders(consultation_id,slot_id,scheduled_at,status) VALUES($1,$2,$3,'pending') ON CONFLICT(consultation_id,slot_id,reminder_type) DO NOTHING`, c.ID, slotID, starts.Add(-24*time.Hour))
	if err != nil {
		return c, err
	}
	if err = tx.Commit(ctx); err != nil {
		return c, err
	}
	c.StartsAt = starts
	c.EndsAt = end
	return c, nil
}

const consultationSelect = `SELECT c.id,c.student_id,c.consultant_id,c.slot_id,s.starts_at,s.ends_at,c.problem_description,c.resume_file_id,c.status,c.reschedule_count,COALESCE(cn.first_name||' '||cn.last_name,''),COALESCE(st.first_name||' '||st.last_name,''),c.created_at,c.updated_at FROM consultations c JOIN consultation_slots s ON s.id=c.slot_id JOIN consultant_profiles cp ON cp.id=c.consultant_id JOIN users cn ON cn.id=cp.user_id JOIN users st ON st.id=c.student_id`

func scanConsultation(row pgx.Row) (models.Consultation, error) {
	var c models.Consultation
	err := row.Scan(&c.ID, &c.StudentID, &c.ConsultantID, &c.SlotID, &c.StartsAt, &c.EndsAt, &c.ProblemDescription, &c.ResumeFileID, &c.Status, &c.RescheduleCount, &c.ConsultantName, &c.StudentName, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}
func (r *Postgres) GetConsultation(ctx context.Context, actorID uuid.UUID, role string, id uuid.UUID) (models.Consultation, error) {
	query := consultationSelect + ` WHERE c.id=$1 AND (c.student_id=$2 OR (c.consultant_id IN (SELECT id FROM consultant_profiles WHERE user_id=$2) AND $3='admin'))`
	c, err := scanConsultation(r.Pool.QueryRow(ctx, query, id, actorID, role))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}
func (r *Postgres) ListMyConsultations(ctx context.Context, userID uuid.UUID) ([]models.Consultation, error) {
	return r.listConsultations(ctx, consultationSelect+` WHERE c.student_id=$1 ORDER BY s.starts_at DESC`, userID)
}
func (r *Postgres) ListAdminConsultations(ctx context.Context, userID uuid.UUID, status string, from, to time.Time, limit, offset int) ([]models.Consultation, error) {
	cid, err := r.GetConsultantID(ctx, userID)
	if err != nil {
		return nil, err
	}
	query := consultationSelect + ` WHERE c.consultant_id=$1 AND s.starts_at >= $2 AND s.starts_at < $3`
	args := []any{cid, from, to}
	if status != "" {
		query += ` AND c.status=$4`
		args = append(args, status)
		query += fmt.Sprintf(` ORDER BY s.starts_at LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	} else {
		query += ` ORDER BY s.starts_at LIMIT $4 OFFSET $5`
	}
	if status != "" {
		args = append(args, limit, offset)
	} else {
		args = append(args, limit, offset)
	}
	return r.listConsultations(ctx, query, args...)
}
func (r *Postgres) listConsultations(ctx context.Context, query string, args ...any) ([]models.Consultation, error) {
	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Consultation, 0)
	for rows.Next() {
		c, err := scanConsultation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *Postgres) Reschedule(ctx context.Context, userID, consultationID, newSlotID uuid.UUID) (models.Consultation, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return models.Consultation{}, err
	}
	defer tx.Rollback(ctx)
	var studentID, oldSlotID, consultantID uuid.UUID
	var status string
	var count int
	var oldStartsAt time.Time
	err = tx.QueryRow(ctx, `SELECT c.student_id,c.slot_id,c.consultant_id,c.status,c.reschedule_count,s.starts_at FROM consultations c JOIN consultation_slots s ON s.id=c.slot_id WHERE c.id=$1 FOR UPDATE OF c`, consultationID).Scan(&studentID, &oldSlotID, &consultantID, &status, &count, &oldStartsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Consultation{}, ErrNotFound
	}
	if err != nil {
		return models.Consultation{}, err
	}
	if studentID != userID {
		return models.Consultation{}, ErrForbidden
	}
	if count >= 1 {
		return models.Consultation{}, ErrRescheduleLimit
	}
	if !oldStartsAt.After(time.Now()) {
		return models.Consultation{}, ErrConflict
	}
	if status != "pending" && status != "confirmed" {
		return models.Consultation{}, ErrConflict
	}
	if oldSlotID == newSlotID {
		return models.Consultation{}, ErrConflict
	}
	var newConsultant uuid.UUID
	var newStatus string
	var start, end time.Time
	var consultantActive bool
	err = tx.QueryRow(ctx, `SELECT s.consultant_id,s.status,s.starts_at,s.ends_at,cp.is_active FROM consultation_slots s JOIN consultant_profiles cp ON cp.id=s.consultant_id WHERE s.id=$1 FOR UPDATE OF s`, newSlotID).Scan(&newConsultant, &newStatus, &start, &end, &consultantActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Consultation{}, ErrNotFound
	}
	if err != nil {
		return models.Consultation{}, err
	}
	if newStatus != "available" || !consultantActive || !start.After(time.Now()) {
		return models.Consultation{}, ErrSlotUnavailable
	}
	_, err = tx.Exec(ctx, `UPDATE consultation_slots SET status='available',updated_at=now() WHERE id=$1 AND status='booked'`, oldSlotID)
	if err != nil {
		return models.Consultation{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE consultation_slots SET status='booked',updated_at=now() WHERE id=$1 AND status='available'`, newSlotID)
	if err != nil {
		return models.Consultation{}, err
	}
	oldJSON, _ := json.Marshal(map[string]any{"slot_id": oldSlotID, "status": status})
	newJSON, _ := json.Marshal(map[string]any{"slot_id": newSlotID, "status": "pending"})
	var c models.Consultation
	err = tx.QueryRow(ctx, `UPDATE consultations SET slot_id=$2,consultant_id=$3,status='pending',reschedule_count=reschedule_count+1,updated_at=now() WHERE id=$1 RETURNING id,student_id,consultant_id,slot_id,problem_description,resume_file_id,status,reschedule_count,created_at,updated_at`, consultationID, newSlotID, newConsultant).Scan(&c.ID, &c.StudentID, &c.ConsultantID, &c.SlotID, &c.ProblemDescription, &c.ResumeFileID, &c.Status, &c.RescheduleCount, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO consultation_history(consultation_id,actor_user_id,action,old_value,new_value) VALUES($1,$2,'rescheduled',$3,$4)`, consultationID, userID, oldJSON, newJSON)
	if err != nil {
		return c, err
	}
	_, err = tx.Exec(ctx, `UPDATE consultation_reminders SET status='cancelled',updated_at=now() WHERE consultation_id=$1 AND status IN ('pending','sending')`, consultationID)
	if err != nil {
		return c, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO consultation_reminders(consultation_id,slot_id,scheduled_at,status) VALUES($1,$2,$3,'pending') ON CONFLICT(consultation_id,slot_id,reminder_type) DO UPDATE SET scheduled_at=EXCLUDED.scheduled_at,status='pending',attempts=0,last_error=NULL,sent_at=NULL,updated_at=now()`, consultationID, newSlotID, start.Add(-24*time.Hour))
	if err != nil {
		return c, err
	}
	if err = tx.Commit(ctx); err != nil {
		return c, err
	}
	c.StartsAt = start
	c.EndsAt = end
	return c, nil
}
func (r *Postgres) CancelConsultation(ctx context.Context, userID, consultationID uuid.UUID) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var studentID, slotID uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `SELECT student_id,slot_id,status FROM consultations WHERE id=$1 FOR UPDATE`, consultationID).Scan(&studentID, &slotID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if studentID != userID {
		return ErrForbidden
	}
	if status == "cancelled" || status == "completed" {
		return ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE consultations SET status='cancelled',cancelled_at=now(),updated_at=now() WHERE id=$1`, consultationID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE consultation_slots SET status='available',updated_at=now() WHERE id=$1 AND starts_at>now() AND status='booked'`, slotID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE consultation_reminders SET status='cancelled',updated_at=now() WHERE consultation_id=$1 AND status IN ('pending','sending')`, consultationID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO consultation_history(consultation_id,actor_user_id,action,old_value,new_value) VALUES($1,$2,'cancelled',jsonb_build_object('status',$3),jsonb_build_object('status','cancelled'))`, consultationID, userID, status)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Postgres) SetConsultationStatus(ctx context.Context, userID, consultationID uuid.UUID, target string) error {
	cid, err := r.GetConsultantID(ctx, userID)
	if err != nil {
		return err
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM consultations WHERE id=$1 AND consultant_id=$2 FOR UPDATE`, consultationID, cid).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	allowed := target == "confirmed" && status == "pending" || target == "completed" && status == "confirmed"
	if !allowed {
		return ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE consultations SET status=$2,updated_at=now() WHERE id=$1`, consultationID, target)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO consultation_history(consultation_id,actor_user_id,action,old_value,new_value) VALUES($1,$2,$3,jsonb_build_object('status',$4),jsonb_build_object('status',$3))`, consultationID, userID, target, status)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Postgres) GetAdminProfile(ctx context.Context, userID uuid.UUID) (models.Consultant, error) {
	var c models.Consultant
	err := r.Pool.QueryRow(ctx, `SELECT cp.id,cp.user_id,u.first_name,u.last_name,u.avatar_url,cp.specialization,cp.description,cp.is_active FROM consultant_profiles cp JOIN users u ON u.id=cp.user_id WHERE cp.user_id=$1`, userID).Scan(&c.ID, &c.UserID, &c.FirstName, &c.LastName, &c.AvatarURL, &c.Specialization, &c.Description, &c.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}
func (r *Postgres) UpdateAdminProfile(ctx context.Context, userID uuid.UUID, specialization, description string) (models.Consultant, error) {
	_, err := r.Pool.Exec(ctx, `UPDATE consultant_profiles SET specialization=$2,description=$3,updated_at=now() WHERE user_id=$1`, userID, specialization, description)
	if err != nil {
		return models.Consultant{}, err
	}
	return r.GetAdminProfile(ctx, userID)
}
func (r *Postgres) CreateFile(ctx context.Context, file models.UploadedFile) (models.UploadedFile, error) {
	err := r.Pool.QueryRow(ctx, `INSERT INTO uploaded_files(id,owner_user_id,original_filename,storage_key,mime_type,size_bytes) VALUES($1,$2,$3,$4,$5,$6) RETURNING created_at`, file.ID, file.OwnerUserID, file.OriginalFilename, file.StorageKey, file.MIMEType, file.SizeBytes).Scan(&file.CreatedAt)
	return file, err
}
func (r *Postgres) GetFile(ctx context.Context, id, owner uuid.UUID) (models.UploadedFile, error) {
	var f models.UploadedFile
	err := r.Pool.QueryRow(ctx, `SELECT id,owner_user_id,original_filename,storage_key,mime_type,size_bytes,created_at FROM uploaded_files WHERE id=$1 AND owner_user_id=$2`, id, owner).Scan(&f.ID, &f.OwnerUserID, &f.OriginalFilename, &f.StorageKey, &f.MIMEType, &f.SizeBytes, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}
func (r *Postgres) DeleteFile(ctx context.Context, id, owner uuid.UUID) (models.UploadedFile, error) {
	var f models.UploadedFile
	err := r.Pool.QueryRow(ctx, `DELETE FROM uploaded_files f WHERE f.id=$1 AND f.owner_user_id=$2 AND NOT EXISTS(SELECT 1 FROM consultations c WHERE c.resume_file_id=f.id) AND NOT EXISTS(SELECT 1 FROM resume_analyses a WHERE a.file_id=f.id) RETURNING id,owner_user_id,original_filename,storage_key,mime_type,size_bytes,created_at`, id, owner).Scan(&f.ID, &f.OwnerUserID, &f.OriginalFilename, &f.StorageKey, &f.MIMEType, &f.SizeBytes, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}
func (r *Postgres) CreateAnalysis(ctx context.Context, userID, fileID uuid.UUID, externalAIConsent bool) (models.ResumeAnalysis, error) {
	if !externalAIConsent {
		return models.ResumeAnalysis{}, ErrConsentRequired
	}
	var owner uuid.UUID
	err := r.Pool.QueryRow(ctx, `SELECT owner_user_id FROM uploaded_files WHERE id=$1`, fileID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ResumeAnalysis{}, ErrNotFound
	}
	if err != nil {
		return models.ResumeAnalysis{}, err
	}
	if owner != userID {
		return models.ResumeAnalysis{}, ErrForbidden
	}
	var a models.ResumeAnalysis
	err = r.Pool.QueryRow(ctx, `INSERT INTO resume_analyses(user_id,file_id,external_ai_consent,external_ai_consent_at) VALUES($1,$2,true,now()) RETURNING id,user_id,file_id,status,result,error_code,attempt_count,created_at,started_at,completed_at,updated_at`, userID, fileID).Scan(&a.ID, &a.UserID, &a.FileID, &a.Status, &a.Result, &a.ErrorCode, &a.AttemptCount, &a.CreatedAt, &a.StartedAt, &a.CompletedAt, &a.UpdatedAt)
	return a, err
}
func (r *Postgres) GetAnalysis(ctx context.Context, userID, id uuid.UUID) (models.ResumeAnalysis, error) {
	var a models.ResumeAnalysis
	err := r.Pool.QueryRow(ctx, `SELECT id,user_id,file_id,status,result,error_code,attempt_count,created_at,started_at,completed_at,updated_at FROM resume_analyses WHERE id=$1 AND user_id=$2`, id, userID).Scan(&a.ID, &a.UserID, &a.FileID, &a.Status, &a.Result, &a.ErrorCode, &a.AttemptCount, &a.CreatedAt, &a.StartedAt, &a.CompletedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}
func (r *Postgres) ListAnalyses(ctx context.Context, userID uuid.UUID) ([]models.ResumeAnalysis, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id,user_id,file_id,status,result,error_code,attempt_count,created_at,started_at,completed_at,updated_at FROM resume_analyses WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ResumeAnalysis, 0)
	for rows.Next() {
		var a models.ResumeAnalysis
		if err := rows.Scan(&a.ID, &a.UserID, &a.FileID, &a.Status, &a.Result, &a.ErrorCode, &a.AttemptCount, &a.CreatedAt, &a.StartedAt, &a.CompletedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *Postgres) ClaimAnalysis(ctx context.Context) (models.ResumeAnalysis, models.UploadedFile, bool, error) {
	_, err := r.Pool.Exec(ctx, `UPDATE resume_analyses SET status='queued',updated_at=now() WHERE status='processing' AND external_ai_consent=true AND updated_at < now()-interval '10 minutes'`)
	if err != nil {
		return models.ResumeAnalysis{}, models.UploadedFile{}, false, err
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return models.ResumeAnalysis{}, models.UploadedFile{}, false, err
	}
	defer tx.Rollback(ctx)
	var a models.ResumeAnalysis
	var f models.UploadedFile
	err = tx.QueryRow(ctx, `SELECT id,user_id,file_id FROM resume_analyses WHERE status='queued' AND external_ai_consent=true ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&a.ID, &a.UserID, &a.FileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, f, false, nil
	}
	if err != nil {
		return a, f, false, err
	}
	err = tx.QueryRow(ctx, `SELECT id,owner_user_id,original_filename,storage_key,mime_type,size_bytes,created_at FROM uploaded_files WHERE id=$1`, a.FileID).Scan(&f.ID, &f.OwnerUserID, &f.OriginalFilename, &f.StorageKey, &f.MIMEType, &f.SizeBytes, &f.CreatedAt)
	if err != nil {
		return a, f, false, err
	}
	err = tx.QueryRow(ctx, `UPDATE resume_analyses SET status='processing',started_at=now(),attempt_count=attempt_count+1,updated_at=now() WHERE id=$1 RETURNING status,attempt_count,created_at,started_at,updated_at`, a.ID).Scan(&a.Status, &a.AttemptCount, &a.CreatedAt, &a.StartedAt, &a.UpdatedAt)
	if err != nil {
		return a, f, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return a, f, false, err
	}
	return a, f, true, nil
}
func (r *Postgres) CompleteAnalysis(ctx context.Context, id uuid.UUID, result json.RawMessage) error {
	_, err := r.Pool.Exec(ctx, `UPDATE resume_analyses SET status='completed',result=$2::jsonb,error_code=NULL,completed_at=now(),updated_at=now() WHERE id=$1`, id, string(result))
	return err
}
func (r *Postgres) FailAnalysis(ctx context.Context, id uuid.UUID, code string, retry bool) error {
	status := "failed"
	if retry {
		status = "queued"
	}
	_, err := r.Pool.Exec(ctx, `UPDATE resume_analyses SET status=$2,error_code=$3,completed_at=CASE WHEN $2='failed' THEN now() ELSE NULL END,updated_at=now() WHERE id=$1`, id, status, code)
	return err
}
func isConstraint(err error) bool {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		return strings.HasPrefix(pgerr.Code, "23")
	}
	return false
}

func (r *Postgres) SetNotificationConsent(ctx context.Context, userID uuid.UUID, enabled bool) (models.User, error) {
	var u models.User
	err := r.Pool.QueryRow(ctx, `UPDATE users SET notification_consent=$2,notification_consent_at=CASE WHEN $2 THEN now() ELSE NULL END,updated_at=now() WHERE id=$1 RETURNING id,vk_id,first_name,last_name,avatar_url,role,created_at,updated_at,last_login_at,notification_consent,notification_consent_at`, userID, enabled).Scan(&u.ID, &u.VKID, &u.FirstName, &u.LastName, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &u.NotificationConsent, &u.NotificationConsentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// DueReminder is a notification job claimed by a worker under a row lock.
type DueReminder struct {
	ID               uuid.UUID
	ConsultationID   uuid.UUID
	VKID             int64
	StudentFirstName string
	ConsultantName   string
	StartsAt         time.Time
	Attempts         int
}

func (r *Postgres) ClaimDueReminder(ctx context.Context) (DueReminder, bool, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return DueReminder{}, false, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE consultation_reminders SET status='pending',updated_at=now() WHERE status='sending' AND updated_at < now()-interval '5 minutes'`)
	if err != nil {
		return DueReminder{}, false, err
	}
	_, err = tx.Exec(ctx, `UPDATE consultation_reminders cr SET status='cancelled',updated_at=now() FROM consultations c JOIN consultation_slots s ON s.id=c.slot_id WHERE cr.consultation_id=c.id AND cr.status='pending' AND (c.status IN ('cancelled','completed') OR s.starts_at<=now() OR cr.slot_id<>c.slot_id)`)
	if err != nil {
		return DueReminder{}, false, err
	}
	var item DueReminder
	err = tx.QueryRow(ctx, `SELECT cr.id,c.id,u.vk_id,u.first_name,cn.first_name||' '||cn.last_name,s.starts_at,cr.attempts
FROM consultation_reminders cr JOIN consultations c ON c.id=cr.consultation_id JOIN consultation_slots s ON s.id=c.slot_id
JOIN users u ON u.id=c.student_id JOIN consultant_profiles cp ON cp.id=c.consultant_id JOIN users cn ON cn.id=cp.user_id
WHERE cr.status='pending' AND cr.scheduled_at<=now() AND c.status IN ('pending','confirmed') AND s.starts_at>now() AND cr.slot_id=c.slot_id AND u.notification_consent=true
ORDER BY cr.scheduled_at LIMIT 1 FOR UPDATE OF cr SKIP LOCKED`).Scan(&item.ID, &item.ConsultationID, &item.VKID, &item.StudentFirstName, &item.ConsultantName, &item.StartsAt, &item.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		// Recovery and cancellation are maintenance work even when no job is sendable.
		return DueReminder{}, false, tx.Commit(ctx)
	}
	if err != nil {
		return DueReminder{}, false, err
	}
	item.Attempts++
	_, err = tx.Exec(ctx, `UPDATE consultation_reminders SET status='sending',attempts=$2,last_error=NULL,updated_at=now() WHERE id=$1`, item.ID, item.Attempts)
	if err != nil {
		return DueReminder{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DueReminder{}, false, err
	}
	return item, true, nil
}

func (r *Postgres) MarkReminderSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.Pool.Exec(ctx, `UPDATE consultation_reminders SET status='sent',sent_at=now(),last_error=NULL,updated_at=now() WHERE id=$1 AND status='sending'`, id)
	return err
}

func (r *Postgres) FailReminder(ctx context.Context, id uuid.UUID, message string, attempts int) error {
	status := "pending"
	if attempts >= 5 {
		status = "failed"
	}
	if len(message) > 500 {
		message = message[:500]
	}
	_, err := r.Pool.Exec(ctx, `UPDATE consultation_reminders SET status=$2,scheduled_at=CASE WHEN $2='pending' THEN now()+interval '5 minutes' ELSE scheduled_at END,last_error=$3,updated_at=now() WHERE id=$1 AND status='sending'`, id, status, message)
	return err
}

func (r *Postgres) GetFileAccessible(ctx context.Context, id, actorID uuid.UUID, role string) (models.UploadedFile, error) {
	var f models.UploadedFile
	err := r.Pool.QueryRow(ctx, `SELECT f.id,f.owner_user_id,f.original_filename,f.storage_key,f.mime_type,f.size_bytes,f.created_at
FROM uploaded_files f WHERE f.id=$1 AND (f.owner_user_id=$2 OR ($3='admin' AND EXISTS (
  SELECT 1 FROM consultations c JOIN consultant_profiles cp ON cp.id=c.consultant_id
  WHERE c.resume_file_id=f.id AND cp.user_id=$2
)))`, id, actorID, role).Scan(&f.ID, &f.OwnerUserID, &f.OriginalFilename, &f.StorageKey, &f.MIMEType, &f.SizeBytes, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}
