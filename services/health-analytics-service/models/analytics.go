package models

import (
	"time"

	"github.com/google/uuid"
)

type DiseaseSeverity string

const (
	SeverityMild     DiseaseSeverity = "mild"
	SeverityModerate DiseaseSeverity = "moderate"
	SeveritySevere   DiseaseSeverity = "severe"
	SeverityCritical DiseaseSeverity = "critical"
)

type OutbreakStatus string

const (
	OutbreakActive   OutbreakStatus = "active"
	OutbreakMonitor  OutbreakStatus = "monitoring"
	OutbreakResolved OutbreakStatus = "resolved"
)

type DiseaseReport struct {
	ID           uuid.UUID `json:"id"`
	ClinicID     uuid.UUID `json:"clinic_id"`
	Country      string    `json:"country"`
	Region       string    `json:"region"`
	ICD10Code    string    `json:"icd10_code"`
	DiseaseName  string    `json:"disease_name"`
	CaseCount    int       `json:"case_count"`
	Period       string    `json:"period"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	Severity     DiseaseSeverity `json:"severity"`
	CreatedAt    time.Time `json:"created_at"`
}

type OutbreakAlert struct {
	ID          uuid.UUID      `json:"id"`
	AlertRef    string         `json:"alert_ref"`
	ClinicID    uuid.UUID      `json:"clinic_id"`
	Country     string         `json:"country"`
	Region      string         `json:"region"`
	ICD10Code   string         `json:"icd10_code"`
	DiseaseName string         `json:"disease_name"`
	CaseCount   int            `json:"case_count"`
	Threshold   int            `json:"threshold"`
	Status      OutbreakStatus `json:"status"`
	DetectedAt  time.Time      `json:"detected_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
}

type ClinicMetric struct {
	ID                       uuid.UUID `json:"id"`
	ClinicID                 uuid.UUID `json:"clinic_id"`
	Country                  string    `json:"country"`
	Period                   string    `json:"period"`
	PeriodStart              time.Time `json:"period_start"`
	PeriodEnd                time.Time `json:"period_end"`
	TotalPatients            int       `json:"total_patients"`
	AvgVisitMinutes          float64   `json:"avg_visit_minutes"`
	ReferralCount            int       `json:"referral_count"`
	ReferralSuccessRate      float64   `json:"referral_success_rate"`
	PatientOutcomeImproved   int       `json:"patient_outcome_improved"`
	PatientOutcomeStable     int       `json:"patient_outcome_stable"`
	PatientOutcomeWorsened   int       `json:"patient_outcome_worsened"`
	CostPerVisitUSD          float64   `json:"cost_per_visit_usd"`
	CreatedAt                time.Time `json:"created_at"`
}

type ImpactSummary struct {
	Country                string    `json:"country"`
	Period                 string    `json:"period"`
	TotalPatients          int       `json:"total_patients"`
	OutcomeImprovementRate float64   `json:"outcome_improvement_rate"`
	AvgCostPerVisitUSD     float64   `json:"avg_cost_per_visit_usd"`
	CostReductionPct       float64   `json:"cost_reduction_pct"`
	ActiveOutbreaks        int       `json:"active_outbreaks"`
	TotalClinics           int       `json:"total_clinics"`
	GeneratedAt            time.Time `json:"generated_at"`
}

type HealthAnalyticsAuditLog struct {
	ID        uuid.UUID `json:"id"`
	ActorID   uuid.UUID `json:"actor_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	CreatedAt time.Time `json:"created_at"`
}

type ReportDiseaseRequest struct {
	ClinicID    string `json:"clinic_id"`
	Country     string `json:"country"`
	Region      string `json:"region"`
	ICD10Code   string `json:"icd10_code"`
	DiseaseName string `json:"disease_name"`
	CaseCount   int    `json:"case_count"`
	Period      string `json:"period"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	Severity    string `json:"severity"`
}

type RecordClinicMetricRequest struct {
	ClinicID                 string  `json:"clinic_id"`
	Country                  string  `json:"country"`
	Period                   string  `json:"period"`
	PeriodStart              string  `json:"period_start"`
	PeriodEnd                string  `json:"period_end"`
	TotalPatients            int     `json:"total_patients"`
	AvgVisitMinutes          float64 `json:"avg_visit_minutes"`
	ReferralCount            int     `json:"referral_count"`
	ReferralSuccessRate      float64 `json:"referral_success_rate"`
	PatientOutcomeImproved   int     `json:"patient_outcome_improved"`
	PatientOutcomeStable     int     `json:"patient_outcome_stable"`
	PatientOutcomeWorsened   int     `json:"patient_outcome_worsened"`
	CostPerVisitUSD          float64 `json:"cost_per_visit_usd"`
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
