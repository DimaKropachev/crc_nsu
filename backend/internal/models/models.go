package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                    uuid.UUID  `json:"id"`
	VKID                  int64      `json:"vk_id"`
	FirstName             string     `json:"first_name"`
	LastName              string     `json:"last_name"`
	AvatarURL             *string    `json:"avatar_url"`
	Role                  string     `json:"role"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastLoginAt           *time.Time `json:"last_login_at,omitempty"`
	NotificationConsent   bool       `json:"notification_consent"`
	NotificationConsentAt *time.Time `json:"notification_consent_at,omitempty"`
}

type Consultant struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	AvatarURL      *string   `json:"avatar_url"`
	Specialization string    `json:"specialization"`
	Description    string    `json:"description"`
	IsActive       bool      `json:"is_active"`
}

type Slot struct {
	ID           uuid.UUID `json:"id"`
	ConsultantID uuid.UUID `json:"consultant_id"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Consultation struct {
	ID                 uuid.UUID  `json:"id"`
	StudentID          uuid.UUID  `json:"student_id"`
	ConsultantID       uuid.UUID  `json:"consultant_id"`
	SlotID             uuid.UUID  `json:"slot_id"`
	StartsAt           time.Time  `json:"starts_at"`
	EndsAt             time.Time  `json:"ends_at"`
	ProblemDescription string     `json:"problem_description"`
	ResumeFileID       *uuid.UUID `json:"resume_file_id,omitempty"`
	Status             string     `json:"status"`
	RescheduleCount    int        `json:"reschedule_count"`
	ConsultantName     string     `json:"consultant_name,omitempty"`
	StudentName        string     `json:"student_name,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type UploadedFile struct {
	ID               uuid.UUID `json:"id"`
	OwnerUserID      uuid.UUID `json:"-"`
	OriginalFilename string    `json:"original_filename"`
	StorageKey       string    `json:"-"`
	MIMEType         string    `json:"mime_type"`
	SizeBytes        int64     `json:"size_bytes"`
	CreatedAt        time.Time `json:"created_at"`
}

type ResumeAnalysis struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"-"`
	FileID       uuid.UUID       `json:"file_id"`
	Status       string          `json:"status"`
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorCode    *string         `json:"error_code,omitempty"`
	AttemptCount int             `json:"attempt_count"`
	CreatedAt    time.Time       `json:"created_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type APIError struct {
	Error APIErrorDetail `json:"error"`
}
type APIErrorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}
