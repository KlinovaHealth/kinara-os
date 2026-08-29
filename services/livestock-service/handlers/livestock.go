package handlers

import (
	"encoding/json"
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

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "livestock_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/livestock").Subrouter()
	api.HandleFunc("/animals", h.register).Methods(http.MethodPost)
	api.HandleFunc("/animals/{id}", h.getAnimal).Methods(http.MethodGet)
	api.HandleFunc("/animals/{id}/health", h.updateHealth).Methods(http.MethodPut)
	api.HandleFunc("/animals/{id}/production", h.recordProduction).Methods(http.MethodPost)
	api.HandleFunc("/farmer/{farmer_id}/animals", h.listByFarmer).Methods(http.MethodGet)
	api.HandleFunc("/farmer/{farmer_id}/production", h.productionSummary).Methods(http.MethodGet)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req models.RegisterAnimalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	now := time.Now().UTC()
	a := models.Animal{
		ID:           id,
		TagRef:       "ANM-" + strings.ToUpper(id.String()[:8]),
		FarmerID:     req.FarmerID,
		Species:      req.Species,
		Breed:        req.Breed,
		BirthDate:    req.BirthDate,
		WeightKg:     req.WeightKg,
		HealthStatus: models.HealthHealthy,
		IsActive:     true,
		TenantID:     claims.TenantID,
		RegisteredAt: now,
		UpdatedAt:    now,
	}
	if err := h.queries.RegisterAnimal(r.Context(), a); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "register failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/livestock/animals", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": a})
}

func (h *Handler) getAnimal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	a, err := h.queries.GetAnimal(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": a})
}

func (h *Handler) updateHealth(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.UpdateHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.queries.UpdateHealth(r.Context(), id, req.HealthStatus, req.WeightKg, time.Now().UTC()); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

func (h *Handler) recordProduction(w http.ResponseWriter, r *http.Request) {
	animalID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	animal, err := h.queries.GetAnimal(r.Context(), animalID)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "animal not found"})
		return
	}
	var req models.RecordProductionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	p := models.ProductionRecord{
		ID:          uuid.New(),
		AnimalID:    animalID,
		FarmerID:    animal.FarmerID,
		ProductType: req.ProductType,
		QuantityKg:  req.QuantityKg,
		RecordedAt:  req.RecordedAt,
		Notes:       req.Notes,
	}
	if err := h.queries.RecordProduction(r.Context(), p); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "record failed"})
		return
	}
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": p})
}

func (h *Handler) listByFarmer(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid farmer_id"})
		return
	}
	animals, err := h.queries.ListByFarmer(r.Context(), farmerID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if animals == nil {
		animals = []models.Animal{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": animals})
}

func (h *Handler) productionSummary(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid farmer_id"})
		return
	}
	since := time.Now().UTC().AddDate(0, -1, 0)
	total, err := h.queries.SumProduction(r.Context(), farmerID, since)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{
		"success":              true,
		"farmer_id":            farmerID,
		"period":               "last_30_days",
		"total_production_kg":  total,
	})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
