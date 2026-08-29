package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Enumerations ─────────────────────────────────────────────────────────────

type ComplianceStatus string

const (
	CompliancePending    ComplianceStatus = "pending"
	ComplianceCompliant  ComplianceStatus = "compliant"
	ComplianceViolation  ComplianceStatus = "violation"
	ComplianceExempted   ComplianceStatus = "exempted"
)

type ReportType string

const (
	ReportEpidemiology ReportType = "epidemiology"
	ReportCompliance   ReportType = "compliance"
	ReportDiseaseBurden ReportType = "disease_burden"
	ReportOutbreak     ReportType = "outbreak"
	ReportMortality    ReportType = "mortality"
	ReportImmunization ReportType = "immunization"
)

type ReportFrequency string

const (
	FrequencyDaily    ReportFrequency = "daily"
	FrequencyWeekly   ReportFrequency = "weekly"
	FrequencyMonthly  ReportFrequency = "monthly"
	FrequencyQuarterly ReportFrequency = "quarterly"
	FrequencyAnnual   ReportFrequency = "annual"
	FrequencyOnDemand ReportFrequency = "on_demand"
)

type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "info"
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)

type AlertStatus string

const (
	AlertOpen       AlertStatus = "open"
	AlertAcknowledged AlertStatus = "acknowledged"
	AlertResolved   AlertStatus = "resolved"
)

type RuleType string

const (
	RuleReportingThreshold RuleType = "reporting_threshold"
	RuleOutbreakThreshold  RuleType = "outbreak_threshold"
	RuleDataRetention      RuleType = "data_retention"
	RuleAccessPolicy       RuleType = "access_policy"
	RuleNotification       RuleType = "notification"
)

type AuditAction string

const (
	AuditCreate AuditAction = "create"
	AuditRead   AuditAction = "read"
	AuditUpdate AuditAction = "update"
	AuditDelete AuditAction = "delete"
)

// ─── Domain models ────────────────────────────────────────────────────────────

