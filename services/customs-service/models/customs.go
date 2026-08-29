package models

import (
	"time"
	"github.com/google/uuid"
)

type ClearanceStatus string
type TariffCategory string

const (
	ClearancePending    ClearanceStatus = "pending"
	ClearanceUnderReview ClearanceStatus = "under_review"
	ClearanceApproved   ClearanceStatus = "approved"
	ClearanceRejected   ClearanceStatus = "rejected"
	ClearanceOnHold     ClearanceStatus = "on_hold"

	TariffAg    TariffCategory = "agricultural"
	TariffManuf TariffCategory = "manufactured"
	TariffChem  TariffCategory = "chemical"
	TariffEnergy TariffCategory = "energy"
	TariffLux   TariffCategory = "luxury"
	TariffExempt TariffCategory = "exempt"
)

type TariffCode struct {
	ID          uuid.UUID      `json:"id"`
	HSCode      string         `json:"hs_code"`
	Description string         `json:"description"`
	Category    TariffCategory `json:"category"`
	DutyRate    float64        `json:"duty_rate_pct"`
	VATRate     float64        `json:"vat_rate_pct"`
	Country     string         `json:"country"`
	IsRestricted bool          `json:"is_restricted"`
	Notes       string         `json:"notes,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type ClearanceRequest struct {
	ID              uuid.UUID       `json:"id"`
	ReferenceNo     string          `json:"reference_no"`
	ImporterName    string          `json:"importer_name"`
	ImporterID      string          `json:"importer_id"`
	ManifestID      uuid.UUID       `json:"manifest_id"`
	VesselID        uuid.UUID       `json:"vessel_id"`
	PortID          uuid.UUID       `json:"port_id"`
	HSCode          string          `json:"hs_code"`
	GoodsDescription string         `json:"goods_description"`
	DeclaredValue   float64         `json:"declared_value"`
	Currency        string          `json:"currency"`
	WeightKg        float64         `json:"weight_kg"`
	DutyAmount      float64         `json:"duty_amount"`
	VATAmount       float64         `json:"vat_amount"`
	TotalDue        float64         `json:"total_due"`
	Status          ClearanceStatus `json:"status"`
	ReviewedBy      string          `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time      `json:"reviewed_at,omitempty"`
	RejectionReason string          `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CustomsAuditLog struct {
	ID         uuid.UUID `json:"id"`
	PortID     uuid.UUID `json:"port_id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateTariffRequest struct {
	HSCode      string  `json:"hs_code"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	DutyRate    float64 `json:"duty_rate_pct"`
	VATRate     float64 `json:"vat_rate_pct"`
	Country     string  `json:"country"`
	IsRestricted bool   `json:"is_restricted"`
	Notes       string  `json:"notes,omitempty"`
}

type LookupTariffRequest struct {
	HSCode  string `json:"hs_code"`
	Country string `json:"country"`
}

type CreateClearanceRequest struct {
	ImporterName    string  `json:"importer_name"`
	ImporterID      string  `json:"importer_id"`
	ManifestID      string  `json:"manifest_id"`
	VesselID        string  `json:"vessel_id"`
	PortID          string  `json:"port_id"`
	HSCode          string  `json:"hs_code"`
	GoodsDescription string `json:"goods_description"`
	DeclaredValue   float64 `json:"declared_value"`
	Currency        string  `json:"currency"`
	WeightKg        float64 `json:"weight_kg"`
}

type UpdateClearanceStatusRequest struct {
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
