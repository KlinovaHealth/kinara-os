package models

import (
	"time"
	"github.com/google/uuid"
)

type ShipmentType string
type FreightStatus string
type BOLStatus string

const (
	ShipFCL ShipmentType = "fcl"
	ShipLCL ShipmentType = "lcl"

	FreightPending   FreightStatus = "pending"
	FreightBooked    FreightStatus = "booked"
	FreightLoaded    FreightStatus = "loaded"
	FreightInTransit FreightStatus = "in_transit"
	FreightDelivered FreightStatus = "delivered"
	FreightOnHold    FreightStatus = "on_hold"

	BOLDraft     BOLStatus = "draft"
	BOLIssued    BOLStatus = "issued"
	BOLSurrendered BOLStatus = "surrendered"
	BOLCancelled BOLStatus = "cancelled"
)

type FreightBooking struct {
	ID              uuid.UUID     `json:"id"`
	BookingRef      string        `json:"booking_ref"`
	ShipperID       uuid.UUID     `json:"shipper_id"`
	ShipperName     string        `json:"shipper_name"`
	ConsigneeName   string        `json:"consignee_name"`
	ShipmentType    ShipmentType  `json:"shipment_type"`
	PortOfLoading   uuid.UUID     `json:"port_of_loading"`
	PortOfDischarge uuid.UUID     `json:"port_of_discharge"`
	VesselID        *uuid.UUID    `json:"vessel_id,omitempty"`
	CommodityDesc   string        `json:"commodity_description"`
	ContainerCount  int           `json:"container_count"`
	WeightKg        float64       `json:"weight_kg"`
	FreightRate     float64       `json:"freight_rate_usd"`
	InsurancePct    float64       `json:"insurance_pct"`
	InsuranceAmount float64       `json:"insurance_amount"`
	DeclaredValue   float64       `json:"declared_value"`
	TotalFreight    float64       `json:"total_freight"`
	Currency        string        `json:"currency"`
	Status          FreightStatus `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type BillOfLading struct {
	ID              uuid.UUID `json:"id"`
	BOLNumber       string    `json:"bol_number"`
	BookingID       uuid.UUID `json:"booking_id"`
	VesselName      string    `json:"vessel_name"`
	VoyageNo        string    `json:"voyage_no"`
	ShipperName     string    `json:"shipper_name"`
	ConsigneeName   string    `json:"consignee_name"`
	NotifyParty     string    `json:"notify_party,omitempty"`
	POL             string    `json:"port_of_loading"`
	POD             string    `json:"port_of_discharge"`
	CommodityDesc   string    `json:"commodity_description"`
	ContainerCount  int       `json:"container_count"`
	GrossWeightKg   float64   `json:"gross_weight_kg"`
	FreightPrepaid  bool      `json:"freight_prepaid"`
	Status          BOLStatus `json:"status"`
	IssuedAt        *time.Time `json:"issued_at,omitempty"`
	SurrenderedAt   *time.Time `json:"surrendered_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DemurrageRecord struct {
	ID          uuid.UUID `json:"id"`
	BookingID   uuid.UUID `json:"booking_id"`
	ContainerNo string    `json:"container_no"`
	FreeDays    int       `json:"free_days"`
	UsedDays    int       `json:"used_days"`
	DailyRate   float64   `json:"daily_rate_usd"`
	TotalCharge float64   `json:"total_charge"`
	Currency    string    `json:"currency"`
	PortID      uuid.UUID `json:"port_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type ShippingAuditLog struct {
	ID         uuid.UUID `json:"id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateBookingRequest struct {
	ShipperName     string  `json:"shipper_name"`
	ConsigneeName   string  `json:"consignee_name"`
	ShipmentType    string  `json:"shipment_type"`
	PortOfLoading   string  `json:"port_of_loading"`
	PortOfDischarge string  `json:"port_of_discharge"`
	CommodityDesc   string  `json:"commodity_description"`
	ContainerCount  int     `json:"container_count"`
	WeightKg        float64 `json:"weight_kg"`
	FreightRate     float64 `json:"freight_rate_usd"`
	DeclaredValue   float64 `json:"declared_value"`
	InsurancePct    float64 `json:"insurance_pct"`
	Currency        string  `json:"currency"`
}

type IssueBOLRequest struct {
	VesselName    string  `json:"vessel_name"`
	VoyageNo      string  `json:"voyage_no"`
	NotifyParty   string  `json:"notify_party,omitempty"`
	POL           string  `json:"port_of_loading"`
	POD           string  `json:"port_of_discharge"`
	FreightPrepaid bool   `json:"freight_prepaid"`
}

type RecordDemurrageRequest struct {
	ContainerNo string  `json:"container_no"`
	FreeDays    int     `json:"free_days"`
	UsedDays    int     `json:"used_days"`
	DailyRate   float64 `json:"daily_rate_usd"`
	PortID      string  `json:"port_id"`
	Currency    string  `json:"currency"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
