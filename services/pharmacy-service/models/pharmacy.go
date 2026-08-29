package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Prescription ─────────────────────────────────────────────────────────────

type PrescriptionStatus string

const (
	PrescriptionPending    PrescriptionStatus = "pending"
	PrescriptionDispensed  PrescriptionStatus = "dispensed"
	PrescriptionPartial    PrescriptionStatus = "partial"
	PrescriptionCancelled  PrescriptionStatus = "cancelled"
	PrescriptionExpired    PrescriptionStatus = "expired"
)

// PrescriptionRow is the encrypted DB record linked to a Clinical Service prescription.
type PrescriptionRow struct {
	ID              uuid.UUID          `db:"id"`
	ClinicalID      uuid.UUID          `db:"clinical_id"` // prescription_id from clinical-service
	PatientID       uuid.UUID          `db:"patient_id"`
	ClinicID        uuid.UUID          `db:"clinic_id"`
	MedicationID    uuid.UUID          `db:"medication_id"`
	PatientNameEnc  string             `db:"patient_name_enc"`
	DosageEnc       string             `db:"dosage_enc"`
	Quantity        int                `db:"quantity"`
	QuantityUnit    string             `db:"quantity_unit"`
	Instructions    string             `db:"instructions"`
	Status          PrescriptionStatus `db:"status"`
	IssuedAt        time.Time          `db:"issued_at"`
	ExpiresAt       time.Time          `db:"expires_at"`
	CreatedAt       time.Time          `db:"created_at"`
	UpdatedAt       time.Time          `db:"updated_at"`
}

type Prescription struct {
	ID           uuid.UUID          `json:"id"`
	ClinicalID   uuid.UUID          `json:"clinical_id"`
	PatientID    uuid.UUID          `json:"patient_id"`
	ClinicID     uuid.UUID          `json:"clinic_id"`
	MedicationID uuid.UUID          `json:"medication_id"`
	PatientName  string             `json:"patient_name"`
	Dosage       string             `json:"dosage"`
	Quantity     int                `json:"quantity"`
	QuantityUnit string             `json:"quantity_unit"`
	Instructions string             `json:"instructions"`
	Status       PrescriptionStatus `json:"status"`
	IssuedAt     time.Time          `json:"issued_at"`
	ExpiresAt    time.Time          `json:"expires_at"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// ─── Medication (inventory item) ──────────────────────────────────────────────

type MedicationRow struct {
	ID             uuid.UUID  `db:"id"`
	Name           string     `db:"name"`
	GenericName    string     `db:"generic_name"`
	Description    string     `db:"description"`
	UnitPrice      float64    `db:"unit_price"`
	Currency       string     `db:"currency"`
	StockLevel     int        `db:"stock_level"`
	ReorderPoint   int        `db:"reorder_point"`
	ReorderQty     int        `db:"reorder_qty"`
	Unit           string     `db:"unit"` // tablet, ml, vial, etc.
	SupplierID     *uuid.UUID `db:"supplier_id"`
	ExpirationDate *time.Time `db:"expiration_date"`
	BatchNumber    string     `db:"batch_number"`
	RequiresCold   bool       `db:"requires_cold"`
	IsActive       bool       `db:"is_active"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// ─── Dispensing ───────────────────────────────────────────────────────────────

type DispensingRow struct {
	ID                 uuid.UUID `db:"id"`
	PrescriptionID     uuid.UUID `db:"prescription_id"`
	MedicationID       uuid.UUID `db:"medication_id"`
	DispensedByUserID  uuid.UUID `db:"dispensed_by_user_id"`
	QuantityDispensed  int       `db:"quantity_dispensed"`
	BatchNumber        string    `db:"batch_number"`
	CostAmount         float64   `db:"cost_amount"`
	Currency           string    `db:"currency"`
	PatientCostShare   float64   `db:"patient_cost_share"`
	Notes              string    `db:"notes"`
	DispensedAt        time.Time `db:"dispensed_at"`
}

// ─── Supply order ─────────────────────────────────────────────────────────────

type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderApproved  OrderStatus = "approved"
	OrderShipped   OrderStatus = "shipped"
	OrderReceived  OrderStatus = "received"
	OrderCancelled OrderStatus = "cancelled"
)