// ComplianceReport is a periodic compliance submission from a facility or ministry.
type ComplianceReport struct {
	ID              uuid.UUID        `json:"id"`
	FacilityID      *uuid.UUID       `json:"facility_id,omitempty"`
	MinistryID      *uuid.UUID       `json:"ministry_id,omitempty"`
	ReportType      ReportType       `json:"report_type"`
	Frequency       ReportFrequency  `json:"frequency"`
	PeriodStart     time.Time        `json:"period_start"`
	PeriodEnd       time.Time        `json:"period_end"`
	Status          ComplianceStatus `json:"status"`
	Country         string           `json:"country"`
	Region          string           `json:"region,omitempty"`
	Summary         string           `json:"summary"`          // PHI-free summary — plaintext
	DataPayload     map[string]interface{} `json:"data_payload"` // encrypted aggregate metrics
	SubmittedBy     uuid.UUID        `json:"submitted_by"`
	SubmittedAt     *time.Time       `json:"submitted_at,omitempty"`
	ReviewedBy      *uuid.UUID       `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time       `json:"reviewed_at,omitempty"`
	ViolationNotes  string           `json:"violation_notes,omitempty"` // encrypted
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// ComplianceReportRow is the encrypted database form.
type ComplianceReportRow struct {
	ID                  uuid.UUID        `db:"id"`
	FacilityID          *uuid.UUID       `db:"facility_id"`
	MinistryID          *uuid.UUID       `db:"ministry_id"`
	ReportType          ReportType       `db:"report_type"`
	Frequency           ReportFrequency  `db:"frequency"`
	PeriodStart         time.Time        `db:"period_start"`
	PeriodEnd           time.Time        `db:"period_end"`
	Status              ComplianceStatus `db:"status"`
	Country             string           `db:"country"`
	Region              *string          `db:"region"`
	Summary             string           `db:"summary"`
	DataPayloadEnc      string           `db:"data_payload_enc"`
	SubmittedBy         uuid.UUID        `db:"submitted_by"`
	SubmittedAt         *time.Time       `db:"submitted_at"`
	ReviewedBy          *uuid.UUID       `db:"reviewed_by"`
	ReviewedAt          *time.Time       `db:"reviewed_at"`
	ViolationNotesEnc   *string          `db:"violation_notes_enc"`
	CreatedAt           time.Time        `db:"created_at"`
	UpdatedAt           time.Time        `db:"updated_at"`
}

// EpidemiologyRecord tracks disease occurrence at population level. No PHI — aggregate only.
type EpidemiologyRecord struct {
	ID              uuid.UUID  `json:"id"`
	ICD10Code       string     `json:"icd10_code"`
	ICD10Desc       string     `json:"icd10_description"`
	Country         string     `json:"country"`
	Region          string     `json:"region,omitempty"`
	District        string     `json:"district,omitempty"`
	CaseCount       int        `json:"case_count"`
	DeathCount      int        `json:"death_count"`
	RecoveredCount  int        `json:"recovered_count"`
	PeriodStart     time.Time  `json:"period_start"`
	PeriodEnd       time.Time  `json:"period_end"`
	AgeGroup        string     `json:"age_group,omitempty"` // e.g. "0-4", "5-17", "18-64", "65+"
	Gender          string     `json:"gender,omitempty"`    // "male", "female", "all"
	FacilityID      *uuid.UUID `json:"facility_id,omitempty"`
	ReportedBy      uuid.UUID  `json:"reported_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CoordinationRule defines a governance rule enforced across the platform.
type CoordinationRule struct {
	ID              uuid.UUID        `json:"id"`
	RuleType        RuleType         `json:"rule_type"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Country         string           `json:"country"`
	Region          string           `json:"region,omitempty"`
	Parameters      map[string]interface{} `json:"parameters"` // rule-specific config
	IsActive        bool             `json:"is_active"`
	EffectiveFrom   time.Time        `json:"effective_from"`
	EffectiveUntil  *time.Time       `json:"effective_until,omitempty"`
	CreatedBy       uuid.UUID        `json:"created_by"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// CoordinationRuleRow is the DB form with encrypted parameters.
type CoordinationRuleRow struct {
	ID              uuid.UUID  `db:"id"`
	RuleType        RuleType   `db:"rule_type"`
	Name            string     `db:"name"`
	Description     string     `db:"description"`
	Country         string     `db:"country"`
	Region          *string    `db:"region"`
	ParametersEnc   string     `db:"parameters_enc"`
	IsActive        bool       `db:"is_active"`
	EffectiveFrom   time.Time  `db:"effective_from"`
	EffectiveUntil  *time.Time `db:"effective_until"`
	CreatedBy       uuid.UUID  `db:"created_by"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

// GovernanceAlert is raised when a coordination rule threshold is breached.
type GovernanceAlert struct {
	ID          uuid.UUID     `json:"id"`
	RuleID      *uuid.UUID    `json:"rule_id,omitempty"`
	Severity    AlertSeverity `json:"severity"`
	Status      AlertStatus   `json:"status"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Country     string        `json:"country"`
	Region      string        `json:"region,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RaisedBy    uuid.UUID     `json:"raised_by"`
	AcknowledgedBy *uuid.UUID `json:"acknowledged_by,omitempty"`
	ResolvedBy  *uuid.UUID    `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// GovernanceAlertRow is the encrypted DB form.
type GovernanceAlertRow struct {
	ID             uuid.UUID     `db:"id"`
	RuleID         *uuid.UUID    `db:"rule_id"`
	Severity       AlertSeverity `db:"severity"`
	Status         AlertStatus   `db:"status"`
	Title          string        `db:"title"`
	Description    string        `db:"description"`
	Country        string        `db:"country"`
	Region         *string       `db:"region"`
	MetadataEnc    *string       `db:"metadata_enc"`
	RaisedBy       uuid.UUID     `db:"raised_by"`
	AcknowledgedBy *uuid.UUID    `db:"acknowledged_by"`
	ResolvedBy     *uuid.UUID    `db:"resolved_by"`
	ResolvedAt     *time.Time    `db:"resolved_at"`
	CreatedAt      time.Time     `db:"created_at"`
	UpdatedAt      time.Time     `db:"updated_at"`
}

// GovernanceAuditLog tracks all governance actions.
type GovernanceAuditLog struct {
	ID           uuid.UUID              `json:"id"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   uuid.UUID              `json:"resource_id"`
	Action       AuditAction            `json:"action"`
	AccessorID   uuid.UUID              `json:"accessor_id"`
	AccessorRole string                 `json:"accessor_role"`
	IPAddress    string                 `json:"ip_address"`
	RequestID    string                 `json:"request_id"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ─── Request / Response types ─────────────────────────────────────────────────

type CreateComplianceReportRequest struct {
	FacilityID  *uuid.UUID             `json:"facility_id,omitempty"`
	MinistryID  *uuid.UUID             `json:"ministry_id,omitempty"`
	ReportType  ReportType             `json:"report_type"`
	Frequency   ReportFrequency        `json:"frequency"`
	PeriodStart time.Time              `json:"period_start"`
	PeriodEnd   time.Time              `json:"period_end"`
	Country     string                 `json:"country"`
	Region      string                 `json:"region,omitempty"`
	Summary     string                 `json:"summary"`
	DataPayload map[string]interface{} `json:"data_payload"`
}

type ReviewComplianceReportRequest struct {
	Status         ComplianceStatus `json:"status"`
	ViolationNotes string           `json:"violation_notes,omitempty"`
}

type CreateEpidemiologyRecordRequest struct {
	ICD10Code      string     `json:"icd10_code"`
	ICD10Desc      string     `json:"icd10_description"`
	Country        string     `json:"country"`
	Region         string     `json:"region,omitempty"`
	District       string     `json:"district,omitempty"`
	CaseCount      int        `json:"case_count"`
	DeathCount     int        `json:"death_count"`
	RecoveredCount int        `json:"recovered_count"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	AgeGroup       string     `json:"age_group,omitempty"`
	Gender         string     `json:"gender,omitempty"`
	FacilityID     *uuid.UUID `json:"facility_id,omitempty"`
}

type CreateRuleRequest struct {
	RuleType       RuleType               `json:"rule_type"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Country        string                 `json:"country"`
	Region         string                 `json:"region,omitempty"`
	Parameters     map[string]interface{} `json:"parameters"`
	EffectiveFrom  time.Time              `json:"effective_from"`
	EffectiveUntil *time.Time             `json:"effective_until,omitempty"`
}

type CreateAlertRequest struct {
	RuleID      *uuid.UUID             `json:"rule_id,omitempty"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Country     string                 `json:"country"`
	Region      string                 `json:"region,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
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
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}
