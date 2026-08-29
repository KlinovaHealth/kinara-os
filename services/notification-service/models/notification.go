package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Channel ──────────────────────────────────────────────────────────────────

type NotificationChannel string

const (
	ChannelSMS       NotificationChannel = "sms"
	ChannelPush      NotificationChannel = "push"
	ChannelWhatsApp  NotificationChannel = "whatsapp"
	ChannelEmail     NotificationChannel = "email"
	ChannelInApp     NotificationChannel = "in_app"
)

// ─── Type ─────────────────────────────────────────────────────────────────────

type NotificationType string

const (
	// Health pillar
	TypeAppointmentReminder NotificationType = "appointment_reminder"
	TypePrescriptionAlert   NotificationType = "prescription_alert"
	TypeReferralStatus      NotificationType = "referral_status"
	TypeLabResult           NotificationType = "lab_result"
	TypeVaccineReminder     NotificationType = "vaccine_reminder"
	// Agriculture pillar
	TypePriceAlert          NotificationType = "price_alert"
	TypeWeatherAlert        NotificationType = "weather_alert"
	TypeMarketOpportunity   NotificationType = "market_opportunity"
	TypeHarvestReminder     NotificationType = "harvest_reminder"
	// Logistics pillar
	TypeFleetAlert          NotificationType = "fleet_alert"
	TypeDeliveryStatus      NotificationType = "delivery_status"
	TypeRouteChange         NotificationType = "route_change"
	// Maritime pillar
	TypePortAlert           NotificationType = "port_alert"
	TypeVesselStatus        NotificationType = "vessel_status"
	TypeCustomsClearance    NotificationType = "customs_clearance"
	// System
	TypeSystemAlert         NotificationType = "system_alert"
	TypeSecurityAlert       NotificationType = "security_alert"
)

// ─── Status ───────────────────────────────────────────────────────────────────

type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusQueued    NotificationStatus = "queued"
	StatusSent      NotificationStatus = "sent"
	StatusDelivered NotificationStatus = "delivered"
	StatusFailed    NotificationStatus = "failed"
	StatusCancelled NotificationStatus = "cancelled"
)

// ─── Priority ─────────────────────────────────────────────────────────────────

type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// ─── Notification ─────────────────────────────────────────────────────────────

// NotificationRow is the DB record with encrypted message content.
type NotificationRow struct {
	ID             uuid.UUID            `db:"id"`
	UserID         uuid.UUID            `db:"user_id"`
	Type           NotificationType     `db:"type"`
	Channel        NotificationChannel  `db:"channel"`
	Priority       NotificationPriority `db:"priority"`
	MessageEnc     string               `db:"message_enc"`
	SubjectEnc     string               `db:"subject_enc"`
	RecipientEnc   string               `db:"recipient_enc"` // phone, email, or push token
	TemplateID     *uuid.UUID           `db:"template_id"`
	Status         NotificationStatus   `db:"status"`
	RetryCount     int                  `db:"retry_count"`
	ScheduledAt    *time.Time           `db:"scheduled_at"`
	SentAt         *time.Time           `db:"sent_at"`
	DeliveredAt    *time.Time           `db:"delivered_at"`
	FailureReason  string               `db:"failure_reason"`
	ExternalID     string               `db:"external_id"` // provider message ID
	CreatedAt      time.Time            `db:"created_at"`
	UpdatedAt      time.Time            `db:"updated_at"`
}

