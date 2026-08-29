package models

import (
	"time"
	"github.com/google/uuid"
)

type LCType string
type LCStatus string
type PaymentTerms string

const (
	LCStandard  LCType = "documentary"
	LCStandby   LCType = "standby"
	LCRevocable LCType = "revocable"

	LCDraft     LCStatus = "draft"
	LCIssued    LCStatus = "issued"
	LCConfirmed LCStatus = "confirmed"
	LCAmended   LCStatus = "amended"
	LCRealized  LCStatus = "realized"
	LCExpired   LCStatus = "expired"
	LCCancelled LCStatus = "cancelled"

	PayNet30 PaymentTerms = "net_30"
	PayNet60 PaymentTerms = "net_60"
	PayNet90 PaymentTerms = "net_90"
	PayCAD   PaymentTerms = "cash_against_documents"
	PayTT    PaymentTerms = "telegraphic_transfer"
)

type LetterOfCredit struct {
	ID              uuid.UUID    `json:"id"`
	LCNumber        string       `json:"lc_number"`
	LCType          LCType       `json:"lc_type"`
	ApplicantID     uuid.UUID    `json:"applicant_id"`
	ApplicantName   string       `json:"applicant_name"`
	BeneficiaryName string       `json:"beneficiary_name"`
	IssuingBank     string       `json:"issuing_bank"`
	AdvisingBank    string       `json:"advising_bank,omitempty"`
	Amount          float64      `json:"amount"`
	Currency        string       `json:"currency"`
	ExpiryDate      time.Time    `json:"expiry_date"`
	ShipmentPOL     string       `json:"shipment_pol"`
	ShipmentPOD     string       `json:"shipment_pod"`
	GoodsDescription string      `json:"goods_description"`
	DocumentsRequired []string   `json:"documents_required"`
	Status          LCStatus     `json:"status"`
	IssuedAt        *time.Time   `json:"issued_at,omitempty"`
	RealizedAt      *time.Time   `json:"realized_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type FinancingRequest struct {
	ID              uuid.UUID    `json:"id"`
	RefNo           string       `json:"reference_no"`
	ApplicantID     uuid.UUID    `json:"applicant_id"`
	BookingID       *uuid.UUID   `json:"booking_id,omitempty"`
	LCID            *uuid.UUID   `json:"lc_id,omitempty"`
	RequestedAmount float64      `json:"requested_amount"`
	Currency        string       `json:"currency"`
	PaymentTerms    PaymentTerms `json:"payment_terms"`
	InterestRatePct float64      `json:"interest_rate_pct"`
	InterestAmount  float64      `json:"interest_amount"`
	TotalRepayable  float64      `json:"total_repayable"`
	Status          string       `json:"status"`
	ApprovedAt      *time.Time   `json:"approved_at,omitempty"`
	DisbursedAt     *time.Time   `json:"disbursed_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type TradeFinanceAuditLog struct {
	ID         uuid.UUID `json:"id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateLCRequest struct {
	LCType          string   `json:"lc_type"`
	ApplicantName   string   `json:"applicant_name"`
	BeneficiaryName string   `json:"beneficiary_name"`
	IssuingBank     string   `json:"issuing_bank"`
	AdvisingBank    string   `json:"advising_bank,omitempty"`
	Amount          float64  `json:"amount"`
	Currency        string   `json:"currency"`
	ExpiryDate      string   `json:"expiry_date"`
	ShipmentPOL     string   `json:"shipment_pol"`
	ShipmentPOD     string   `json:"shipment_pod"`
	GoodsDescription string  `json:"goods_description"`
	DocumentsRequired []string `json:"documents_required"`
}

type UpdateLCStatusRequest struct {
	Status string `json:"status"`
}

type CreateFinancingRequest struct {
	BookingID       string  `json:"booking_id,omitempty"`
	LCID            string  `json:"lc_id,omitempty"`
	RequestedAmount float64 `json:"requested_amount"`
	Currency        string  `json:"currency"`
	PaymentTerms    string  `json:"payment_terms"`
	InterestRatePct float64 `json:"interest_rate_pct"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
