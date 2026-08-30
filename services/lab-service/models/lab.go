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
	FlagAbnormal ResultFlag = "abnormal"
	FlagCritical ResultFlag = "critical"
	FlagHigh     ResultFlag = "high"
	FlagLow      ResultFlag = "low"
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
	ID          uuid.UUID `json:"id"`
	OrderID     uuid.UUID `json:"order_id"`
	ResultValue float64   `json:"result_value"`
	Unit        string    `json:"unit"`
	NormalLow   float64   `json:"normal_low"`
	NormalHigh  float64   `json:"normal_high"`
	Flag        string    `json:"flag"` // "normal", "abnormal", "critical"
	Notes       string    `json:"notes,omitempty"`
	RecordedBy  uuid.UUID `json:"recorded_by"`
	RecordedAt  time.Time `json:"recorded_at"`
}

type LabResultWithOrder struct {
	Order  LabOrder   `json:"order"`
	Result *LabResult `json:"result,omitempty"`
}

type TestCatalogEntry struct {
	TestCode   string  `json:"test_code"`
	TestName   string  `json:"test_name"`
	NormalLow  float64 `json:"normal_low"`
	NormalHigh float64 `json:"normal_high"`
	Unit       string  `json:"unit"`
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

type UploadResultRequest struct {
	ResultValue float64 `json:"result_value"`
	Unit        string  `json:"unit"`
	NormalLow   float64 `json:"normal_low"`
	NormalHigh  float64 `json:"normal_high"`
	Notes       string  `json:"notes"`
}

// RecordResultRequest kept as alias for backward compat.
type RecordResultRequest = UploadResultRequest
