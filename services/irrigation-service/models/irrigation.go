package models

import (
	"time"

	"github.com/google/uuid"
)

type IrrigationSystem struct {
	ID             uuid.UUID `json:"id"`
	FarmID         string    `json:"farm_id"`
	SystemType     string    `json:"system_type"` // "drip", "sprinkler", "flood"
	CapacityLiters float64   `json:"capacity_liters"`
	SensorID       string    `json:"sensor_id,omitempty"`
	TenantID       string    `json:"tenant_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type WateringSchedule struct {
	ID             uuid.UUID `json:"id"`
	FarmID         string    `json:"farm_id"`
	CronExpression string    `json:"cron_expression"`
	DurationMin    int       `json:"duration_minutes"`
	CropType       string    `json:"crop_type"`
	TenantID       string    `json:"tenant_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type SoilMoistureReading struct {
	ID          uuid.UUID `json:"id"`
	FarmID      string    `json:"farm_id"`
	MoisturePct float64   `json:"moisture_percent"`
	SensorID    string    `json:"sensor_id,omitempty"`
	RecordedAt  time.Time `json:"recorded_at"`
}

type WateringHistory struct {
	ID          uuid.UUID `json:"id"`
	FarmID      string    `json:"farm_id"`
	DurationMin int       `json:"duration_minutes"`
	AmountL     float64   `json:"amount_liters"`
	TriggerType string    `json:"trigger_type"` // "manual", "scheduled", "auto"
	IrrigatedAt time.Time `json:"irrigated_at"`
}

type IrrigationAlert struct {
	ID        uuid.UUID `json:"id"`
	FarmID    string    `json:"farm_id"`
	Message   string    `json:"message"`
	AlertType string    `json:"alert_type"` // "low_moisture", "overdue", "system_fault"
	SentAt    time.Time `json:"sent_at"`
}

type IrrigationRec struct {
	ShouldIrrigate         bool    `json:"should_irrigate"`
	Reason                 string  `json:"reason"`
	RecommendedDurationMin int     `json:"recommended_duration_min"`
	OptimalTime            string  `json:"optimal_time,omitempty"`
	CurrentMoisturePct     float64 `json:"current_moisture_percent"`
}

type FarmStatus struct {
	FarmID         string               `json:"farm_id"`
	System         *IrrigationSystem    `json:"system,omitempty"`
	LatestMoisture *SoilMoistureReading `json:"latest_moisture,omitempty"`
}

type RegisterSystemRequest struct {
	SystemType     string  `json:"system_type"`
	CapacityLiters float64 `json:"capacity_liters"`
	SensorID       string  `json:"sensor_id"`
}

type ScheduleRequest struct {
	CronExpression string `json:"cron_expression"`
	DurationMin    int    `json:"duration_minutes"`
	CropType       string `json:"crop_type"`
}

type AlertRequest struct {
	Message   string `json:"message"`
	AlertType string `json:"alert_type"`
}

type MoistureRequest struct {
	MoisturePct float64 `json:"moisture_percent"`
	SensorID    string  `json:"sensor_id"`
}
