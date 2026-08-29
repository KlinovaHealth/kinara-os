package models

import (
	"time"
	"github.com/google/uuid"
)

type VesselType string
type VesselFlag string
type VesselCondition string
type MaintenanceType string

const (
	VesselContainerShip VesselType = "container_ship"
	VesselBulkCarrier   VesselType = "bulk_carrier"
	VesselTanker        VesselType = "tanker"
	VesselRoRo          VesselType = "ro_ro"
	VesselGeneral       VesselType = "general_cargo"
	VesselFerry         VesselType = "ferry"

	ConditionExcellent VesselCondition = "excellent"
	ConditionGood      VesselCondition = "good"
	ConditionFair      VesselCondition = "fair"
	ConditionPoor      VesselCondition = "poor"

	MaintScheduled    MaintenanceType = "scheduled"
	MaintEmergency    MaintenanceType = "emergency"
	MaintDryDock      MaintenanceType = "dry_dock"
	MaintInspection   MaintenanceType = "inspection"
)

type Vessel struct {
	ID             uuid.UUID       `json:"id"`
	IMONumber      string          `json:"imo_number"`
	Name           string          `json:"name"`
	VesselType     VesselType      `json:"vessel_type"`
	Flag           string          `json:"flag"`
	Owner          string          `json:"owner"`
	OperatorID     uuid.UUID       `json:"operator_id"`
	YearBuilt      int             `json:"year_built"`
	GrossTonnage   float64         `json:"gross_tonnage_t"`
	DeadweightT    float64         `json:"deadweight_t"`
	LengthM        float64         `json:"length_m"`
	BeamM          float64         `json:"beam_m"`
	MaxDraftM      float64         `json:"max_draft_m"`
	MaxSpeed       float64         `json:"max_speed_knots"`
	Condition      VesselCondition `json:"condition"`
	CurrentPortID  *uuid.UUID      `json:"current_port_id,omitempty"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type VoyageRecord struct {
	ID          uuid.UUID `json:"id"`
	VesselID    uuid.UUID `json:"vessel_id"`
	VoyageCode  string    `json:"voyage_code"`
	DeparturePortID uuid.UUID `json:"departure_port_id"`
	ArrivalPortID   uuid.UUID `json:"arrival_port_id"`
	DepartedAt  *time.Time `json:"departed_at,omitempty"`
	ArrivedAt   *time.Time `json:"arrived_at,omitempty"`
	DistanceNM  float64    `json:"distance_nm"`
	CargoTonnage float64   `json:"cargo_tonnage_t"`
	Notes       string     `json:"notes,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type MaintenanceRecord struct {
	ID              uuid.UUID       `json:"id"`
	VesselID        uuid.UUID       `json:"vessel_id"`
	MaintenanceType MaintenanceType `json:"maintenance_type"`
	Description     string          `json:"description"`
	StartDate       time.Time       `json:"start_date"`
	EndDate         *time.Time      `json:"end_date,omitempty"`
	Cost            float64         `json:"cost"`
	Currency        string          `json:"currency"`
	Vendor          string          `json:"vendor,omitempty"`
	Completed       bool            `json:"completed"`
	CreatedAt       time.Time       `json:"created_at"`
}

type VesselAuditLog struct {
	ID        uuid.UUID `json:"id"`
	VesselID  uuid.UUID `json:"vessel_id"`
	ActorID   string    `json:"actor_id"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterVesselRequest struct {
	IMONumber    string  `json:"imo_number"`
	Name         string  `json:"name"`
	VesselType   string  `json:"vessel_type"`
	Flag         string  `json:"flag"`
	Owner        string  `json:"owner"`
	YearBuilt    int     `json:"year_built"`
	GrossTonnage float64 `json:"gross_tonnage_t"`
	DeadweightT  float64 `json:"deadweight_t"`
	LengthM      float64 `json:"length_m"`
	BeamM        float64 `json:"beam_m"`
	MaxDraftM    float64 `json:"max_draft_m"`
	MaxSpeed     float64 `json:"max_speed_knots"`
}

type LogVoyageRequest struct {
	DeparturePortID string  `json:"departure_port_id"`
	ArrivalPortID   string  `json:"arrival_port_id"`
	DepartedAt      string  `json:"departed_at,omitempty"`
	ArrivedAt       string  `json:"arrived_at,omitempty"`
	DistanceNM      float64 `json:"distance_nm"`
	CargoTonnage    float64 `json:"cargo_tonnage_t"`
	Notes           string  `json:"notes,omitempty"`
}

type LogMaintenanceRequest struct {
	MaintenanceType string  `json:"maintenance_type"`
	Description     string  `json:"description"`
	StartDate       string  `json:"start_date"`
	Cost            float64 `json:"cost"`
	Currency        string  `json:"currency"`
	Vendor          string  `json:"vendor,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
