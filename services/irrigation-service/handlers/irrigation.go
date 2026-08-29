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

	"github.com/klinova/kinara-os/irrigation-service/db"
	"github.com/klinova/kinara-os/irrigation-service/middleware"
	"github.com/klinova/kinara-os/irrigation-service/models"
)

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "irrigation_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/irrigation").Subrouter()
	api.HandleFunc("/schedules", h.createSchedule).Methods(http.MethodPost)
	api.HandleFunc("/schedules/farmer/{farmer_id}", h.listSchedules).Methods(http.MethodGet)
	api.HandleFunc("/events", h.logEvent).Methods(http.MethodPost)
	api.HandleFunc("/events/farmer/{farmer_id}", h.listEvents).Methods(http.MethodGet)
	api.HandleFunc("/farmer/{farmer_id}/water-usage", h.waterUsage).Methods(http.MethodGet)
}

func (h *Handler) createSchedule(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req models.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	now := time.Now().UTC()
	s := models.IrrigationSchedule{
		ID:            id,
		ScheduleRef:   "IRR-" + strings.ToUpper(id.String()[:8]),
		FarmerID:      req.FarmerID,
		FieldID:       req.FieldID,
		CropType:      req.CropType,
		Method:        req.Method,
		FrequencyDays: req.FrequencyDays,
		DurationMin:   req.DurationMin,
		WaterLiters:   req.WaterLiters,
		IsActive:      true,
		TenantID:      claims.TenantID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := h.queries.CreateSchedule(r.Context(), s); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/irrigation/schedules", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": s})
}

func (h *Handler) listSchedules(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid farmer_id"})
		return
	}
	items, err := h.queries.ListSchedules(r.Context(), farmerID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.IrrigationSchedule{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

func (h *Handler) logEvent(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req struct {
		ScheduleID uuid.UUID         `json:"schedule_id"`
		FarmerID   uuid.UUID         `json:"farmer_id"`
		FieldID    string            `json:"field_id"`
		models.LogEventRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	_ = claims
	e := models.IrrigationEvent{
		ID:          uuid.New(),
		ScheduleID:  req.ScheduleID,
		FarmerID:    req.FarmerID,
		FieldID:     req.FieldID,
		ScheduledAt: req.ScheduledAt,
		WaterUsedL:  req.WaterUsedL,
		Status:      req.Status,
		Notes:       req.Notes,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.queries.LogEvent(r.Context(), e); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "log failed"})
		return
	}
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": e})
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid farmer_id"})
		return
	}
	events, err := h.queries.ListEvents(r.Context(), farmerID, 50)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if events == nil {
		events = []models.IrrigationEvent{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": events})
}

func (h *Handler) waterUsage(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid farmer_id"})
		return
	}
	since := time.Now().UTC().AddDate(0, -1, 0)
	total, err := h.queries.SumWaterUsed(r.Context(), farmerID, since)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{
		"success":           true,
		"farmer_id":         farmerID,
		"period":            "last_30_days",
		"water_used_liters": total,
	})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
