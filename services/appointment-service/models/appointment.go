package models

import (
	"time"

	"github.com/google/uuid"
)

type AppointmentStatus string
type AppointmentType string

const (
	StatusScheduled  AppointmentStatus = "scheduled"
	StatusConfirmed  AppointmentStatus = "confirmed"
	StatusCompleted  AppointmentStatus = "completed"
	StatusCancelled  AppointmentStatus = "cancelled"
	StatusNoShow     AppointmentStatus = "no_show"

	TypeConsultation AppointmentType = "consultation"
	TypeFollowUp     AppointmentType = "follow_up"
	TypeProcedure    AppointmentType = "procedure"
	TypeEmergency    AppointmentType = "emergency"
)

type Appointment struct {
	ID             uuid.UUID         `json:"id"`
	AppointmentRef string            `json:"appointment_ref"`
	PatientID      uuid.UUID         `json:"patient_id"`
	DoctorID       uuid.UUID         `json:"doctor_id"`
	ClinicID       string            `json:"clinic_id"`
	ScheduledAt    time.Time         `json:"scheduled_at"`
	DurationMin    int               `json:"duration_minutes"`
	Type           AppointmentType   `json:"type"`
	Status         AppointmentStatus `json:"status"`
	Notes          string            `json:"notes,omitempty"`
	TenantID       string            `json:"tenant_id"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type CreateAppointmentRequest struct {
	PatientID   uuid.UUID       `json:"patient_id"`
	DoctorID    uuid.UUID       `json:"doctor_id"`
	ClinicID    string          `json:"clinic_id"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	DurationMin int             `json:"duration_minutes"`
	Type        AppointmentType `json:"type"`
	Notes       string          `json:"notes"`
}

type UpdateStatusRequest struct {
	Status AppointmentStatus `json:"status"`
	Notes  string            `json:"notes"`
}
