package models

import (
	"time"

	"github.com/google/uuid"
)

type ExtensionResource struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	ContentSummary string    `json:"content_summary"`
	CropType       string    `json:"crop_type"`
	Language       string    `json:"language"`     // "en", "fr"
	ResourceType   string    `json:"resource_type"` // "guide", "video", "checklist"
	ViewedCount    int       `json:"viewed_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type Consultation struct {
	ID            uuid.UUID  `json:"id"`
	ConsultRef    string     `json:"consult_ref"`
	FarmerID      uuid.UUID  `json:"farmer_id"`
	OfficerID     *uuid.UUID `json:"officer_id,omitempty"`
	Topic         string     `json:"topic"`
	CropType      string     `json:"crop_type,omitempty"`
	PreferredDate *time.Time `json:"preferred_date,omitempty"`
	Status        string     `json:"status"` // "pending", "scheduled", "completed"
	Notes         string     `json:"notes,omitempty"`
	TenantID      string     `json:"tenant_id"`
	BookedAt      time.Time  `json:"booked_at"`
}

type ExtensionFeedback struct {
	ID             uuid.UUID `json:"id"`
	ConsultationID uuid.UUID `json:"consultation_id"`
	FarmerID       uuid.UUID `json:"farmer_id"`
	Rating         int       `json:"rating"` // 1-5
	Notes          string    `json:"notes,omitempty"`
	Result         string    `json:"result,omitempty"`
	SubmittedAt    time.Time `json:"submitted_at"`
}

type BestPractice struct {
	ID                       uuid.UUID `json:"id"`
	CropType                 string    `json:"crop_type"`
	Technique                string    `json:"technique"`
	Description              string    `json:"description"`
	ExpectedYieldImprovement float64   `json:"expected_yield_improvement_pct"`
	Climate                  string    `json:"climate"` // "arid", "tropical", "semi-arid"
}

type BookConsultationRequest struct {
	Topic         string     `json:"topic"`
	CropType      string     `json:"crop_type"`
	PreferredDate *time.Time `json:"preferred_date"`
	FarmID        string     `json:"farm_id"`
}

type FeedbackRequest struct {
	Rating int    `json:"rating"`
	Notes  string `json:"notes"`
	Result string `json:"result"`
}
