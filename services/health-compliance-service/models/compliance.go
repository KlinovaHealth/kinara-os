package models

import (
	"time"

	"github.com/google/uuid"
)

type AuditEntryType string

const (
	EntryCreate   AuditEntryType = "create"
	EntryRead     AuditEntryType = "read"
	EntryUpdate   AuditEntryType = "update"
	EntryDelete   AuditEntryType = "delete"
	EntryExport   AuditEntryType = "export"
	EntryBreach   AuditEntryType = "breach_attempt"
)

type RegulatoryStandard string

const (
	StandardHIPAA  RegulatoryStandard = "HIPAA"
	StandardGDPR   RegulatoryStandard = "GDPR"
	StandardNIST   RegulatoryStandard = "NIST"
	StandardISO27001 RegulatoryStandard = "ISO27001"
)

// AuditEntry is an immutable, cryptographically signed record of a health data access event.
type AuditEntry struct {
	ID           uuid.UUID  `json:"id"`
	EntryRef     string     `json:"entry_ref"`
	Service      string     `json:"service"`
	ResourceType string     `json:"resource_type"`
	ResourceID   uuid.UUID  `json:"resource_id"`
	ActorID      uuid.UUID  `json:"actor_id"`
	ActorRole    string     `json:"actor_role"`
	Action       AuditEntryType `json:"action"`
	Detail       string     `json:"detail"`
	IPAddress    string     `json:"ip_address"`
	Signature    string     `json:"signature"`
	CreatedAt    time.Time  `json:"created_at"`
}

type BreachAttempt struct {
	ID          uuid.UUID `json:"id"`
	Service     string    `json:"service"`
	ActorID     *uuid.UUID `json:"actor_id,omitempty"`
	IPAddress   string    `json:"ip_address"`
	Reason      string    `json:"reason"`
	DetectedAt  time.Time `json:"detected_at"`
	Resolved    bool      `json:"resolved"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

type EncryptionStatus struct {
	Service         string    `json:"service"`
	TotalFields     int       `json:"total_fields"`
	EncryptedFields int       `json:"encrypted_fields"`
	Algorithm       string    `json:"algorithm"`
	LastVerifiedAt  time.Time `json:"last_verified_at"`
	IsCompliant     bool      `json:"is_compliant"`
}

type ComplianceReport struct {
	ID            uuid.UUID  `json:"id"`
	ReportRef     string     `json:"report_ref"`
	Standard      RegulatoryStandard `json:"standard"`
	Country       string     `json:"country"`
	PeriodStart   time.Time  `json:"period_start"`
	PeriodEnd     time.Time  `json:"period_end"`
	TotalEvents   int        `json:"total_events"`
	BreachCount   int        `json:"breach_count"`
	IsCompliant   bool       `json:"is_compliant"`
	Findings      string     `json:"findings"`
	GeneratedAt   time.Time  `json:"generated_at"`
	GeneratedBy   uuid.UUID  `json:"generated_by"`
}

type LogEntryRequest struct {
	Service      string `json:"service"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	ActorID      string `json:"actor_id"`
	ActorRole    string `json:"actor_role"`
	Action       string `json:"action"`
	Detail       string `json:"detail"`
	IPAddress    string `json:"ip_address"`
}

type GenerateReportRequest struct {
	Standard    string `json:"standard"`
	Country     string `json:"country"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
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
