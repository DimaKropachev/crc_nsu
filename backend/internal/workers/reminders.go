package workers

import (
	"context"
	"fmt"
	"time"

	"career-consultations-backend/internal/notifications"
	"career-consultations-backend/internal/repository"
	"go.uber.org/zap"
)

type ReminderWorker struct {
	repo   *repository.Postgres
	sender notifications.Sender
	log    *zap.Logger
}

func NewReminderWorker(repo *repository.Postgres, sender notifications.Sender, log *zap.Logger) *ReminderWorker {
	return &ReminderWorker{repo: repo, sender: sender, log: log}
}
func (w *ReminderWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	w.process(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}
func (w *ReminderWorker) process(ctx context.Context) {
	for i := 0; i < 20; i++ {
		item, ok, err := w.repo.ClaimDueReminder(ctx)
		if err != nil {
			w.log.Error("claim reminder failed", zap.Error(err))
			return
		}
		if !ok {
			return
		}
		loc, err := time.LoadLocation("Asia/Novosibirsk")
		if err != nil {
			loc = time.UTC
		}
		message := fmt.Sprintf("Напоминание: карьерная консультация запланирована на %s. Консультант: %s. Откройте VK Mini App, чтобы посмотреть детали.", item.StartsAt.In(loc).Format("02.01.2006 в 15:04 MST"), item.ConsultantName)
		sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = w.sender.SendReminder(sendCtx, item.VKID, item.ID, message)
		cancel()
		if err != nil {
			w.log.Warn("reminder delivery failed", zap.String("reminder_id", item.ID.String()), zap.Error(err))
			if saveErr := w.repo.FailReminder(ctx, item.ID, err.Error(), item.Attempts); saveErr != nil {
				w.log.Error("save reminder failure failed", zap.Error(saveErr))
			}
			continue
		}
		if err := w.repo.MarkReminderSent(ctx, item.ID); err != nil {
			w.log.Error("mark reminder sent failed", zap.String("reminder_id", item.ID.String()), zap.Error(err))
		}
	}
}
