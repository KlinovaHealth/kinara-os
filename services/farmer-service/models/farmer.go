package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Farm size ────────────────────────────────────────────────────────────────

type FarmSize string

const (
	FarmSmallholder FarmSize = "smallholder" // < 2 ha
	FarmSmall       FarmSize = "small"        // 2–10 ha
	FarmMedium      FarmSize = "medium"       // 10–100 ha
	FarmLarge       FarmSize = "large"        // > 100 ha
)

// ─── Farmer ───────────────────────────────────────────────────────────────────

// FarmerRow is the DB record with encrypted PII fields.
type FarmerRow struct {
	ID                uuid.UUID  `db:"id"`
	UserID            *uuid.UUID `db:"user_id"` // links to auth-service user
	FullNameEnc       string     `db:"full_name_enc"`
	PhoneEnc          string     `db:"phone_enc"`
	NationalIDEnc     string     `db:"national_id_enc"`
	Country           string     `db:"country"`
	Region            string     `db:"region"`
	District          string     `db:"district"`
	GPSLat            *float64   `db:"gps_lat"`
	GPSLng            *float64   `db:"gps_lng"`
	FarmSizeHa        float64    `db:"farm_size_ha"`
	FarmSize          FarmSize   `db:"farm_size"`
	PrimaryLanguage   string     `db:"primary_language"`
	IsVerified        bool       `db:"is_verified"`
	IsActive          bool       `db:"is_active"`
	CooperativeID     *uuid.UUID `db:"cooperative_id"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

// Farmer is the decrypted API representation.
type Farmer struct {
	ID              uuid.UUID  `json:"id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	FullName        string     `json:"full_name"`
	Phone           string     `json:"phone"`
	NationalID      string     `json:"national_id,omitempty"`
	Country         string     `json:"country"`
	Region          string     `json:"region"`
	District        string     `json:"district"`
	GPSLat          *float64   `json:"gps_lat,omitempty"`
	GPSLng          *float64   `json:"gps_lng,omitempty"`
	FarmSizeHa      float64    `json:"farm_size_ha"`
	FarmSize        FarmSize   `json:"farm_size"`
	PrimaryLanguage string     `json:"primary_language"`
	IsVerified      bool       `json:"is_verified"`
	IsActive        bool       `json:"is_active"`
	CooperativeID   *uuid.UUID `json:"cooperative_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ─── Farm plot ────────────────────────────────────────────────────────────────

type FarmPlot struct {
	ID          uuid.UUID `json:"id" db:"id"`
	FarmerID    uuid.UUID `json:"farmer_id" db:"farmer_id"`
	Name        string    `json:"name" db:"name"`
	SizeHa      float64   `json:"size_ha" db:"size_ha"`
	SoilType    string    `json:"soil_type" db:"soil_type"`
	Irrigation  bool      `json:"irrigation" db:"irrigation"`
	GPSPolygon  string    `json:"gps_polygon,omitempty" db:"gps_polygon"` // GeoJSON
	CurrentCrop string    `json:"current_crop,omitempty" db:"current_crop"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ─── Crop record ──────────────────────────────────────────────────────────────

type CropStatus string

const (
	CropPlanted   CropStatus = "planted"
	CropGrowing   CropStatus = "growing"
	CropHarvested CropStatus = "harvested"
	CropFailed    CropStatus = "failed"
)

type CropRecord struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	FarmerID       uuid.UUID  `json:"farmer_id" db:"farmer_id"`
	PlotID         *uuid.UUID `json:"plot_id,omitempty" db:"plot_id"`
	CropType       string     `json:"crop_type" db:"crop_type"`
	Variety        string     `json:"variety,omitempty" db:"variety"`
	AreaHa         float64    `json:"area_ha" db:"area_ha"`
	PlantedAt      time.Time  `json:"planted_at" db:"planted_at"`
	ExpectedHarvest time.Time `json:"expected_harvest" db:"expected_harvest"`
	ActualHarvest  *time.Time `json:"actual_harvest,omitempty" db:"actual_harvest"`
	YieldKg        *float64   `json:"yield_kg,omitempty" db:"yield_kg"`
	Status         CropStatus `json:"status" db:"status"`
	Notes          string     `json:"notes,omitempty" db:"notes"`
	Season         string     `json:"season" db:"season"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// ─── Audit log ────────────────────────────────────────────────────────────────

type FarmerAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	FarmerID  *uuid.UUID `json:"farmer_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

// ─── Request types ────────────────────────────────────────────────────────────

type RegisterFarmerRequest struct {
	FullName        string   `json:"full_name"`
	Phone           string   `json:"phone"`
	NationalID      string   `json:"national_id,omitempty"`
	Country         string   `json:"country"`
	Region          string   `json:"region"`
	District        string   `json:"district"`
	GPSLat          *float64 `json:"gps_lat,omitempty"`
	GPSLng          *float64 `json:"gps_lng,omitempty"`
	FarmSizeHa      float64  `json:"farm_size_ha"`
	PrimaryLanguage string   `json:"primary_language"`
	CooperativeID   *string  `json:"cooperative_id,omitempty"`
}

type UpdateFarmerRequest struct {
	Phone           *string  `json:"phone,omitempty"`
	Region          *string  `json:"region,omitempty"`
	District        *string  `json:"district,omitempty"`
	GPSLat          *float64 `json:"gps_lat,omitempty"`
	GPSLng          *float64 `json:"gps_lng,omitempty"`
	FarmSizeHa      *float64 `json:"farm_size_ha,omitempty"`
	PrimaryLanguage *string  `json:"primary_language,omitempty"`
	CooperativeID   *string  `json:"cooperative_id,omitempty"`
	IsActive        *bool    `json:"is_active,omitempty"`
}

type AddPlotRequest struct {
	Name       string  `json:"name"`
	SizeHa     float64 `json:"size_ha"`
	SoilType   string  `json:"soil_type,omitempty"`
	Irrigation bool    `json:"irrigation"`
	GPSPolygon string  `json:"gps_polygon,omitempty"`
}

type RecordCropRequest struct {
	PlotID          *string `json:"plot_id,omitempty"`
	CropType        string  `json:"crop_type"`
	Variety         string  `json:"variety,omitempty"`
	AreaHa          float64 `json:"area_ha"`
	PlantedAt       string  `json:"planted_at"` // RFC3339
	ExpectedHarvest string  `json:"expected_harvest"` // RFC3339
	Season          string  `json:"season"`
	Notes           string  `json:"notes,omitempty"`
}

type UpdateCropRequest struct {
	Status        CropStatus `json:"status"`
	ActualHarvest *string    `json:"actual_harvest,omitempty"`
	YieldKg       *float64   `json:"yield_kg,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
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
