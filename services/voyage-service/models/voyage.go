package models

import (
	"time"

	"github.com/google/uuid"
)

type VoyageStatus string

const (
	VoyagePlanned    VoyageStatus = "planned"
	VoyageDeparted   VoyageStatus = "departed"
	VoyageInTransit  VoyageStatus = "in_transit"
	VoyageArrived    VoyageStatus = "arrived"
	VoyageCompleted  VoyageStatus = "completed"
	VoyageCancelled  VoyageStatus = "cancelled"
)

type Voyage struct {
	ID              uuid.UUID    `json:"id"`
	VoyageRef       string       `json:"voyage_ref"`
	VesselID        uuid.UUID    `json:"vessel_id"`
	OriginPort      string       `json:"origin_port"`
	DestinationPort string       `json:"destination_port"`
	CargoType       string       `json:"cargo_type"`
	CargoTons       float64      `json:"cargo_tons"`
	Status          VoyageStatus `json:"status"`
	DepartureAt     *time.Time   `json:"departure_at,omitempty"`
	EstArrivalAt    *time.Time   `json:"estimated_arrival_at,omitempty"`
	ActualArrivalAt *time.Time   `json:"actual_arrival_at,omitempty"`
	DistanceNM      float64      `json:"distance_nautical_miles"`
	FuelTons        float64      `json:"fuel_tons"`
	TenantID        string       `json:"tenant_id"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type VoyageEvent struct {
	ID          uuid.UUID `json:"id"`
	VoyageID    uuid.UUID `json:"voyage_id"`
	EventType   string    `json:"event_type"`
	Description string    `json:"description"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type CreateVoyageRequest struct {
	VesselID        uuid.UUID  `json:"vessel_id"`
	OriginPort      string     `json:"origin_port"`
	DestinationPort string     `json:"destination_port"`
	CargoType       string     `json:"cargo_type"`
	CargoTons       float64    `json:"cargo_tons"`
	DepartureAt     *time.Time `json:"departure_at"`
	EstArrivalAt    *time.Time `json:"estimated_arrival_at"`
	DistanceNM      float64    `json:"distance_nautical_miles"`
	FuelTons        float64    `json:"fuel_tons"`
}

type UpdateStatusRequest struct {
	Status          VoyageStatus `json:"status"`
	ActualArrivalAt *time.Time   `json:"actual_arrival_at"`
}

type LogEventRequest struct {
	EventType   string    `json:"event_type"`
	Description string    `json:"description"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	OccurredAt  time.Time `json:"occurred_at"`
}
