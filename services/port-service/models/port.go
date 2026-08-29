package models

import (
	"time"
	"github.com/google/uuid"
)

type BerthStatus string
type VesselStatus string
type PortAlertLevel string

const (
	BerthAvailable    BerthStatus = "available"
	BerthOccupied     BerthStatus = "occupied"
	BerthMaintenance  BerthStatus = "maintenance"
	BerthReserved     BerthStatus = "reserved"

	VesselExpected  VesselStatus = "expected"
	VesselArrived   VesselStatus = "arrived"
	VesselDeparted  VesselStatus = "departed"
	VesselDelayed   VesselStatus = "delayed"

	AlertNormal   PortAlertLevel = "normal"
	AlertModerate PortAlertLevel = "moderate"
	AlertHigh     PortAlertLevel = "high"
	AlertCritical PortAlertLevel = "critical"
)

type Port struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	MaxDraft    float64   `json:"max_draft_m"`
	TotalBerths int       `json:"total_berths"`
	AlertLevel  PortAlertLevel `json:"alert_level"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Berth struct {
	ID          uuid.UUID   `json:"id"`
	PortID      uuid.UUID   `json:"port_id"`
	BerthNumber string      `json:"berth_number"`
	Status      BerthStatus `json:"status"`
	MaxLengthM  float64     `json:"max_length_m"`
	MaxDraftM   float64     `json:"max_draft_m"`
	MaxTonnage  float64     `json:"max_tonnage_t"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type BerthSchedule struct {
	ID          uuid.UUID    `json:"id"`
	BerthID     uuid.UUID    `json:"berth_id"`
	VesselID    uuid.UUID    `json:"vessel_id"`
	VesselName  string       `json:"vessel_name"`
	Status      VesselStatus `json:"status"`
	ETA         time.Time    `json:"eta"`
	ETD         time.Time    `json:"etd"`
	ActualArrival  *time.Time `json:"actual_arrival,omitempty"`
	ActualDeparture *time.Time `json:"actual_departure,omitempty"`
	CargoType   string       `json:"cargo_type"`
	TonnageT    float64      `json:"tonnage_t"`
	Notes       string       `json:"notes,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type CongestionAlert struct {
	ID         uuid.UUID      `json:"id"`
	PortID     uuid.UUID      `json:"port_id"`
	AlertLevel PortAlertLevel `json:"alert_level"`
	Message    string         `json:"message"`
	OccupiedBerths int        `json:"occupied_berths"`
	TotalBerths    int        `json:"total_berths"`
	OccupancyPct   float64    `json:"occupancy_pct"`
	ResolvedAt *time.Time     `json:"resolved_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type PortAuditLog struct {
	ID         uuid.UUID `json:"id"`
	PortID     uuid.UUID `json:"port_id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreatePortRequest struct {
	Name        string  `json:"name"`
	Country     string  `json:"country"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	MaxDraft    float64 `json:"max_draft_m"`
	TotalBerths int     `json:"total_berths"`
}

type CreateBerthRequest struct {
	BerthNumber string  `json:"berth_number"`
	MaxLengthM  float64 `json:"max_length_m"`
	MaxDraftM   float64 `json:"max_draft_m"`
	MaxTonnage  float64 `json:"max_tonnage_t"`
}

type ScheduleBerthRequest struct {
	VesselID   string  `json:"vessel_id"`
	VesselName string  `json:"vessel_name"`
	ETA        string  `json:"eta"`
	ETD        string  `json:"etd"`
	CargoType  string  `json:"cargo_type"`
	TonnageT   float64 `json:"tonnage_t"`
	Notes      string  `json:"notes,omitempty"`
}

type UpdateScheduleStatusRequest struct {
	Status string `json:"status"`
}

type CreateAlertRequest struct {
	AlertLevel string `json:"alert_level"`
	Message    string `json:"message"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
