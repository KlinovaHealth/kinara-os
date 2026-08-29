package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Listing ──────────────────────────────────────────────────────────────────

type ListingStatus string

const (
	ListingActive    ListingStatus = "active"
	ListingReserved  ListingStatus = "reserved"
	ListingSold      ListingStatus = "sold"
	ListingExpired   ListingStatus = "expired"
	ListingCancelled ListingStatus = "cancelled"
)

type PriceUnit string

const (
	UnitKg     PriceUnit = "kg"
	UnitTonne  PriceUnit = "tonne"
	UnitBag    PriceUnit = "bag"
	UnitCrate  PriceUnit = "crate"
	UnitBushel PriceUnit = "bushel"
	UnitLitre  PriceUnit = "litre"
)

// MarketListing represents a commodity for sale from a farmer.
type MarketListing struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	FarmerID       uuid.UUID     `json:"farmer_id" db:"farmer_id"`
	CropType       string        `json:"crop_type" db:"crop_type"`
	Variety        string        `json:"variety,omitempty" db:"variety"`
	QuantityKg     float64       `json:"quantity_kg" db:"quantity_kg"`
	QuantityAvail  float64       `json:"quantity_available" db:"quantity_available"`
	PricePerUnit   float64       `json:"price_per_unit" db:"price_per_unit"`
	Currency       string        `json:"currency" db:"currency"`
	PriceUnit      PriceUnit     `json:"price_unit" db:"price_unit"`
	QualityGrade   string        `json:"quality_grade" db:"quality_grade"` // A, B, C
	Country        string        `json:"country" db:"country"`
	Region         string        `json:"region" db:"region"`
	Market         string        `json:"market" db:"market"` // market name/location
	HarvestedAt    *time.Time    `json:"harvested_at,omitempty" db:"harvested_at"`
	AvailableFrom  time.Time     `json:"available_from" db:"available_from"`
	AvailableUntil *time.Time    `json:"available_until,omitempty" db:"available_until"`
	Status         ListingStatus `json:"status" db:"status"`
	Description    string        `json:"description,omitempty" db:"description"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
}

// ─── Price record (historical) ────────────────────────────────────────────────

type PriceRecord struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CropType    string    `json:"crop_type" db:"crop_type"`
	Market      string    `json:"market" db:"market"`
	Country     string    `json:"country" db:"country"`
	Region      string    `json:"region" db:"region"`
	PricePerKg  float64   `json:"price_per_kg" db:"price_per_kg"`
	Currency    string    `json:"currency" db:"currency"`
	Source      string    `json:"source" db:"source"` // "listing", "reported", "official"
	RecordedAt  time.Time `json:"recorded_at" db:"recorded_at"`
	RecordedBy  uuid.UUID `json:"recorded_by" db:"recorded_by"`
}

// ─── Bid / Match ──────────────────────────────────────────────────────────────

type BidStatus string

const (
	BidPending   BidStatus = "pending"
	BidAccepted  BidStatus = "accepted"
	BidRejected  BidStatus = "rejected"
	BidWithdrawn BidStatus = "withdrawn"
	BidExpired   BidStatus = "expired"
)

type MarketBid struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ListingID   uuid.UUID `json:"listing_id" db:"listing_id"`
	BuyerID     uuid.UUID `json:"buyer_id" db:"buyer_id"`
	QuantityKg  float64   `json:"quantity_kg" db:"quantity_kg"`
	BidPrice    float64   `json:"bid_price" db:"bid_price"`
	Currency    string    `json:"currency" db:"currency"`
	Status      BidStatus `json:"status" db:"status"`
	Message     string    `json:"message,omitempty" db:"message"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ─── Audit log ────────────────────────────────────────────────────────────────

type MarketAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

// ─── Request types ────────────────────────────────────────────────────────────

type CreateListingRequest struct {
	CropType       string    `json:"crop_type"`
	Variety        string    `json:"variety,omitempty"`
	QuantityKg     float64   `json:"quantity_kg"`
	PricePerUnit   float64   `json:"price_per_unit"`
	Currency       string    `json:"currency"`
	PriceUnit      PriceUnit `json:"price_unit"`
	QualityGrade   string    `json:"quality_grade"`
	Market         string    `json:"market"`
	Region         string    `json:"region"`
	HarvestedAt    *string   `json:"harvested_at,omitempty"`
	AvailableFrom  string    `json:"available_from"`
	AvailableUntil *string   `json:"available_until,omitempty"`
	Description    string    `json:"description,omitempty"`
}

type UpdateListingRequest struct {
	PricePerUnit   *float64  `json:"price_per_unit,omitempty"`
	QuantityAvail  *float64  `json:"quantity_available,omitempty"`
	Status         *ListingStatus `json:"status,omitempty"`
	AvailableUntil *string   `json:"available_until,omitempty"`
	Description    *string   `json:"description,omitempty"`
}

type PlaceBidRequest struct {
	QuantityKg float64  `json:"quantity_kg"`
	BidPrice   float64  `json:"bid_price"`
	Currency   string   `json:"currency"`
	Message    string   `json:"message,omitempty"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
}

type RespondBidRequest struct {
	Status  BidStatus `json:"status"` // accepted or rejected
	Message string    `json:"message,omitempty"`
}

type RecordPriceRequest struct {
	CropType   string  `json:"crop_type"`
	Market     string  `json:"market"`
	Country    string  `json:"country"`
	Region     string  `json:"region"`
	PricePerKg float64 `json:"price_per_kg"`
	Currency   string  `json:"currency"`
	Source     string  `json:"source"`
}

// ─── Price summary ────────────────────────────────────────────────────────────

type PriceSummary struct {
	CropType  string  `json:"crop_type"`
	Market    string  `json:"market"`
	Country   string  `json:"country"`
	MinPrice  float64 `json:"min_price_per_kg"`
	MaxPrice  float64 `json:"max_price_per_kg"`
	AvgPrice  float64 `json:"avg_price_per_kg"`
	Currency  string  `json:"currency"`
	DataPoints int    `json:"data_points"`
	Period    string  `json:"period"`
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