type SupplyOrderRow struct {
	ID             uuid.UUID   `db:"id"`
	SupplierID     uuid.UUID   `db:"supplier_id"`
	MedicationID   uuid.UUID   `db:"medication_id"`
	QuantityOrdered int        `db:"quantity_ordered"`
	QuantityReceived int       `db:"quantity_received"`
	UnitCost       float64     `db:"unit_cost"`
	Currency       string      `db:"currency"`
	Status         OrderStatus `db:"status"`
	OrderedByID    uuid.UUID   `db:"ordered_by_id"`
	ExpectedAt     *time.Time  `db:"expected_at"`
	ReceivedAt     *time.Time  `db:"received_at"`
	Notes          string      `db:"notes"`
	CreatedAt      time.Time   `db:"created_at"`
	UpdatedAt      time.Time   `db:"updated_at"`
}

// ─── Audit log ────────────────────────────────────────────────────────────────

type PharmacyAuditLog struct {
	ID         uuid.UUID  `json:"id"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty"`
	UserID     uuid.UUID  `json:"user_id"`
	Action     string     `json:"action"`
	Resource   string     `json:"resource"`
	IPAddress  string     `json:"ip_address"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ─── Request types ────────────────────────────────────────────────────────────

type RegisterPrescriptionRequest struct {
	ClinicalID   string `json:"clinical_id"`
	PatientID    string `json:"patient_id"`
	PatientName  string `json:"patient_name"`
	ClinicID     string `json:"clinic_id"`
	MedicationID string `json:"medication_id"`
	Dosage       string `json:"dosage"`
	Quantity     int    `json:"quantity"`
	QuantityUnit string `json:"quantity_unit"`
	Instructions string `json:"instructions"`
	ExpiresAt    string `json:"expires_at"` // RFC3339
}

type DispenseRequest struct {
	QuantityDispensed int     `json:"quantity_dispensed"`
	BatchNumber       string  `json:"batch_number"`
	CostAmount        float64 `json:"cost_amount"`
	PatientCostShare  float64 `json:"patient_cost_share"`
	Notes             string  `json:"notes,omitempty"`
}

type UpdateStockRequest struct {
	StockLevel     *int     `json:"stock_level,omitempty"`
	ReorderPoint   *int     `json:"reorder_point,omitempty"`
	ReorderQty     *int     `json:"reorder_qty,omitempty"`
	UnitPrice      *float64 `json:"unit_price,omitempty"`
	BatchNumber    *string  `json:"batch_number,omitempty"`
	ExpirationDate *string  `json:"expiration_date,omitempty"` // RFC3339
}

type CreateOrderRequest struct {
	SupplierID      string  `json:"supplier_id"`
	MedicationID    string  `json:"medication_id"`
	QuantityOrdered int     `json:"quantity_ordered"`
	UnitCost        float64 `json:"unit_cost"`
	Currency        string  `json:"currency"`
	ExpectedAt      *string `json:"expected_at,omitempty"` // RFC3339
	Notes           string  `json:"notes,omitempty"`
}

type CreateMedicationRequest struct {
	Name           string   `json:"name"`
	GenericName    string   `json:"generic_name"`
	Description    string   `json:"description"`
	UnitPrice      float64  `json:"unit_price"`
	Currency       string   `json:"currency"`
	StockLevel     int      `json:"stock_level"`
	ReorderPoint   int      `json:"reorder_point"`
	ReorderQty     int      `json:"reorder_qty"`
	Unit           string   `json:"unit"`
	SupplierID     *string  `json:"supplier_id,omitempty"`
	ExpirationDate *string  `json:"expiration_date,omitempty"`
	BatchNumber    string   `json:"batch_number"`
	RequiresCold   bool     `json:"requires_cold"`
}

// ─── Cost summary ─────────────────────────────────────────────────────────────

type CostSummary struct {
	ClinicID          string  `json:"clinic_id"`
	Period            string  `json:"period"`
	TotalDispensed    int     `json:"total_dispensed"`
	TotalCost         float64 `json:"total_cost"`
	PatientCostShare  float64 `json:"patient_cost_share"`
	FacilityCost      float64 `json:"facility_cost"`
	Currency          string  `json:"currency"`
}

// ─── Stock alert ──────────────────────────────────────────────────────────────

type StockAlert struct {
	MedicationID   uuid.UUID `json:"medication_id"`
	MedicationName string    `json:"medication_name"`
	AlertType      string    `json:"alert_type"` // low_stock, expired, expiring_soon
	StockLevel     int       `json:"stock_level"`
	ReorderPoint   int       `json:"reorder_point"`
	Message        string    `json:"message"`
}

// ─── Standard response types ──────────────────────────────────────────────────

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
