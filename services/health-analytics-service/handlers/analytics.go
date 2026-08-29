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
	"github.com/klinova/kinara-os/health-analytics-service/db"
	"github.com/klinova/kinara-os/health-analytics-service/models"
)

type Store interface {
	ReportDisease(ctx context.Context, r models.DiseaseReport) error
	ListDiseases(ctx context.Context, p db.ListDiseaseParams) ([]models.DiseaseReport, error)
	CreateOutbreakAlert(ctx context.Context, a models.OutbreakAlert) error
	ListActiveAlerts(ctx context.Context, country *string) ([]models.OutbreakAlert, error)
	ResolveAlert(ctx context.Context, id uuid.UUID, now time.Time) error
	RecordClinicMetric(ctx context.Context, m models.ClinicMetric) error
	GetClinicMetrics(ctx context.Context, clinicID uuid.UUID, limit int) ([]models.ClinicMetric, error)
	GetImpactSummary(ctx context.Context, country string) (*models.ImpactSummary, error)
	InsertAuditLog(ctx context.Context, l models.HealthAnalyticsAuditLog) error
}

// outbreakThreshold: 3+ cases of same disease in one clinic within period triggers alert.
const outbreakThreshold = 3

type Handler struct {
	store  Store
	logger *slog.Logger
}

func NewHandler(q *db.Queries, logger *slog.Logger) *Handler {
	return &Handler{store: q, logger: logger}
}

func NewHandlerWithStore(s Store) *Handler {
	return &Handler{store: s, logger: slog.Default()}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.health).Methods(http.MethodGet)
	api := r.PathPrefix("/api/v1/health-analytics").Subrouter()
	api.HandleFunc("/diseases", h.reportDisease).Methods(http.MethodPost)
	api.HandleFunc("/diseases", h.listDiseases).Methods(http.MethodGet)
	api.HandleFunc("/outbreaks", h.listActiveAlerts).Methods(http.MethodGet)
	api.HandleFunc("/outbreaks/{id}/resolve", h.resolveAlert).Methods(http.MethodPut)
	api.HandleFunc("/clinics/{id}/metrics", h.recordClinicMetric).Methods(http.MethodPost)
	api.HandleFunc("/clinics/{id}/metrics", h.getClinicMetrics).Methods(http.MethodGet)
	api.HandleFunc("/clinics/{id}/performance", h.getClinicPerformance).Methods(http.MethodGet)
	api.HandleFunc("/impact", h.getImpact).Methods(http.MethodGet)
	api.HandleFunc("/population", h.getPopulationHealth).Methods(http.MethodGet)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "health-analytics-service"})
}

func (h *Handler) reportDisease(w http.ResponseWriter, r *http.Request) {
	var req models.ReportDiseaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Country == "" || req.ICD10Code == "" || req.DiseaseName == "" || req.ClinicID == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "country, icd10_code, disease_name, and clinic_id are required")
		return
	}
	clinicID, err := uuid.Parse(req.ClinicID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid clinic_id")
		return
	}
	period := req.Period
	if period == "" {
		period = "daily"
	}
	now := time.Now().UTC()
	periodStart := now.Truncate(24 * time.Hour)
	periodEnd := periodStart.Add(24 * time.Hour)
	if req.PeriodStart != "" {
		if t, err := time.Parse(time.RFC3339, req.PeriodStart); err == nil {
			periodStart = t
		}
	}
	if req.PeriodEnd != "" {
		if t, err := time.Parse(time.RFC3339, req.PeriodEnd); err == nil {
			periodEnd = t
		}
	}
	severity := models.DiseaseSeverity(req.Severity)
	if severity == "" {
		severity = models.SeverityMild
	}
	caseCount := req.CaseCount
	if caseCount <= 0 {
		caseCount = 1
	}
	report := models.DiseaseReport{
		ID:          uuid.New(),
		ClinicID:    clinicID,
		Country:     strings.ToUpper(req.Country),
		Region:      req.Region,
		ICD10Code:   req.ICD10Code,
		DiseaseName: req.DiseaseName,
		CaseCount:   caseCount,
		Period:      period,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Severity:    severity,
		CreatedAt:   now,
	}
	if err := h.store.ReportDisease(r.Context(), report); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to record disease report")
		return
	}
	// Auto-detect outbreak: threshold cases in same clinic/disease
	if caseCount >= outbreakThreshold || severity == models.SeverityCritical {
		alertID := uuid.New()
		alert := models.OutbreakAlert{
			ID:          alertID,
			AlertRef:    "OA-" + strings.ToUpper(alertID.String()[:8]),
			ClinicID:    clinicID,
			Country:     strings.ToUpper(req.Country),
			Region:      req.Region,
			ICD10Code:   req.ICD10Code,
			DiseaseName: req.DiseaseName,
			CaseCount:   caseCount,
			Threshold:   outbreakThreshold,
			Status:      models.OutbreakActive,
			DetectedAt:  now,
		}
		_ = h.store.CreateOutbreakAlert(r.Context(), alert)
	}
	h.audit(r.Context(), "report_disease", "disease_report:"+report.ID.String())
	respond(w, http.StatusCreated, report)
}

