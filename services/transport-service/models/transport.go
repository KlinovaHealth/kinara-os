package models

import (
	"time"

	"github.com/google/uuid"
)

type TripStatus string

const (
	TripScheduled TripStatus = "scheduled"
	TripEnRoute   TripStatus = "en_route"
	TripDelivered TripStatus = "delivered"
	TripDelayed   TripStatus = "delayed"
	TripCancelled TripStatus = "cancelled"
)

type TransportTrip struct {
	ID                 uuid.UUID  `json:"id"`
	TripCode           string     `json:"trip_code"`
	RouteID            *uuid.UUID `json:"route_id,omitempty"`
	VehicleID          uuid.UUID  `json:"vehicle_id"`
	DriverID           uuid.UUID  `json:"driver_id"`
	Status             TripStatus `json:"status"`
	Country            string     `json:"country"`
	OriginAddress      string     `json:"origin_address"`
	OriginLat          float64    `json:"origin_lat"`
	OriginLng          float64    `json:"origin_lng"`
	DestAddress        string     `json:"destination_address"`
	DestLat            float64    `json:"destination_lat"`
	DestLng            float64    `json:"destination_lng"`
	ScheduledPickup    time.Time  `json:"scheduled_pickup"`
	ScheduledDelivery  *time.Time `json:"scheduled_delivery,omitempty"`
	ActualPickup       *time.Time `json:"actual_pickup,omitempty"`
	ActualDelivery     *time.Time `json:"actual_delivery,omitempty"`
	DistanceKm         float64    `json:"distance_km"`
	CostPerKm          float64    `json:"cost_per_km"`
	TotalCost          float64    `json:"total_cost"`
	Currency           string     `json:"currency"`
	FuelCost           float64    `json:"fuel_cost"`
	CargoID            *uuid.UUID `json:"cargo_id,omitempty"`
	CurrentLat         *float64   `json:"current_lat,omitempty"`
	CurrentLng         *float64   `json:"current_lng,omitempty"`
	LastGPSUpdate      *time.Time `json:"last_gps_update,omitempty"`
	DelayReasonCode    string     `json:"delay_reason_code,omitempty"`
	Notes              string     `json:"notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type GPSUpdate struct {
	ID        uuid.UUID `json:"id"`
	TripID    uuid.UUID `json:"trip_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	SpeedKph  float64   `json:"speed_kph"`
	Heading   float64   `json:"heading"`
	RecordedAt time.Time `json:"recorded_at"`
	CreatedAt time.Time `json:"created_at"`
}

type TransportAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateTripRequest struct {
	VehicleID         string  `json:"vehicle_id"`
	DriverID          string  `json:"driver_id"`
	RouteID           string  `json:"route_id,omitempty"`
	Country           string  `json:"country"`
	OriginAddress     string  `json:"origin_address"`
	OriginLat         float64 `json:"origin_lat"`
	OriginLng         float64 `json:"origin_lng"`
	DestAddress       string  `json:"destination_address"`
	DestLat           float64 `json:"destination_lat"`
	DestLng           float64 `json:"destination_lng"`
	ScheduledPickup   string  `json:"scheduled_pickup"`
	ScheduledDelivery string  `json:"scheduled_delivery,omitempty"`
	DistanceKm        float64 `json:"distance_km"`
	CostPerKm         float64 `json:"cost_per_km"`
	Currency          string  `json:"currency"`
	CargoID           string  `json:"cargo_id,omitempty"`
	Notes             string  `json:"notes,omitempty"`
}

type UpdateTripStatusRequest struct {
	Status          TripStatus `json:"status"`
	DelayReasonCode string     `json:"delay_reason_code,omitempty"`
}

type UpdateGPSRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	SpeedKph  float64 `json:"speed_kph"`
	Heading   float64 `json:"heading"`
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
