package models

import (
	"time"

	"github.com/google/uuid"
)

type RouteStatus string

const (
	RouteActive    RouteStatus = "active"
	RouteInactive  RouteStatus = "inactive"
	RouteArchived  RouteStatus = "archived"
)

type RouteType string

const (
	RouteFixed    RouteType = "fixed"
	RouteDynamic  RouteType = "dynamic"
	RouteCircular RouteType = "circular"
)

type Route struct {
	ID            uuid.UUID   `json:"id"`
	Name          string      `json:"name"`
	RouteCode     string      `json:"route_code"`
	RouteType     RouteType   `json:"route_type"`
	Status        RouteStatus `json:"status"`
	Country       string      `json:"country"`
	OriginName    string      `json:"origin_name"`
	OriginLat     float64     `json:"origin_lat"`
	OriginLng     float64     `json:"origin_lng"`
	DestName      string      `json:"destination_name"`
	DestLat       float64     `json:"destination_lat"`
	DestLng       float64     `json:"destination_lng"`
	DistanceKm    float64     `json:"distance_km"`
	EstHours      float64     `json:"estimated_hours"`
	Waypoints     []Waypoint  `json:"waypoints,omitempty"`
	FreightClass  string      `json:"freight_class"`
	Notes         string      `json:"notes,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type Waypoint struct {
	Sequence int     `json:"sequence"`
	Name     string  `json:"name"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type RouteSchedule struct {
	ID            uuid.UUID  `json:"id"`
	RouteID       uuid.UUID  `json:"route_id"`
	VehicleID     *uuid.UUID `json:"vehicle_id,omitempty"`
	DriverID      *uuid.UUID `json:"driver_id,omitempty"`
	DepartureTime time.Time  `json:"departure_time"`
	ArrivalTime   *time.Time `json:"arrival_time,omitempty"`
	Status        string     `json:"status"`
	ActualDeptAt  *time.Time `json:"actual_departure_at,omitempty"`
	ActualArrAt   *time.Time `json:"actual_arrival_at,omitempty"`
	Notes         string     `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type RouteAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateRouteRequest struct {
	Name       string     `json:"name"`
	RouteCode  string     `json:"route_code"`
	RouteType  RouteType  `json:"route_type"`
	Country    string     `json:"country"`
	OriginName string     `json:"origin_name"`
	OriginLat  float64    `json:"origin_lat"`
	OriginLng  float64    `json:"origin_lng"`
	DestName   string     `json:"destination_name"`
	DestLat    float64    `json:"destination_lat"`
	DestLng    float64    `json:"destination_lng"`
	DistanceKm float64    `json:"distance_km"`
	EstHours   float64    `json:"estimated_hours"`
	Waypoints  []Waypoint `json:"waypoints,omitempty"`
	FreightClass string   `json:"freight_class,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}

type ScheduleRouteRequest struct {
	VehicleID     string `json:"vehicle_id,omitempty"`
	DriverID      string `json:"driver_id,omitempty"`
	DepartureTime string `json:"departure_time"`
	Notes         string `json:"notes,omitempty"`
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
