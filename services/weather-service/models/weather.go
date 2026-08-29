package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Forecast ─────────────────────────────────────────────────────────────────

type ForecastType string

const (
	ForecastDaily   ForecastType = "daily"
	ForecastHourly  ForecastType = "hourly"
	ForecastSeasonal ForecastType = "seasonal"
)

type WeatherCondition string

const (
	ConditionSunny         WeatherCondition = "sunny"
	ConditionPartlyCloudy  WeatherCondition = "partly_cloudy"
	ConditionCloudy        WeatherCondition = "cloudy"
	ConditionRainy         WeatherCondition = "rainy"
	ConditionHeavyRain     WeatherCondition = "heavy_rain"
	ConditionThunderstorm  WeatherCondition = "thunderstorm"
	ConditionDrizzle       WeatherCondition = "drizzle"
	ConditionFoggy         WeatherCondition = "foggy"
	ConditionWindy         WeatherCondition = "windy"
	ConditionHaze          WeatherCondition = "haze"
)

type WeatherForecast struct {
	ID              uuid.UUID        `json:"id"`
	Country         string           `json:"country"`
	Region          string           `json:"region"`
	District        string           `json:"district"`
	Latitude        float64          `json:"latitude"`
	Longitude       float64          `json:"longitude"`
	ForecastType    ForecastType     `json:"forecast_type"`
	ForecastDate    time.Time        `json:"forecast_date"`
	Condition       WeatherCondition `json:"condition"`
	TempMinC        float64          `json:"temp_min_c"`
	TempMaxC        float64          `json:"temp_max_c"`
	TempAvgC        float64          `json:"temp_avg_c"`
	HumidityPct     float64          `json:"humidity_pct"`
	WindSpeedKmh    float64          `json:"wind_speed_kmh"`
	WindDirection   string           `json:"wind_direction"`
	RainfallMm      float64          `json:"rainfall_mm"`
	RainfallProb    float64          `json:"rainfall_probability"`
	UVIndex         float64          `json:"uv_index"`
	DataSource      string           `json:"data_source"`
	ValidUntil      time.Time        `json:"valid_until"`
	CreatedAt       time.Time        `json:"created_at"`
}

// ─── Alert ────────────────────────────────────────────────────────────────────

type AlertType string

const (
	AlertFlood        AlertType = "flood"
	AlertDrought      AlertType = "drought"
	AlertFrost        AlertType = "frost"
	AlertHeatWave     AlertType = "heat_wave"
	AlertHighWind     AlertType = "high_wind"
	AlertHeavyRain    AlertType = "heavy_rain"
	AlertPestRisk     AlertType = "pest_risk"
	AlertDiseaseRisk  AlertType = "disease_risk"
	AlertLocust       AlertType = "locust"
	AlertFireRisk     AlertType = "fire_risk"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWatch    AlertSeverity = "watch"
	SeverityWarning  AlertSeverity = "warning"
	SeverityEmergency AlertSeverity = "emergency"
)

