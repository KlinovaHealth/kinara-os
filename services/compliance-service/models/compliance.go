package models

import (
	"time"

	"github.com/google/uuid"
)

type PermitStatus string

const (
	PermitActive    PermitStatus = "active"
	PermitExpired   PermitStatus = "expired"
	PermitRevoked   PermitStatus = "revoked"
	PermitPending   PermitStatus = "pending"
)

type PermitType string

const (
	PermitTransit      PermitType = "transit"
	PermitBorderCross  PermitType = "border_crossing"
	PermitOversize     PermitType = "oversize"
	PermitHazmat       PermitType = "hazmat"
	PermitColdChain    PermitType = "cold_chain"
)

type TransitPermit struct {
	ID              uuid.UUID    `json:"id"`
	PermitNo        string       `json:"permit_no"`
	VehicleID       uuid.UUID    `json:"vehicle_id"`
	DriverID        *uuid.UUID   `json:"driver_id,omitempty"`
	PermitType      PermitType   `json:"permit_type"`
	Status          PermitStatus `json:"status"`
	IssuedBy        string       `json:"issued_by"`
	Country         string       `json:"country"`
	RouteRestriction string      `json:"route_restriction,omitempty"`
	MaxWeightKg     float64      `json:"max_weight_kg"`
	ValidFrom       time.Time    `json:"valid_from"`
	ValidUntil      time.Time    `json:"valid_until"`
	Notes           string       `json:"notes,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type BorderCrossing struct {
	ID          uuid.UUID `json:"id"`
	VehicleID   uuid.UUID `json:"vehicle_id"`
	DriverID    uuid.UUID `json:"driver_id"`
	FromCountry string    `json:"from_country"`
	ToCountry   string    `json:"to_country"`
	BorderPost  string    `json:"border_post"`
	CargoDesc   string    `json:"cargo_description"`
	GrossWeightKg float64 `json:"gross_weight_kg"`
	CrossedAt   time.Time `json:"crossed_at"`
	ExitPermitNo string   `json:"exit_permit_no"`
	EntryPermitNo string  `json:"entry_permit_no"`
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type WeightCheck struct {
	ID          uuid.UUID `json:"id"`
	VehicleID   uuid.UUID `json:"vehicle_id"`
	Country     string    `json:"country"`
	CheckStation string   `json:"check_station"`
	GrossWeightKg float64 `json:"gross_weight_kg"`
	LegalLimitKg  float64 `json:"legal_limit_kg"`
	IsCompliant   bool    `json:"is_compliant"`
	FineAmount    float64 `json:"fine_amount"`
	Currency      string  `json:"currency"`
	CheckedAt     time.Time `json:"checked_at"`
	Notes         string  `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type ComplianceAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreatePermitRequest struct {
	VehicleID        string     `json:"vehicle_id"`
	DriverID         string     `json:"driver_id,omitempty"`
	PermitType       PermitType `json:"permit_type"`
	IssuedBy         string     `json:"issued_by"`
	Country          string     `json:"country"`
	RouteRestriction string     `json:"route_restriction,omitempty"`
	MaxWeightKg      float64    `json:"max_weight_kg"`
	ValidFrom        string     `json:"valid_from"`
	ValidUntil       string     `json:"valid_until"`
	Notes            string     `json:"notes,omitempty"`
}

type CreateBorderCrossingRequest struct {
	VehicleID     string  `json:"vehicle_id"`
	DriverID      string  `json:"driver_id"`
	FromCountry   string  `json:"from_country"`
	ToCountry     string  `json:"to_country"`
	BorderPost    string  `json:"border_post"`
	CargoDesc     string  `json:"cargo_description"`
	GrossWeightKg float64 `json:"gross_weight_kg"`
	ExitPermitNo  string  `json:"exit_permit_no"`
	EntryPermitNo string  `json:"entry_permit_no"`
	Notes         string  `json:"notes,omitempty"`
}

type CreateWeightCheckRequest struct {
	VehicleID     string  `json:"vehicle_id"`
	Country       string  `json:"country"`
	CheckStation  string  `json:"check_station"`
	GrossWeightKg float64 `json:"gross_weight_kg"`
	LegalLimitKg  float64 `json:"legal_limit_kg"`
	FineAmount    float64 `json:"fine_amount"`
	Currency      string  `json:"currency"`
	Notes         string  `json:"notes,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
