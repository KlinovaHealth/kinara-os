package models

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string
type ResultFlag string

const (
	OrderPending    OrderStatus = "pending"
	OrderCollected  OrderStatus = "collected"
	OrderProcessing OrderStatus = "processing"
	OrderCompleted  OrderStatus = "completed"
	OrderCancelled  OrderStatus = "cancelled"

	FlagNormal   ResultFlag = "normal"
	FlagHigh     ResultFlag = "high"
	FlagLow      ResultFlag = "low"
	FlagCritical ResultFlag = "critical"
)

type LabOrder struct {
	ID          uuid.UUID   `json:"id"`
	OrderRef    string      `json:"order_ref"`
	PatientID   uuid.UUID   `json:"patient_id"`
	OrderedBy   uuid.UUID   `json:"ordered_by"`
	ClinicID    string      `json:"clinic_id"`
	TestCode    string      `json:"test_code"`
	TestName    string      `json:"test_name"`
	Priority    string      `json:"priority"`
	Status      OrderStatus `json:"status"`
	Notes       string      `json:"notes,omitempty"`
	TenantID    string      `json:"tenant_id"`
	OrderedAt   time.Time   `json:"ordered_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

type LabResult struct {
	ID             uuid.UUID  `json:"id"`
	OrderID        uuid.UUID  `json:"order_id"`
	PatientID      uuid.UUID  `json:"patient_id"`
	TestCode       string     `json:"test_code"`
	ResultValue    string     `json:"result_value"`
	Unit           string     `json:"unit"`
	ReferenceRange string     `json:"reference_range"`
	Flag           ResultFlag `json:"flag"`
	AnalyzedBy     uuid.UUID  `json:"analyzed_by"`
	ResultAt       time.Time  `json:"result_at"`
	TenantID       string     `json:"tenant_id"`
}

type CreateOrderRequest struct {
	PatientID uuid.UUID `json:"patient_id"`
	OrderedBy uuid.UUID `json:"ordered_by"`
	ClinicID  string    `json:"clinic_id"`
	TestCode  string    `json:"test_code"`
	TestName  string    `json:"test_name"`
	Priority  string    `json:"priority"`
	Notes     string    `json:"notes"`
}

type RecordResultRequest struct {
	ResultValue    string     `json:"result_value"`
	Unit           string     `json:"unit"`
	ReferenceRange string     `json:"reference_range"`
	Flag           ResultFlag `json:"flag"`
	AnalyzedBy     uuid.UUID  `json:"analyzed_by"`
}
