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
	ID               uuid.UUID  `json:"id"`
	RecordRef        string     `json:"record_ref"`
	PatientID        uuid.UUID  `json:"patient_id"`
	VaccineCode      string     `json:"vaccine_code"`
	VaccineName      string     `json:"vaccine_name"`
	DoseNumber       int        `json:"dose_number"`
	AdministeredBy   uuid.UUID  `json:"administered_by"`
	AdministeredAt   time.Time  `json:"administered_at"`
	LotNumber        string     `json:"lot_number"`
	ExpiryDate       *time.Time `json:"expiry_date,omitempty"`
	SiteOfInjection  string     `json:"site_of_injection"`
	ClinicID         string     `json:"clinic_id"`
	NextDoseDate     *time.Time `json:"next_dose_date,omitempty"`
	Status           DoseStatus `json:"status"`
	TenantID         string     `json:"tenant_id"`
	CreatedAt        time.Time  `json:"created_at"`
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
	PatientID        uuid.UUID            `json:"patient_id"`
	TotalDoses       int                  `json:"total_doses"`
	OverdueDoses     int                  `json:"overdue_doses"`
	Records          []ImmunizationRecord `json:"records"`
}
