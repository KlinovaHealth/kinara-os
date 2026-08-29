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

	"github.com/klinova/kinara-os/appointment-service/db"
	"github.com/klinova/kinara-os/appointment-service/middleware"
	"github.com/klinova/kinara-os/appointment-service/models"
)

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "appointment_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/appointments").Subrouter()
	api.HandleFunc("", h.create).Methods(http.MethodPost)
	api.HandleFunc("", h.list).Methods(http.MethodGet)
	api.HandleFunc("/{id}", h.get).Methods(http.MethodGet)
	api.HandleFunc("/{id}/status", h.updateStatus).Methods(http.MethodPut)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "nurse", "doctor", "frontdesk") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req models.CreateAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.PatientID == uuid.Nil || req.DoctorID == uuid.Nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "patient_id and doctor_id required"})
		return
	}
	if req.DurationMin <= 0 {
		req.DurationMin = 30
	}
	id := uuid.New()
	now := time.Now().UTC()
	a := models.Appointment{
		ID:             id,
		AppointmentRef: "APT-" + strings.ToUpper(id.String()[:8]),
		PatientID:      req.PatientID,
		DoctorID:       req.DoctorID,
		ClinicID:       req.ClinicID,
		ScheduledAt:    req.ScheduledAt,
		DurationMin:    req.DurationMin,
		Type:           req.Type,
		Status:         models.StatusScheduled,
		Notes:          req.Notes,
		TenantID:       claims.TenantID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := h.queries.CreateAppointment(r.Context(), a); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/appointments", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": a})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var patientID, doctorID *uuid.UUID
	if v := r.URL.Query().Get("patient_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			patientID = &id
		}
	}
	if v := r.URL.Query().Get("doctor_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			doctorID = &id
		}
	}
	items, err := h.queries.ListAppointments(r.Context(), patientID, doctorID, claims.TenantID, 50)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.Appointment{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	a, err := h.queries.GetAppointment(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": a})
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	_ = claims
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.queries.UpdateStatus(r.Context(), id, req.Status, req.Notes, time.Now().UTC()); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
