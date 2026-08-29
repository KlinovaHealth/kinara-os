package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Enumerations ─────────────────────────────────────────────────────────────

type ConsultationStatus string

const (
	ConsultScheduled  ConsultationStatus = "scheduled"
	ConsultInProgress ConsultationStatus = "in_progress"
	ConsultCompleted  ConsultationStatus = "completed"
	ConsultCancelled  ConsultationStatus = "cancelled"
	ConsultNoShow     ConsultationStatus = "no_show"
)

type ConsultationType string

const (
	TypeVideo     ConsultationType = "video"
	TypeAudio     ConsultationType = "audio"
	TypeChat      ConsultationType = "chat"
	TypeInPerson  ConsultationType = "in_person"
	TypeWhatsApp  ConsultationType = "whatsapp"
)

type Severity string

const (
	SeverityMild     Severity = "mild"
	SeverityModerate Severity = "moderate"
	SeveritySevere   Severity = "severe"
	SeverityCritical Severity = "critical"
)

type TreatmentType string

const (
	TreatmentMedication  TreatmentType = "medication"
	TreatmentProcedure   TreatmentType = "procedure"
	TreatmentReferral    TreatmentType = "referral"
	TreatmentLifestyle   TreatmentType = "lifestyle"
	TreatmentMonitoring  TreatmentType = "monitoring"
)

type TreatmentStatus string

const (
	TreatmentActive        TreatmentStatus = "active"
	TreatmentCompleted     TreatmentStatus = "completed"
	TreatmentDiscontinued  TreatmentStatus = "discontinued"
)

type NoteType string

const (
	NoteSubjective NoteType = "subjective" // patient-reported symptoms
	NoteObjective  NoteType = "objective"  // clinician observations
	NoteAssessment NoteType = "assessment" // clinical assessment
	NotePlan       NoteType = "plan"       // treatment plan
	NoteGeneral    NoteType = "general"
)

type PrescriptionStatus string

const (
	PrescriptionPending   PrescriptionStatus = "pending"
	PrescriptionSent      PrescriptionStatus = "sent"
	PrescriptionDispensed PrescriptionStatus = "dispensed"
	PrescriptionCancelled PrescriptionStatus = "cancelled"
)

type AuditAction string

const (
	AuditCreate AuditAction = "create"
	AuditRead   AuditAction = "read"
	AuditUpdate AuditAction = "update"
	AuditDelete AuditAction = "delete"
)

// ─── Domain models (decrypted / in-memory) ────────────────────────────────────