func (h *Handler) listDiseases(w http.ResponseWriter, r *http.Request) {
	p := db.ListDiseaseParams{Page: 1, Limit: 50}
	if c := r.URL.Query().Get("country"); c != "" {
		upper := strings.ToUpper(c)
		p.Country = &upper
	}
	if code := r.URL.Query().Get("icd10"); code != "" {
		p.ICD10 = &code
	}
	reports, err := h.store.ListDiseases(r.Context(), p)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list diseases")
		return
	}
	if reports == nil {
		reports = []models.DiseaseReport{}
	}
	h.audit(r.Context(), "list_diseases", "")
	respond(w, http.StatusOK, reports)
}

func (h *Handler) listActiveAlerts(w http.ResponseWriter, r *http.Request) {
	var country *string
	if c := r.URL.Query().Get("country"); c != "" {
		upper := strings.ToUpper(c)
		country = &upper
	}
	alerts, err := h.store.ListActiveAlerts(r.Context(), country)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list alerts")
		return
	}
	if alerts == nil {
		alerts = []models.OutbreakAlert{}
	}
	h.audit(r.Context(), "list_outbreak_alerts", "")
	respond(w, http.StatusOK, alerts)
}

func (h *Handler) resolveAlert(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid alert id")
		return
	}
	now := time.Now().UTC()
	if err := h.store.ResolveAlert(r.Context(), id, now); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to resolve alert")
		return
	}
	h.audit(r.Context(), "resolve_outbreak_alert", "alert:"+id.String())
	respond(w, http.StatusOK, map[string]interface{}{"alert_id": id, "status": "resolved", "resolved_at": now})
}

func (h *Handler) recordClinicMetric(w http.ResponseWriter, r *http.Request) {
	clinicID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid clinic id")
		return
	}
	var req models.RecordClinicMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Country == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "country is required")
		return
	}
	period := req.Period
	if period == "" {
		period = "weekly"
	}
	now := time.Now().UTC()
	pStart := now.AddDate(0, 0, -7)
	pEnd := now
	if req.PeriodStart != "" {
		if t, err := time.Parse(time.RFC3339, req.PeriodStart); err == nil {
			pStart = t
		}
	}
	if req.PeriodEnd != "" {
		if t, err := time.Parse(time.RFC3339, req.PeriodEnd); err == nil {
			pEnd = t
		}
	}
	m := models.ClinicMetric{
		ID:                     uuid.New(),
		ClinicID:               clinicID,
		Country:                strings.ToUpper(req.Country),
		Period:                 period,
		PeriodStart:            pStart,
		PeriodEnd:              pEnd,
		TotalPatients:          req.TotalPatients,
		AvgVisitMinutes:        req.AvgVisitMinutes,
		ReferralCount:          req.ReferralCount,
		ReferralSuccessRate:    req.ReferralSuccessRate,
		PatientOutcomeImproved: req.PatientOutcomeImproved,
		PatientOutcomeStable:   req.PatientOutcomeStable,
		PatientOutcomeWorsened: req.PatientOutcomeWorsened,
		CostPerVisitUSD:        req.CostPerVisitUSD,
		CreatedAt:              now,
	}
	if err := h.store.RecordClinicMetric(r.Context(), m); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to record clinic metric")
		return
	}
	h.audit(r.Context(), "record_clinic_metric", "clinic:"+clinicID.String())
	respond(w, http.StatusCreated, m)
}

