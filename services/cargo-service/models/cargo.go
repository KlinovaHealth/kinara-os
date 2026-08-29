package models

import (
	"time"

	"github.com/google/uuid"
)

type CargoStatus string

const (
	CargoPending    CargoStatus = "pending"
	CargoPickedUp   CargoStatus = "picked_up"
	CargoInTransit  CargoStatus = "in_transit"
	CargoDelivered  CargoStatus = "delivered"
	CargoCancelled  CargoStatus = "cancelled"
	CargoFailed     CargoStatus = "failed"
)

type CargoType string

const (
	CargoGeneral      CargoType = "general"
	CargoPerishable   CargoType = "perishable"
	CargoFragile      CargoType = "fragile"
	CargoHazardous    CargoType = "hazardous"
	CargoLivestock    CargoType = "livestock"
	CargoBulkGrain    CargoType = "bulk_grain"
	CargoMedical      CargoType = "medical"
	CargoRefrigerated CargoType = "refrigerated"
)

type CargoBooking struct {
	ID                uuid.UUID   `json:"id"`
	BookingRef        string      `json:"booking_ref"`
	ShipperID         uuid.UUID   `json:"shipper_id"`
	ConsigneeID       *uuid.UUID  `json:"consignee_id,omitempty"`
	CargoType         CargoType   `json:"cargo_type"`
	Description       string      `json:"description"`
	WeightKg          float64     `json:"weight_kg"`
	VolumeM3          float64     `json:"volume_m3"`
	Status            CargoStatus `json:"status"`
	OriginAddress     string      `json:"origin_address"`
	OriginLat         float64     `json:"origin_lat"`
	OriginLng         float64     `json:"origin_lng"`
	DestinationAddress string     `json:"destination_address"`
	DestinationLat    float64     `json:"destination_lat"`
	DestinationLng    float64     `json:"destination_lng"`
	PickupAt          *time.Time  `json:"pickup_at,omitempty"`
	DeliveredAt       *time.Time  `json:"delivered_at,omitempty"`
	AssignedVehicleID *uuid.UUID  `json:"assigned_vehicle_id,omitempty"`
	AssignedDriverID  *uuid.UUID  `json:"assigned_driver_id,omitempty"`
	EstimatedDelivery *time.Time  `json:"estimated_delivery,omitempty"`
	FreightCost       float64     `json:"freight_cost"`
	Currency          string      `json:"currency"`
	Notes             string      `json:"notes,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

type TrackingEvent struct {
	ID        uuid.UUID `json:"id"`
	CargoID   uuid.UUID `json:"cargo_id"`
	Status    CargoStatus `json:"status"`
	Location  string    `json:"location"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Notes     string    `json:"notes,omitempty"`
	EventTime time.Time `json:"event_time"`
	CreatedAt time.Time `json:"created_at"`
}

type CargoAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateCargoRequest struct {
	CargoType          CargoType `json:"cargo_type"`
	Description        string    `json:"description"`
	WeightKg           float64   `json:"weight_kg"`
	VolumeM3           float64   `json:"volume_m3"`
	OriginAddress      string    `json:"origin_address"`
	OriginLat          float64   `json:"origin_lat"`
	OriginLng          float64   `json:"origin_lng"`
	DestinationAddress string    `json:"destination_address"`
	DestinationLat     float64   `json:"destination_lat"`
	DestinationLng     float64   `json:"destination_lng"`
	EstimatedDelivery  *string   `json:"estimated_delivery,omitempty"`
	FreightCost        float64   `json:"freight_cost"`
	Currency           string    `json:"currency"`
	Notes              string    `json:"notes,omitempty"`
}

type AssignCargoRequest struct {
	VehicleID string `json:"vehicle_id"`
	DriverID  string `json:"driver_id"`
}

type TrackEventRequest struct {
	Status    CargoStatus `json:"status"`
	Location  string      `json:"location"`
	Latitude  float64     `json:"latitude"`
	Longitude float64     `json:"longitude"`
	Notes     string      `json:"notes,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *PageMeta   `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PageMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
