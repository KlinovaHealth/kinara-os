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

	"github.com/klinova/kinara-os/immunization-service/db"
	"github.com/klinova/kinara-os/immunization-service/middleware"
	"github.com/klinova/kinara-os/immunization-service/models"
)

// Store is the interface the handler requires. *db.Queries satisfies it.
type Store interface {
	RecordImmunization(ctx context.Context, r models.ImmunizationRecord) error
	ListByPatient(ctx context.Context, patientID uuid.UUID) ([]models.ImmunizationRecord, error)
	GetSchedule(ctx context.Context, patientID uuid.UUID) ([]models.VaccineDue, error)
	InsertAlert(ctx context.Context, alert models.ImmunizationAlert) error
	GetClinicCompliance(ctx context.Context, clinicID string) (models.ComplianceReport, error)
	GetPopulationCoverage(ctx context.Context) ([]models.CoverageItem, error)
	InsertAudit(ctx context.Context, rec models.ImmunizationRecord, actorID string) error
}

var (
	immunizationsRecorded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "immunizations_recorded_total",
		Help: "Total number of immunizations recorded.",
	})
	alertsSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "immunization_alerts_sent_total",
		Help: "Total number of immunization alerts sent.",
	})
)

type Handler struct{ store Store }

// New constructs a Handler backed by a real *db.Queries.
func New(q *db.Queries) *Handler { return &Handler{store: q} }

// NewWithStore constructs a Handler backed by any Store (useful for testing).
func NewWithStore(s Store) *Handler { return &Handler{store: s} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/immunizations").Subrouter()
	api.HandleFunc("/record", h.record).Methods(http.MethodPost)
	api.HandleFunc("/patient/{patient_id}", h.listByPatient).Methods(http.MethodGet)
	api.HandleFunc("/patient/{patient_id}/schedule", h.getSchedule).Methods(http.MethodGet)
	api.HandleFunc("/alert", h.sendAlert).Methods(http.MethodPost)
	api.HandleFunc("/compliance/clinic/{clinic_id}", h.clinicCompliance).Methods(http.MethodGet)
	api.HandleFunc("/coverage", h.populationCoverage).Methods(http.MethodGet)
}

// POST /api/v1/immunizations/record
func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("nurse", "doctor", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req models.CreateImmunizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	id := uuid.New()
	rec := models.ImmunizationRecord{
		ID:              id,
		RecordRef:       "IMM-" + strings.ToUpper(id.String()[:8]),
		PatientID:       req.PatientID,
		VaccineCode:     req.VaccineCode,
		VaccineName:     req.VaccineName,
		DoseNumber:      req.DoseNumber,
		AdministeredBy:  req.AdministeredBy,
		AdministeredAt:  req.AdministeredAt,
		LotNumber:       req.LotNumber,
		ExpiryDate:      req.ExpiryDate,
		SiteOfInjection: req.SiteOfInjection,
		ClinicID:        req.ClinicID,
		NextDoseDate:    req.NextDoseDate,
		Status:          models.DoseAdministered,
		TenantID:        claims.TenantID,
		CreatedAt:       time.Now().UTC(),
	}

	if err := h.store.RecordImmunization(r.Context(), rec); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}

	// Audit log: INSERT, never query first.
	_ = h.store.InsertAudit(r.Context(), rec, claims.UserID.String())

	immunizationsRecorded.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": rec})
}

// GET /api/v1/immunizations/patient/{patient_id}
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

	records, err := h.store.ListByPatient(r.Context(), patientID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if records == nil {
		records = []models.ImmunizationRecord{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": records})
}

// GET /api/v1/immunizations/patient/{patient_id}/schedule
func (h *Handler) getSchedule(w http.ResponseWriter, r *http.Request) {
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

	schedule, err := h.store.GetSchedule(r.Context(), patientID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if schedule == nil {
		schedule = []models.VaccineDue{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": schedule})
}

// POST /api/v1/immunizations/alert
func (h *Handler) sendAlert(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req models.SendAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	alert := models.ImmunizationAlert{
		ID:        uuid.New(),
		PatientID: req.PatientID,
		Message:   req.Message,
		SentAt:    time.Now().UTC(),
	}

	if err := h.store.InsertAlert(r.Context(), alert); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}

	alertsSent.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": alert})
}

// GET /api/v1/immunizations/compliance/clinic/{clinic_id}
func (h *Handler) clinicCompliance(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	clinicID := mux.Vars(r)["clinic_id"]
	if clinicID == "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid clinic_id"})
		return
	}

	report, err := h.store.GetClinicCompliance(r.Context(), clinicID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": report})
}

// GET /api/v1/immunizations/coverage
func (h *Handler) populationCoverage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	coverage, err := h.store.GetPopulationCoverage(r.Context())
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if coverage == nil {
		coverage = []models.CoverageItem{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": coverage})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
