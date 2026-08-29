package models

import (
	"time"

	"github.com/google/uuid"
)

type MetricPeriod string

const (
	PeriodDaily   MetricPeriod = "daily"
	PeriodWeekly  MetricPeriod = "weekly"
	PeriodMonthly MetricPeriod = "monthly"
)

type LogisticsMetric struct {
	ID                  uuid.UUID    `json:"id"`
	Period              MetricPeriod `json:"period"`
	PeriodStart         time.Time    `json:"period_start"`
	PeriodEnd           time.Time    `json:"period_end"`
	Country             string       `json:"country"`
	TotalTrips          int          `json:"total_trips"`
	TotalDistanceKm     float64      `json:"total_distance_km"`
	TotalDeliveries     int          `json:"total_deliveries"`
	SuccessfulDeliveries int         `json:"successful_deliveries"`
	OnTimeDeliveries    int          `json:"on_time_deliveries"`
	OnTimeRate          float64      `json:"on_time_rate"`
	AvgCostPerKm        float64      `json:"avg_cost_per_km"`
	AvgCostPerDelivery  float64      `json:"avg_cost_per_delivery"`
	TotalRevenue        float64      `json:"total_revenue"`
	Currency            string       `json:"currency"`
	BottleneckRoute     string       `json:"bottleneck_route,omitempty"`
	BottleneckWarehouse string       `json:"bottleneck_warehouse,omitempty"`
	CreatedAt           time.Time    `json:"created_at"`
}

type DemandForecast struct {
	ID            uuid.UUID `json:"id"`
	Country       string    `json:"country"`
	Route         string    `json:"route"`
	ForecastDate  time.Time `json:"forecast_date"`
	PredictedVolume float64 `json:"predicted_volume_tons"`
	PredictedTrips  int     `json:"predicted_trips"`
	ConfidencePct   float64 `json:"confidence_pct"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AnalyticsAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type RecordMetricRequest struct {
	Period              MetricPeriod `json:"period"`
	PeriodStart         string       `json:"period_start"`
	PeriodEnd           string       `json:"period_end"`
	Country             string       `json:"country"`
	TotalTrips          int          `json:"total_trips"`
	TotalDistanceKm     float64      `json:"total_distance_km"`
	TotalDeliveries     int          `json:"total_deliveries"`
	SuccessfulDeliveries int         `json:"successful_deliveries"`
	OnTimeDeliveries    int          `json:"on_time_deliveries"`
	AvgCostPerKm        float64      `json:"avg_cost_per_km"`
	TotalRevenue        float64      `json:"total_revenue"`
	Currency            string       `json:"currency"`
	BottleneckRoute     string       `json:"bottleneck_route,omitempty"`
	BottleneckWarehouse string       `json:"bottleneck_warehouse,omitempty"`
}

type CreateForecastRequest struct {
	Country         string  `json:"country"`
	Route           string  `json:"route"`
	ForecastDate    string  `json:"forecast_date"`
	PredictedVolume float64 `json:"predicted_volume_tons"`
	PredictedTrips  int     `json:"predicted_trips"`
	ConfidencePct   float64 `json:"confidence_pct"`
	Notes           string  `json:"notes,omitempty"`
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
