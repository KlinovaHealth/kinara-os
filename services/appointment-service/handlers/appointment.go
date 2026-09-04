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

	"github.com/klinova/kinara-os/appointment-service/db"
	"github.com/klinova/kinara-os/appointment-service/middleware"
	"github.com/klinova/kinara-os/appointment-service/models"
)

// Prometheus counters.
var (
	apptCreated     = promauto.NewCounter(prometheus.CounterOpts{Name: "appointment_created_total"})
	apptCancelled   = promauto.NewCounter(prometheus.CounterOpts{Name: "appointment_cancelled_total"})
	apptCompleted   = promauto.NewCounter(prometheus.CounterOpts{Name: "appointment_completed_total"})
	apptRescheduled = promauto.NewCounter(prometheus.CounterOpts{Name: "appointment_rescheduled_total"})
)

// Store is the database interface used by Handler, enabling mock injection in tests.
type Store interface {
	CreateAppointment(ctx context.Context, a models.Appointment) error
	GetAppointment(ctx context.Context, id uuid.UUID) (*models.Appointment, error)
	ListAppointments(ctx context.Context, patientID, doctorID *uuid.UUID, tenantID string, limit int) ([]models.Appointment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.AppointmentStatus, notes string, now time.Time) error
	RescheduleAppointment(ctx context.Context, id uuid.UUID, newTime time.Time, durationMin int, now time.Time) error
	CancelAppointment(ctx context.Context, id uuid.UUID, reason, actorID string, now time.Time) error
	CompleteAppointment(ctx context.Context, id uuid.UUID, notes, actorID string, now time.Time) error
	ListByClinic(ctx context.Context, clinicID, tenantID string, date *time.Time, status *string, limit int) ([]models.Appointment, error)
	ListByPatient(ctx context.Context, patientID uuid.UUID, tenantID string, limit int) ([]models.Appointment, error)
	InsertAudit(ctx context.Context, entry models.AuditEntry) error
	GetAuditHistory(ctx context.Context, apptID uuid.UUID) ([]models.AuditEntry, error)
}

// Handler holds the store and exposes HTTP handlers.
type Handler struct{ store Store }

// New wires up a real db.Queries-backed handler.
func New(q *db.Queries) *Handler { return &Handler{store: q} }

// NewWithStore allows injecting a mock Store in tests.
func NewWithStore(s Store) *Handler { return &Handler{store: s} }

// Register mounts all routes. Specific sub-paths must precede /{id} to avoid mux conflicts.
func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/appointments").Subrouter()

	// Static-sub-path routes first.
	api.HandleFunc("/clinic/{clinic_id}", h.listByClinic).Methods(http.MethodGet)
	api.HandleFunc("/patient/{patient_id}", h.listByPatient).Methods(http.MethodGet)

	// Collection route.
	api.HandleFunc("", h.create).Methods(http.MethodPost)

	// Dynamic-ID routes.
	api.HandleFunc("/{id}", h.get).Methods(http.MethodGet)
	api.HandleFunc("/{id}", h.reschedule).Methods(http.MethodPut)
	api.HandleFunc("/{id}/cancel", h.cancel).Methods(http.MethodPost)
	api.HandleFunc("/{id}/complete", h.complete).Methods(http.MethodPut)
	api.HandleFunc("/{id}/history", h.history).Methods(http.MethodGet)
}

// ---------- POST /api/v1/appointments ----------

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
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
	if req.PatientID == uuid.Nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "patient_id required"})
		return
	}
	if req.DoctorID == uuid.Nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "doctor_id required"})
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
		TenantID:       claims.TenantID.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := h.store.CreateAppointment(r.Context(), a); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	h.audit(r.Context(), id.String(), claims.UserID.String(), "created", "", string(models.StatusScheduled))
	apptCreated.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": a})
}

// ---------- GET /api/v1/appointments/{id} ----------

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
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
	a, err := h.store.GetAppointment(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": a})
}

// ---------- PUT /api/v1/appointments/{id} (reschedule) ----------

func (h *Handler) reschedule(w http.ResponseWriter, r *http.Request) {
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
	var req models.RescheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.ScheduledAt.IsZero() {
		respond(w, http.StatusBadRequest, map[string]string{"error": "scheduled_at required"})
		return
	}
	if req.DurationMin <= 0 {
		req.DurationMin = 30
	}
	now := time.Now().UTC()
	if err := h.store.RescheduleAppointment(r.Context(), id, req.ScheduledAt, req.DurationMin, now); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "reschedule failed"})
		return
	}
	h.audit(r.Context(), id.String(), claims.UserID.String(), "rescheduled", "", "")
	apptRescheduled.Inc()
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

// ---------- GET /api/v1/appointments/clinic/{clinic_id} ----------

func (h *Handler) listByClinic(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	clinicID := mux.Vars(r)["clinic_id"]
	if clinicID == "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": "clinic_id required"})
		return
	}

	var date *time.Time
	if v := r.URL.Query().Get("date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			date = &t
		}
	}

	var status *string
	if v := r.URL.Query().Get("status"); v != "" {
		status = &v
	}

	items, err := h.store.ListByClinic(r.Context(), clinicID, claims.TenantID.String(), date, status, 100)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.Appointment{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

// ---------- GET /api/v1/appointments/patient/{patient_id} ----------

func (h *Handler) listByPatient(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	patientID, err := uuid.Parse(mux.Vars(r)["patient_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid patient_id"})
		return
	}
	items, err := h.store.ListByPatient(r.Context(), patientID, claims.TenantID.String(), 100)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.Appointment{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

// ---------- POST /api/v1/appointments/{id}/cancel ----------

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "doctor", "nurse", "frontdesk") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	// Fetch current status for audit.
	existing, err := h.store.GetAppointment(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	oldStatus := string(existing.Status)

	var req models.CancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	actorID := claims.UserID.String()
	now := time.Now().UTC()
	if err := h.store.CancelAppointment(r.Context(), id, req.Reason, actorID, now); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "cancel failed"})
		return
	}
	h.audit(r.Context(), id.String(), actorID, "cancelled", oldStatus, string(models.StatusCancelled))
	apptCancelled.Inc()
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

// ---------- PUT /api/v1/appointments/{id}/complete ----------

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "doctor", "nurse") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	// Fetch current status for audit.
	existing, err := h.store.GetAppointment(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	oldStatus := string(existing.Status)

	var req models.CompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	actorID := claims.UserID.String()
	now := time.Now().UTC()
	if err := h.store.CompleteAppointment(r.Context(), id, req.Notes, actorID, now); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "complete failed"})
		return
	}
	h.audit(r.Context(), id.String(), actorID, "completed", oldStatus, string(models.StatusCompleted))
	apptCompleted.Inc()
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

// ---------- GET /api/v1/appointments/{id}/history ----------

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
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
	entries, err := h.store.GetAuditHistory(r.Context(), id)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if entries == nil {
		entries = []models.AuditEntry{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": entries})
}

// ---------- helpers ----------

// audit fire-and-forgets an audit log insert. Errors are non-fatal.
func (h *Handler) audit(ctx context.Context, apptID, actorID, action, oldStatus, newStatus string) {
	h.store.InsertAudit(ctx, models.AuditEntry{
		AppointmentID: apptID,
		Action:        action,
		ActorID:       actorID,
		OldStatus:     oldStatus,
		NewStatus:     newStatus,
	})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
