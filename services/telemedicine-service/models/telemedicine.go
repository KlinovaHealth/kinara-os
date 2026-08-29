package models

import (
	"time"

	"github.com/google/uuid"
)

type ConsultationStatus string

const (
	StatusScheduled  ConsultationStatus = "scheduled"
	StatusInProgress ConsultationStatus = "in_progress"
	StatusCompleted  ConsultationStatus = "completed"
	StatusCancelled  ConsultationStatus = "cancelled"
	StatusNoShow     ConsultationStatus = "no_show"
)

type ConsultationType string

const (
	TypeVideo    ConsultationType = "video"
	TypeAudio    ConsultationType = "audio"
	TypeChat     ConsultationType = "chat"
	TypeInPerson ConsultationType = "in_person"
)

type Consultation struct {
	ID              uuid.UUID          `json:"id"`
	ConsultRef      string             `json:"consult_ref"`
	PatientID       uuid.UUID          `json:"patient_id"`
	DoctorID        uuid.UUID          `json:"doctor_id"`
	ClinicID        uuid.UUID          `json:"clinic_id"`
	Type            ConsultationType   `json:"type"`
	Status          ConsultationStatus `json:"status"`
	ChiefComplaint  string             `json:"chief_complaint"`
	ScheduledAt     time.Time          `json:"scheduled_at"`
	StartedAt       *time.Time         `json:"started_at,omitempty"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	DurationMinutes *int               `json:"duration_minutes,omitempty"`
	CostUSD         float64            `json:"cost_usd"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type Doctor struct {
	ID             uuid.UUID `json:"id"`
	ClinicID       uuid.UUID `json:"clinic_id"`
	FullName       string    `json:"full_name"`
	Specialization string    `json:"specialization"`
	LicenseNumber  string    `json:"license_number"`
	IsAvailable    bool      `json:"is_available"`
	CreatedAt      time.Time `json:"created_at"`
}

type Prescription struct {
	ID             uuid.UUID `json:"id"`
	ConsultationID uuid.UUID `json:"consultation_id"`
	PatientID      uuid.UUID `json:"patient_id"`
	DoctorID       uuid.UUID `json:"doctor_id"`
	Medication     string    `json:"medication_name"`
	Dosage         string    `json:"dosage"`
	FrequencyDays  int       `json:"frequency_days"`
	Instructions   string    `json:"instructions"`
	IssuedAt       time.Time `json:"issued_at"`
}

type RecordingMetadata struct {
	ID              uuid.UUID `json:"id"`
	ConsultationID  uuid.UUID `json:"consultation_id"`
	StoragePath     string    `json:"storage_path"`
	DurationSeconds int       `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

type TelemedicineAuditLog struct {
	ID             uuid.UUID `json:"id"`
	ConsultationID uuid.UUID `json:"consultation_id"`
	ActorID        uuid.UUID `json:"actor_id"`
	Action         string    `json:"action"`
	Detail         string    `json:"detail"`
	CreatedAt      time.Time `json:"created_at"`
}

type VideoToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	RoomID    string    `json:"room_id"`
}

type BookConsultationRequest struct {
	PatientID      string `json:"patient_id"`
	DoctorID       string `json:"doctor_id"`
	ClinicID       string `json:"clinic_id"`
	Type           string `json:"type"`
	ChiefComplaint string `json:"chief_complaint"`
	ScheduledAt    string `json:"scheduled_at"`
}

type IssuePrescriptionRequest struct {
	Medication    string `json:"medication_name"`
	Dosage        string `json:"dosage"`
	FrequencyDays int    `json:"frequency_days"`
	Instructions  string `json:"instructions"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
