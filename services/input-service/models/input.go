package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Form struct {
	ID        uuid.UUID       `json:"id"`
	FormType  string          `json:"form_type"`
	Title     string          `json:"title"`
	Schema    json.RawMessage `json:"schema"`
	Version   int             `json:"version"`
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
}

type FormSubmission struct {
	ID            uuid.UUID       `json:"id"`
	SubmissionRef string          `json:"submission_ref"`
	PatientID     uuid.UUID       `json:"patient_id"`
	FormType      string          `json:"form_type"`
	FormVersion   int             `json:"form_version"`
	Data          json.RawMessage `json:"data"`
	SubmittedBy   uuid.UUID       `json:"submitted_by"`
	TenantID      string          `json:"tenant_id"`
	SubmittedAt   time.Time       `json:"submitted_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type SubmitRequest struct {
	PatientID uuid.UUID       `json:"patient_id"`
	FormType  string          `json:"form_type"`
	Data      json.RawMessage `json:"data"`
}

type UpdateRequest struct {
	Data json.RawMessage `json:"data"`
}
