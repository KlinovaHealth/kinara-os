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
	"github.com/klinova/kinara-os/cargo-service/auth"
	"github.com/klinova/kinara-os/cargo-service/db"
	"github.com/klinova/kinara-os/cargo-service/handlers"
	"github.com/klinova/kinara-os/cargo-service/middleware"
	"github.com/klinova/kinara-os/cargo-service/models"
)

type memStore struct {
	mu       sync.RWMutex
	bookings map[uuid.UUID]*models.CargoBooking
	refs     map[string]uuid.UUID
	events   []models.TrackingEvent
	audit    []models.CargoAuditLog
}

func newMemStore() *memStore {
	return &memStore{bookings: map[uuid.UUID]*models.CargoBooking{}, refs: map[string]uuid.UUID{}}
}

func (s *memStore) CreateBooking(_ context.Context, b models.CargoBooking) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.bookings[b.ID] = &b; s.refs[b.BookingRef] = b.ID; return nil
}
func (s *memStore) GetBooking(_ context.Context, id uuid.UUID) (*models.CargoBooking, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	b, ok := s.bookings[id]
	if !ok { return nil, errNotFound }
	cp := *b; return &cp, nil
}
func (s *memStore) GetBookingByRef(_ context.Context, ref string) (*models.CargoBooking, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	id, ok := s.refs[ref]
	if !ok { return nil, errNotFound }
	b := s.bookings[id]; cp := *b; return &cp, nil
}
func (s *memStore) ListBookings(_ context.Context, p db.ListCargoParams) ([]models.CargoBooking, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.CargoBooking
	for _, b := range s.bookings {
		if p.Status != nil && b.Status != *p.Status { continue }
		result = append(result, *b)
	}
	return result, nil
}
func (s *memStore) UpdateBookingStatus(_ context.Context, id uuid.UUID, status models.CargoStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	b, ok := s.bookings[id]
	if !ok { return errNotFound }
	b.Status = status; b.UpdatedAt = now
	if status == models.CargoPickedUp { b.PickupAt = &now }
	if status == models.CargoDelivered { b.DeliveredAt = &now }
	return nil
}
func (s *memStore) AssignCargo(_ context.Context, id, vehicleID, driverID uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	b, ok := s.bookings[id]
	if !ok { return errNotFound }
	b.AssignedVehicleID = &vehicleID; b.AssignedDriverID = &driverID; b.UpdatedAt = now; return nil
}
func (s *memStore) AddTrackingEvent(_ context.Context, e models.TrackingEvent) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.events = append(s.events, e); return nil
}
func (s *memStore) ListTracking(_ context.Context, cargoID uuid.UUID) ([]models.TrackingEvent, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.TrackingEvent
	for _, e := range s.events { if e.CargoID == cargoID { result = append(result, e) } }
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.CargoAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewCargoHandlerWithStore(store)
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

func TestCreateBooking(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"cargo_type":"general","description":"Maize bags","weight_kg":5000,"volume_m3":8,"origin_address":"Kumasi Central Market","destination_address":"Accra Tema Port","freight_cost":350}`
	resp, _ := http.Post(srv.URL+"/api/v1/cargo", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreateBooking_MissingWeight(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"origin_address":"Kumasi","destination_address":"Accra"}`
	resp, _ := http.Post(srv.URL+"/api/v1/cargo", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetBooking(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateBooking(context.Background(), models.CargoBooking{ID: id, BookingRef: "KN-abc12345", ShipperID: uuid.New(), CargoType: models.CargoBulkGrain, WeightKg: 10000, OriginAddress: "Mombasa", DestinationAddress: "Nairobi", Status: models.CargoPending, Currency: "USD", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/cargo/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetBookingByRef(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	ref := "KN-testref1"
	store.CreateBooking(context.Background(), models.CargoBooking{ID: id, BookingRef: ref, ShipperID: uuid.New(), CargoType: models.CargoGeneral, WeightKg: 500, OriginAddress: "Dakar", DestinationAddress: "Saint-Louis", Status: models.CargoPending, Currency: "XOF", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/cargo/ref/" + ref)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestUpdateStatus_PickedUp(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateBooking(context.Background(), models.CargoBooking{ID: id, BookingRef: "KN-pickup01", ShipperID: uuid.New(), CargoType: models.CargoPerishable, WeightKg: 2000, OriginAddress: "Kampala", DestinationAddress: "Entebbe", Status: models.CargoPending, Currency: "UGX", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body := `{"status":"picked_up"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/cargo/"+id.String()+"/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	b, _ := store.GetBooking(context.Background(), id)
	if b.PickupAt == nil { t.Fatal("pickup_at should be set when status is picked_up") }
}

func TestAssignCargo(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); vid := uuid.New(); did := uuid.New()
	store.CreateBooking(context.Background(), models.CargoBooking{ID: id, BookingRef: "KN-assign01", ShipperID: uuid.New(), CargoType: models.CargoGeneral, WeightKg: 1000, OriginAddress: "Lomé", DestinationAddress: "Cotonou", Status: models.CargoPending, Currency: "XOF", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body, _ := json.Marshal(models.AssignCargoRequest{VehicleID: vid.String(), DriverID: did.String()})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/cargo/"+id.String()+"/assign", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestAddTrackingEvent(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateBooking(context.Background(), models.CargoBooking{ID: id, BookingRef: "KN-track01", ShipperID: uuid.New(), CargoType: models.CargoMedical, WeightKg: 50, OriginAddress: "Kigali", DestinationAddress: "Butare", Status: models.CargoPending, Currency: "RWF", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body := `{"status":"in_transit","location":"Gitarama","latitude":-2.0758,"longitude":29.7567,"notes":"On schedule"}`
	resp, _ := http.Post(srv.URL+"/api/v1/cargo/"+id.String()+"/track", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestGetBooking_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/cargo/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}
