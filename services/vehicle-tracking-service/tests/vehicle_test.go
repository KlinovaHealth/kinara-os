package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/klinova/kinara-os/vehicle-tracking-service/auth"
	"github.com/klinova/kinara-os/vehicle-tracking-service/handlers"
	"github.com/klinova/kinara-os/vehicle-tracking-service/middleware"
	"github.com/klinova/kinara-os/vehicle-tracking-service/models"
)

// ---------- mock store ----------

type mockStore struct {
	locations map[uuid.UUID]*models.GPSLocation
	routes    map[uuid.UUID]*models.VehicleRoute
	vehicles  map[uuid.UUID]*models.Vehicle
	alerts    []models.VehicleAlert
	fleet     []models.FleetVehicleStatus
	err       error
}

func newMockStore() *mockStore {
	return &mockStore{
		locations: make(map[uuid.UUID]*models.GPSLocation),
		routes:    make(map[uuid.UUID]*models.VehicleRoute),
		vehicles:  make(map[uuid.UUID]*models.Vehicle),
	}
}

func (m *mockStore) InsertPing(_ context.Context, loc models.GPSLocation) error {
	if m.err != nil {
		return m.err
	}
	cp := loc
	m.locations[loc.VehicleID] = &cp
	return nil
}

func (m *mockStore) GetLatestLocation(_ context.Context, vehicleID uuid.UUID) (*models.GPSLocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	loc, ok := m.locations[vehicleID]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	cp := *loc
	return &cp, nil
}

func (m *mockStore) GetActiveRoute(_ context.Context, vehicleID uuid.UUID) (*models.VehicleRoute, error) {
	if m.err != nil {
		return nil, m.err
	}
	rt, ok := m.routes[vehicleID]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	cp := *rt
	return &cp, nil
}

func (m *mockStore) GetFleetStatus(_ context.Context, _ string) ([]models.FleetVehicleStatus, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.fleet, nil
}

func (m *mockStore) InsertAlert(_ context.Context, a models.VehicleAlert) error {
	if m.err != nil {
		return m.err
	}
	m.alerts = append(m.alerts, a)
	return nil
}

func (m *mockStore) GetVehicle(_ context.Context, id uuid.UUID) (*models.Vehicle, error) {
	if m.err != nil {
		return nil, m.err
	}
	v, ok := m.vehicles[id]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	cp := *v
	return &cp, nil
}

// ---------- helpers ----------

func newRouter(store handlers.Store) *mux.Router {
	r := mux.NewRouter()
	handlers.NewWithStore(store).Register(r)
	return r
}

func withClaims(r *http.Request, role string) *http.Request {
	claims := &auth.Claims{UserID: uuid.New(), Role: role, TenantID: "TG"}
	ctx := middleware.SetClaims(r.Context(), claims)
	return r.WithContext(ctx)
}

func mustMarshal(t *testing.T, v interface{}) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

// seedLocation adds a GPS location for a vehicle in the store.
func seedLocation(store *mockStore, vehicleID uuid.UUID, lat, lng, speed float64) {
	store.locations[vehicleID] = &models.GPSLocation{
		ID:        uuid.New(),
		VehicleID: vehicleID,
		Latitude:  lat,
		Longitude: lng,
		SpeedKmh:  speed,
		PingedAt:  time.Now().UTC(),
	}
}

// seedRoute adds an active route for a vehicle in the store.
func seedRoute(store *mockStore, vehicleID uuid.UUID) {
	store.routes[vehicleID] = &models.VehicleRoute{
		ID:          uuid.New(),
		VehicleID:   vehicleID,
		OriginLat:   6.1375,
		OriginLng:   1.2123,
		DestLat:     5.5560,
		DestLng:     -0.1969,
		Description: "Lomé to Accra",
		Active:      true,
		AssignedAt:  time.Now().UTC(),
	}
}

// ---------- tests ----------

func TestPing_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	vehicleID := uuid.New()

	body := mustMarshal(t, models.PingRequest{
		VehicleID:  vehicleID,
		Latitude:   6.1375,
		Longitude:  1.2123,
		SpeedKmh:   60.0,
		HeadingDeg: 180.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicle/ping", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "driver")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.locations[vehicleID] == nil {
		t.Error("expected location to be stored")
	}
}

func TestPing_Unauthorized(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	body := mustMarshal(t, models.PingRequest{VehicleID: uuid.New(), Latitude: 6.0, Longitude: 1.0})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicle/ping", body)
	req.Header.Set("Content-Type", "application/json")
	// No claims in context.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestGetLocation_Success(t *testing.T) {
	store := newMockStore()
	vehicleID := uuid.New()
	seedLocation(store, vehicleID, 6.1375, 1.2123, 50.0)
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicle/"+vehicleID.String()+"/location", nil)
	req = withClaims(req, "dispatcher")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	if resp["success"] != true {
		t.Errorf("expected success true")
	}
}

