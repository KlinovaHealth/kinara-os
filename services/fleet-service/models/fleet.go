package models

import (
	"time"

	"github.com/google/uuid"
)

type VehicleType string

const (
	VehicleTruck       VehicleType = "truck"
	VehiclePickup      VehicleType = "pickup"
	VehicleMotorcycle  VehicleType = "motorcycle"
	VehicleVan         VehicleType = "van"
	VehicleBus         VehicleType = "bus"
	VehicleAmbulance   VehicleType = "ambulance"
	VehicleRefrigerated VehicleType = "refrigerated"
	VehicleTanker      VehicleType = "tanker"
)

type VehicleStatus string

const (
	VehicleActive      VehicleStatus = "active"
	VehicleInRepair    VehicleStatus = "in_repair"
	VehicleRetired     VehicleStatus = "retired"
	VehicleAvailable   VehicleStatus = "available"
	VehicleAssigned    VehicleStatus = "assigned"
	VehicleInTransit   VehicleStatus = "in_transit"
)

type FuelType string

const (
	FuelPetrol  FuelType = "petrol"
	FuelDiesel  FuelType = "diesel"
	FuelCNG     FuelType = "cng"
	FuelElectric FuelType = "electric"
	FuelHybrid  FuelType = "hybrid"
)

type Vehicle struct {
	ID                uuid.UUID     `json:"id"`
	RegistrationNo    string        `json:"registration_no"`
	VehicleType       VehicleType   `json:"vehicle_type"`
	Make              string        `json:"make"`
	Model             string        `json:"model"`
	Year              int           `json:"year"`
	FuelType          FuelType      `json:"fuel_type"`
	PayloadCapacityKg float64       `json:"payload_capacity_kg"`
	VolumeCapacityM3  float64       `json:"volume_capacity_m3"`
	Status            VehicleStatus `json:"status"`
	Country           string        `json:"country"`
	BaseLocation      string        `json:"base_location"`
	CurrentOdometerKm float64       `json:"current_odometer_km"`
	LastServiceKm     float64       `json:"last_service_km"`
	NextServiceKm     float64       `json:"next_service_km"`
	InsuranceExpiry   *time.Time    `json:"insurance_expiry,omitempty"`
	InspectionExpiry  *time.Time    `json:"inspection_expiry,omitempty"`
	AssignedDriverID  *uuid.UUID    `json:"assigned_driver_id,omitempty"`
	Notes             string        `json:"notes,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type MaintenanceRecord struct {
	ID            uuid.UUID `json:"id"`
	VehicleID     uuid.UUID `json:"vehicle_id"`
	ServiceType   string    `json:"service_type"`
	Description   string    `json:"description"`
	OdometerKm    float64   `json:"odometer_km"`
	Cost          float64   `json:"cost"`
	Currency      string    `json:"currency"`
	ServicedBy    string    `json:"serviced_by"`
	ServicedAt    time.Time `json:"serviced_at"`
	NextServiceKm float64   `json:"next_service_km"`
	CreatedAt     time.Time `json:"created_at"`
}

type FuelLog struct {
	ID         uuid.UUID `json:"id"`
	VehicleID  uuid.UUID `json:"vehicle_id"`
	DriverID   *uuid.UUID `json:"driver_id,omitempty"`
	LitresFilled float64 `json:"litres_filled"`
	CostPerLitre float64 `json:"cost_per_litre"`
	TotalCost  float64   `json:"total_cost"`
	Currency   string    `json:"currency"`
	OdometerKm float64   `json:"odometer_km"`
	Station    string    `json:"station"`
	FilledAt   time.Time `json:"filled_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type FleetAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateVehicleRequest struct {
	RegistrationNo    string      `json:"registration_no"`
	VehicleType       VehicleType `json:"vehicle_type"`
	Make              string      `json:"make"`
	Model             string      `json:"model"`
	Year              int         `json:"year"`
	FuelType          FuelType    `json:"fuel_type"`
	PayloadCapacityKg float64     `json:"payload_capacity_kg"`
	VolumeCapacityM3  float64     `json:"volume_capacity_m3"`
	Country           string      `json:"country"`
	BaseLocation      string      `json:"base_location"`
	InsuranceExpiry   *string     `json:"insurance_expiry,omitempty"`
	InspectionExpiry  *string     `json:"inspection_expiry,omitempty"`
	Notes             string      `json:"notes,omitempty"`
}

type UpdateVehicleRequest struct {
	Status           *VehicleStatus `json:"status,omitempty"`
	AssignedDriverID *string        `json:"assigned_driver_id,omitempty"`
	CurrentOdometer  *float64       `json:"current_odometer_km,omitempty"`
	Notes            *string        `json:"notes,omitempty"`
}

type LogMaintenanceRequest struct {
	ServiceType   string  `json:"service_type"`
	Description   string  `json:"description"`
	OdometerKm    float64 `json:"odometer_km"`
	Cost          float64 `json:"cost"`
	Currency      string  `json:"currency"`
	ServicedBy    string  `json:"serviced_by"`
	ServicedAt    string  `json:"serviced_at"`
	NextServiceKm float64 `json:"next_service_km"`
}

type LogFuelRequest struct {
	LitresFilled float64 `json:"litres_filled"`
	CostPerLitre float64 `json:"cost_per_litre"`
	Currency     string  `json:"currency"`
	OdometerKm   float64 `json:"odometer_km"`
	Station      string  `json:"station"`
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
