package models

import (
	"time"

	"github.com/google/uuid"
)

type InputType string
type InputUnit string

const (
	InputSeed       InputType = "seed"
	InputFertilizer InputType = "fertilizer"
	InputPesticide  InputType = "pesticide"
	InputEquipment  InputType = "equipment"
	InputFuel       InputType = "fuel"

	UnitKG    InputUnit = "kg"
	UnitLiter InputUnit = "liter"
	UnitPiece InputUnit = "piece"
	UnitBag   InputUnit = "bag"
)

type InputPurchase struct {
	ID           uuid.UUID `json:"id"`
	PurchaseRef  string    `json:"purchase_ref"`
	FarmerID     uuid.UUID `json:"farmer_id"`
	CoopID       *uuid.UUID `json:"coop_id,omitempty"`
	InputType    InputType `json:"input_type"`
	InputName    string    `json:"input_name"`
	Quantity     float64   `json:"quantity"`
	Unit         InputUnit `json:"unit"`
	CostXOF      float64   `json:"cost_xof"`
	Supplier     string    `json:"supplier"`
	PurchasedAt  time.Time `json:"purchased_at"`
	TenantID     string    `json:"tenant_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type InputUsage struct {
	ID          uuid.UUID `json:"id"`
	PurchaseID  uuid.UUID `json:"purchase_id"`
	FarmerID    uuid.UUID `json:"farmer_id"`
	FieldID     string    `json:"field_id"`
	Quantity    float64   `json:"quantity"`
	UsedAt      time.Time `json:"used_at"`
	Notes       string    `json:"notes,omitempty"`
}

type CreatePurchaseRequest struct {
	FarmerID    uuid.UUID  `json:"farmer_id"`
	CoopID      *uuid.UUID `json:"coop_id"`
	InputType   InputType  `json:"input_type"`
	InputName   string     `json:"input_name"`
	Quantity    float64    `json:"quantity"`
	Unit        InputUnit  `json:"unit"`
	CostXOF     float64    `json:"cost_xof"`
	Supplier    string     `json:"supplier"`
	PurchasedAt time.Time  `json:"purchased_at"`
}

type RecordUsageRequest struct {
	FieldID  string    `json:"field_id"`
	Quantity float64   `json:"quantity"`
	UsedAt   time.Time `json:"used_at"`
	Notes    string    `json:"notes"`
}
