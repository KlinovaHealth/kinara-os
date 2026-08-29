package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/documentation-service/middleware"
	"github.com/klinova/kinara-os/documentation-service/models"
)

type Store interface {
	CreateDocument(ctx context.Context, d models.TradeDocument) error
	GetDocument(ctx context.Context, id uuid.UUID) (*models.TradeDocument, error)
	ListDocuments(ctx context.Context, docType *models.DocType, bookingRef *string) ([]models.TradeDocument, error)
	IssueDocument(ctx context.Context, id uuid.UUID, fileURL string, now time.Time) error
	RevokeDocument(ctx context.Context, id uuid.UUID, now time.Time) error
	InsertAuditLog(ctx context.Context, l models.DocumentAuditLog) error
}

type Handler struct{ store Store }

func NewHandler(store Store) *Handler         { return &Handler{store: store} }
func NewHandlerWithStore(s Store) *Handler    { return &Handler{store: s} }

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/documents", h.CreateDocument).Methods("POST")
	r.HandleFunc("/documents", h.ListDocuments).Methods("GET")
	r.HandleFunc("/documents/{id}", h.GetDocument).Methods("GET")
	r.HandleFunc("/documents/{id}/issue", h.IssueDocument).Methods("PUT")
	r.HandleFunc("/documents/{id}/revoke", h.RevokeDocument).Methods("PUT")
}

func respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req models.CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid request"})
		return
	}
	if req.DocType == "" || req.ShipperName == "" || req.ConsigneeName == "" || req.IssuingCountry == "" || req.IssuingAuthority == "" {
		respond(w, 400, models.APIResponse{Error: "document_type, shipper_name, consignee_name, issuing_country, issuing_authority required"})
		return
	}
	now := time.Now().UTC()
	id := uuid.New()
	ref := "TD-" + strings.ToUpper(id.String()[:10])
	doc := models.TradeDocument{
		ID:               id,
		DocumentRef:      ref,
		DocType:          models.DocType(req.DocType),
		ShipperName:      req.ShipperName,
		ConsigneeName:    req.ConsigneeName,
		BookingRef:       req.BookingRef,
		ManifestRef:      req.ManifestRef,
		IssuingCountry:   req.IssuingCountry,
		IssuingAuthority: req.IssuingAuthority,
		GoodsDescription: req.GoodsDescription,
		Value:            req.Value,
		Currency:         req.Currency,
		WeightKg:         req.WeightKg,
		NetWeightKg:      req.NetWeightKg,
		Packages:         req.Packages,
		Status:           models.DocDraft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if req.Currency == "" {
		doc.Currency = "USD"
	}
	if req.ExpiresAt != "" {
		t, _ := time.Parse(time.RFC3339, req.ExpiresAt)
		doc.ExpiresAt = &t
	}
	if err := h.store.CreateDocument(r.Context(), doc); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to create document"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.DocumentAuditLog{ID: uuid.New(), ActorID: claims.UserID, Action: "create_document", EntityType: "trade_document", EntityID: id, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: doc})
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	doc, err := h.store.GetDocument(r.Context(), id)
	if err != nil {
		respond(w, 404, models.APIResponse{Error: "document not found"})
		return
	}
	respond(w, 200, models.APIResponse{Success: true, Data: doc})
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	var docType *models.DocType
	var bookingRef *string
	if dt := r.URL.Query().Get("document_type"); dt != "" {
		t := models.DocType(dt)
		docType = &t
	}
	if br := r.URL.Query().Get("booking_ref"); br != "" {
		bookingRef = &br
	}
	docs, err := h.store.ListDocuments(r.Context(), docType, bookingRef)
	if err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to list documents"})
		return
	}
	if docs == nil {
		docs = []models.TradeDocument{}
	}
	respond(w, 200, models.APIResponse{Success: true, Data: docs})
}

func (h *Handler) IssueDocument(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	var req models.IssueDocumentRequest
	json.NewDecoder(r.Body).Decode(&req)
	now := time.Now().UTC()
	if err := h.store.IssueDocument(r.Context(), id, req.FileURL, now); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to issue document"})
		return
	}
	doc, _ := h.store.GetDocument(r.Context(), id)
	h.store.InsertAuditLog(r.Context(), models.DocumentAuditLog{ID: uuid.New(), ActorID: claims.UserID, Action: "issue_document", EntityType: "trade_document", EntityID: id, CreatedAt: now})
	respond(w, 200, models.APIResponse{Success: true, Data: doc})
}

func (h *Handler) RevokeDocument(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	now := time.Now().UTC()
	if err := h.store.RevokeDocument(r.Context(), id, now); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to revoke document"})
		return
	}
	doc, _ := h.store.GetDocument(r.Context(), id)
	h.store.InsertAuditLog(r.Context(), models.DocumentAuditLog{ID: uuid.New(), ActorID: claims.UserID, Action: "revoke_document", EntityType: "trade_document", EntityID: id, CreatedAt: now})
	respond(w, 200, models.APIResponse{Success: true, Data: doc})
}
