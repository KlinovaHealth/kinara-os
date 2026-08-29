package models

import (
	"time"

	"github.com/google/uuid"
)

type ShipmentStatus string

const (
	ShipmentCreated        ShipmentStatus = "created"
	ShipmentPicked         ShipmentStatus = "picked"
	ShipmentInTransit      ShipmentStatus = "in_transit"
	ShipmentOutForDelivery ShipmentStatus = "out_for_delivery"
	ShipmentDelivered      ShipmentStatus = "delivered"
	ShipmentReturned       ShipmentStatus = "returned"
	ShipmentCancelled      ShipmentStatus = "cancelled"
)

type ServiceLevel string

const (
	ServiceStandard  ServiceLevel = "standard"
	ServiceExpress   ServiceLevel = "express"
	ServiceOvernight ServiceLevel = "overnight"
	ServiceEconomy   ServiceLevel = "economy"
)

type Shipment struct {
	ID              uuid.UUID      `json:"id"`
	TrackingCode    string         `json:"tracking_code"`
	SenderID        uuid.UUID      `json:"sender_id"`
	RecipientName   string         `json:"recipient_name"`
	RecipientPhone  string         `json:"recipient_phone"`
	OriginAddress   string         `json:"origin_address"`
	OriginCountry   string         `json:"origin_country"`
	DestAddress     string         `json:"destination_address"`
	DestCountry     string         `json:"destination_country"`
	WeightKg        float64        `json:"weight_kg"`
	LengthCm        float64        `json:"length_cm"`
	WidthCm         float64        `json:"width_cm"`
	HeightCm        float64        `json:"height_cm"`
	DeclaredValue   float64        `json:"declared_value"`
	Currency        string         `json:"currency"`
	ServiceLevel    ServiceLevel   `json:"service_level"`
	Status          ShipmentStatus `json:"status"`
	FreightCharge   float64        `json:"freight_charge"`
	InsuranceCharge float64        `json:"insurance_charge"`
	TotalCharge     float64        `json:"total_charge"`
	PickedAt        *time.Time     `json:"picked_at,omitempty"`
	DeliveredAt     *time.Time     `json:"delivered_at,omitempty"`
	EstDelivery     *time.Time     `json:"est_delivery,omitempty"`
	Notes           string         `json:"notes,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type ShipmentEvent struct {
	ID         uuid.UUID      `json:"id"`
	ShipmentID uuid.UUID      `json:"shipment_id"`
	Status     ShipmentStatus `json:"status"`
	Location   string         `json:"location"`
	Notes      string         `json:"notes,omitempty"`
	EventTime  time.Time      `json:"event_time"`
	CreatedAt  time.Time      `json:"created_at"`
}

type ShipmentAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateShipmentRequest struct {
	RecipientName  string       `json:"recipient_name"`
	RecipientPhone string       `json:"recipient_phone"`
	OriginAddress  string       `json:"origin_address"`
	OriginCountry  string       `json:"origin_country"`
	DestAddress    string       `json:"destination_address"`
	DestCountry    string       `json:"destination_country"`
	WeightKg       float64      `json:"weight_kg"`
	LengthCm       float64      `json:"length_cm"`
	WidthCm        float64      `json:"width_cm"`
	HeightCm       float64      `json:"height_cm"`
	DeclaredValue  float64      `json:"declared_value"`
	Currency       string       `json:"currency"`
	ServiceLevel   ServiceLevel `json:"service_level"`
	Notes          string       `json:"notes,omitempty"`
}

type AddShipmentEventRequest struct {
	Status   ShipmentStatus `json:"status"`
	Location string         `json:"location"`
	Notes    string         `json:"notes,omitempty"`
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
