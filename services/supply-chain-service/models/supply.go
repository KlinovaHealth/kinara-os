package models

import (
	"time"

	"github.com/google/uuid"
)

type ShipmentStatus string

const (
	ShipmentPending   ShipmentStatus = "pending"
	ShipmentPickedUp  ShipmentStatus = "picked_up"
	ShipmentInTransit ShipmentStatus = "in_transit"
	ShipmentAtWarehouse ShipmentStatus = "at_warehouse"
	ShipmentDelivered ShipmentStatus = "delivered"
	ShipmentCancelled ShipmentStatus = "cancelled"
)

type PillarHandoff string

const (
	HandoffAgriToLogistics  PillarHandoff = "agri_to_logistics"
	HandoffLogisticsToPort  PillarHandoff = "logistics_to_port"
	HandoffPortToMarket     PillarHandoff = "port_to_market"
)

type Shipment struct {
	ID              uuid.UUID      `json:"id"`
	ShipmentRef     string         `json:"shipment_ref"`
	FarmerID        uuid.UUID      `json:"farmer_id"`
	CooperativeID   *uuid.UUID     `json:"cooperative_id,omitempty"`
	CommodityName   string         `json:"commodity_name"`
	QuantityKg      float64        `json:"quantity_kg"`
	OriginLocation  string         `json:"origin_location"`
	DestLocation    string         `json:"destination_location"`
	BuyerID         *uuid.UUID     `json:"buyer_id,omitempty"`
	Status          ShipmentStatus `json:"status"`
	PillarHandoff   PillarHandoff  `json:"pillar_handoff"`
	EstimatedCostUSD float64       `json:"estimated_cost_usd"`
	ActualCostUSD   *float64       `json:"actual_cost_usd,omitempty"`
	PickedUpAt      *time.Time     `json:"picked_up_at,omitempty"`
	DeliveredAt     *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type TrackingEvent struct {
	ID          uuid.UUID `json:"id"`
	ShipmentID  uuid.UUID `json:"shipment_id"`
	Status      ShipmentStatus `json:"status"`
	Location    string    `json:"location"`
	Note        string    `json:"note"`
	RecordedAt  time.Time `json:"recorded_at"`
}

type CostEstimate struct {
	OriginLocation string  `json:"origin_location"`
	DestLocation   string  `json:"destination_location"`
	QuantityKg     float64 `json:"quantity_kg"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	CostPerKgUSD   float64 `json:"cost_per_kg_usd"`
	DistanceKm     float64 `json:"estimated_distance_km"`
}

type SupplyAuditLog struct {
	ID         uuid.UUID `json:"id"`
	ShipmentID uuid.UUID `json:"shipment_id"`
	ActorID    uuid.UUID `json:"actor_id"`
	Action     string    `json:"action"`
	Detail     string    `json:"detail"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateShipmentRequest struct {
	FarmerID        string  `json:"farmer_id"`
	CooperativeID   string  `json:"cooperative_id"`
	CommodityName   string  `json:"commodity_name"`
	QuantityKg      float64 `json:"quantity_kg"`
	OriginLocation  string  `json:"origin_location"`
	DestLocation    string  `json:"destination_location"`
	BuyerID         string  `json:"buyer_id"`
}

type UpdateStatusRequest struct {
	Status   string  `json:"status"`
	Location string  `json:"location"`
	Note     string  `json:"note"`
	ActualCostUSD *float64 `json:"actual_cost_usd"`
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
