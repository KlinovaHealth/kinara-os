package models

import (
	"time"

	"github.com/google/uuid"
)

// PatientStatus represents the lifecycle state of a patient record.
type PatientStatus string

const (
	StatusActive     PatientStatus = "active"
	StatusInactive   PatientStatus = "inactive"
	StatusDeceased   PatientStatus = "deceased"
	StatusTransferred PatientStatus = "transferred"
)

// Gender values aligned with WHO standards.
type Gender string

const (
	GenderMale          Gender = "male"
	GenderFemale        Gender = "female"
	GenderOther         Gender = "other"
	GenderPreferNotSay  Gender = "prefer_not_to_say"
)

// AuditAction enumerates all operations tracked in the immutable audit log.
type AuditAction string

const (
	AuditCreate AuditAction = "create"
	AuditRead   AuditAction = "read"
	AuditUpdate AuditAction = "update"
	AuditDelete AuditAction = "delete"
	AuditSearch AuditAction = "search"
)

// EmergencyContact is stored as encrypted JSON inside the patients row.
type EmergencyContact struct {
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Relationship string `json:"relationship"`
}

// Patient is the decrypted, in-memory representation of a patient record.
// PHI fields are never stored in plaintext in PostgreSQL.
type Patient struct {
	ID                   uuid.UUID        `json:"id"`
	NationalID           string           `json:"national_id"`
	FullName             string           `json:"full_name"`
	DateOfBirth          time.Time        `json:"date_of_birth"`
	Gender               Gender           `json:"gender"`
	PhoneNumber          string           `json:"phone_number"`
	Email                string           `json:"email,omitempty"`
	Address              string           `json:"address,omitempty"`
	Country              string           `json:"country"`
	Region               string           `json:"region,omitempty"`
	BloodType            string           `json:"blood_type,omitempty"`
	Allergies            []string         `json:"allergies,omitempty"`
	EmergencyContact     EmergencyContact `json:"emergency_contact,omitempty"`
	Status               PatientStatus    `json:"status"`
	CreatedBy            uuid.UUID        `json:"created_by"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

// PatientRow is the encrypted database representation. All *_enc fields
// contain AES-256-GCM ciphertext encoded as base64.
type PatientRow struct {
	ID                       uuid.UUID  `db:"id"`
	NationalIDEnc            string     `db:"national_id_enc"`
	FullNameEnc              string     `db:"full_name_enc"`
	DateOfBirthEnc           string     `db:"date_of_birth_enc"`
	Gender                   Gender     `db:"gender"`
	PhoneNumberEnc           string     `db:"phone_number_enc"`
	EmailEnc                 *string    `db:"email_enc"`
	AddressEnc               *string    `db:"address_enc"`
	Country                  string     `db:"country"`
	Region                   *string    `db:"region"`
	BloodTypeEnc             *string    `db:"blood_type_enc"`
	AllergiesEnc             *string    `db:"allergies_enc"`
	EmergencyContactNameEnc  *string    `db:"emergency_contact_name_enc"`
	EmergencyContactPhoneEnc *string    `db:"emergency_contact_phone_enc"`
	EmergencyContactRelEnc   *string    `db:"emergency_contact_rel_enc"`
	Status                   PatientStatus `db:"status"`
	CreatedBy                uuid.UUID  `db:"created_by"`
	CreatedAt                time.Time  `db:"created_at"`
	UpdatedAt                time.Time  `db:"updated_at"`
	DeletedAt                *time.Time `db:"deleted_at"`
}

// CreatePatientRequest is the JSON body for POST /api/v1/patients.
type CreatePatientRequest struct {
	NationalID       string           `json:"national_id"`
	FullName         string           `json:"full_name"`
	DateOfBirth      string           `json:"date_of_birth"` // RFC3339 date: "2000-01-15"
	Gender           Gender           `json:"gender"`
	PhoneNumber      string           `json:"phone_number"`
	Email            string           `json:"email,omitempty"`
	Address          string           `json:"address,omitempty"`
	Country          string           `json:"country"`
	Region           string           `json:"region,omitempty"`
	BloodType        string           `json:"blood_type,omitempty"`
	Allergies        []string         `json:"allergies,omitempty"`
	EmergencyContact EmergencyContact `json:"emergency_contact,omitempty"`
}

// UpdatePatientRequest is the JSON body for PUT /api/v1/patients/:id.
// All fields are optional (partial update).
type UpdatePatientRequest struct {
	PhoneNumber      *string           `json:"phone_number,omitempty"`
	Email            *string           `json:"email,omitempty"`
	Address          *string           `json:"address,omitempty"`
	Region           *string           `json:"region,omitempty"`
	BloodType        *string           `json:"blood_type,omitempty"`
	Allergies        []string          `json:"allergies,omitempty"`
	EmergencyContact *EmergencyContact `json:"emergency_contact,omitempty"`
	Status           *PatientStatus    `json:"status,omitempty"`
}

// PatientListRequest holds query parameters for GET /api/v1/patients.
type PatientListRequest struct {
	Country string
	Region  string
	Status  PatientStatus
	Page    int
	Limit   int
}

// PatientAuditLog is a single immutable audit entry.
type PatientAuditLog struct {
	ID          uuid.UUID              `json:"id"`
	PatientID   uuid.UUID              `json:"patient_id"`
	Action      AuditAction            `json:"action"`
	AccessorID  uuid.UUID              `json:"accessor_id"`
	AccessorRole string               `json:"accessor_role"`
	IPAddress   string                 `json:"ip_address"`
	RequestID   string                 `json:"request_id"`
	Changes     map[string]interface{} `json:"changes,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// APIResponse wraps all JSON responses from this service.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *PageMeta   `json:"meta,omitempty"`
}

// APIError is the structured error returned to callers.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PageMeta carries pagination metadata in list responses.
type PageMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}
