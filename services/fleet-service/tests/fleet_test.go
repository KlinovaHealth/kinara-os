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
	"github.com/klinova/kinara-os/fleet-service/auth"
	"github.com/klinova/kinara-os/fleet-service/db"
	"github.com/klinova/kinara-os/fleet-service/handlers"
	"github.com/klinova/kinara-os/fleet-service/middleware"
	"github.com/klinova/kinara-os/fleet-service/models"
)

type memStore struct {
	mu       sync.RWMutex
	vehicles map[uuid.UUID]*models.Vehicle
	maint    []models.MaintenanceRecord
	fuel     []models.FuelLog
	audit    []models.FleetAuditLog
}

func newMemStore() *memStore {
	return &memStore{vehicles: map[uuid.UUID]*models.Vehicle{}}
}

func (s *memStore) CreateVehicle(_ context.Context, v models.Vehicle) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.vehicles[v.ID] = &v; return nil
}
func (s *memStore) GetVehicle(_ context.Context, id uuid.UUID) (*models.Vehicle, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	v, ok := s.vehicles[id]
	if !ok { return nil, errNotFound }
	cp := *v; return &cp, nil
}
func (s *memStore) ListVehicles(_ context.Context, p db.ListVehiclesParams) ([]models.Vehicle, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Vehicle
	for _, v := range s.vehicles {
		if p.Status != nil && v.Status != *p.Status { continue }
		if p.Country != nil && v.Country != *p.Country { continue }
		result = append(result, *v)
	}
	return result, nil
}
func (s *memStore) UpdateVehicle(_ context.Context, id uuid.UUID, req models.UpdateVehicleRequest, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	v, ok := s.vehicles[id]
	if !ok { return errNotFound }
	if req.Status != nil { v.Status = *req.Status }
	if req.Notes != nil { v.Notes = *req.Notes }
	v.UpdatedAt = now; return nil
}
func (s *memStore) LogMaintenance(_ context.Context, m models.MaintenanceRecord) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.maint = append(s.maint, m); return nil
}
func (s *memStore) ListMaintenance(_ context.Context, vehicleID uuid.UUID) ([]models.MaintenanceRecord, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.MaintenanceRecord
	for _, m := range s.maint { if m.VehicleID == vehicleID { result = append(result, m) } }
	return result, nil
}
func (s *memStore) LogFuel(_ context.Context, f models.FuelLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.fuel = append(s.fuel, f); return nil
}
func (s *memStore) ListFuelLogs(_ context.Context, vehicleID uuid.UUID) ([]models.FuelLog, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.FuelLog
	for _, f := range s.fuel { if f.VehicleID == vehicleID { result = append(result, f) } }
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.FleetAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewFleetHandlerWithStore(store)
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

func TestCreateVehicle(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"registration_no":"KE-001-A","vehicle_type":"truck","make":"Scania","model":"R500","year":2020,"fuel_type":"diesel","payload_capacity_kg":15000,"country":"KE","base_location":"Nairobi"}`
	resp, _ := http.Post(srv.URL+"/api/v1/vehicles", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreateVehicle_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/vehicles", "application/json", bytes.NewBufferString(`{"make":"Ford"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetVehicle(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateVehicle(context.Background(), models.Vehicle{ID: id, RegistrationNo: "TZ-002", VehicleType: models.VehicleTruck, Status: models.VehicleAvailable, Country: "TZ", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/vehicles/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetVehicle_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/vehicles/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}

func TestListVehicles(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	store.CreateVehicle(context.Background(), models.Vehicle{ID: uuid.New(), RegistrationNo: "GH-001", VehicleType: models.VehiclePickup, Status: models.VehicleAvailable, Country: "GH", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	store.CreateVehicle(context.Background(), models.Vehicle{ID: uuid.New(), RegistrationNo: "GH-002", VehicleType: models.VehicleTruck, Status: models.VehicleInTransit, Country: "GH", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/vehicles")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestUpdateVehicle(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateVehicle(context.Background(), models.Vehicle{ID: id, RegistrationNo: "NG-003", VehicleType: models.VehicleTruck, Status: models.VehicleAvailable, Country: "NG", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	status := models.VehicleInRepair
	body, _ := json.Marshal(models.UpdateVehicleRequest{Status: &status})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/vehicles/"+id.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestLogMaintenance(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateVehicle(context.Background(), models.Vehicle{ID: id, RegistrationNo: "SN-001", VehicleType: models.VehicleTruck, Status: models.VehicleAvailable, Country: "SN", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body := `{"service_type":"oil_change","description":"Regular oil change","odometer_km":50000,"cost":120,"currency":"USD","serviced_by":"Ahmed Garage"}`
	resp, _ := http.Post(srv.URL+"/api/v1/vehicles/"+id.String()+"/maintenance", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestLogFuel(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateVehicle(context.Background(), models.Vehicle{ID: id, RegistrationNo: "CI-001", VehicleType: models.VehicleVan, Status: models.VehicleAvailable, Country: "CI", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body := `{"litres_filled":80,"cost_per_litre":1.20,"currency":"USD","odometer_km":60000,"station":"TotalEnergies Abidjan"}`
	resp, _ := http.Post(srv.URL+"/api/v1/vehicles/"+id.String()+"/fuel", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestLogFuel_InvalidLitres(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateVehicle(context.Background(), models.Vehicle{ID: id, RegistrationNo: "ET-001", VehicleType: models.VehicleMotorcycle, Status: models.VehicleAvailable, Country: "ET", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body := `{"litres_filled":-5,"cost_per_litre":1.5}`
	resp, _ := http.Post(srv.URL+"/api/v1/vehicles/"+id.String()+"/fuel", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}
