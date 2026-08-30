package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/klinova/kinara-os/extension-service/db"
	"github.com/klinova/kinara-os/extension-service/middleware"
	"github.com/klinova/kinara-os/extension-service/models"
)

// Prometheus counters.
var (
	consultationsBooked = promauto.NewCounter(prometheus.CounterOpts{Name: "extension_consultations_booked_total"})
	feedbackSubmitted   = promauto.NewCounter(prometheus.CounterOpts{Name: "extension_feedback_submitted_total"})
	resourcesAccessed   = promauto.NewCounter(prometheus.CounterOpts{Name: "extension_resources_accessed_total"})
)

// Store is the database interface used by Handler, enabling mock injection in tests.
type Store interface {
	ListResources(ctx context.Context, cropType, language string) ([]models.ExtensionResource, error)
	GetRecommendedResources(ctx context.Context, cropType string, limit int) ([]models.ExtensionResource, error)
	BookConsultation(ctx context.Context, c models.Consultation) error
	GetConsultation(ctx context.Context, id uuid.UUID) (*models.Consultation, error)
	InsertFeedback(ctx context.Context, f models.ExtensionFeedback) error
	GetBestPractices(ctx context.Context, cropType string) ([]models.BestPractice, error)
	InsertAudit(ctx context.Context, consultID, actorID, action string) error
}

// Handler holds the store and exposes HTTP handlers.
type Handler struct{ store Store }

// New wires up a real db.Queries-backed handler.
func New(q *db.Queries) *Handler { return &Handler{store: q} }

// NewWithStore allows injecting a mock Store in tests.
func NewWithStore(s Store) *Handler { return &Handler{store: s} }

// Register mounts all routes.
func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/extension").Subrouter()

	// Static sub-paths first to avoid mux conflicts with /{id} variants.
	api.HandleFunc("/resources/recommended", h.recommended).Methods(http.MethodGet)
	api.HandleFunc("/resources", h.listResources).Methods(http.MethodGet)
	api.HandleFunc("/consultations", h.bookConsultation).Methods(http.MethodPost)
	api.HandleFunc("/consultations/{id}", h.getConsultation).Methods(http.MethodGet)
	api.HandleFunc("/consultations/{id}/feedback", h.submitFeedback).Methods(http.MethodPost)
	api.HandleFunc("/best-practices/{crop_type}", h.bestPractices).Methods(http.MethodGet)
}

// ---------- GET /api/v1/extension/resources ----------

func (h *Handler) listResources(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	cropType := r.URL.Query().Get("crop_type")
	language := r.URL.Query().Get("language")
	items, err := h.store.ListResources(r.Context(), cropType, language)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.ExtensionResource{}
	}
	resourcesAccessed.Inc()
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

// ---------- GET /api/v1/extension/resources/recommended ----------

func (h *Handler) recommended(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	cropType := r.URL.Query().Get("crop_type")
	items, err := h.store.GetRecommendedResources(r.Context(), cropType, 5)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.ExtensionResource{}
	}
	resourcesAccessed.Inc()
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

// ---------- POST /api/v1/extension/consultations ----------

func (h *Handler) bookConsultation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("farmer", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req models.BookConsultationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if strings.TrimSpace(req.Topic) == "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": "topic required"})
		return
	}
	id := uuid.New()
	now := time.Now().UTC()
	c := models.Consultation{
		ID:            id,
		ConsultRef:    "EXT-" + strings.ToUpper(id.String()[:8]),
		FarmerID:      claims.UserID,
		Topic:         req.Topic,
		CropType:      req.CropType,
		PreferredDate: req.PreferredDate,
		Status:        "pending",
		TenantID:      claims.TenantID,
		BookedAt:      now,
	}
	if err := h.store.BookConsultation(r.Context(), c); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "book failed"})
		return
	}
	go h.store.InsertAudit(context.Background(), id.String(), claims.UserID.String(), "booked")
	consultationsBooked.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": c})
}

// ---------- GET /api/v1/extension/consultations/{id} ----------

func (h *Handler) getConsultation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	c, err := h.store.GetConsultation(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": c})
}

// ---------- POST /api/v1/extension/consultations/{id}/feedback ----------

func (h *Handler) submitFeedback(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("farmer", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	consultID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid consultation id"})
		return
	}
	var req models.FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		respond(w, http.StatusBadRequest, map[string]string{"error": "rating must be between 1 and 5"})
		return
	}
	fb := models.ExtensionFeedback{
		ID:             uuid.New(),
		ConsultationID: consultID,
		FarmerID:       claims.UserID,
		Rating:         req.Rating,
		Notes:          req.Notes,
		Result:         req.Result,
		SubmittedAt:    time.Now().UTC(),
	}
	if err := h.store.InsertFeedback(r.Context(), fb); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}
	go h.store.InsertAudit(context.Background(), consultID.String(), claims.UserID.String(), "feedback_submitted")
	feedbackSubmitted.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": fb})
}

// ---------- GET /api/v1/extension/best-practices/{crop_type} ----------

func (h *Handler) bestPractices(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	cropType := mux.Vars(r)["crop_type"]
	if cropType == "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": "crop_type required"})
		return
	}
	items, err := h.store.GetBestPractices(r.Context(), cropType)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.BestPractice{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

// ---------- helpers ----------

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
