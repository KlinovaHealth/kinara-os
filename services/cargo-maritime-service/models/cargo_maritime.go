package models

import (
	"time"
	"github.com/google/uuid"
)

type ContainerType string
type ContainerStatus string
type DamageLevel string

const (
	Container20Dry  ContainerType = "20ft_dry"
	Container40Dry  ContainerType = "40ft_dry"
	Container40HC   ContainerType = "40ft_hc"
	Container20Reefer ContainerType = "20ft_reefer"
	Container40Reefer ContainerType = "40ft_reefer"
	ContainerTank   ContainerType = "tank"
	ContainerFlat   ContainerType = "flat_rack"

	StatusEmpty     ContainerStatus = "empty"
	StatusLoaded    ContainerStatus = "loaded"
	StatusInTransit ContainerStatus = "in_transit"
	StatusAtPort    ContainerStatus = "at_port"
	StatusDelivered ContainerStatus = "delivered"
	StatusOnHold    ContainerStatus = "on_hold"

	DamageNone     DamageLevel = "none"
	DamageMinor    DamageLevel = "minor"
	DamageMajor    DamageLevel = "major"
	DamageTotalLoss DamageLevel = "total_loss"
)

type Container struct {
	ID            uuid.UUID       `json:"id"`
	ContainerNo   string          `json:"container_no"`
	ContainerType ContainerType   `json:"container_type"`
	OwnerID       uuid.UUID       `json:"owner_id"`
	Status        ContainerStatus `json:"status"`
	CurrentPortID *uuid.UUID      `json:"current_port_id,omitempty"`
	VesselID      *uuid.UUID      `json:"vessel_id,omitempty"`
	WeightKg      float64         `json:"weight_kg"`
	TareWeightKg  float64         `json:"tare_weight_kg"`
	PayloadKg     float64         `json:"payload_kg"`
	SealNo        string          `json:"seal_no,omitempty"`
	Temperature   *float64        `json:"temperature_c,omitempty"`
	IsHazmat      bool            `json:"is_hazmat"`
	HazmatClass   string          `json:"hazmat_class,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type CargoManifest struct {
	ID            uuid.UUID `json:"id"`
	ManifestNo    string    `json:"manifest_no"`
	VoyageID      uuid.UUID `json:"voyage_id"`
	VesselID      uuid.UUID `json:"vessel_id"`
	PortOfLoading uuid.UUID `json:"port_of_loading"`
	PortOfDischarge uuid.UUID `json:"port_of_discharge"`
	ShipperName   string    `json:"shipper_name"`
	ConsigneeName string    `json:"consignee_name"`
	TotalContainers int     `json:"total_containers"`
	TotalWeightKg float64   `json:"total_weight_kg"`
	Commodity     string    `json:"commodity"`
	IsFinalized   bool      `json:"is_finalized"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ManifestContainer struct {
	ID           uuid.UUID `json:"id"`
	ManifestID   uuid.UUID `json:"manifest_id"`
	ContainerID  uuid.UUID `json:"container_id"`
	ContainerNo  string    `json:"container_no"`
	AddedAt      time.Time `json:"added_at"`
}

type DamageReport struct {
	ID           uuid.UUID   `json:"id"`
	ContainerID  uuid.UUID   `json:"container_id"`
	ContainerNo  string      `json:"container_no"`
	DamageLevel  DamageLevel `json:"damage_level"`
	Description  string      `json:"description"`
	PhotoURL     string      `json:"photo_url,omitempty"`
	ReportedBy   string      `json:"reported_by"`
	EstimatedCost float64    `json:"estimated_cost"`
	Currency     string      `json:"currency"`
	PortID       uuid.UUID   `json:"port_id"`
	CreatedAt    time.Time   `json:"created_at"`
}

type CargoMaritimeAuditLog struct {
	ID         uuid.UUID `json:"id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type RegisterContainerRequest struct {
	ContainerNo   string  `json:"container_no"`
	ContainerType string  `json:"container_type"`
	TareWeightKg  float64 `json:"tare_weight_kg"`
	IsHazmat      bool    `json:"is_hazmat"`
	HazmatClass   string  `json:"hazmat_class,omitempty"`
}

type UpdateContainerStatusRequest struct {
	Status    string `json:"status"`
	SealNo    string `json:"seal_no,omitempty"`
	PortID    string `json:"port_id,omitempty"`
	VesselID  string `json:"vessel_id,omitempty"`
}

type CreateManifestRequest struct {
	VoyageID      string  `json:"voyage_id"`
	VesselID      string  `json:"vessel_id"`
	PortOfLoading string  `json:"port_of_loading"`
	PortOfDischarge string `json:"port_of_discharge"`
	ShipperName   string  `json:"shipper_name"`
	ConsigneeName string  `json:"consignee_name"`
	Commodity     string  `json:"commodity"`
}

type AddContainerToManifestRequest struct {
	ContainerID string `json:"container_id"`
}

type ReportDamageRequest struct {
	DamageLevel   string  `json:"damage_level"`
	Description   string  `json:"description"`
	PhotoURL      string  `json:"photo_url,omitempty"`
	EstimatedCost float64 `json:"estimated_cost"`
	Currency      string  `json:"currency"`
	PortID        string  `json:"port_id"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
