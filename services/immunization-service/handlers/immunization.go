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

	"github.com/klinova/kinara-os/immunization-service/db"
	"github.com/klinova/kinara-os/immunization-service/middleware"
	"github.com/klinova/kinara-os/immunization-service/models"
)

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "immunization_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/immunizations").Subrouter()
	api.HandleFunc("", h.record).Methods(http.MethodPost)
	api.HandleFunc("/patient/{patient_id}", h.listByPatient).Methods(http.MethodGet)
	api.HandleFunc("/patient/{patient_id}/summary", h.summary).Methods(http.MethodGet)
}

func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "nurse", "doctor") {
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
	if err := h.queries.CreateRecord(r.Context(), rec); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/immunizations", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": rec})
}

func (h *Handler) listByPatient(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(mux.Vars(r)["patient_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid patient_id"})
		return
	}
	records, err := h.queries.ListByPatient(r.Context(), patientID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if records == nil {
		records = []models.ImmunizationRecord{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": records})
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(mux.Vars(r)["patient_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid patient_id"})
		return
	}
	records, err := h.queries.ListByPatient(r.Context(), patientID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	overdue, _ := h.queries.CountOverdue(r.Context(), patientID)
	if records == nil {
		records = []models.ImmunizationRecord{}
	}
	s := models.ImmunizationSummary{
		PatientID:    patientID,
		TotalDoses:   len(records),
		OverdueDoses: overdue,
		Records:      records,
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": s})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
