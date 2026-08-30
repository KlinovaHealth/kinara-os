package models

import (
	"time"

	"github.com/google/uuid"
)

type Vehicle struct {
	ID           uuid.UUID  `json:"id"`
	VehicleRef   string     `json:"vehicle_ref"`
	FleetID      string     `json:"fleet_id"`
	VehicleType  string     `json:"vehicle_type"` // "truck", "bike", "van", "motorcycle"
	Capacity     float64    `json:"capacity_kg"`
	DriverName   string     `json:"driver_name,omitempty"`
	DriverID     *uuid.UUID `json:"driver_id,omitempty"`
	Status       string     `json:"status"` // "active", "idle", "maintenance", "offline"
	TenantID     string     `json:"tenant_id"`
	RegisteredAt time.Time  `json:"registered_at"`
}

type GPSLocation struct {
	ID         uuid.UUID `json:"id"`
	VehicleID  uuid.UUID `json:"vehicle_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	SpeedKmh   float64   `json:"speed_kmh"`
	HeadingDeg float64   `json:"heading_deg"`
	PingedAt   time.Time `json:"pinged_at"`
}

type VehicleRoute struct {
	ID          uuid.UUID  `json:"id"`
	VehicleID   uuid.UUID  `json:"vehicle_id"`
	OriginLat   float64    `json:"origin_lat"`
	OriginLng   float64    `json:"origin_lng"`
	DestLat     float64    `json:"dest_lat"`
	DestLng     float64    `json:"dest_lng"`
	Description string     `json:"description"`
	Active      bool       `json:"active"`
	AssignedAt  time.Time  `json:"assigned_at"`
	ETA         *time.Time `json:"eta,omitempty"`
}

type FleetVehicleStatus struct {
	VehicleID   uuid.UUID  `json:"vehicle_id"`
	VehicleRef  string     `json:"vehicle_ref"`
	VehicleType string     `json:"vehicle_type"`
	Status      string     `json:"status"`
	DriverName  string     `json:"driver_name,omitempty"`
	LastPing    *time.Time `json:"last_ping,omitempty"`
	Latitude    *float64   `json:"latitude,omitempty"`
	Longitude   *float64   `json:"longitude,omitempty"`
}

type VehicleAlert struct {
	ID        uuid.UUID `json:"id"`
	VehicleID uuid.UUID `json:"vehicle_id"`
	AlertType string    `json:"alert_type"` // "off_route", "speeding", "no_signal", "manual"
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type ETAResponse struct {
	VehicleID        uuid.UUID `json:"vehicle_id"`
	CurrentLat       float64   `json:"current_lat"`
	CurrentLng       float64   `json:"current_lng"`
	DestinationLat   float64   `json:"destination_lat"`
	DestinationLng   float64   `json:"destination_lng"`
	DistanceKm       float64   `json:"distance_km"`
	EstimatedMinutes int       `json:"estimated_minutes"`
	ETAUTC           time.Time `json:"eta_utc"`
}

type PingRequest struct {
	VehicleID  uuid.UUID `json:"vehicle_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	SpeedKmh   float64   `json:"speed_kmh"`
	HeadingDeg float64   `json:"heading_deg"`
}

type ETARequest struct {
	DestinationLat float64 `json:"destination_lat"`
	DestinationLng float64 `json:"destination_lng"`
}

type AlertRequest struct {
	AlertType string `json:"alert_type"`
	Message   string `json:"message"`
}
