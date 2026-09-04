package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/klinova/kinara-os/livestock-service/db"
	"github.com/klinova/kinara-os/livestock-service/middleware"
	"github.com/klinova/kinara-os/livestock-service/models"
)

var (
	animalsRegistered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "livestock_animals_registered_total",
		Help: "Total number of animals registered.",
	})
	healthEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "livestock_health_events_total",
		Help: "Total number of health events logged.",
	})
	productionRecords = promauto.NewCounter(prometheus.CounterOpts{
		Name: "livestock_production_records_total",
		Help: "Total number of production records logged.",
	})
)

// Store is the db.Store interface (re-exported for testability).
type Store = db.Store

// Handler holds the store dependency.
type Handler struct{ store Store }

// New constructs a Handler from the concrete *db.Queries.
func New(q *db.Queries) *Handler { return &Handler{store: q} }

// NewWithStore constructs a Handler from any Store (used in tests).
func NewWithStore(s Store) *Handler { return &Handler{store: s} }

// Register wires all routes.
func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/livestock").Subrouter()
	api.HandleFunc("/animals", h.registerAnimal).Methods(http.MethodPost)
	api.HandleFunc("/animals/{id}", h.getAnimal).Methods(http.MethodGet)
	api.HandleFunc("/animals/{id}/health", h.logHealthEvent).Methods(http.MethodPost)
	api.HandleFunc("/animals/{id}/health-history", h.getHealthHistory).Methods(http.MethodGet)
	api.HandleFunc("/animals/{id}/production", h.logProduction).Methods(http.MethodPost)
	api.HandleFunc("/farmers/{farmer_id}/herd", h.listHerd).Methods(http.MethodGet)
	api.HandleFunc("/farmers/{farmer_id}/analytics", h.herdAnalytics).Methods(http.MethodGet)
}

// POST /api/v1/livestock/animals
func (h *Handler) registerAnimal(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("farmer", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req models.RegisterAnimalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	animal := models.Animal{
		ID:           id,
		AnimalRef:    "ANM-" + strings.ToUpper(id.String()[:8]),
		FarmerID:     req.FarmerID,
		AnimalType:   req.AnimalType,
		Breed:        req.Breed,
		AgeMonths:    req.AgeMonths,
		Sex:          req.Sex,
		EarTag:       req.EarTag,
		TenantID:     claims.TenantID.String(),
		RegisteredAt: time.Now().UTC(),
	}
	if err := h.store.RegisterAnimal(r.Context(), animal); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "register failed"})
		return
	}
	animalsRegistered.Inc()
	go func() {
		if err := h.store.InsertAudit(r.Context(), id.String(), claims.UserID.String(), "register_animal"); err != nil {
			slog.Error("audit insert failed", "error", err)
		}
	}()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": animal})
}

// GET /api/v1/livestock/animals/{id}
func (h *Handler) getAnimal(w http.ResponseWriter, r *http.Request) {
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
	animal, err := h.store.GetAnimal(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": animal})
}

// POST /api/v1/livestock/animals/{id}/health
func (h *Handler) logHealthEvent(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("farmer", "vet", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	animalID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.HealthEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	event := models.HealthEvent{
		ID:             uuid.New(),
		AnimalID:       animalID,
		EventType:      req.EventType,
		Description:    req.Description,
		Treatment:      req.Treatment,
		VeterinarianID: req.VeterinarianID,
		EventDate:      time.Now().UTC(),
		CreatedBy:      claims.UserID,
	}
	if err := h.store.LogHealthEvent(r.Context(), event); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "log failed"})
		return
	}
	healthEventsTotal.Inc()
	// If illness, insert veterinary alert (fire-and-forget).
	if req.EventType == "illness" {
		go func() {
			alert := models.VeterinaryAlert{
				ID:        uuid.New(),
				AnimalID:  animalID,
				AlertType: "illness_detected",
				Priority:  "high",
				CreatedAt: time.Now().UTC(),
			}
			if err := h.store.InsertVetAlert(r.Context(), alert); err != nil {
				slog.Error("vet alert insert failed", "error", err)
			}
		}()
	}
	go func() {
		if err := h.store.InsertAudit(r.Context(), animalID.String(), claims.UserID.String(), "log_health_event"); err != nil {
			slog.Error("audit insert failed", "error", err)
		}
	}()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": event})
}

// GET /api/v1/livestock/animals/{id}/health-history
func (h *Handler) getHealthHistory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	animalID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	history, err := h.store.GetHealthHistory(r.Context(), animalID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if history == nil {
		history = []models.HealthEvent{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": history})
}

// POST /api/v1/livestock/animals/{id}/production
func (h *Handler) logProduction(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("farmer", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	animalID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.ProductionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	recordedDate := req.RecordedDate
	if recordedDate.IsZero() {
		recordedDate = time.Now().UTC()
	}
	rec := models.ProductionRecord{
		ID:             uuid.New(),
		AnimalID:       animalID,
		ProductionType: req.ProductionType,
		Quantity:       req.Quantity,
		Unit:           req.Unit,
		RecordedDate:   recordedDate,
		RecordedBy:     claims.UserID,
	}
	if err := h.store.LogProduction(r.Context(), rec); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "log failed"})
		return
	}
	productionRecords.Inc()
	go func() {
		if err := h.store.InsertAudit(r.Context(), animalID.String(), claims.UserID.String(), "log_production"); err != nil {
			slog.Error("audit insert failed", "error", err)
		}
	}()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": rec})
}

// GET /api/v1/livestock/farmers/{farmer_id}/herd
func (h *Handler) listHerd(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid farmer_id"})
		return
	}
	animals, err := h.store.ListHerd(r.Context(), farmerID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if animals == nil {
		animals = []models.Animal{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": animals})
}

// GET /api/v1/livestock/farmers/{farmer_id}/analytics
func (h *Handler) herdAnalytics(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid farmer_id"})
		return
	}
	analytics, err := h.store.GetHerdAnalytics(r.Context(), farmerID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": analytics})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
