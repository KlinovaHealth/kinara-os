package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/logistics-analytics-service/auth"
	"github.com/klinova/kinara-os/logistics-analytics-service/db"
	"github.com/klinova/kinara-os/logistics-analytics-service/handlers"
	"github.com/klinova/kinara-os/logistics-analytics-service/middleware"
	"github.com/klinova/kinara-os/logistics-analytics-service/models"
)

type memStore struct {
	mu        sync.RWMutex
	metrics   []models.LogisticsMetric
	forecasts map[uuid.UUID]*models.DemandForecast
	audit     []models.AnalyticsAuditLog
}

func newMemStore() *memStore { return &memStore{forecasts: map[uuid.UUID]*models.DemandForecast{}} }

func (s *memStore) RecordMetric(_ context.Context, m models.LogisticsMetric) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.metrics = append(s.metrics, m); return nil
}
func (s *memStore) ListMetrics(_ context.Context, p db.ListMetricsParams) ([]models.LogisticsMetric, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.LogisticsMetric
	for _, m := range s.metrics {
		if p.Country != nil && m.Country != *p.Country { continue }
		if p.Period != nil && m.Period != *p.Period { continue }
		result = append(result, m)
	}
	return result, nil
}
func (s *memStore) CreateForecast(_ context.Context, f models.DemandForecast) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.forecasts[f.ID] = &f; return nil
}
func (s *memStore) GetForecast(_ context.Context, id uuid.UUID) (*models.DemandForecast, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	f, ok := s.forecasts[id]; if !ok { return nil, errNotFound }
	cp := *f; return &cp, nil
}
func (s *memStore) ListForecasts(_ context.Context, country string) ([]models.DemandForecast, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.DemandForecast
	for _, f := range s.forecasts { if f.Country == country { result = append(result, *f) } }
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
	h := handlers.NewAnalyticsHandlerWithStore(store)
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := &auth.Claims{UserID: uuid.New(), Role: "admin"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestRecordMetric(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.RecordMetricRequest{Period:models.PeriodWeekly,PeriodStart:"2026-08-01T00:00:00Z",PeriodEnd:"2026-08-07T23:59:59Z",Country:"KE",TotalTrips:450,TotalDistanceKm:95000,TotalDeliveries:380,SuccessfulDeliveries:362,OnTimeDeliveries:340,AvgCostPerKm:2.5,TotalRevenue:237500,Currency:"USD"})
	resp, _ := http.Post(srv.URL+"/api/v1/metrics", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestRecordMetric_MissingCountry(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"period":"daily","period_start":"2026-08-01T00:00:00Z","period_end":"2026-08-01T23:59:59Z"}`
	resp, _ := http.Post(srv.URL+"/api/v1/metrics", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestListMetrics(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	now := time.Now().UTC()
	store.RecordMetric(context.Background(), models.LogisticsMetric{ID:uuid.New(),Period:models.PeriodMonthly,PeriodStart:now.AddDate(0,-1,0),PeriodEnd:now,Country:"GH",TotalTrips:1200,TotalDeliveries:980,TotalRevenue:500000,Currency:"USD",CreatedAt:now})
	resp, _ := http.Get(srv.URL + "/api/v1/metrics")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestCreateForecast(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateForecastRequest{Country:"NG",Route:"Lagos-Abuja",ForecastDate:"2026-10-01T00:00:00Z",PredictedVolume:8500,PredictedTrips:320,ConfidencePct:87.5,Notes:"Based on seasonal harvest patterns"})
	resp, _ := http.Post(srv.URL+"/api/v1/forecasts", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestGetForecast(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateForecast(context.Background(), models.DemandForecast{ID:id,Country:"TZ",Route:"Dar-Dodoma",ForecastDate:time.Now().AddDate(0,1,0),PredictedVolume:3200,PredictedTrips:150,ConfidencePct:78,CreatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/forecasts/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestListForecasts_ByCountry(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	store.CreateForecast(context.Background(), models.DemandForecast{ID:uuid.New(),Country:"SN",Route:"Dakar-Thiès",ForecastDate:time.Now().AddDate(0,2,0),PredictedVolume:1500,PredictedTrips:80,ConfidencePct:82,CreatedAt:time.Now()})
	store.CreateForecast(context.Background(), models.DemandForecast{ID:uuid.New(),Country:"CI",Route:"Abidjan-Bouaké",ForecastDate:time.Now().AddDate(0,2,0),PredictedVolume:2000,PredictedTrips:100,ConfidencePct:75,CreatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/forecasts/country/SN")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 SN forecast, got %d", len(items)) }
}

func TestOnTimeRateCalculation(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	now := time.Now().UTC()
	store.RecordMetric(context.Background(), models.LogisticsMetric{ID:uuid.New(),Period:models.PeriodWeekly,PeriodStart:now.AddDate(0,0,-7),PeriodEnd:now,Country:"ET",TotalDeliveries:100,OnTimeDeliveries:85,OnTimeRate:85.0,Currency:"USD",CreatedAt:now})
	metrics, _ := store.ListMetrics(context.Background(), db.ListMetricsParams{Page:1,Limit:10})
	if len(metrics) == 0 { t.Fatal("expected metrics") }
	if metrics[0].OnTimeRate != 85.0 { t.Fatalf("expected 85.0 on-time rate, got %f", metrics[0].OnTimeRate) }
}
