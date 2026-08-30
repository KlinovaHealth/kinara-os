package models

import (
	"time"

	"github.com/google/uuid"
)

type Animal struct {
	ID           uuid.UUID `json:"id"`
	AnimalRef    string    `json:"animal_ref"`
	FarmerID     uuid.UUID `json:"farmer_id"`
	AnimalType   string    `json:"animal_type"` // "cattle", "goat", "sheep", "pig", "poultry"
	Breed        string    `json:"breed"`
	AgeMonths    int       `json:"age_months"`
	Sex          string    `json:"sex"` // "M", "F"
	EarTag       string    `json:"ear_tag,omitempty"`
	TenantID     string    `json:"tenant_id"`
	RegisteredAt time.Time `json:"registered_at"`
}

type HealthEvent struct {
	ID             uuid.UUID  `json:"id"`
	AnimalID       uuid.UUID  `json:"animal_id"`
	EventType      string     `json:"event_type"` // "vaccination", "illness", "treatment", "checkup"
	Description    string     `json:"description"`
	Treatment      string     `json:"treatment,omitempty"`
	VeterinarianID *uuid.UUID `json:"veterinarian_id,omitempty"`
	EventDate      time.Time  `json:"event_date"`
	CreatedBy      uuid.UUID  `json:"created_by"`
}

type ProductionRecord struct {
	ID             uuid.UUID `json:"id"`
	AnimalID       uuid.UUID `json:"animal_id"`
	ProductionType string    `json:"production_type"` // "milk", "eggs", "wool"
	Quantity       float64   `json:"quantity"`
	Unit           string    `json:"unit"` // "liters", "units", "kg"
	RecordedDate   time.Time `json:"recorded_date"`
	RecordedBy     uuid.UUID `json:"recorded_by"`
}

type VeterinaryAlert struct {
	ID        uuid.UUID `json:"id"`
	AnimalID  uuid.UUID `json:"animal_id"`
	AlertType string    `json:"alert_type"`
	Priority  string    `json:"priority"` // "low", "medium", "high"
	CreatedAt time.Time `json:"created_at"`
}

type HerdAnalytics struct {
	TotalAnimals         int     `json:"total_animals"`
	HealthyCount         int     `json:"healthy_count"`
	HealthRatePct        float64 `json:"health_rate_percent"`
	TotalProductionMonth float64 `json:"total_production_this_month"`
	TopProducingType     string  `json:"top_producing_type"`
}

type RegisterAnimalRequest struct {
	AnimalType string    `json:"animal_type"`
	Breed      string    `json:"breed"`
	AgeMonths  int       `json:"age_months"`
	Sex        string    `json:"sex"`
	EarTag     string    `json:"ear_tag"`
	FarmerID   uuid.UUID `json:"farmer_id"`
}

type HealthEventRequest struct {
	EventType      string     `json:"event_type"`
	Description    string     `json:"description"`
	Treatment      string     `json:"treatment"`
	VeterinarianID *uuid.UUID `json:"veterinarian_id"`
}

type ProductionRequest struct {
	ProductionType string    `json:"production_type"`
	Quantity       float64   `json:"quantity"`
	Unit           string    `json:"unit"`
	RecordedDate   time.Time `json:"recorded_date"`
}
