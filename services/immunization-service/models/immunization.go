package models

import (
	"time"

	"github.com/google/uuid"
)

type DoseStatus string

const (
	DoseAdministered DoseStatus = "administered"
	DoseScheduled    DoseStatus = "scheduled"
	DoseOverdue      DoseStatus = "overdue"
	DoseMissed       DoseStatus = "missed"
)

type ImmunizationRecord struct {
	ID              uuid.UUID  `json:"id"`
	RecordRef       string     `json:"record_ref"`
	PatientID       uuid.UUID  `json:"patient_id"`
	VaccineCode     string     `json:"vaccine_code"`
	VaccineName     string     `json:"vaccine_name"`
	DoseNumber      int        `json:"dose_number"`
	AdministeredBy  uuid.UUID  `json:"administered_by"`
	AdministeredAt  time.Time  `json:"administered_at"`
	LotNumber       string     `json:"lot_number"`
	ExpiryDate      *time.Time `json:"expiry_date,omitempty"`
	SiteOfInjection string     `json:"site_of_injection"`
	ClinicID        string     `json:"clinic_id"`
	NextDoseDate    *time.Time `json:"next_dose_date,omitempty"`
	Status          DoseStatus `json:"status"`
	TenantID        string     `json:"tenant_id"`
	CreatedAt       time.Time  `json:"created_at"`
}

type CreateImmunizationRequest struct {
	PatientID       uuid.UUID  `json:"patient_id"`
	VaccineCode     string     `json:"vaccine_code"`
	VaccineName     string     `json:"vaccine_name"`
	DoseNumber      int        `json:"dose_number"`
	AdministeredBy  uuid.UUID  `json:"administered_by"`
	AdministeredAt  time.Time  `json:"administered_at"`
	LotNumber       string     `json:"lot_number"`
	ExpiryDate      *time.Time `json:"expiry_date"`
	SiteOfInjection string     `json:"site_of_injection"`
	ClinicID        string     `json:"clinic_id"`
	NextDoseDate    *time.Time `json:"next_dose_date"`
}

type ImmunizationSummary struct {
	PatientID    uuid.UUID            `json:"patient_id"`
	TotalDoses   int                  `json:"total_doses"`
	OverdueDoses int                  `json:"overdue_doses"`
	Records      []ImmunizationRecord `json:"records"`
}

type VaccineDue struct {
	VaccineCode string    `json:"vaccine_code"`
	VaccineName string    `json:"vaccine_name"`
	DueDate     time.Time `json:"due_date"`
	Status      string    `json:"status"` // "overdue", "due", "upcoming"
}

type ImmunizationAlert struct {
	ID        uuid.UUID `json:"id"`
	PatientID uuid.UUID `json:"patient_id"`
	Message   string    `json:"message"`
	SentAt    time.Time `json:"sent_at"`
}

type ComplianceReport struct {
	ClinicID        string  `json:"clinic_id"`
	CompliancePct   float64 `json:"compliance_percent"`
	VaccinatedCount int     `json:"vaccinated_count"`
	TotalEligible   int     `json:"total_eligible"`
}

type CoverageItem struct {
	VaccineCode   string `json:"vaccine_code"`
	CoverageCount int    `json:"coverage_count"`
}

type SendAlertRequest struct {
	PatientID uuid.UUID `json:"patient_id"`
	Message   string    `json:"message"`
}