type WeatherAlert struct {
	ID           uuid.UUID     `json:"id"`
	AlertType    AlertType     `json:"alert_type"`
	Severity     AlertSeverity `json:"severity"`
	Country      string        `json:"country"`
	Region       string        `json:"region"`
	District     string        `json:"district"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Instructions string        `json:"instructions"`
	AffectedCrops []string     `json:"affected_crops,omitempty"`
	IssuedAt     time.Time     `json:"issued_at"`
	ExpiresAt    *time.Time    `json:"expires_at,omitempty"`
	Active       bool          `json:"active"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// ─── Pest/Disease Advisory ────────────────────────────────────────────────────

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskModerate RiskLevel = "moderate"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type PestAdvisory struct {
	ID              uuid.UUID  `json:"id"`
	PestName        string     `json:"pest_name"`
	PestType        string     `json:"pest_type"` // pest or disease
	AffectedCrops   []string   `json:"affected_crops"`
	Country         string     `json:"country"`
	Region          string     `json:"region"`
	RiskLevel       RiskLevel  `json:"risk_level"`
	Description     string     `json:"description"`
	Symptoms        string     `json:"symptoms"`
	Prevention      string     `json:"prevention"`
	Treatment       string     `json:"treatment"`
	ReportedCases   int        `json:"reported_cases"`
	ValidFrom       time.Time  `json:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ─── Observation (ground truth) ───────────────────────────────────────────────

type WeatherObservation struct {
	ID            uuid.UUID `json:"id"`
	ReporterID    uuid.UUID `json:"reporter_id"`
	Country       string    `json:"country"`
	Region        string    `json:"region"`
	District      string    `json:"district"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	ObservedAt    time.Time `json:"observed_at"`
	TempC         *float64  `json:"temp_c,omitempty"`
	RainfallMm    *float64  `json:"rainfall_mm,omitempty"`
	HumidityPct   *float64  `json:"humidity_pct,omitempty"`
	WindSpeedKmh  *float64  `json:"wind_speed_kmh,omitempty"`
	Condition     WeatherCondition `json:"condition"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ─── Audit log ────────────────────────────────────────────────────────────────

type WeatherAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

// ─── Request types ────────────────────────────────────────────────────────────

type CreateForecastRequest struct {
	Country       string           `json:"country"`
	Region        string           `json:"region"`
	District      string           `json:"district,omitempty"`
	Latitude      float64          `json:"latitude"`
	Longitude     float64          `json:"longitude"`
	ForecastType  ForecastType     `json:"forecast_type"`
	ForecastDate  string           `json:"forecast_date"`
	Condition     WeatherCondition `json:"condition"`
	TempMinC      float64          `json:"temp_min_c"`
	TempMaxC      float64          `json:"temp_max_c"`
	HumidityPct   float64          `json:"humidity_pct"`
	WindSpeedKmh  float64          `json:"wind_speed_kmh"`
	WindDirection string           `json:"wind_direction,omitempty"`
	RainfallMm    float64          `json:"rainfall_mm"`
	RainfallProb  float64          `json:"rainfall_probability"`
	UVIndex       float64          `json:"uv_index"`
	DataSource    string           `json:"data_source,omitempty"`
	ValidHours    int              `json:"valid_hours"`
}

type CreateAlertRequest struct {
	AlertType     AlertType     `json:"alert_type"`
	Severity      AlertSeverity `json:"severity"`
	Country       string        `json:"country"`
	Region        string        `json:"region"`
	District      string        `json:"district,omitempty"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Instructions  string        `json:"instructions,omitempty"`
	AffectedCrops []string      `json:"affected_crops,omitempty"`
	ExpiresAt     *string       `json:"expires_at,omitempty"`
}

type CreateAdvisoryRequest struct {
	PestName      string    `json:"pest_name"`
	PestType      string    `json:"pest_type"`
	AffectedCrops []string  `json:"affected_crops"`
	Country       string    `json:"country"`
	Region        string    `json:"region"`
	RiskLevel     RiskLevel `json:"risk_level"`
	Description   string    `json:"description"`
	Symptoms      string    `json:"symptoms,omitempty"`
	Prevention    string    `json:"prevention,omitempty"`
	Treatment     string    `json:"treatment,omitempty"`
	ValidFrom     string    `json:"valid_from,omitempty"`
	ValidUntil    *string   `json:"valid_until,omitempty"`
}

type SubmitObservationRequest struct {
	Country      string           `json:"country"`
	Region       string           `json:"region"`
	District     string           `json:"district,omitempty"`
	Latitude     float64          `json:"latitude"`
	Longitude    float64          `json:"longitude"`
	ObservedAt   string           `json:"observed_at,omitempty"`
	TempC        *float64         `json:"temp_c,omitempty"`
	RainfallMm   *float64         `json:"rainfall_mm,omitempty"`
	HumidityPct  *float64         `json:"humidity_pct,omitempty"`
	WindSpeedKmh *float64         `json:"wind_speed_kmh,omitempty"`
	Condition    WeatherCondition `json:"condition"`
	Notes        string           `json:"notes,omitempty"`
}

// ─── Standard response types ──────────────────────────────────────────────────

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
