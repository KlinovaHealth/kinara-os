package models

import (
	"time"
	"github.com/google/uuid"
)

type OperationType string
type EquipmentStatus string
type EquipmentType string

const (
	OpLoading   OperationType = "loading"
	OpUnloading OperationType = "unloading"
	OpTransfer  OperationType = "transfer"
	OpInspect   OperationType = "inspection"

	EquipAvailable  EquipmentStatus = "available"
	EquipInUse      EquipmentStatus = "in_use"
	EquipMaintain   EquipmentStatus = "maintenance"
	EquipOOS        EquipmentStatus = "out_of_service"

	EquipCrane      EquipmentType = "crane"
	EquipForklift   EquipmentType = "forklift"
	EquipReachStack EquipmentType = "reach_stacker"
	EquipTractor    EquipmentType = "tractor"
	EquipConveyor   EquipmentType = "conveyor"
)

type DockOperation struct {
	ID              uuid.UUID     `json:"id"`
	PortID          uuid.UUID     `json:"port_id"`
	BerthID         uuid.UUID     `json:"berth_id"`
	VesselID        uuid.UUID     `json:"vessel_id"`
	OperationType   OperationType `json:"operation_type"`
	CargoType       string        `json:"cargo_type"`
	TonnageT        float64       `json:"tonnage_t"`
	UnitCount       int           `json:"unit_count"`
	StevedoreTeam   string        `json:"stevedore_team,omitempty"`
	StartedAt       *time.Time    `json:"started_at,omitempty"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	PlannedDuration float64       `json:"planned_duration_hrs"`
	ActualDuration  float64       `json:"actual_duration_hrs,omitempty"`
	SafetyIncident  bool          `json:"safety_incident"`
	IncidentDetails string        `json:"incident_details,omitempty"`
	BillingAmount   float64       `json:"billing_amount"`
	Currency        string        `json:"currency"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type Equipment struct {
	ID            uuid.UUID       `json:"id"`
	PortID        uuid.UUID       `json:"port_id"`
	EquipmentCode string          `json:"equipment_code"`
	EquipmentType EquipmentType   `json:"equipment_type"`
	Model         string          `json:"model"`
	Status        EquipmentStatus `json:"status"`
	CapacityT     float64         `json:"capacity_t"`
	LastServiceAt *time.Time      `json:"last_service_at,omitempty"`
	NextServiceAt *time.Time      `json:"next_service_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type SafetyEvent struct {
	ID              uuid.UUID `json:"id"`
	OperationID     uuid.UUID `json:"operation_id"`
	PortID          uuid.UUID `json:"port_id"`
	EventType       string    `json:"event_type"`
	Severity        string    `json:"severity"`
	Description     string    `json:"description"`
	Injured         int       `json:"injured_count"`
	ReportedBy      string    `json:"reported_by"`
	CreatedAt       time.Time `json:"created_at"`
}

type DockAuditLog struct {
	ID          uuid.UUID `json:"id"`
	PortID      uuid.UUID `json:"port_id"`
	ActorID     string    `json:"actor_id"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entity_type"`
	EntityID    uuid.UUID `json:"entity_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateOperationRequest struct {
	PortID          string        `json:"port_id"`
	BerthID         string        `json:"berth_id"`
	VesselID        string        `json:"vessel_id"`
	OperationType   string        `json:"operation_type"`
	CargoType       string        `json:"cargo_type"`
	TonnageT        float64       `json:"tonnage_t"`
	UnitCount       int           `json:"unit_count"`
	StevedoreTeam   string        `json:"stevedore_team,omitempty"`
	PlannedDuration float64       `json:"planned_duration_hrs"`
	BillingAmount   float64       `json:"billing_amount"`
	Currency        string        `json:"currency"`
}

type StartOperationRequest struct {
	StartedAt string `json:"started_at,omitempty"`
}

type CompleteOperationRequest struct {
	CompletedAt     string  `json:"completed_at,omitempty"`
	ActualDuration  float64 `json:"actual_duration_hrs"`
	SafetyIncident  bool    `json:"safety_incident"`
	IncidentDetails string  `json:"incident_details,omitempty"`
}

type CreateEquipmentRequest struct {
	EquipmentType string  `json:"equipment_type"`
	Model         string  `json:"model"`
	CapacityT     float64 `json:"capacity_t"`
}

type ReportSafetyEventRequest struct {
	EventType   string `json:"event_type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Injured     int    `json:"injured_count"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
