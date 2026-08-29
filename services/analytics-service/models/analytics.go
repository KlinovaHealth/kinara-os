package models

import (
	"time"
	"github.com/google/uuid"
)

type Pillar string
type MetricType string

const (
	PillarHealth    Pillar = "health"
	PillarAgri      Pillar = "agriculture"
	PillarLogistics Pillar = "logistics"
	PillarMaritime  Pillar = "maritime"

	MetricServiceDelivery MetricType = "service_delivery"
	MetricEconomicImpact  MetricType = "economic_impact"
	MetricReach           MetricType = "reach"
	MetricEfficiency      MetricType = "efficiency"
)

type ImpactMetric struct {
	ID             uuid.UUID  `json:"id"`
	Pillar         Pillar     `json:"pillar"`
	Country        string     `json:"country"`
	MetricType     MetricType `json:"metric_type"`
	MetricName     string     `json:"metric_name"`
	MetricValue    float64    `json:"metric_value"`
	MetricUnit     string     `json:"metric_unit"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	BeneficiaryCount int64   `json:"beneficiary_count"`
	Notes          string     `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type CrossPillarSummary struct {
	ID              uuid.UUID `json:"id"`
	Country         string    `json:"country"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	HealthScore     float64   `json:"health_score"`
	AgriScore       float64   `json:"agri_score"`
	LogisticsScore  float64   `json:"logistics_score"`
	MaritimeScore   float64   `json:"maritime_score"`
	OverallScore    float64   `json:"overall_score"`
	TotalBeneficiaries int64  `json:"total_beneficiaries"`
	TotalServicesDelivered int64 `json:"total_services_delivered"`
	CreatedAt       time.Time `json:"created_at"`
}

type GovernmentReport struct {
	ID              uuid.UUID  `json:"id"`
	ReportRef       string     `json:"report_ref"`
	Country         string     `json:"country"`
	ReportType      string     `json:"report_type"`
	PeriodStart     time.Time  `json:"period_start"`
	PeriodEnd       time.Time  `json:"period_end"`
	GeneratedAt     time.Time  `json:"generated_at"`
	SummaryJSON     string     `json:"summary_json"`
	CreatedAt       time.Time  `json:"created_at"`
}

type Bottleneck struct {
	ID              uuid.UUID  `json:"id"`
	Pillar          Pillar     `json:"pillar"`
	Country         string     `json:"country"`
	BottleneckType  string     `json:"bottleneck_type"`
	Description     string     `json:"description"`
	Severity        string     `json:"severity"`
	AffectedUnits   int        `json:"affected_units"`
	RecommendedAction string   `json:"recommended_action"`
	DetectedAt      time.Time  `json:"detected_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type AnalyticsAuditLog struct {
	ID         uuid.UUID `json:"id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type RecordImpactRequest struct {
	Pillar           string  `json:"pillar"`
	Country          string  `json:"country"`
	MetricType       string  `json:"metric_type"`
	MetricName       string  `json:"metric_name"`
	MetricValue      float64 `json:"metric_value"`
	MetricUnit       string  `json:"metric_unit"`
	PeriodStart      string  `json:"period_start"`
	PeriodEnd        string  `json:"period_end"`
	BeneficiaryCount int64   `json:"beneficiary_count"`
	Notes            string  `json:"notes,omitempty"`
}

type CreateSummaryRequest struct {
	Country         string  `json:"country"`
	PeriodStart     string  `json:"period_start"`
	PeriodEnd       string  `json:"period_end"`
	HealthScore     float64 `json:"health_score"`
	AgriScore       float64 `json:"agri_score"`
	LogisticsScore  float64 `json:"logistics_score"`
	MaritimeScore   float64 `json:"maritime_score"`
	TotalBeneficiaries int64 `json:"total_beneficiaries"`
	TotalServicesDelivered int64 `json:"total_services_delivered"`
}

type GenerateReportRequest struct {
	Country     string `json:"country"`
	ReportType  string `json:"report_type"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
}

type ReportBottleneckRequest struct {
	Pillar            string `json:"pillar"`
	Country           string `json:"country"`
	BottleneckType    string `json:"bottleneck_type"`
	Description       string `json:"description"`
	Severity          string `json:"severity"`
	AffectedUnits     int    `json:"affected_units"`
	RecommendedAction string `json:"recommended_action"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
