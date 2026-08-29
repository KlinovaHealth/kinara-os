package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Cooperative ──────────────────────────────────────────────────────────────

type CoopStatus string

const (
	CoopActive    CoopStatus = "active"
	CoopSuspended CoopStatus = "suspended"
	CoopDissolved CoopStatus = "dissolved"
)

type CoopType string

const (
	CoopProduction CoopType = "production"
	CoopMarketing  CoopType = "marketing"
	CoopCredit     CoopType = "credit"
	CoopMulti      CoopType = "multi_purpose"
)

type Cooperative struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	RegistrationNo string     `json:"registration_no"`
	CoopType       CoopType   `json:"coop_type"`
	Status         CoopStatus `json:"status"`
	Country        string     `json:"country"`
	Region         string     `json:"region"`
	District       string     `json:"district"`
	TotalMembers   int        `json:"total_members"`
	TotalFarmHa    float64    `json:"total_farm_ha"`
	Description    string     `json:"description,omitempty"`
	ContactPhone   string     `json:"contact_phone"`
	ContactEmail   string     `json:"contact_email,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ─── Membership ───────────────────────────────────────────────────────────────

type MemberRole string

const (
	RoleMember    MemberRole = "member"
	RoleSecretary MemberRole = "secretary"
	RoleTreasurer MemberRole = "treasurer"
	RoleChairman  MemberRole = "chairman"
)

type MemberStatus string

const (
	MemberActive    MemberStatus = "active"
	MemberSuspended MemberStatus = "suspended"
	MemberExited    MemberStatus = "exited"
)

type CoopMember struct {
	ID            uuid.UUID    `json:"id"`
	CoopID        uuid.UUID    `json:"cooperative_id"`
	FarmerID      uuid.UUID    `json:"farmer_id"`
	Role          MemberRole   `json:"role"`
	Status        MemberStatus `json:"status"`
	SharesHeld    int          `json:"shares_held"`
	JoinedAt      time.Time    `json:"joined_at"`
	ExitedAt      *time.Time   `json:"exited_at,omitempty"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// ─── Collective selling pool ───────────────────────────────────────────────────

type PoolStatus string

const (
	PoolOpen      PoolStatus = "open"
	PoolClosed    PoolStatus = "closed"
	PoolSold      PoolStatus = "sold"
	PoolCancelled PoolStatus = "cancelled"
)

type SellingPool struct {
	ID            uuid.UUID  `json:"id"`
	CoopID        uuid.UUID  `json:"cooperative_id"`
	CropType      string     `json:"crop_type"`
	TargetQtyKg   float64    `json:"target_quantity_kg"`
	CollectedQtyKg float64   `json:"collected_quantity_kg"`
	PricePerKg    float64    `json:"price_per_kg"`
	Currency      string     `json:"currency"`
	Status        PoolStatus `json:"status"`
	OpenUntil     *time.Time `json:"open_until,omitempty"`
	SoldAt        *time.Time `json:"sold_at,omitempty"`
	TotalRevenue  float64    `json:"total_revenue"`
	Description   string     `json:"description,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ─── Pool contribution ────────────────────────────────────────────────────────

type PoolContribution struct {
	ID           uuid.UUID  `json:"id"`
	PoolID       uuid.UUID  `json:"pool_id"`
	FarmerID     uuid.UUID  `json:"farmer_id"`
	QuantityKg   float64    `json:"quantity_kg"`
	PayoutAmount float64    `json:"payout_amount"`
	PayoutPaid   bool       `json:"payout_paid"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ─── Cooperative audit log ────────────────────────────────────────────────────

type CoopAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

// ─── Request types ────────────────────────────────────────────────────────────

type CreateCoopRequest struct {
	Name           string   `json:"name"`
	RegistrationNo string   `json:"registration_no"`
	CoopType       CoopType `json:"coop_type"`
	Country        string   `json:"country"`
	Region         string   `json:"region"`
	District       string   `json:"district,omitempty"`
	Description    string   `json:"description,omitempty"`
	ContactPhone   string   `json:"contact_phone"`
	ContactEmail   string   `json:"contact_email,omitempty"`
}

type AddMemberRequest struct {
	FarmerID   string     `json:"farmer_id"`
	Role       MemberRole `json:"role"`
	SharesHeld int        `json:"shares_held"`
}

type UpdateMemberRequest struct {
	Role       *MemberRole   `json:"role,omitempty"`
	Status     *MemberStatus `json:"status,omitempty"`
	SharesHeld *int          `json:"shares_held,omitempty"`
}

type CreatePoolRequest struct {
	CropType     string     `json:"crop_type"`
	TargetQtyKg  float64    `json:"target_quantity_kg"`
	PricePerKg   float64    `json:"price_per_kg"`
	Currency     string     `json:"currency"`
	OpenUntil    *string    `json:"open_until,omitempty"`
	Description  string     `json:"description,omitempty"`
}

type ContributeRequest struct {
	FarmerID   string  `json:"farmer_id"`
	QuantityKg float64 `json:"quantity_kg"`
}

type RecordSaleRequest struct {
	PricePerKg   float64 `json:"price_per_kg"`
	TotalRevenue float64 `json:"total_revenue"`
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
