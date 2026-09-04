package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/klinova/kinara-os/input-service/db"
	"github.com/klinova/kinara-os/input-service/middleware"
	"github.com/klinova/kinara-os/input-service/models"
)

var (
	formSubmissionsTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "input_form_submissions_total", Help: "Total form submissions"})
	validationErrors     = promauto.NewCounter(prometheus.CounterOpts{Name: "input_validation_errors_total", Help: "Total validation errors on submission"})
)

// Store is the handler's view of the database layer.
type Store = db.Store

// Handler holds the store interface.
type Handler struct{ store Store }

// New creates a Handler backed by a real *db.Queries.
func New(q *db.Queries) *Handler { return &Handler{store: q} }

// NewWithStore creates a Handler with any Store (for tests).
func NewWithStore(s Store) *Handler { return &Handler{store: s} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/input").Subrouter()
	// Form schema (must be before /forms/{form_type} to avoid ambiguity).
	api.HandleFunc("/forms/schema/{form_type}", h.getFormSchema).Methods(http.MethodGet)
	api.HandleFunc("/forms/{form_type}", h.getForm).Methods(http.MethodGet)
	// Submissions.
	api.HandleFunc("/submissions", h.submitForm).Methods(http.MethodPost)
	api.HandleFunc("/submissions/patient/{patient_id}", h.listByPatient).Methods(http.MethodGet)
	api.HandleFunc("/submissions/{id}", h.getSubmission).Methods(http.MethodGet)
	api.HandleFunc("/submissions/{id}", h.updateSubmission).Methods(http.MethodPut)
}

// GET /api/v1/input/forms/{form_type}
func (h *Handler) getForm(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	formType := mux.Vars(r)["form_type"]
	form, err := h.store.GetForm(r.Context(), formType)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": form})
}

// GET /api/v1/input/forms/schema/{form_type}
func (h *Handler) getFormSchema(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	formType := mux.Vars(r)["form_type"]
	form, err := h.store.GetForm(r.Context(), formType)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "schema": form.Schema, "form_type": form.FormType, "version": form.Version})
}

// POST /api/v1/input/submissions
func (h *Handler) submitForm(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("nurse", "frontdesk", "admin", "doctor") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req models.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.PatientID == uuid.Nil {
		validationErrors.Inc()
		respond(w, http.StatusBadRequest, map[string]string{"error": "patient_id is required"})
		return
	}
	if req.FormType == "" {
		validationErrors.Inc()
		respond(w, http.StatusBadRequest, map[string]string{"error": "form_type is required"})
		return
	}

	// Load form to get version (falls back to hardcoded if not in DB).
	form, err := h.store.GetForm(r.Context(), req.FormType)
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "unknown form_type"})
		return
	}

	id := uuid.New()
	now := time.Now().UTC()
	s := models.FormSubmission{
		ID:            id,
		SubmissionRef: "SUB-" + strings.ToUpper(id.String()[:8]),
		PatientID:     req.PatientID,
		FormType:      req.FormType,
		FormVersion:   form.Version,
		Data:          req.Data,
		SubmittedBy:   claims.UserID,
		TenantID:      claims.TenantID.String(),
		SubmittedAt:   now,
		UpdatedAt:     now,
	}

	if err := h.store.CreateSubmission(r.Context(), s); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	formSubmissionsTotal.Inc()

	// Fire-and-forget audit.
	go func() {
		ctx := context.Background()
		if err := h.store.InsertAudit(ctx, id.String(), claims.UserID.String(), "submit_form"); err != nil {
			slog.Error("audit insert", "error", err)
		}
	}()

	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": s})
}

// GET /api/v1/input/submissions/{id}
func (h *Handler) getSubmission(w http.ResponseWriter, r *http.Request) {
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

	sub, err := h.store.GetSubmission(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": sub})
}

// GET /api/v1/input/submissions/patient/{patient_id}
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

	items, err := h.store.ListByPatient(r.Context(), patientID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.FormSubmission{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

// PUT /api/v1/input/submissions/{id}
func (h *Handler) updateSubmission(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("nurse", "frontdesk", "admin", "doctor") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req models.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	now := time.Now().UTC()
	if err := h.store.UpdateSubmission(r.Context(), id, req.Data, now); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}

	// Fire-and-forget audit.
	go func() {
		ctx := context.Background()
		if err := h.store.InsertAudit(ctx, id.String(), claims.UserID.String(), "update_submission"); err != nil {
			slog.Error("audit insert", "error", err)
		}
	}()

	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
