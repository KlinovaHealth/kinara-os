package models

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryStatus string

const (
	DeliveryPending     DeliveryStatus = "pending"
	DeliveryAssigned    DeliveryStatus = "assigned"
	DeliveryEnRoute     DeliveryStatus = "en_route"
	DeliveryAttempted   DeliveryStatus = "attempted"
	DeliveryDelivered   DeliveryStatus = "delivered"
	DeliveryFailed      DeliveryStatus = "failed"
	DeliveryReturned    DeliveryStatus = "returned"
)

type FailureReason string

const (
	FailureNotHome      FailureReason = "not_home"
	FailureWrongAddress FailureReason = "wrong_address"
	FailureRefused      FailureReason = "refused"
	FailureAccessDenied FailureReason = "access_denied"
)

type Delivery struct {
	ID                  uuid.UUID      `json:"id"`
	DeliveryCode        string         `json:"delivery_code"`
	CargoID             *uuid.UUID     `json:"cargo_id,omitempty"`
	DriverID            *uuid.UUID     `json:"driver_id,omitempty"`
	RecipientName       string         `json:"recipient_name"`
	RecipientPhone      string         `json:"recipient_phone"`
	DeliveryAddress     string         `json:"delivery_address"`
	DeliveryLat         float64        `json:"delivery_lat"`
	DeliveryLng         float64        `json:"delivery_lng"`
	Status              DeliveryStatus `json:"status"`
	WindowStart         *time.Time     `json:"window_start,omitempty"`
	WindowEnd           *time.Time     `json:"window_end,omitempty"`
	AttemptCount        int            `json:"attempt_count"`
	DeliveredAt         *time.Time     `json:"delivered_at,omitempty"`
	ProofPhotoURL       string         `json:"proof_photo_url,omitempty"`
	SignatureURL        string         `json:"signature_url,omitempty"`
	FailureReason       FailureReason  `json:"failure_reason,omitempty"`
	NextAttemptAt       *time.Time     `json:"next_attempt_at,omitempty"`
	SMSNotified         bool           `json:"sms_notified"`
	Country             string         `json:"country"`
	Notes               string         `json:"notes,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type LastMileAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateDeliveryRequest struct {
	CargoID         string  `json:"cargo_id,omitempty"`
	RecipientName   string  `json:"recipient_name"`
	RecipientPhone  string  `json:"recipient_phone"`
	DeliveryAddress string  `json:"delivery_address"`
	DeliveryLat     float64 `json:"delivery_lat"`
	DeliveryLng     float64 `json:"delivery_lng"`
	WindowStart     string  `json:"window_start,omitempty"`
	WindowEnd       string  `json:"window_end,omitempty"`
	Country         string  `json:"country"`
	Notes           string  `json:"notes,omitempty"`
}

type AssignDriverRequest struct {
	DriverID string `json:"driver_id"`
}

type RecordDeliveryRequest struct {
	ProofPhotoURL string `json:"proof_photo_url,omitempty"`
	SignatureURL  string `json:"signature_url,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

type RecordFailureRequest struct {
	FailureReason FailureReason `json:"failure_reason"`
	NextAttemptAt string        `json:"next_attempt_at,omitempty"`
	Notes         string        `json:"notes,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
