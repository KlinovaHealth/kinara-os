package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditEvent is a cross-pillar event from any Kinara OS service.
type AuditEvent struct {
	ID          uuid.UUID `json:"id"`
	EventRef    string    `json:"event_ref"`
	Service     string    `json:"service"`
	Pillar      string    `json:"pillar"`
	EventType   string    `json:"event_type"`
	ActorID     uuid.UUID `json:"actor_id"`
	ActorRole   string    `json:"actor_role"`
	ResourceID  string    `json:"resource_id"`
	ResourceType string   `json:"resource_type"`
	Detail      string    `json:"detail"`
	IPAddress   string    `json:"ip_address"`
	TenantID    string    `json:"tenant_id"`
	Signature   string    `json:"signature"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuditReport summarizes audit activity for a time period.
type AuditReport struct {
	ID          uuid.UUID         `json:"id"`
	ReportRef   string            `json:"report_ref"`
	PeriodStart time.Time         `json:"period_start"`
	PeriodEnd   time.Time         `json:"period_end"`
	TotalEvents int               `json:"total_events"`
	ByPillar    map[string]int    `json:"events_by_pillar"`
	ByService   map[string]int    `json:"events_by_service"`
	GeneratedAt time.Time         `json:"generated_at"`
}

type LogEventRequest struct {
	Service      string `json:"service"`
	Pillar       string `json:"pillar"`
	EventType    string `json:"event_type"`
	ActorID      string `json:"actor_id"`
	ActorRole    string `json:"actor_role"`
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	Detail       string `json:"detail"`
	IPAddress    string `json:"ip_address"`
	TenantID     string `json:"tenant_id"`
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
