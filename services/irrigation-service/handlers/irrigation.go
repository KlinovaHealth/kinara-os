package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/klinova/kinara-os/irrigation-service/db"
	"github.com/klinova/kinara-os/irrigation-service/middleware"
	"github.com/klinova/kinara-os/irrigation-service/models"
)

var (
	irrigationSchedulesCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "irrigation_schedules_created_total",
		Help: "Total number of watering schedules created.",
	})
	irrigationAlertsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "irrigation_alerts_sent_total",
		Help: "Total number of irrigation alerts sent.",
	})
	irrigationRecsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "irrigation_recommendations_total",
		Help: "Total number of irrigation recommendations generated.",
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
	api := r.PathPrefix("/api/v1/irrigation").Subrouter()
	api.HandleFunc("/farms/{farm_id}/system", h.registerSystem).Methods(http.MethodPost)
	api.HandleFunc("/farms/{farm_id}/status", h.getFarmStatus).Methods(http.MethodGet)
	api.HandleFunc("/farms/{farm_id}/schedule", h.createSchedule).Methods(http.MethodPost)
	api.HandleFunc("/farms/{farm_id}/recommendation", h.getRecommendation).Methods(http.MethodGet)
	api.HandleFunc("/farms/{farm_id}/alert", h.sendAlert).Methods(http.MethodPost)
	api.HandleFunc("/farms/{farm_id}/history", h.getHistory).Methods(http.MethodGet)
	api.HandleFunc("/farms/{farm_id}/moisture", h.recordMoisture).Methods(http.MethodPost)
}

// POST /api/v1/irrigation/farms/{farm_id}/system
func (h *Handler) registerSystem(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmID := mux.Vars(r)["farm_id"]
	var req models.RegisterSystemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	sys := models.IrrigationSystem{
		ID:             id,
		FarmID:         farmID,
		SystemType:     req.SystemType,
		CapacityLiters: req.CapacityLiters,
		SensorID:       req.SensorID,
		TenantID:       claims.TenantID.String(),
		CreatedAt:      time.Now().UTC(),
	}
	if err := h.store.RegisterSystem(r.Context(), sys); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "register failed"})
		return
	}
	go func() {
		if err := h.store.InsertAudit(r.Context(), farmID, claims.UserID.String(), "register_system"); err != nil {
			slog.Error("audit insert failed", "error", err)
		}
	}()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": sys})
}

// GET /api/v1/irrigation/farms/{farm_id}/status
func (h *Handler) getFarmStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmID := mux.Vars(r)["farm_id"]
	sys, _ := h.store.GetSystem(r.Context(), farmID)
	moisture, _ := h.store.GetLatestMoisture(r.Context(), farmID)
	status := models.FarmStatus{
		FarmID:         farmID,
		System:         sys,
		LatestMoisture: moisture,
	}
	if sys == nil && moisture == nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "no data found for farm"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": status})
}

// POST /api/v1/irrigation/farms/{farm_id}/schedule
func (h *Handler) createSchedule(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmID := mux.Vars(r)["farm_id"]
	var req models.ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	sched := models.WateringSchedule{
		ID:             id,
		FarmID:         farmID,
		CronExpression: req.CronExpression,
		DurationMin:    req.DurationMin,
		CropType:       req.CropType,
		TenantID:       claims.TenantID.String(),
		CreatedAt:      time.Now().UTC(),
	}
	if err := h.store.CreateSchedule(r.Context(), sched); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create schedule failed"})
		return
	}
	irrigationSchedulesCreated.Inc()
	go func() {
		if err := h.store.InsertAudit(r.Context(), farmID, claims.UserID.String(), "create_schedule"); err != nil {
			slog.Error("audit insert failed", "error", err)
		}
	}()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": sched})
}

// GET /api/v1/irrigation/farms/{farm_id}/recommendation
func (h *Handler) getRecommendation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmID := mux.Vars(r)["farm_id"]

	moisture, err := h.store.GetLatestMoisture(r.Context(), farmID)
	var moisturePct float64
	if err == nil && moisture != nil {
		moisturePct = moisture.MoisturePct
	}

	// Placeholder weather data: simulate no rain expected by default.
	// In production this would call a weather API.
	rainExpected := false // weather_code != RAIN placeholder

	// Derive crop type from latest schedule if available.
	cropType := "unknown"

	rec := h.GetIrrigationRecommendation(moisturePct, rainExpected, cropType)
	rec.CurrentMoisturePct = moisturePct

	irrigationRecsTotal.Inc()
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": rec})
}

// GetIrrigationRecommendation is the pure recommendation logic, exported for test access.
func (h *Handler) GetIrrigationRecommendation(moisturePct float64, rainExpected bool, cropType string) models.IrrigationRec {
	return h.getIrrigationRecommendation(moisturePct, rainExpected, cropType)
}

// getIrrigationRecommendation is the internal recommendation logic.
func (h *Handler) getIrrigationRecommendation(moisturePct float64, rainExpected bool, cropType string) models.IrrigationRec {
	switch {
	case rainExpected:
		return models.IrrigationRec{ShouldIrrigate: false, Reason: "rain expected within 48h", RecommendedDurationMin: 0}
	case moisturePct > 80:
		return models.IrrigationRec{ShouldIrrigate: false, Reason: "soil saturated", RecommendedDurationMin: 0}
	case moisturePct < 30:
		return models.IrrigationRec{ShouldIrrigate: true, Reason: "soil moisture critically low", RecommendedDurationMin: 45, OptimalTime: "06:00"}
	case moisturePct < 50:
		return models.IrrigationRec{ShouldIrrigate: true, Reason: "soil moisture below optimal", RecommendedDurationMin: 30, OptimalTime: "06:00"}
	default:
		return models.IrrigationRec{ShouldIrrigate: false, Reason: "soil moisture adequate", RecommendedDurationMin: 0}
	}
}

// POST /api/v1/irrigation/farms/{farm_id}/alert
func (h *Handler) sendAlert(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmID := mux.Vars(r)["farm_id"]
	var req models.AlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	alert := models.IrrigationAlert{
		ID:        id,
		FarmID:    farmID,
		Message:   req.Message,
		AlertType: req.AlertType,
		SentAt:    time.Now().UTC(),
	}
	if err := h.store.InsertAlert(r.Context(), alert); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "alert failed"})
		return
	}
	irrigationAlertsTotal.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": alert})
}

// GET /api/v1/irrigation/farms/{farm_id}/history
func (h *Handler) getHistory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmID := mux.Vars(r)["farm_id"]
	history, err := h.store.GetHistory(r.Context(), farmID, 30)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if history == nil {
		history = []models.WateringHistory{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": history})
}

// POST /api/v1/irrigation/farms/{farm_id}/moisture
func (h *Handler) recordMoisture(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	farmID := mux.Vars(r)["farm_id"]
	var req models.MoistureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	reading := models.SoilMoistureReading{
		ID:          uuid.New(),
		FarmID:      farmID,
		MoisturePct: req.MoisturePct,
		SensorID:    req.SensorID,
		RecordedAt:  time.Now().UTC(),
	}
	if err := h.store.InsertMoisture(r.Context(), reading); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": reading})
}

// irrigationRef generates a short uppercase reference from a UUID.
func irrigationRef(id uuid.UUID) string {
	return "IRR-" + strings.ToUpper(id.String()[:8])
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
