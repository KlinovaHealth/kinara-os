package models

import (
	"time"
	"github.com/google/uuid"
)

type DocType string
type DocStatus string

const (
	DocCommercialInvoice DocType = "commercial_invoice"
	DocCertOrigin        DocType = "certificate_of_origin"
	DocPackingList       DocType = "packing_list"
	DocInsuranceCert     DocType = "insurance_certificate"
	DocBillOfLading      DocType = "bill_of_lading"
	DocPhytosanitary     DocType = "phytosanitary_certificate"
	DocHealthCert        DocType = "health_certificate"
	DocCustomsDecl       DocType = "customs_declaration"

	DocDraft     DocStatus = "draft"
	DocIssued    DocStatus = "issued"
	DocAmended   DocStatus = "amended"
	DocRevoked   DocStatus = "revoked"
)

type TradeDocument struct {
	ID              uuid.UUID `json:"id"`
	DocumentRef     string    `json:"document_ref"`
	DocType         DocType   `json:"document_type"`
	ShipperName     string    `json:"shipper_name"`
	ConsigneeName   string    `json:"consignee_name"`
	BookingRef      string    `json:"booking_ref,omitempty"`
	ManifestRef     string    `json:"manifest_ref,omitempty"`
	IssuingCountry  string    `json:"issuing_country"`
	IssuingAuthority string   `json:"issuing_authority"`
	GoodsDescription string   `json:"goods_description"`
	Value           float64   `json:"value"`
	Currency        string    `json:"currency"`
	WeightKg        float64   `json:"weight_kg"`
	NetWeightKg     float64   `json:"net_weight_kg"`
	Packages        int       `json:"packages"`
	Status          DocStatus `json:"status"`
	IssuedAt        *time.Time `json:"issued_at,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	FileURL         string    `json:"file_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type DocumentAuditLog struct {
	ID         uuid.UUID `json:"id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateDocumentRequest struct {
	DocType          string  `json:"document_type"`
	ShipperName      string  `json:"shipper_name"`
	ConsigneeName    string  `json:"consignee_name"`
	BookingRef       string  `json:"booking_ref,omitempty"`
	ManifestRef      string  `json:"manifest_ref,omitempty"`
	IssuingCountry   string  `json:"issuing_country"`
	IssuingAuthority string  `json:"issuing_authority"`
	GoodsDescription string  `json:"goods_description"`
	Value            float64 `json:"value"`
	Currency         string  `json:"currency"`
	WeightKg         float64 `json:"weight_kg"`
	NetWeightKg      float64 `json:"net_weight_kg"`
	Packages         int     `json:"packages"`
	ExpiresAt        string  `json:"expires_at,omitempty"`
}

type IssueDocumentRequest struct {
	FileURL string `json:"file_url,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
