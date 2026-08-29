package models

import (
	"time"

	"github.com/google/uuid"
)

type DriverStatus string

const (
	DriverActive    DriverStatus = "active"
	DriverSuspended DriverStatus = "suspended"
	DriverInactive  DriverStatus = "inactive"
	DriverAvailable DriverStatus = "available"
	DriverOnDuty    DriverStatus = "on_duty"
	DriverOffDuty   DriverStatus = "off_duty"
)

type LicenseClass string

const (
	LicenseA  LicenseClass = "A"  // motorcycle
	LicenseB  LicenseClass = "B"  // light vehicle
	LicenseC  LicenseClass = "C"  // heavy vehicle
	LicenseD  LicenseClass = "D"  // passenger
	LicenseE  LicenseClass = "E"  // articulated
)

type DriverRow struct {
	ID              uuid.UUID    `db:"id"`
	FullNameEnc     string       `db:"full_name_enc"`
	PhoneEnc        string       `db:"phone_enc"`
	NationalIDEnc   string       `db:"national_id_enc"`
	LicenseNo       string       `db:"license_no"`
	LicenseClass    LicenseClass `db:"license_class"`
	LicenseExpiry   time.Time    `db:"license_expiry"`
	Status          DriverStatus `db:"status"`
	Country         string       `db:"country"`
	BaseLocation    string       `db:"base_location"`
	TotalTrips      int          `db:"total_trips"`
	TotalKm         float64      `db:"total_km"`
	Rating          float64      `db:"rating"`
	AssignedVehicleID *uuid.UUID `db:"assigned_vehicle_id"`
	CreatedAt       time.Time    `db:"created_at"`
	UpdatedAt       time.Time    `db:"updated_at"`
}

type Driver struct {
	ID              uuid.UUID    `json:"id"`
	FullName        string       `json:"full_name"`
	Phone           string       `json:"phone"`
	NationalID      string       `json:"national_id,omitempty"`
	LicenseNo       string       `json:"license_no"`
	LicenseClass    LicenseClass `json:"license_class"`
	LicenseExpiry   time.Time    `json:"license_expiry"`
	Status          DriverStatus `json:"status"`
	Country         string       `json:"country"`
	BaseLocation    string       `json:"base_location"`
	TotalTrips      int          `json:"total_trips"`
	TotalKm         float64      `json:"total_km"`
	Rating          float64      `json:"rating"`
	AssignedVehicleID *uuid.UUID `json:"assigned_vehicle_id,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type DriverTrip struct {
	ID         uuid.UUID  `json:"id"`
	DriverID   uuid.UUID  `json:"driver_id"`
	VehicleID  uuid.UUID  `json:"vehicle_id"`
	RouteID    *uuid.UUID `json:"route_id,omitempty"`
	DistanceKm float64    `json:"distance_km"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Status     string     `json:"status"`
	Notes      string     `json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type DriverAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateDriverRequest struct {
	FullName     string       `json:"full_name"`
	Phone        string       `json:"phone"`
	NationalID   string       `json:"national_id"`
	LicenseNo    string       `json:"license_no"`
	LicenseClass LicenseClass `json:"license_class"`
	LicenseExpiry string      `json:"license_expiry"`
	Country      string       `json:"country"`
	BaseLocation string       `json:"base_location"`
}

type UpdateDriverRequest struct {
	Status            *DriverStatus `json:"status,omitempty"`
	AssignedVehicleID *string       `json:"assigned_vehicle_id,omitempty"`
	BaseLocation      *string       `json:"base_location,omitempty"`
}

type LogTripRequest struct {
	VehicleID  string  `json:"vehicle_id"`
	RouteID    string  `json:"route_id,omitempty"`
	DistanceKm float64 `json:"distance_km"`
	StartTime  string  `json:"start_time"`
	Notes      string  `json:"notes,omitempty"`
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
