package models

import (
	"time"

	"github.com/google/uuid"
)

type ResponseStatus string
type ActionType string

const (
	ResponseActive     ResponseStatus = "active"
	ResponseContained  ResponseStatus = "contained"
	ResponseResolved   ResponseStatus = "resolved"

	ActionQuarantine   ActionType = "quarantine"
	ActionVaccination  ActionType = "mass_vaccination"
	ActionSurveillance ActionType = "enhanced_surveillance"
	ActionTreatment    ActionType = "treatment_protocol"
	ActionCommunity    ActionType = "community_alert"
)

type OutbreakResponse struct {
	ID              uuid.UUID      `json:"id"`
	ResponseRef     string         `json:"response_ref"`
	AlertRef        string         `json:"alert_ref"`
	DiseaseName     string         `json:"disease_name"`
	Country         string         `json:"country"`
	Region          string         `json:"region"`
	Status          ResponseStatus `json:"status"`
	LeadCoordinator uuid.UUID      `json:"lead_coordinator"`
	TeamSize        int            `json:"team_size"`
	CasesTargeted   int            `json:"cases_targeted"`
	Population      int            `json:"population_at_risk"`
	TenantID        string         `json:"tenant_id"`
	StartedAt       time.Time      `json:"started_at"`
	ContainedAt     *time.Time     `json:"contained_at,omitempty"`
	ResolvedAt      *time.Time     `json:"resolved_at,omitempty"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type ResponseAction struct {
	ID         uuid.UUID  `json:"id"`
	ResponseID uuid.UUID  `json:"response_id"`
	ActionType ActionType `json:"action_type"`
	Description string    `json:"description"`
	AssignedTo  uuid.UUID `json:"assigned_to"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateResponseRequest struct {
	AlertRef        string    `json:"alert_ref"`
	DiseaseName     string    `json:"disease_name"`
	Country         string    `json:"country"`
	Region          string    `json:"region"`
	LeadCoordinator uuid.UUID `json:"lead_coordinator"`
	TeamSize        int       `json:"team_size"`
	CasesTargeted   int       `json:"cases_targeted"`
	Population      int       `json:"population_at_risk"`
}

type AddActionRequest struct {
	ActionType  ActionType `json:"action_type"`
	Description string     `json:"description"`
	AssignedTo  uuid.UUID  `json:"assigned_to"`
}
