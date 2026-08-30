package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/analytics-service/db"
	"github.com/klinova/kinara-os/analytics-service/middleware"
	"github.com/klinova/kinara-os/analytics-service/models"
)

type Store interface {
	RecordImpact(ctx context.Context, m models.ImpactMetric) error
	ListImpact(ctx context.Context, p db.ListImpactParams) ([]models.ImpactMetric, error)
	CreateSummary(ctx context.Context, s models.CrossPillarSummary) error
	GetSummary(ctx context.Context, id uuid.UUID) (*models.CrossPillarSummary, error)
	ListSummaries(ctx context.Context, country string) ([]models.CrossPillarSummary, error)
	CreateReport(ctx context.Context, r models.GovernmentReport) error
	GetReport(ctx context.Context, id uuid.UUID) (*models.GovernmentReport, error)
	ListReports(ctx context.Context, country string) ([]models.GovernmentReport, error)
	ReportBottleneck(ctx context.Context, b models.Bottleneck) error
	ListBottlenecks(ctx context.Context, pillar *models.Pillar, country *string) ([]models.Bottleneck, error)
	InsertAuditLog(ctx context.Context, l models.AnalyticsAuditLog) error
}

type Handler struct{ store Store }

func NewHandler(store Store) *Handler          { return &Handler{store: store} }
func NewHandlerWithStore(s Store) *Handler     { return &Handler{store: s} }

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/impact", h.RecordImpact).Methods("POST")
	r.HandleFunc("/impact", h.ListImpact).Methods("GET")
	r.HandleFunc("/summaries", h.CreateSummary).Methods("POST")
	r.HandleFunc("/summaries/{id}", h.GetSummary).Methods("GET")
	r.HandleFunc("/summaries/country/{country}", h.ListSummaries).Methods("GET")
	r.HandleFunc("/reports", h.GenerateReport).Methods("POST")
	r.HandleFunc("/reports/{id}", h.GetReport).Methods("GET")
	r.HandleFunc("/reports/country/{country}", h.ListReports).Methods("GET")
	r.HandleFunc("/bottlenecks", h.ReportBottleneck).Methods("POST")
	r.HandleFunc("/bottlenecks", h.ListBottlenecks).Methods("GET")
}

func respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) RecordImpact(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.RecordImpactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid request"})
		return
	}
	if req.Pillar == "" || req.Country == "" || req.MetricName == "" {
		respond(w, 400, models.APIResponse{Error: "pillar, country, metric_name required"})
		return
	}
	ps, err1 := time.Parse(time.RFC3339, req.PeriodStart)
	pe, err2 := time.Parse(time.RFC3339, req.PeriodEnd)
	if err1 != nil || err2 != nil {
		respond(w, 400, models.APIResponse{Error: "period_start and period_end must be RFC3339"})
		return
	}
	now := time.Now().UTC()
	id := uuid.New()
	m := models.ImpactMetric{
		ID: id, Pillar: models.Pillar(req.Pillar), Country: req.Country,
		MetricType: models.MetricType(req.MetricType), MetricName: req.MetricName,
		MetricValue: req.MetricValue, MetricUnit: req.MetricUnit,
		PeriodStart: ps, PeriodEnd: pe,
		BeneficiaryCount: req.BeneficiaryCount, Notes: req.Notes, CreatedAt: now,
	}
	if err := h.store.RecordImpact(r.Context(), m); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to record metric"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.AnalyticsAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "record_impact", EntityType: "impact_metric", EntityID: id, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: m})
}

func (h *Handler) ListImpact(w http.ResponseWriter, r *http.Request) {
	var pillar *models.Pillar
	var country *string
	if p := r.URL.Query().Get("pillar"); p != "" {
		v := models.Pillar(p)
		pillar = &v
	}
	if c := r.URL.Query().Get("country"); c != "" {
		country = &c
	}
	metrics, err := h.store.ListImpact(r.Context(), db.ListImpactParams{Pillar: pillar, Country: country, Page: 1, Limit: 100})
	if err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to list impact metrics"})
		return
	}
	if metrics == nil { metrics = []models.ImpactMetric{} }
	respond(w, 200, models.APIResponse{Success: true, Data: metrics})
}

