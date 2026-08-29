package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/analytics-service/auth"
	"github.com/klinova/kinara-os/analytics-service/db"
	"github.com/klinova/kinara-os/analytics-service/handlers"
	"github.com/klinova/kinara-os/analytics-service/middleware"
	"github.com/klinova/kinara-os/analytics-service/models"
)

type memStore struct {
	mu          sync.RWMutex
	metrics     []models.ImpactMetric
	summaries   map[uuid.UUID]*models.CrossPillarSummary
	reports     map[uuid.UUID]*models.GovernmentReport
	bottlenecks []models.Bottleneck
	audit       []models.AnalyticsAuditLog
}

func newMemStore() *memStore {
	return &memStore{
		summaries: map[uuid.UUID]*models.CrossPillarSummary{},
		reports:   map[uuid.UUID]*models.GovernmentReport{},
	}
}

func (s *memStore) RecordImpact(_ context.Context, m models.ImpactMetric) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.metrics = append(s.metrics, m); return nil
}
func (s *memStore) ListImpact(_ context.Context, p db.ListImpactParams) ([]models.ImpactMetric, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.ImpactMetric
	for _, m := range s.metrics {
		if p.Pillar != nil && m.Pillar != *p.Pillar { continue }
		if p.Country != nil && m.Country != *p.Country { continue }
		result = append(result, m)
	}
	return result, nil
}
func (s *memStore) CreateSummary(_ context.Context, sm models.CrossPillarSummary) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.summaries[sm.ID] = &sm; return nil
}
func (s *memStore) GetSummary(_ context.Context, id uuid.UUID) (*models.CrossPillarSummary, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	sm, ok := s.summaries[id]; if !ok { return nil, errNotFound }
	cp := *sm; return &cp, nil
}
func (s *memStore) ListSummaries(_ context.Context, country string) ([]models.CrossPillarSummary, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.CrossPillarSummary
	for _, sm := range s.summaries { if sm.Country == country { result = append(result, *sm) } }
	return result, nil
}
func (s *memStore) CreateReport(_ context.Context, r models.GovernmentReport) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.reports[r.ID] = &r; return nil
}
func (s *memStore) GetReport(_ context.Context, id uuid.UUID) (*models.GovernmentReport, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	r, ok := s.reports[id]; if !ok { return nil, errNotFound }
	cp := *r; return &cp, nil
}
func (s *memStore) ListReports(_ context.Context, country string) ([]models.GovernmentReport, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.GovernmentReport
	for _, r := range s.reports { if r.Country == country { result = append(result, *r) } }
	return result, nil
}
func (s *memStore) ReportBottleneck(_ context.Context, b models.Bottleneck) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.bottlenecks = append(s.bottlenecks, b); return nil
}
func (s *memStore) ListBottlenecks(_ context.Context, pillar *models.Pillar, country *string) ([]models.Bottleneck, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Bottleneck
	for _, b := range s.bottlenecks {
		if b.ResolvedAt != nil { continue }
		if pillar != nil && b.Pillar != *pillar { continue }
		if country != nil && b.Country != *country { continue }
		result = append(result, b)
	}
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.AnalyticsAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewHandlerWithStore(store)
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := &auth.Claims{UserID: uuid.New().String(), Role: "admin"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestRecordImpact(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.RecordImpactRequest{Pillar:"health",Country:"KE",MetricType:"service_delivery",MetricName:"Patients Served",MetricValue:12500,MetricUnit:"patients",PeriodStart:"2026-08-01T00:00:00Z",PeriodEnd:"2026-08-31T23:59:59Z",BeneficiaryCount:12500})
	resp, _ := http.Post(srv.URL+"/api/v1/impact", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestRecordImpact_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"pillar":"agriculture"}`
	resp, _ := http.Post(srv.URL+"/api/v1/impact", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestListImpact_ByPillar(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	now := time.Now().UTC()
	store.RecordImpact(context.Background(), models.ImpactMetric{ID:uuid.New(),Pillar:models.PillarHealth,Country:"GH",MetricType:models.MetricReach,MetricName:"Patients",MetricValue:5000,MetricUnit:"patients",PeriodStart:now.AddDate(0,-1,0),PeriodEnd:now,BeneficiaryCount:5000,CreatedAt:now})
	store.RecordImpact(context.Background(), models.ImpactMetric{ID:uuid.New(),Pillar:models.PillarAgri,Country:"GH",MetricType:models.MetricEconomicImpact,MetricName:"Revenue",MetricValue:200000,MetricUnit:"USD",PeriodStart:now.AddDate(0,-1,0),PeriodEnd:now,CreatedAt:now})
	resp, _ := http.Get(srv.URL + "/api/v1/impact?pillar=health")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 health metric, got %d", len(items)) }
}

func TestCreateSummary_OverallScoreCalc(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateSummaryRequest{Country:"NG",PeriodStart:"2026-08-01T00:00:00Z",PeriodEnd:"2026-08-31T23:59:59Z",HealthScore:82.0,AgriScore:75.0,LogisticsScore:88.0,MaritimeScore:70.0,TotalBeneficiaries:450000,TotalServicesDelivered:125000})
	resp, _ := http.Post(srv.URL+"/api/v1/summaries", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	// overall = (82+75+88+70)/4 = 78.75
	overall := data["overall_score"].(float64)
	if overall < 78 || overall > 79 { t.Fatalf("expected ~78.75 overall, got %.2f", overall) }
}

func TestGetSummary(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); now := time.Now().UTC()
	store.CreateSummary(context.Background(), models.CrossPillarSummary{ID:id,Country:"TZ",PeriodStart:now.AddDate(0,-1,0),PeriodEnd:now,HealthScore:78,AgriScore:65,LogisticsScore:72,MaritimeScore:80,OverallScore:73.75,TotalBeneficiaries:320000,TotalServicesDelivered:95000,CreatedAt:now})
	resp, _ := http.Get(srv.URL + "/api/v1/summaries/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGenerateReport(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.GenerateReportRequest{Country:"KE",ReportType:"quarterly_impact",PeriodStart:"2026-07-01T00:00:00Z",PeriodEnd:"2026-09-30T23:59:59Z"})
	resp, _ := http.Post(srv.URL+"/api/v1/reports", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if !strings.HasPrefix(data["report_ref"].(string), "GR-") { t.Fatal("report_ref must start with GR-") }
}

func TestReportBottleneck(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.ReportBottleneckRequest{Pillar:"logistics",Country:"SN",BottleneckType:"road_congestion",Description:"N1 highway congestion causing 3-hour delays",Severity:"high",AffectedUnits:45,RecommendedAction:"Deploy via N2 alternate route"})
	resp, _ := http.Post(srv.URL+"/api/v1/bottlenecks", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }

	// List bottlenecks
	resp2, _ := http.Get(srv.URL + "/api/v1/bottlenecks?pillar=logistics&country=SN")
	if resp2.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp2.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp2.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 bottleneck, got %d", len(items)) }
	bn := items[0].(map[string]interface{})
	if bn["severity"].(string) != "high" { t.Fatal("expected severity high") }
	_ = store
}

func TestListSummaries_ByCountry(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	now := time.Now().UTC()
	store.CreateSummary(context.Background(), models.CrossPillarSummary{ID:uuid.New(),Country:"ET",PeriodStart:now.AddDate(0,-2,0),PeriodEnd:now.AddDate(0,-1,0),OverallScore:71.0,CreatedAt:now})
	store.CreateSummary(context.Background(), models.CrossPillarSummary{ID:uuid.New(),Country:"ET",PeriodStart:now.AddDate(0,-1,0),PeriodEnd:now,OverallScore:74.5,CreatedAt:now})
	store.CreateSummary(context.Background(), models.CrossPillarSummary{ID:uuid.New(),Country:"RW",PeriodStart:now.AddDate(0,-1,0),PeriodEnd:now,OverallScore:85.0,CreatedAt:now})
	resp, _ := http.Get(srv.URL + "/api/v1/summaries/country/ET")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 2 { t.Fatalf("expected 2 ET summaries, got %d", len(items)) }
}
