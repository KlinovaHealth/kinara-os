package models

import (
	"time"

	"github.com/google/uuid"
)

type CrewRank string
type CertStatus string

const (
	RankCaptain     CrewRank = "captain"
	RankFirstOfficer CrewRank = "first_officer"
	RankEngineer    CrewRank = "chief_engineer"
	RankDeckhand    CrewRank = "deckhand"
	RankCook        CrewRank = "cook"
	RankSteward     CrewRank = "steward"

	CertValid   CertStatus = "valid"
	CertExpiring CertStatus = "expiring_soon"
	CertExpired CertStatus = "expired"
)

type CrewMember struct {
	ID             uuid.UUID `json:"id"`
	CrewRef        string    `json:"crew_ref"`
	FullName       string    `json:"full_name"`
	Nationality    string    `json:"nationality"`
	PassportNumber string    `json:"passport_number,omitempty"`
	Rank           CrewRank  `json:"rank"`
	VesselID       *uuid.UUID `json:"vessel_id,omitempty"`
	IsActive       bool      `json:"is_active"`
	TenantID       string    `json:"tenant_id"`
	JoinedAt       time.Time `json:"joined_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CrewCertification struct {
	ID          uuid.UUID  `json:"id"`
	CrewID      uuid.UUID  `json:"crew_id"`
	CertType    string     `json:"cert_type"`
	CertNumber  string     `json:"cert_number"`
	IssuedBy    string     `json:"issued_by"`
	IssuedAt    time.Time  `json:"issued_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	Status      CertStatus `json:"status"`
}

type RegisterCrewRequest struct {
	FullName       string     `json:"full_name"`
	Nationality    string     `json:"nationality"`
	PassportNumber string     `json:"passport_number"`
	Rank           CrewRank   `json:"rank"`
	VesselID       *uuid.UUID `json:"vessel_id"`
}

type AddCertificationRequest struct {
	CertType   string    `json:"cert_type"`
	CertNumber string    `json:"cert_number"`
	IssuedBy   string    `json:"issued_by"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type AssignVesselRequest struct {
	VesselID uuid.UUID `json:"vessel_id"`
}