func (h *Handler) CreateSummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.CreateSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid request"})
		return
	}
	if req.Country == "" {
		respond(w, 400, models.APIResponse{Error: "country required"})
		return
	}
	ps, _ := time.Parse(time.RFC3339, req.PeriodStart)
	pe, _ := time.Parse(time.RFC3339, req.PeriodEnd)
	// Overall score = average of non-zero pillar scores
	scores := []float64{}
	if req.HealthScore > 0    { scores = append(scores, req.HealthScore) }
	if req.AgriScore > 0      { scores = append(scores, req.AgriScore) }
	if req.LogisticsScore > 0 { scores = append(scores, req.LogisticsScore) }
	if req.MaritimeScore > 0  { scores = append(scores, req.MaritimeScore) }
	var overall float64
	for _, s := range scores { overall += s }
	if len(scores) > 0 { overall /= float64(len(scores)) }

	now := time.Now().UTC()
	id := uuid.New()
	summary := models.CrossPillarSummary{
		ID: id, Country: req.Country, PeriodStart: ps, PeriodEnd: pe,
		HealthScore: req.HealthScore, AgriScore: req.AgriScore,
		LogisticsScore: req.LogisticsScore, MaritimeScore: req.MaritimeScore,
		OverallScore: overall, TotalBeneficiaries: req.TotalBeneficiaries,
		TotalServicesDelivered: req.TotalServicesDelivered, CreatedAt: now,
	}
	if err := h.store.CreateSummary(r.Context(), summary); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to create summary"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.AnalyticsAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "create_summary", EntityType: "cross_pillar_summary", EntityID: id, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: summary})
}

func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	s, err := h.store.GetSummary(r.Context(), id)
	if err != nil {
		respond(w, 404, models.APIResponse{Error: "summary not found"})
		return
	}
	respond(w, 200, models.APIResponse{Success: true, Data: s})
}

func (h *Handler) ListSummaries(w http.ResponseWriter, r *http.Request) {
	country := mux.Vars(r)["country"]
	summaries, err := h.store.ListSummaries(r.Context(), country)
	if err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to list summaries"})
		return
	}
	if summaries == nil { summaries = []models.CrossPillarSummary{} }
	respond(w, 200, models.APIResponse{Success: true, Data: summaries})
}

func (h *Handler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.GenerateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid request"})
		return
	}
	if req.Country == "" || req.ReportType == "" {
		respond(w, 400, models.APIResponse{Error: "country, report_type required"})
		return
	}
	ps, _ := time.Parse(time.RFC3339, req.PeriodStart)
	pe, _ := time.Parse(time.RFC3339, req.PeriodEnd)
	now := time.Now().UTC()
	id := uuid.New()
	ref := "GR-" + strings.ToUpper(id.String()[:10])
	summaryJSON := fmt.Sprintf(`{"country":"%s","report_type":"%s","generated_by":"kinara-analytics","period_start":"%s","period_end":"%s"}`,
		req.Country, req.ReportType, ps.Format(time.RFC3339), pe.Format(time.RFC3339))
	report := models.GovernmentReport{
		ID: id, ReportRef: ref, Country: req.Country, ReportType: req.ReportType,
		PeriodStart: ps, PeriodEnd: pe, GeneratedAt: now,
		SummaryJSON: summaryJSON, CreatedAt: now,
	}
	if err := h.store.CreateReport(r.Context(), report); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to generate report"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.AnalyticsAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "generate_report", EntityType: "government_report", EntityID: id, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: report})
}

func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	report, err := h.store.GetReport(r.Context(), id)
	if err != nil {
		respond(w, 404, models.APIResponse{Error: "report not found"})
		return
	}
	respond(w, 200, models.APIResponse{Success: true, Data: report})
}

func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
	country := mux.Vars(r)["country"]
	reports, err := h.store.ListReports(r.Context(), country)
	if err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to list reports"})
		return
	}
	if reports == nil { reports = []models.GovernmentReport{} }
	respond(w, 200, models.APIResponse{Success: true, Data: reports})
}

func (h *Handler) ReportBottleneck(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.ReportBottleneckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid request"})
		return
	}
	if req.Pillar == "" || req.Country == "" || req.Description == "" {
		respond(w, 400, models.APIResponse{Error: "pillar, country, description required"})
		return
	}
	if req.Severity == "" { req.Severity = "medium" }
	now := time.Now().UTC()
	id := uuid.New()
	b := models.Bottleneck{
		ID: id, Pillar: models.Pillar(req.Pillar), Country: req.Country,
		BottleneckType: req.BottleneckType, Description: req.Description,
		Severity: req.Severity, AffectedUnits: req.AffectedUnits,
		RecommendedAction: req.RecommendedAction,
		DetectedAt: now, CreatedAt: now,
	}
	if err := h.store.ReportBottleneck(r.Context(), b); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to report bottleneck"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.AnalyticsAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "report_bottleneck", EntityType: "bottleneck", EntityID: id, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: b})
}

func (h *Handler) ListBottlenecks(w http.ResponseWriter, r *http.Request) {
	var pillar *models.Pillar
	var country *string
	if p := r.URL.Query().Get("pillar"); p != "" {
		v := models.Pillar(p)
		pillar = &v
	}
	if c := r.URL.Query().Get("country"); c != "" {
		country = &c
	}
	bottlenecks, err := h.store.ListBottlenecks(r.Context(), pillar, country)
	if err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to list bottlenecks"})
		return
	}
	if bottlenecks == nil { bottlenecks = []models.Bottleneck{} }
	respond(w, 200, models.APIResponse{Success: true, Data: bottlenecks})
}