func (h *Handler) getClinicMetrics(w http.ResponseWriter, r *http.Request) {
	clinicID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid clinic id")
		return
	}
	metrics, err := h.store.GetClinicMetrics(r.Context(), clinicID, 10)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to get metrics")
		return
	}
	if metrics == nil {
		metrics = []models.ClinicMetric{}
	}
	respond(w, http.StatusOK, metrics)
}

func (h *Handler) getClinicPerformance(w http.ResponseWriter, r *http.Request) {
	clinicID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid clinic id")
		return
	}
	metrics, err := h.store.GetClinicMetrics(r.Context(), clinicID, 4)
	if err != nil || len(metrics) == 0 {
		respond(w, http.StatusOK, map[string]interface{}{"clinic_id": clinicID, "message": "no data yet"})
		return
	}
	latest := metrics[0]
	var improveRate float64
	total := latest.PatientOutcomeImproved + latest.PatientOutcomeStable + latest.PatientOutcomeWorsened
	if total > 0 {
		improveRate = float64(latest.PatientOutcomeImproved) / float64(total) * 100
	}
	respond(w, http.StatusOK, map[string]interface{}{
		"clinic_id":              clinicID,
		"total_patients":         latest.TotalPatients,
		"avg_visit_minutes":      latest.AvgVisitMinutes,
		"outcome_improvement_pct": improveRate,
		"referral_success_rate":  latest.ReferralSuccessRate,
		"cost_per_visit_usd":     latest.CostPerVisitUSD,
		"period":                 latest.Period,
	})
}

func (h *Handler) getImpact(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	if country == "" {
		country = "TG"
	}
	country = strings.ToUpper(country)
	summary, err := h.store.GetImpactSummary(r.Context(), country)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to get impact summary")
		return
	}
	h.audit(r.Context(), "get_impact_summary", "country:"+country)
	respond(w, http.StatusOK, summary)
}

func (h *Handler) getPopulationHealth(w http.ResponseWriter, r *http.Request) {
	country := strings.ToUpper(r.URL.Query().Get("country"))
	if country == "" {
		country = "TG"
	}
	reports, err := h.store.ListDiseases(r.Context(), db.ListDiseaseParams{Page: 1, Limit: 100, Country: &country})
	if err != nil {
		reports = []models.DiseaseReport{}
	}
	totalCases := 0
	diseaseMap := map[string]int{}
	for _, rep := range reports {
		totalCases += rep.CaseCount
		diseaseMap[rep.DiseaseName] += rep.CaseCount
	}
	respond(w, http.StatusOK, map[string]interface{}{
		"country":      country,
		"total_cases":  totalCases,
		"diseases":     diseaseMap,
		"report_count": len(reports),
		"generated_at": time.Now().UTC(),
	})
}

func (h *Handler) audit(ctx context.Context, action, resource string) {
	_ = h.store.InsertAuditLog(ctx, models.HealthAnalyticsAuditLog{
		ID:        uuid.New(),
		ActorID:   uuid.Nil,
		Action:    action,
		Resource:  resource,
		CreatedAt: time.Now().UTC(),
	})
}

func respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: data})
}

func respondError(w http.ResponseWriter, code int, errCode, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: errCode, Message: msg},
	})
}
