package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/weather-service/auth"
	"github.com/klinova/kinara-os/weather-service/db"
	"github.com/klinova/kinara-os/weather-service/handlers"
	"github.com/klinova/kinara-os/weather-service/middleware"
	"github.com/klinova/kinara-os/weather-service/models"
)

// ─── In-memory store ─────────────────────────────────────────────────────────

type memStore struct {
	forecasts    map[uuid.UUID]*models.WeatherForecast
	alerts       map[uuid.UUID]*models.WeatherAlert
	advisories   map[uuid.UUID]*models.PestAdvisory
	observations []models.WeatherObservation
}

func newMemStore() *memStore {
	return &memStore{
		forecasts:  map[uuid.UUID]*models.WeatherForecast{},
		alerts:     map[uuid.UUID]*models.WeatherAlert{},
		advisories: map[uuid.UUID]*models.PestAdvisory{},
	}
}

var errNotFound = &notFoundErr{}

type notFoundErr struct{}

func (e *notFoundErr) Error() string { return "not found" }

func (m *memStore) CreateForecast(_ context.Context, f models.WeatherForecast) error {
	m.forecasts[f.ID] = &f
	return nil
}
func (m *memStore) GetForecast(_ context.Context, id uuid.UUID) (*models.WeatherForecast, error) {
	f, ok := m.forecasts[id]
	if !ok {
		return nil, errNotFound
	}
	return f, nil
}
func (m *memStore) ListForecasts(_ context.Context, _ db.ListForecastsParams) ([]models.WeatherForecast, error) {
	var result []models.WeatherForecast
	for _, f := range m.forecasts {
		result = append(result, *f)
	}
	return result, nil
}

func (m *memStore) CreateAlert(_ context.Context, a models.WeatherAlert) error {
	m.alerts[a.ID] = &a
	return nil
}
func (m *memStore) GetAlert(_ context.Context, id uuid.UUID) (*models.WeatherAlert, error) {
	a, ok := m.alerts[id]
	if !ok {
		return nil, errNotFound
	}
	return a, nil
}
func (m *memStore) ListActiveAlerts(_ context.Context, _, _ string) ([]models.WeatherAlert, error) {
	var result []models.WeatherAlert
	for _, a := range m.alerts {
		if a.Active {
			result = append(result, *a)
		}
	}
	return result, nil
}
func (m *memStore) DeactivateAlert(_ context.Context, id uuid.UUID, now time.Time) error {
	a, ok := m.alerts[id]
	if !ok {
		return errNotFound
	}
	a.Active = false
	a.UpdatedAt = now
	return nil
}

func (m *memStore) CreateAdvisory(_ context.Context, a models.PestAdvisory) error {
	m.advisories[a.ID] = &a
	return nil
}
func (m *memStore) GetAdvisory(_ context.Context, id uuid.UUID) (*models.PestAdvisory, error) {
	a, ok := m.advisories[id]
	if !ok {
		return nil, errNotFound
	}
	return a, nil
}
func (m *memStore) ListAdvisories(_ context.Context, _, _, _ string, _ *models.RiskLevel) ([]models.PestAdvisory, error) {
	var result []models.PestAdvisory
	for _, a := range m.advisories {
		result = append(result, *a)
	}
	return result, nil
}

func (m *memStore) CreateObservation(_ context.Context, o models.WeatherObservation) error {
	m.observations = append(m.observations, o)
	return nil
}
func (m *memStore) ListObservations(_ context.Context, _, _ string, _ time.Time, _ int) ([]models.WeatherObservation, error) {
	return m.observations, nil
}
func (m *memStore) InsertAuditLog(_ context.Context, _ models.WeatherAuditLog) error { return nil }

// ─── Router ───────────────────────────────────────────────────────────────────

func setupRouter(store handlers.Store) *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &auth.Claims{UserID: uuid.New().String(), Role: "agronomist", FacilityID: "KE"}
			ctx := middleware.SetClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	h := handlers.NewWeatherHandlerWithStore(store)
	h.RegisterRoutes(api)
	return r
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestCreateForecast(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{
		"country":              "KE",
		"region":               "Rift Valley",
		"latitude":             0.5150,
		"longitude":            35.2698,
		"forecast_type":        "daily",
		"forecast_date":        time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"condition":            "partly_cloudy",
		"temp_min_c":           18.5,
		"temp_max_c":           29.0,
		"humidity_pct":         65.0,
		"wind_speed_kmh":       12.0,
		"rainfall_mm":          5.2,
		"rainfall_probability": 40.0,
		"uv_index":             7.0,
		"valid_hours":          24,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/forecasts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.forecasts) != 1 {
		t.Fatal("expected forecast to be stored")
	}
}

func TestCreateForecastMissingFields(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{"country": "KE"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/forecasts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetForecastNotFound(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/forecasts/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCreateAlert(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{
		"alert_type":  "flood",
		"severity":    "emergency",
		"country":     "NG",
		"region":      "Niger Delta",
		"title":       "Severe Flooding Risk",
		"description": "Flash flood expected within 48 hours due to heavy upstream rains",
		"instructions": "Move livestock to higher ground immediately",
		"affected_crops": []string{"rice", "yam"},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	for _, a := range store.alerts {
		if !a.Active {
			t.Fatal("expected alert to be active")
		}
	}
}

func TestDeactivateAlert(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	alertID := uuid.New()
	store.alerts[alertID] = &models.WeatherAlert{
		ID: alertID, AlertType: models.AlertDrought, Severity: models.SeverityWatch,
		Country: "ET", Title: "Drought Warning", Active: true,
		IssuedAt: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		AffectedCrops: []string{},
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts/"+alertID.String()+"/deactivate", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.alerts[alertID].Active {
		t.Fatal("expected alert to be deactivated")
	}
}

func TestListActiveAlerts(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	store.alerts[uuid.New()] = &models.WeatherAlert{
		ID: uuid.New(), AlertType: models.AlertPestRisk, Severity: models.SeverityWarning,
		Country: "GH", Title: "Armyworm Alert", Active: true,
		IssuedAt: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		AffectedCrops: []string{"maize"},
	}
	store.alerts[uuid.New()] = &models.WeatherAlert{
		ID: uuid.New(), AlertType: models.AlertFlood, Severity: models.SeverityInfo,
		Country: "GH", Title: "Flood Watch", Active: false,
		IssuedAt: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		AffectedCrops: []string{},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?country=GH", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp models.APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestCreateAdvisory(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{
		"pest_name":      "Fall Armyworm",
		"pest_type":      "pest",
		"affected_crops": []string{"maize", "sorghum", "millet"},
		"country":        "KE",
		"region":         "Western Kenya",
		"risk_level":     "high",
		"description":    "Spodoptera frugiperda outbreak detected in multiple counties",
		"symptoms":       "Ragged holes in leaves, frass on whorls",
		"prevention":     "Early planting, intercropping with legumes",
		"treatment":      "Apply emamectin benzoate 1.9EC at 200ml/ha",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/advisories", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitObservation(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{
		"country":   "TZ",
		"region":    "Dodoma",
		"latitude":  -6.1722,
		"longitude": 35.7395,
		"condition": "rainy",
		"rainfall_mm": 22.5,
		"notes":     "Heavy rain started at 14:00 local time",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/observations", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.observations) != 1 {
		t.Fatal("expected observation to be stored")
	}
}

func TestSubmitObservationMissingFields(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{"country": "TZ"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/observations", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
