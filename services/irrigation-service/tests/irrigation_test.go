package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/klinova/kinara-os/irrigation-service/auth"
	"github.com/klinova/kinara-os/irrigation-service/handlers"
	"github.com/klinova/kinara-os/irrigation-service/middleware"
	"github.com/klinova/kinara-os/irrigation-service/models"
)

// mockStore is a zero-allocation in-memory Store used by all handler tests.
type mockStore struct {
	systems   map[string]*models.IrrigationSystem
	moisture  map[string]*models.SoilMoistureReading
	schedules []models.WateringSchedule
	alerts    []models.IrrigationAlert
	history   []models.WateringHistory
	audits    []string
	errOn     string // set to method name to simulate DB error
}

func newMock() *mockStore {
	return &mockStore{
		systems:  make(map[string]*models.IrrigationSystem),
		moisture: make(map[string]*models.SoilMoistureReading),
	}
}

func (m *mockStore) RegisterSystem(_ context.Context, s models.IrrigationSystem) error {
	if m.errOn == "RegisterSystem" {
		return errMock
	}
	m.systems[s.FarmID] = &s
	return nil
}
func (m *mockStore) GetSystem(_ context.Context, farmID string) (*models.IrrigationSystem, error) {
	if m.errOn == "GetSystem" {
		return nil, errMock
	}
	s, ok := m.systems[farmID]
	if !ok {
		return nil, errNotFound
	}
	return s, nil
}
func (m *mockStore) GetLatestMoisture(_ context.Context, farmID string) (*models.SoilMoistureReading, error) {
	r, ok := m.moisture[farmID]
	if !ok {
		return nil, errNotFound
	}
	return r, nil
}
func (m *mockStore) CreateSchedule(_ context.Context, s models.WateringSchedule) error {
	if m.errOn == "CreateSchedule" {
		return errMock
	}
	m.schedules = append(m.schedules, s)
	return nil
}
func (m *mockStore) InsertMoisture(_ context.Context, r models.SoilMoistureReading) error {
	if m.errOn == "InsertMoisture" {
		return errMock
	}
	m.moisture[r.FarmID] = &r
	return nil
}
func (m *mockStore) InsertAlert(_ context.Context, a models.IrrigationAlert) error {
	if m.errOn == "InsertAlert" {
		return errMock
	}
	m.alerts = append(m.alerts, a)
	return nil
}
func (m *mockStore) GetHistory(_ context.Context, farmID string, _ int) ([]models.WateringHistory, error) {
	var out []models.WateringHistory
	for _, h := range m.history {
		if h.FarmID == farmID {
			out = append(out, h)
		}
	}
	return out, nil
}
func (m *mockStore) InsertAudit(_ context.Context, farmID, actorID, action string) error {
	m.audits = append(m.audits, farmID+":"+actorID+":"+action)
	return nil
}

// sentinel errors
type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

const errMock sentinelErr = "mock db error"
const errNotFound sentinelErr = "not found"

// authCtx injects a *auth.Claims into a request context.
func authCtx(r *http.Request, role string) *http.Request {
	claims := &auth.Claims{
		UserID:   uuid.New(),
		Role:     role,
		TenantID: "tenant-test",
	}
	return r.WithContext(middleware.SetClaims(r.Context(), claims))
}

