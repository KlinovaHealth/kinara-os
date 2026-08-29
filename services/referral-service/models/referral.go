package models

import (
	"time"

	"github.com/google/uuid"
)

type ReferralStatus string

const (
	ReferralPending    ReferralStatus = "pending"
	ReferralAccepted   ReferralStatus = "accepted"
	ReferralInProgress ReferralStatus = "in_progress"
	ReferralCompleted  ReferralStatus = "completed"
	ReferralRejected   ReferralStatus = "rejected"
	ReferralCancelled  ReferralStatus = "cancelled"
)

type ReferralUrgency string

const (
	UrgencyRoutine    ReferralUrgency = "routine"
	UrgencySemiUrgent ReferralUrgency = "semi_urgent"
	UrgencyUrgent     ReferralUrgency = "urgent"
	UrgencyEmergency  ReferralUrgency = "emergency"
)

// ReferralRow is the encrypted DB representation.
type ReferralRow struct {
	ID                 uuid.UUID       `db:"id"`
	PatientID          uuid.UUID       `db:"patient_id"`
	FromClinicID       uuid.UUID       `db:"from_clinic_id"`
	ToClinicID         uuid.UUID       `db:"to_clinic_id"`
	FromClinicianID    uuid.UUID       `db:"from_clinician_id"`
	ToClinicianID      *uuid.UUID      `db:"to_clinician_id"`
	ReasonEnc          string          `db:"reason_enc"`
	PatientNameEnc     string          `db:"patient_name_enc"`
	Urgency            ReferralUrgency `db:"urgency"`
	Status             ReferralStatus  `db:"status"`
	FollowUpDate       *time.Time      `db:"follow_up_date"`
	FollowUpNotesEnc   *string         `db:"follow_up_notes_enc"`
	AcceptedAt         *time.Time      `db:"accepted_at"`
	CompletedAt        *time.Time      `db:"completed_at"`
	RejectedAt         *time.Time      `db:"rejected_at"`
	RejectionReasonEnc *string         `db:"rejection_reason_enc"`
	CreatedAt          time.Time       `db:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at"`
}

// Referral is the decrypted API representation.
type Referral struct {
	ID              uuid.UUID       `json:"id"`
	PatientID       uuid.UUID       `json:"patient_id"`
	FromClinicID    uuid.UUID       `json:"from_clinic_id"`
	ToClinicID      uuid.UUID       `json:"to_clinic_id"`
	FromClinicianID uuid.UUID       `json:"from_clinician_id"`
	ToClinicianID   *uuid.UUID      `json:"to_clinician_id,omitempty"`
	Reason          string          `json:"reason"`
	PatientName     string          `json:"patient_name"`
	Urgency         ReferralUrgency `json:"urgency"`
	Status          ReferralStatus  `json:"status"`
	FollowUpDate    *time.Time      `json:"follow_up_date,omitempty"`
	FollowUpNotes   *string         `json:"follow_up_notes,omitempty"`
	AcceptedAt      *time.Time      `json:"accepted_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	RejectedAt      *time.Time      `json:"rejected_at,omitempty"`
	RejectionReason *string         `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ReferralNoteRow is the encrypted DB representation of a note.
type ReferralNoteRow struct {
	ID              uuid.UUID `db:"id"`
	ReferralID      uuid.UUID `db:"referral_id"`
	NoteEnc         string    `db:"note_enc"`
	CreatedByUserID uuid.UUID `db:"created_by_user_id"`
	CreatedAt       time.Time `db:"created_at"`
}

// ReferralNote is the decrypted API representation.
type ReferralNote struct {
	ID              uuid.UUID `json:"id"`
	ReferralID      uuid.UUID `json:"referral_id"`
	Note            string    `json:"note"`
	CreatedByUserID uuid.UUID `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
}

// ReferralHistory records every status transition. Immutable.
type ReferralHistory struct {
	ID              uuid.UUID `json:"id"`
	ReferralID      uuid.UUID `json:"referral_id"`
	StatusBefore    *string   `json:"status_before,omitempty"`
	StatusAfter     string    `json:"status_after"`
	ChangedByUserID uuid.UUID `json:"changed_by_user_id"`
	ChangedByRole   string    `json:"changed_by_role"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ReferralAuditLog records every access. Immutable.
type ReferralAuditLog struct {
	ID         uuid.UUID  `json:"id"`
	ReferralID *uuid.UUID `json:"referral_id,omitempty"`
	UserID     uuid.UUID  `json:"user_id"`
	Action     string     `json:"action"`
	Resource   string     `json:"resource"`
	IPAddress  string     `json:"ip_address"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ─── Request types ─────────────────────────────────────────────────────────────

type CreateReferralRequest struct {
	PatientID   string          `json:"patient_id"`
	PatientName string          `json:"patient_name"`
	ToClinicID  string          `json:"to_clinic_id"`
	Reason      string          `json:"reason"`
	Urgency     ReferralUrgency `json:"urgency"`
}

type UpdateReferralStatusRequest struct {
	Status          ReferralStatus `json:"status"`
	ToClinicianID   *string        `json:"to_clinician_id,omitempty"`
	RejectionReason *string        `json:"rejection_reason,omitempty"`
	Notes           *string        `json:"notes,omitempty"`
}

type AddNoteRequest struct {
	Note string `json:"note"`
}

type ScheduleFollowUpRequest struct {
	FollowUpDate string  `json:"follow_up_date"` // RFC3339
	Notes        *string `json:"notes,omitempty"`
}

// ─── Response types ────────────────────────────────────────────────────────────

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *PageMeta   `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PageMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
