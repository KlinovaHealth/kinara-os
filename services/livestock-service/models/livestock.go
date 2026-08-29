package models

import (
	"time"

	"github.com/google/uuid"
)

type AnimalSpecies string
type HealthStatus string

const (
	SpeciesCattle  AnimalSpecies = "cattle"
	SpeciesGoat    AnimalSpecies = "goat"
	SpeciesSheep   AnimalSpecies = "sheep"
	SpeciesPig     AnimalSpecies = "pig"
	SpeciesPoultry AnimalSpecies = "poultry"

	HealthHealthy  HealthStatus = "healthy"
	HealthSick     HealthStatus = "sick"
	HealthQuarantine HealthStatus = "quarantine"
	HealthDeceased HealthStatus = "deceased"
)

type Animal struct {
	ID           uuid.UUID     `json:"id"`
	TagRef       string        `json:"tag_ref"`
	FarmerID     uuid.UUID     `json:"farmer_id"`
	Species      AnimalSpecies `json:"species"`
	Breed        string        `json:"breed"`
	BirthDate    *time.Time    `json:"birth_date,omitempty"`
	WeightKg     float64       `json:"weight_kg"`
	HealthStatus HealthStatus  `json:"health_status"`
	IsActive     bool          `json:"is_active"`
	TenantID     string        `json:"tenant_id"`
	RegisteredAt time.Time     `json:"registered_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type ProductionRecord struct {
	ID          uuid.UUID `json:"id"`
	AnimalID    uuid.UUID `json:"animal_id"`
	FarmerID    uuid.UUID `json:"farmer_id"`
	ProductType string    `json:"product_type"`
	QuantityKg  float64   `json:"quantity_kg"`
	RecordedAt  time.Time `json:"recorded_at"`
	Notes       string    `json:"notes,omitempty"`
}

type RegisterAnimalRequest struct {
	FarmerID  uuid.UUID     `json:"farmer_id"`
	Species   AnimalSpecies `json:"species"`
	Breed     string        `json:"breed"`
	BirthDate *time.Time    `json:"birth_date"`
	WeightKg  float64       `json:"weight_kg"`
}

type RecordProductionRequest struct {
	ProductType string    `json:"product_type"`
	QuantityKg  float64   `json:"quantity_kg"`
	RecordedAt  time.Time `json:"recorded_at"`
	Notes       string    `json:"notes"`
}

type UpdateHealthRequest struct {
	HealthStatus HealthStatus `json:"health_status"`
	WeightKg     float64      `json:"weight_kg"`
}