func TestGetLocation_NotFound(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicle/"+uuid.New().String()+"/location", nil)
	req = withClaims(req, "dispatcher")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetRoute_Success(t *testing.T) {
	store := newMockStore()
	vehicleID := uuid.New()
	seedRoute(store, vehicleID)
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicle/"+vehicleID.String()+"/route", nil)
	req = withClaims(req, "dispatcher")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCalculateETA_Success(t *testing.T) {
	store := newMockStore()
	vehicleID := uuid.New()
	// Same origin and destination → distance=0, ETA=0 minutes.
	seedLocation(store, vehicleID, 6.1375, 1.2123, 0)
	r := newRouter(store)

	body := mustMarshal(t, models.ETARequest{DestinationLat: 6.1375, DestinationLng: 1.2123})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicle/"+vehicleID.String()+"/eta", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "dispatcher")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", resp["data"])
	}
	minutes, _ := data["estimated_minutes"].(float64)
	if int(minutes) != 0 {
		t.Errorf("expected 0 minutes for same location, got %d", int(minutes))
	}
}

func TestCalculateETA_100km(t *testing.T) {
	// Seed a vehicle at a position roughly 100km from destination.
	// We simulate by seeding location and a destination such that haversine ≈ 100km,
	// then at 60km/h the result should be 100 minutes.
	// Use two points that are approximately 100km apart.
	// Lomé (6.1375, 1.2123) to a point roughly 100km north.
	store := newMockStore()
	vehicleID := uuid.New()
	// At 0 speed, handler uses 60 km/h default.
	// Place vehicle at one point and destination 100km away.
	// Roughly 1 degree latitude ≈ 111 km; 0.9 degrees ≈ 100 km.
	seedLocation(store, vehicleID, 6.0, 1.0, 0)
	r := newRouter(store)

	body := mustMarshal(t, models.ETARequest{DestinationLat: 6.9, DestinationLng: 1.0})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicle/"+vehicleID.String()+"/eta", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "admin")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", resp["data"])
	}
	// 0.9 degrees lat ≈ 100km → at 60km/h ≈ 100 minutes. Allow ±15 min tolerance.
	minutes, _ := data["estimated_minutes"].(float64)
	if int(minutes) < 85 || int(minutes) > 115 {
		t.Errorf("expected ~100 minutes for ~100km at 60km/h, got %d", int(minutes))
	}
}

func TestHaversine_Accuracy(t *testing.T) {
	// Lomé (6.1375, 1.2123) to Accra (5.5560, -0.1969) ≈ 118-130 km.
	dist := haversineKm(6.1375, 1.2123, 5.5560, -0.1969)
	if dist < 118 || dist > 130 {
		t.Errorf("expected Lomé-Accra distance 118-130 km, got %.2f km", dist)
	}
}

func TestGetFleetStatus_Success(t *testing.T) {
	store := newMockStore()
	store.fleet = []models.FleetVehicleStatus{
		{VehicleID: uuid.New(), VehicleRef: "VEH-AAA", VehicleType: "truck", Status: "active"},
		{VehicleID: uuid.New(), VehicleRef: "VEH-BBB", VehicleType: "bike", Status: "idle"},
	}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicle/fleet/status", nil)
	req = withClaims(req, "admin")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array, got %T", resp["data"])
	}
	if len(data) != 2 {
		t.Errorf("expected 2 fleet vehicles, got %d", len(data))
	}
}

func TestSendAlert_Success(t *testing.T) {
	store := newMockStore()
	vehicleID := uuid.New()
	r := newRouter(store)

	body := mustMarshal(t, models.AlertRequest{AlertType: "speeding", Message: "Vehicle exceeded 120 km/h"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicle/"+vehicleID.String()+"/alert", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "admin")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.alerts) != 1 {
		t.Errorf("expected 1 alert in store, got %d", len(store.alerts))
	}
	if store.alerts[0].AlertType != "speeding" {
		t.Errorf("expected alert_type speeding, got %s", store.alerts[0].AlertType)
	}
}

func TestSendAlert_ForbiddenRole(t *testing.T) {
	store := newMockStore()
	vehicleID := uuid.New()
	r := newRouter(store)

	body := mustMarshal(t, models.AlertRequest{AlertType: "manual", Message: "Check vehicle"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicle/"+vehicleID.String()+"/alert", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "driver") // driver is not allowed to send alerts
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for driver role on alert, got %d", rec.Code)
	}
}

func TestVehicleRef_Format(t *testing.T) {
	id := uuid.New()
	ref := "VEH-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "VEH-") {
		t.Errorf("vehicle_ref must start with VEH-, got %s", ref)
	}
	// "VEH-" (4) + 8 hex chars = 12
	if len(ref) != 12 {
		t.Errorf("vehicle_ref length must be 12, got %d (%s)", len(ref), ref)
	}
}

// haversineKm is re-exposed here for direct unit testing without depending on the handlers package.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
