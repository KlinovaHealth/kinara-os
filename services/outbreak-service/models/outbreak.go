package models

import (
	"time"

	"github.com/google/uuid"
)

type SuspectedCase struct {
	ID          uuid.UUID `json:"id"`
	CaseRef     string    `json:"case_ref"`
	PatientID   uuid.UUID `json:"patient_id"`
	DiseaseCode string    `json:"disease_code"`
	DiseaseName string    `json:"disease_name"`
	ClinicID    string    `json:"clinic_id"`
	Location    string    `json:"location"`
	Symptoms    string    `json:"symptoms"`
	ReportedBy  uuid.UUID `json:"reported_by"`
	TenantID    string    `json:"tenant_id"`
	ReportedAt  time.Time `json:"reported_at"`
}

type ConfirmedOutbreak struct {
	ID          uuid.UUID  `json:"id"`
	AlertRef    string     `json:"alert_ref"`
	DiseaseCode string     `json:"disease_code"`
	DiseaseName string     `json:"disease_name"`
	ClinicID    string     `json:"clinic_id"`
	CaseCount   int        `json:"case_count"`
	Status      string     `json:"status"` // "active", "confirmed", "contained"
	TenantID    string     `json:"tenant_id"`
	DetectedAt  time.Time  `json:"detected_at"`
	ContainedAt *time.Time `json:"contained_at,omitempty"`
}

type DiseaseCluster struct {
	DiseaseCode string `json:"disease_code"`
	DiseaseName string `json:"disease_name"`
	ClinicID    string `json:"clinic_id"`
	CaseCount   int    `json:"case_count"`
}

type DiseaseTrend struct {
	DiseaseCode string    `json:"disease_code"`
	Date        time.Time `json:"date"`
	CaseCount   int       `json:"case_count"`
}

type OutbreakNotification struct {
	ID         uuid.UUID `json:"id"`
	OutbreakID uuid.UUID `json:"outbreak_id"`
	Message    string    `json:"message"`
	Recipients string    `json:"recipients"`
	SentBy     uuid.UUID `json:"sent_by"`
	SentAt     time.Time `json:"sent_at"`
}

type ReportCaseRequest struct {
	PatientID   uuid.UUID `json:"patient_id"`
	DiseaseCode string    `json:"disease_code"`
	DiseaseName string    `json:"disease_name"`
	ClinicID    string    `json:"clinic_id"`
	Location    string    `json:"location"`
	Symptoms    string    `json:"symptoms"`
}

type NotifyRequest struct {
	OutbreakID uuid.UUID `json:"outbreak_id"`
	Message    string    `json:"message"`
	Recipients string    `json:"recipients"`
}
