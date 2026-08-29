package models

import (
	"time"

	"github.com/google/uuid"
)

type IrrigationMethod string
type EventStatus string

const (
	MethodDrip       IrrigationMethod = "drip"
	MethodSprinkler  IrrigationMethod = "sprinkler"
	MethodFlood      IrrigationMethod = "flood"
	MethodFurrow     IrrigationMethod = "furrow"

	EventScheduled  EventStatus = "scheduled"
	EventActive     EventStatus = "active"
	EventCompleted  EventStatus = "completed"
	EventSkipped    EventStatus = "skipped"
)

type IrrigationSchedule struct {
	ID            uuid.UUID        `json:"id"`
	ScheduleRef   string           `json:"schedule_ref"`
	FarmerID      uuid.UUID        `json:"farmer_id"`
	FieldID       string           `json:"field_id"`
	CropType      string           `json:"crop_type"`
	Method        IrrigationMethod `json:"method"`
	FrequencyDays int              `json:"frequency_days"`
	DurationMin   int              `json:"duration_minutes"`
	WaterLiters   float64          `json:"water_liters_per_event"`
	IsActive      bool             `json:"is_active"`
	TenantID      string           `json:"tenant_id"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type IrrigationEvent struct {
	ID           uuid.UUID   `json:"id"`
	ScheduleID   uuid.UUID   `json:"schedule_id"`
	FarmerID     uuid.UUID   `json:"farmer_id"`
	FieldID      string      `json:"field_id"`
	ScheduledAt  time.Time   `json:"scheduled_at"`
	StartedAt    *time.Time  `json:"started_at,omitempty"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
	WaterUsedL   float64     `json:"water_used_liters"`
	Status       EventStatus `json:"status"`
	Notes        string      `json:"notes,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

type CreateScheduleRequest struct {
	FarmerID      uuid.UUID        `json:"farmer_id"`
	FieldID       string           `json:"field_id"`
	CropType      string           `json:"crop_type"`
	Method        IrrigationMethod `json:"method"`
	FrequencyDays int              `json:"frequency_days"`
	DurationMin   int              `json:"duration_minutes"`
	WaterLiters   float64          `json:"water_liters_per_event"`
}

type LogEventRequest struct {
	ScheduledAt time.Time   `json:"scheduled_at"`
	WaterUsedL  float64     `json:"water_used_liters"`
	Status      EventStatus `json:"status"`
	Notes       string      `json:"notes"`
}