// Notification is the decrypted API representation.
type Notification struct {
	ID            uuid.UUID            `json:"id"`
	UserID        uuid.UUID            `json:"user_id"`
	Type          NotificationType     `json:"type"`
	Channel       NotificationChannel  `json:"channel"`
	Priority      NotificationPriority `json:"priority"`
	Message       string               `json:"message"`
	Subject       string               `json:"subject,omitempty"`
	Recipient     string               `json:"recipient,omitempty"`
	TemplateID    *uuid.UUID           `json:"template_id,omitempty"`
	Status        NotificationStatus   `json:"status"`
	RetryCount    int                  `json:"retry_count"`
	ScheduledAt   *time.Time           `json:"scheduled_at,omitempty"`
	SentAt        *time.Time           `json:"sent_at,omitempty"`
	DeliveredAt   *time.Time           `json:"delivered_at,omitempty"`
	FailureReason string               `json:"failure_reason,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
}

// ─── Template ─────────────────────────────────────────────────────────────────

type NotificationTemplate struct {
	ID           uuid.UUID        `json:"id" db:"id"`
	Type         NotificationType `json:"type" db:"type"`
	Channel      NotificationChannel `json:"channel" db:"channel"`
	Name         string           `json:"name" db:"name"`
	SubjectTpl   string           `json:"subject_template" db:"subject_template"`
	BodyTpl      string           `json:"body_template" db:"body_template"`
	Variables    []string         `json:"variables" db:"variables"`
	Language     string           `json:"language" db:"language"`
	IsActive     bool             `json:"is_active" db:"is_active"`
	CreatedAt    time.Time        `json:"created_at" db:"created_at"`
}

// ─── User preferences ─────────────────────────────────────────────────────────

type UserPreferences struct {
	ID               uuid.UUID `json:"id" db:"id"`
	UserID           uuid.UUID `json:"user_id" db:"user_id"`
	SMSEnabled       bool      `json:"sms_enabled" db:"sms_enabled"`
	PushEnabled      bool      `json:"push_enabled" db:"push_enabled"`
	WhatsAppEnabled  bool      `json:"whatsapp_enabled" db:"whatsapp_enabled"`
	EmailEnabled     bool      `json:"email_enabled" db:"email_enabled"`
	InAppEnabled     bool      `json:"in_app_enabled" db:"in_app_enabled"`
	QuietHoursStart  *string   `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"` // "22:00"
	QuietHoursEnd    *string   `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"`     // "07:00"
	TimeZone         string    `json:"timezone" db:"timezone"`
	Language         string    `json:"language" db:"language"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// ─── Audit log ────────────────────────────────────────────────────────────────

type NotificationAuditLog struct {
	ID             uuid.UUID  `json:"id"`
	NotificationID *uuid.UUID `json:"notification_id,omitempty"`
	UserID         uuid.UUID  `json:"user_id"`
	Action         string     `json:"action"`
	Resource       string     `json:"resource"`
	IPAddress      string     `json:"ip_address"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ─── Request types ────────────────────────────────────────────────────────────

type SendNotificationRequest struct {
	UserID    string               `json:"user_id"`
	Type      NotificationType     `json:"type"`
	Channel   NotificationChannel  `json:"channel"`
	Priority  NotificationPriority `json:"priority"`
	Subject   string               `json:"subject,omitempty"`
	Message   string               `json:"message"`
	Recipient string               `json:"recipient"` // phone/email/push token
}

type ScheduleNotificationRequest struct {
	UserID      string               `json:"user_id"`
	Type        NotificationType     `json:"type"`
	Channel     NotificationChannel  `json:"channel"`
	Priority    NotificationPriority `json:"priority"`
	Subject     string               `json:"subject,omitempty"`
	Message     string               `json:"message"`
	Recipient   string               `json:"recipient"`
	ScheduledAt string               `json:"scheduled_at"` // RFC3339
}

type BulkSendRequest struct {
	Type      NotificationType     `json:"type"`
	Channel   NotificationChannel  `json:"channel"`
	Priority  NotificationPriority `json:"priority"`
	Subject   string               `json:"subject,omitempty"`
	Message   string               `json:"message"`
	UserIDs   []string             `json:"user_ids"`
}

type UpdatePreferencesRequest struct {
	SMSEnabled      *bool   `json:"sms_enabled,omitempty"`
	PushEnabled     *bool   `json:"push_enabled,omitempty"`
	WhatsAppEnabled *bool   `json:"whatsapp_enabled,omitempty"`
	EmailEnabled    *bool   `json:"email_enabled,omitempty"`
	InAppEnabled    *bool   `json:"in_app_enabled,omitempty"`
	QuietHoursStart *string `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd   *string `json:"quiet_hours_end,omitempty"`
	TimeZone        *string `json:"timezone,omitempty"`
	Language        *string `json:"language,omitempty"`
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