func routeRequest(r *http.Request, h *handlers.Handler) *httptest.ResponseRecorder {
	router := mux.NewRouter()
	h.Register(router)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

// --- Tests ---

func TestRegisterSystem_Success(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.RegisterSystemRequest{
		SystemType:     "drip",
		CapacityLiters: 1000,
		SensorID:       "sensor-01",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/irrigation/farms/farm-001/system", bytes.NewReader(body))
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if store.systems["farm-001"] == nil {
		t.Error("system not stored")
	}
}

func TestRegisterSystem_Unauthorized(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.RegisterSystemRequest{SystemType: "drip", CapacityLiters: 500})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/irrigation/farms/farm-001/system", bytes.NewReader(body))
	// No auth context injected
	w := routeRequest(req, h)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetFarmStatus_Success(t *testing.T) {
	store := newMock()
	store.systems["farm-002"] = &models.IrrigationSystem{
		ID:     uuid.New(),
		FarmID: "farm-002",
	}
	store.moisture["farm-002"] = &models.SoilMoistureReading{
		ID:          uuid.New(),
		FarmID:      "farm-002",
		MoisturePct: 55.0,
		RecordedAt:  time.Now(),
	}
	h := handlers.NewWithStore(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/irrigation/farms/farm-002/status", nil)
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetFarmStatus_NotFound(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/irrigation/farms/nonexistent/status", nil)
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreateSchedule_Success(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.ScheduleRequest{
		CronExpression: "0 6 * * *",
		DurationMin:    30,
		CropType:       "maize",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/irrigation/farms/farm-003/schedule", bytes.NewReader(body))
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if len(store.schedules) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(store.schedules))
	}
}

func TestGetRecommendation_ShouldIrrigate(t *testing.T) {
	h := handlers.NewWithStore(newMock())
	rec := h.GetIrrigationRecommendation(20.0, false, "maize")
	if !rec.ShouldIrrigate {
		t.Error("expected should_irrigate=true for moisture < 30%")
	}
	if rec.RecommendedDurationMin != 45 {
		t.Errorf("expected 45 min, got %d", rec.RecommendedDurationMin)
	}
	if rec.OptimalTime != "06:00" {
		t.Errorf("expected optimal time 06:00, got %s", rec.OptimalTime)
	}
}

func TestGetRecommendation_ShouldSkip_RainExpected(t *testing.T) {
	h := handlers.NewWithStore(newMock())
	rec := h.GetIrrigationRecommendation(25.0, true, "maize")
	if rec.ShouldIrrigate {
		t.Error("expected should_irrigate=false when rain expected")
	}
	if !strings.Contains(rec.Reason, "rain") {
		t.Errorf("expected rain reason, got: %s", rec.Reason)
	}
}

func TestGetRecommendation_ShouldSkip_Saturated(t *testing.T) {
	h := handlers.NewWithStore(newMock())
	rec := h.GetIrrigationRecommendation(85.0, false, "rice")
	if rec.ShouldIrrigate {
		t.Error("expected should_irrigate=false when moisture > 80%")
	}
	if !strings.Contains(rec.Reason, "saturated") {
		t.Errorf("expected saturated reason, got: %s", rec.Reason)
	}
}

func TestGetRecommendation_ShouldIrrigate_Low(t *testing.T) {
	h := handlers.NewWithStore(newMock())
	rec := h.GetIrrigationRecommendation(40.0, false, "wheat")
	if !rec.ShouldIrrigate {
		t.Error("expected should_irrigate=true for moisture 30-50%")
	}
	if rec.RecommendedDurationMin != 30 {
		t.Errorf("expected 30 min, got %d", rec.RecommendedDurationMin)
	}
}

func TestSendAlert_Success(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.AlertRequest{
		Message:   "Soil moisture critically low",
		AlertType: "low_moisture",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/irrigation/farms/farm-004/alert", bytes.NewReader(body))
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if len(store.alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(store.alerts))
	}
	if store.alerts[0].AlertType != "low_moisture" {
		t.Errorf("expected low_moisture alert, got %s", store.alerts[0].AlertType)
	}
}

func TestGetHistory_Success(t *testing.T) {
	store := newMock()
	store.history = []models.WateringHistory{
		{ID: uuid.New(), FarmID: "farm-005", DurationMin: 30, TriggerType: "manual", IrrigatedAt: time.Now()},
		{ID: uuid.New(), FarmID: "farm-005", DurationMin: 45, TriggerType: "scheduled", IrrigatedAt: time.Now()},
	}
	h := handlers.NewWithStore(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/irrigation/farms/farm-005/history", nil)
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("expected 2 history records, got %d", len(data))
	}
}

func TestRecordMoisture_Success(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.MoistureRequest{
		MoisturePct: 42.5,
		SensorID:    "sensor-07",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/irrigation/farms/farm-006/moisture", bytes.NewReader(body))
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if store.moisture["farm-006"] == nil {
		t.Error("moisture reading not stored")
	}
	if store.moisture["farm-006"].MoisturePct != 42.5 {
		t.Errorf("expected 42.5%%, got %.2f", store.moisture["farm-006"].MoisturePct)
	}
}
