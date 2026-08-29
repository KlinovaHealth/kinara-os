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
	"github.com/klinova/kinara-os/health-analytics-service/db"
	"github.com/klinova/kinara-os/health-analytics-service/handlers"
	"github.com/klinova/kinara-os/health-analytics-service/models"
)

type memStore struct {
	mu       sync.RWMutex
	diseases []models.DiseaseReport
	alerts   map[uuid.UUID]*models.OutbreakAlert
	metrics  []models.ClinicMetric
	audit    []models.HealthAnalyticsAuditLog
}

func newMemStore() *memStore {
	return &memStore{alerts: map[uuid.UUID]*models.OutbreakAlert{}}
}

func (s *memStore) ReportDisease(_ context.Context, r models.DiseaseReport) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.diseases = append(s.diseases, r); return nil
}
func (s *memStore) ListDiseases(_ context.Context, p db.ListDiseaseParams) ([]models.DiseaseReport, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.DiseaseReport
	for _, r := range s.diseases {
		if p.Country != nil && r.Country != *p.Country {
			continue
		}
		if p.ICD10 != nil && r.ICD10Code != *p.ICD10 {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}
func (s *memStore) CreateOutbreakAlert(_ context.Context, a models.OutbreakAlert) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.alerts[a.ID] = &a; return nil
}
func (s *memStore) ListActiveAlerts(_ context.Context, country *string) ([]models.OutbreakAlert, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.OutbreakAlert
	for _, a := range s.alerts {
		if a.Status == "resolved" {
			continue
		}
		if country != nil && a.Country != *country {
			continue
		}
		result = append(result, *a)
	}
	return result, nil
}
func (s *memStore) ResolveAlert(_ context.Context, id uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if a, ok := s.alerts[id]; ok {
		a.Status = "resolved"
		a.ResolvedAt = &now
	}
	return nil
}
func (s *memStore) RecordClinicMetric(_ context.Context, m models.ClinicMetric) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.metrics = append(s.metrics, m); return nil
}
func (s *memStore) GetClinicMetrics(_ context.Context, clinicID uuid.UUID, limit int) ([]models.ClinicMetric, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.ClinicMetric
	for _, m := range s.metrics {
		if m.ClinicID == clinicID {
			result = append(result, m)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}
func (s *memStore) GetImpactSummary(_ context.Context, country string) (*models.ImpactSummary, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	summary := &models.ImpactSummary{Country: country, GeneratedAt: time.Now().UTC()}
	for _, m := range s.metrics {
		if m.Country == country {
			summary.TotalPatients += m.TotalPatients
			summary.TotalClinics++
		}
	}
	for _, a := range s.alerts {
		if a.Country == country && a.Status != "resolved" {
			summary.ActiveOutbreaks++
		}
	}
	return summary, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.HealthAnalyticsAuditLog) error {
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
	h.RegisterRoutes(r)
	return httptest.NewServer(r), store
}

func TestReportDisease(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	body, _ := json.Marshal(models.ReportDiseaseRequest{
		ClinicID:    uuid.New().String(),
		Country:     "TG",
		Region:      "Maritime",
		ICD10Code:   "B54",
		DiseaseName: "Malaria",
		CaseCount:   5,
		Period:      "weekly",
		PeriodStart: time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		PeriodEnd:   time.Now().Format(time.RFC3339),
		Severity:    "moderate",
	})
	resp, _ := http.Post(srv.URL+"/api/v1/health-analytics/diseases", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestReportDisease_MissingFields(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	body := `{"country":"TG"}`
	resp, _ := http.Post(srv.URL+"/api/v1/health-analytics/diseases", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestOutbreakAutoDetection(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	// 5 cases should trigger outbreak alert (threshold=3)
	body, _ := json.Marshal(models.ReportDiseaseRequest{
		ClinicID:    uuid.New().String(),
		Country:     "TG",
		ICD10Code:   "B54",
		DiseaseName: "Malaria",
		CaseCount:   5,
		Period:      "weekly",
		Severity:    "moderate",
	})
	http.Post(srv.URL+"/api/v1/health-analytics/diseases", "application/json", bytes.NewBuffer(body))

	store.mu.RLock()
	alertCount := len(store.alerts)
	store.mu.RUnlock()
	if alertCount == 0 {
		t.Fatal("expected outbreak alert to be created for 5 cases (threshold=3)")
	}
}

func TestListActiveAlerts(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	store.CreateOutbreakAlert(context.Background(), models.OutbreakAlert{
		ID:          uuid.New(),
		AlertRef:    "OA-TEST001",
		ClinicID:    uuid.New(),
		Country:     "TG",
		ICD10Code:   "A00",
		DiseaseName: "Cholera",
		CaseCount:   8,
		Threshold:   3,
		Status:      "active",
		DetectedAt:  time.Now(),
	})
	resp, _ := http.Get(srv.URL + "/api/v1/health-analytics/outbreaks?country=TG")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 active alert for TG, got %d", len(items))
	}
}

func TestRecordAndGetClinicMetric(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	clinicID := uuid.New()
	body, _ := json.Marshal(models.RecordClinicMetricRequest{
		Country:                "GH",
		Period:                 "weekly",
		PeriodStart:            time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		PeriodEnd:              time.Now().Format(time.RFC3339),
		TotalPatients:          120,
		AvgVisitMinutes:        22.5,
		ReferralCount:          8,
		ReferralSuccessRate:    87.5,
		PatientOutcomeImproved: 95,
		PatientOutcomeStable:   18,
		PatientOutcomeWorsened: 7,
		CostPerVisitUSD:        3.50,
	})
	resp, _ := http.Post(srv.URL+"/api/v1/health-analytics/clinics/"+clinicID.String()+"/metrics",
		"application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp2, _ := http.Get(srv.URL + "/api/v1/health-analytics/clinics/" + clinicID.String() + "/metrics")
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
}

func TestGetImpactSummary(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	clinicID := uuid.New()
	store.RecordClinicMetric(context.Background(), models.ClinicMetric{
		ID:                     uuid.New(),
		ClinicID:               clinicID,
		Country:                "TG",
		Period:                 "monthly",
		PeriodStart:            time.Now().AddDate(0, -1, 0),
		PeriodEnd:              time.Now(),
		TotalPatients:          500,
		PatientOutcomeImproved: 420,
		PatientOutcomeStable:   60,
		PatientOutcomeWorsened: 20,
		CostPerVisitUSD:        4.50,
		CreatedAt:              time.Now(),
	})
	resp, _ := http.Get(srv.URL + "/api/v1/health-analytics/impact?country=TG")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["country"] != "TG" {
		t.Fatalf("expected country TG, got %v", data["country"])
	}
	totalPatients := int(data["total_patients"].(float64))
	if totalPatients != 500 {
		t.Fatalf("expected 500 total patients, got %d", totalPatients)
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/health")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
