package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"career-consultations-backend/internal/models"
	"career-consultations-backend/internal/repository"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid business input")

// ConsultationRepository exposes the atomic persistence operations used by the domain service.
type ConsultationRepository interface {
	CreateConsultation(context.Context, uuid.UUID, uuid.UUID, string, *uuid.UUID) (models.Consultation, error)
	Reschedule(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (models.Consultation, error)
	CancelConsultation(context.Context, uuid.UUID, uuid.UUID) error
	CreateSlot(context.Context, uuid.UUID, time.Time, time.Time) (models.Slot, error)
	UpdateSlot(context.Context, uuid.UUID, uuid.UUID, time.Time, time.Time) (models.Slot, error)
	BlockSlot(context.Context, uuid.UUID, uuid.UUID) error
	SetConsultationStatus(context.Context, uuid.UUID, uuid.UUID, string) error
}

type Consultations struct{ repo ConsultationRepository }

func NewConsultations(repo ConsultationRepository) *Consultations { return &Consultations{repo: repo} }

func (s *Consultations) Create(ctx context.Context, studentID, slotID uuid.UUID, problem string, resumeFileID *uuid.UUID) (models.Consultation, error) {
	if studentID == uuid.Nil || slotID == uuid.Nil {
		return models.Consultation{}, ErrInvalidInput
	}
	problem = strings.TrimSpace(problem)
	if len(problem) > 5000 {
		return models.Consultation{}, ErrInvalidInput
	}
	return s.repo.CreateConsultation(ctx, studentID, slotID, problem, resumeFileID)
}
func (s *Consultations) Reschedule(ctx context.Context, studentID, consultationID, newSlotID uuid.UUID) (models.Consultation, error) {
	if studentID == uuid.Nil || consultationID == uuid.Nil || newSlotID == uuid.Nil {
		return models.Consultation{}, ErrInvalidInput
	}
	return s.repo.Reschedule(ctx, studentID, consultationID, newSlotID)
}
func (s *Consultations) Cancel(ctx context.Context, studentID, consultationID uuid.UUID) error {
	if studentID == uuid.Nil || consultationID == uuid.Nil {
		return ErrInvalidInput
	}
	return s.repo.CancelConsultation(ctx, studentID, consultationID)
}
func (s *Consultations) CreateSlot(ctx context.Context, adminID uuid.UUID, start, end time.Time) (models.Slot, error) {
	if adminID == uuid.Nil || !end.After(start) || !start.After(time.Now()) || end.Sub(start) > 4*time.Hour {
		return models.Slot{}, ErrInvalidInput
	}
	return s.repo.CreateSlot(ctx, adminID, start.UTC(), end.UTC())
}
func (s *Consultations) UpdateSlot(ctx context.Context, adminID, slotID uuid.UUID, start, end time.Time) (models.Slot, error) {
	if adminID == uuid.Nil || slotID == uuid.Nil || !end.After(start) || !start.After(time.Now()) || end.Sub(start) > 4*time.Hour {
		return models.Slot{}, ErrInvalidInput
	}
	return s.repo.UpdateSlot(ctx, adminID, slotID, start.UTC(), end.UTC())
}
func (s *Consultations) BlockSlot(ctx context.Context, adminID, slotID uuid.UUID) error {
	if adminID == uuid.Nil || slotID == uuid.Nil {
		return ErrInvalidInput
	}
	return s.repo.BlockSlot(ctx, adminID, slotID)
}
func (s *Consultations) ChangeStatus(ctx context.Context, adminID, consultationID uuid.UUID, target string) error {
	if adminID == uuid.Nil || consultationID == uuid.Nil || (target != "confirmed" && target != "completed") {
		return ErrInvalidInput
	}
	return s.repo.SetConsultationStatus(ctx, adminID, consultationID, target)
}

var _ ConsultationRepository = (*repository.Postgres)(nil)