// Consultation is a clinical encounter between a patient and a clinician.
type Consultation struct {
	ID               uuid.UUID          `json:"id"`
	PatientID        uuid.UUID          `json:"patient_id"`
	DoctorID         uuid.UUID          `json:"doctor_id"`
	Status           ConsultationStatus `json:"status"`
	ConsultationType ConsultationType   `json:"consultation_type"`
	ChiefComplaint   string             `json:"chief_complaint"`      // PHI — decrypted
	ScheduledAt      *time.Time         `json:"scheduled_at,omitempty"`
	StartedAt        *time.Time         `json:"started_at,omitempty"`
	EndedAt          *time.Time         `json:"ended_at,omitempty"`
	Country          string             `json:"country"`
	Region           string             `json:"region,omitempty"`
	FacilityID       *uuid.UUID         `json:"facility_id,omitempty"`
	CreatedBy        uuid.UUID          `json:"created_by"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// ConsultationRow is the encrypted database representation.
type ConsultationRow struct {
	ID                  uuid.UUID          `db:"id"`
	PatientID           uuid.UUID          `db:"patient_id"`
	DoctorID            uuid.UUID          `db:"doctor_id"`
	Status              ConsultationStatus `db:"status"`
	ConsultationType    ConsultationType   `db:"consultation_type"`
	ChiefComplaintEnc   string             `db:"chief_complaint_enc"`
	ScheduledAt         *time.Time         `db:"scheduled_at"`
	StartedAt           *time.Time         `db:"started_at"`
	EndedAt             *time.Time         `db:"ended_at"`
	Country             string             `db:"country"`
	Region              *string            `db:"region"`
	FacilityID          *uuid.UUID         `db:"facility_id"`
	CreatedBy           uuid.UUID          `db:"created_by"`
	CreatedAt           time.Time          `db:"created_at"`
	UpdatedAt           time.Time          `db:"updated_at"`
}

// Diagnosis captures a clinical diagnosis tied to a consultation.
type Diagnosis struct {
	ID              uuid.UUID  `json:"id"`
	ConsultationID  uuid.UUID  `json:"consultation_id"`
	PatientID       uuid.UUID  `json:"patient_id"`
	DoctorID        uuid.UUID  `json:"doctor_id"`
	ICD10Code       string     `json:"icd10_code"`
	ICD10Desc       string     `json:"icd10_description"`
	ClinicalNotes   string     `json:"clinical_notes"`           // PHI — decrypted
	Severity        Severity   `json:"severity"`
	IsPrimary       bool       `json:"is_primary"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// DiagnosisRow is the encrypted form stored in PostgreSQL.
type DiagnosisRow struct {
	ID                uuid.UUID `db:"id"`
	ConsultationID    uuid.UUID `db:"consultation_id"`
	PatientID         uuid.UUID `db:"patient_id"`
	DoctorID          uuid.UUID `db:"doctor_id"`
	ICD10Code         string    `db:"icd10_code"`
	ICD10Desc         string    `db:"icd10_description"`
	ClinicalNotesEnc  string    `db:"clinical_notes_enc"`
	Severity          Severity  `db:"severity"`
	IsPrimary         bool      `db:"is_primary"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// Treatment is a clinical action prescribed during a consultation.
type Treatment struct {
	ID             uuid.UUID       `json:"id"`
	ConsultationID uuid.UUID       `json:"consultation_id"`
	PatientID      uuid.UUID       `json:"patient_id"`
	DoctorID       uuid.UUID       `json:"doctor_id"`
	TreatmentType  TreatmentType   `json:"treatment_type"`
	Instructions   string          `json:"instructions"`      // PHI — decrypted
	DurationDays   int             `json:"duration_days,omitempty"`
	FollowUpDate   *time.Time      `json:"follow_up_date,omitempty"`
	Status         TreatmentStatus `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// TreatmentRow is the encrypted database form.
type TreatmentRow struct {
	ID               uuid.UUID       `db:"id"`
	ConsultationID   uuid.UUID       `db:"consultation_id"`
	PatientID        uuid.UUID       `db:"patient_id"`
	DoctorID         uuid.UUID       `db:"doctor_id"`
	TreatmentType    TreatmentType   `db:"treatment_type"`
	InstructionsEnc  string          `db:"instructions_enc"`
	DurationDays     int             `db:"duration_days"`
	FollowUpDate     *time.Time      `db:"follow_up_date"`
	Status           TreatmentStatus `db:"status"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
}

// ClinicalNote is a SOAP-style note entry. Notes are immutable once written.
type ClinicalNote struct {
	ID             uuid.UUID `json:"id"`
	ConsultationID uuid.UUID `json:"consultation_id"`
	PatientID      uuid.UUID `json:"patient_id"`
	AuthorID       uuid.UUID `json:"author_id"`
	NoteType       NoteType  `json:"note_type"`
	Content        string    `json:"content"`     // PHI — decrypted
	CreatedAt      time.Time `json:"created_at"`
}

// ClinicalNoteRow is the encrypted form.
type ClinicalNoteRow struct {
	ID             uuid.UUID `db:"id"`
	ConsultationID uuid.UUID `db:"consultation_id"`
	PatientID      uuid.UUID `db:"patient_id"`
	AuthorID       uuid.UUID `db:"author_id"`
	NoteType       NoteType  `db:"note_type"`
	ContentEnc     string    `db:"content_enc"`
	CreatedAt      time.Time `db:"created_at"`
}

// Medication is one item in a prescription.
type Medication struct {
	Name      string `json:"name"`
	Dosage    string `json:"dosage"`
	Frequency string `json:"frequency"`
	Duration  string `json:"duration"`
	Route     string `json:"route"` // oral, IV, topical, etc.
}

// Prescription links a consultation to a pharmacy order.
type Prescription struct {
	ID             uuid.UUID          `json:"id"`
	ConsultationID uuid.UUID          `json:"consultation_id"`
	PatientID      uuid.UUID          `json:"patient_id"`
	DoctorID       uuid.UUID          `json:"doctor_id"`
	PharmacyID     *uuid.UUID         `json:"pharmacy_id,omitempty"`
	Medications    []Medication       `json:"medications"`       // PHI — decrypted
	Notes          string             `json:"notes,omitempty"`   // PHI — decrypted
	Status         PrescriptionStatus `json:"status"`
	DispensedAt    *time.Time         `json:"dispensed_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// PrescriptionRow is the encrypted form.
type PrescriptionRow struct {
	ID              uuid.UUID          `db:"id"`
	ConsultationID  uuid.UUID          `db:"consultation_id"`
	PatientID       uuid.UUID          `db:"patient_id"`
	DoctorID        uuid.UUID          `db:"doctor_id"`
	PharmacyID      *uuid.UUID         `db:"pharmacy_id"`
	MedicationsEnc  string             `db:"medications_enc"`
	NotesEnc        *string            `db:"notes_enc"`
	Status          PrescriptionStatus `db:"status"`
	DispensedAt     *time.Time         `db:"dispensed_at"`
	CreatedAt       time.Time          `db:"created_at"`
	UpdatedAt       time.Time          `db:"updated_at"`
}

// ClinicalAuditLog is one immutable audit entry for any clinical resource.
type ClinicalAuditLog struct {
	ID           uuid.UUID              `json:"id"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   uuid.UUID              `json:"resource_id"`
	PatientID    uuid.UUID              `json:"patient_id"`
	Action       AuditAction            `json:"action"`
	AccessorID   uuid.UUID              `json:"accessor_id"`
	AccessorRole string                 `json:"accessor_role"`
	IPAddress    string                 `json:"ip_address"`
	RequestID    string                 `json:"request_id"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ─── Request / Response types ─────────────────────────────────────────────────

type CreateConsultationRequest struct {
	PatientID        uuid.UUID        `json:"patient_id"`
	DoctorID         uuid.UUID        `json:"doctor_id"`
	ConsultationType ConsultationType `json:"consultation_type"`
	ChiefComplaint   string           `json:"chief_complaint"`
	ScheduledAt      *time.Time       `json:"scheduled_at,omitempty"`
	Country          string           `json:"country"`
	Region           string           `json:"region,omitempty"`
	FacilityID       *uuid.UUID       `json:"facility_id,omitempty"`
}

type UpdateConsultationRequest struct {
	Status         *ConsultationStatus `json:"status,omitempty"`
	ChiefComplaint *string             `json:"chief_complaint,omitempty"`
	StartedAt      *time.Time          `json:"started_at,omitempty"`
	EndedAt        *time.Time          `json:"ended_at,omitempty"`
}

type CreateDiagnosisRequest struct {
	ICD10Code     string   `json:"icd10_code"`
	ICD10Desc     string   `json:"icd10_description"`
	ClinicalNotes string   `json:"clinical_notes"`
	Severity      Severity `json:"severity"`
	IsPrimary     bool     `json:"is_primary"`
}

type CreateTreatmentRequest struct {
	TreatmentType TreatmentType `json:"treatment_type"`
	Instructions  string        `json:"instructions"`
	DurationDays  int           `json:"duration_days,omitempty"`
	FollowUpDate  *time.Time    `json:"follow_up_date,omitempty"`
}

type CreateNoteRequest struct {
	NoteType NoteType `json:"note_type"`
	Content  string   `json:"content"`
}

type CreatePrescriptionRequest struct {
	PharmacyID  *uuid.UUID   `json:"pharmacy_id,omitempty"`
	Medications []Medication `json:"medications"`
	Notes       string       `json:"notes,omitempty"`
}

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
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}
