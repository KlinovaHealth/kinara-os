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
	"github.com/klinova/kinara-os/transport-service/auth"
	"github.com/klinova/kinara-os/transport-service/db"
	"github.com/klinova/kinara-os/transport-service/handlers"
	"github.com/klinova/kinara-os/transport-service/middleware"
	"github.com/klinova/kinara-os/transport-service/models"
)

type memStore struct {
	mu     sync.RWMutex
	trips  map[uuid.UUID]*models.TransportTrip
	gps    []models.GPSUpdate
	audit  []models.TransportAuditLog
}

func newMemStore() *memStore { return &memStore{trips: map[uuid.UUID]*models.TransportTrip{}} }

func (s *memStore) CreateTrip(_ context.Context, t models.TransportTrip) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.trips[t.ID] = &t; return nil
}
func (s *memStore) GetTrip(_ context.Context, id uuid.UUID) (*models.TransportTrip, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	t, ok := s.trips[id]
	if !ok { return nil, errNotFound }
	cp := *t; return &cp, nil
}
func (s *memStore) ListTrips(_ context.Context, p db.ListTripsParams) ([]models.TransportTrip, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.TransportTrip
	for _, t := range s.trips {
		if p.Status != nil && t.Status != *p.Status { continue }
		if p.Country != nil && t.Country != *p.Country { continue }
		result = append(result, *t)
	}
	return result, nil
}
func (s *memStore) UpdateTripStatus(_ context.Context, id uuid.UUID, status models.TripStatus, delay string, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	t, ok := s.trips[id]; if !ok { return errNotFound }
	t.Status = status; t.DelayReasonCode = delay; t.UpdatedAt = now
	if status == models.TripEnRoute { t.ActualPickup = &now }
	if status == models.TripDelivered { t.ActualDelivery = &now }
	return nil
}
func (s *memStore) UpdateGPS(_ context.Context, id uuid.UUID, lat, lng float64, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	t, ok := s.trips[id]; if !ok { return errNotFound }
	t.CurrentLat = &lat; t.CurrentLng = &lng; t.LastGPSUpdate = &now; return nil
}
func (s *memStore) AddGPSUpdate(_ context.Context, g models.GPSUpdate) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.gps = append(s.gps, g); return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.TransportAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewTransportHandlerWithStore(store)
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

func TestCreateTrip(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	vid := uuid.New(); did := uuid.New()
	body, _ := json.Marshal(models.CreateTripRequest{VehicleID:vid.String(),DriverID:did.String(),Country:"KE",OriginAddress:"Nairobi",DestAddress:"Mombasa",ScheduledPickup:"2026-09-01T08:00:00Z",DistanceKm:480,CostPerKm:2.5})
	resp, _ := http.Post(srv.URL+"/api/v1/trips", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreateTrip_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/trips", "application/json", bytes.NewBufferString(`{"country":"KE"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetTrip(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); vid := uuid.New(); did := uuid.New()
	now := time.Now().UTC()
	store.CreateTrip(context.Background(), models.TransportTrip{ID:id,TripCode:"TR-test001",VehicleID:vid,DriverID:did,Status:models.TripScheduled,Country:"GH",OriginAddress:"Accra",DestAddress:"Tamale",ScheduledPickup:now.Add(time.Hour),Currency:"USD",CreatedAt:now,UpdatedAt:now})
	resp, _ := http.Get(srv.URL + "/api/v1/trips/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetTrip_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/trips/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}

func TestUpdateStatus_EnRoute(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); vid := uuid.New(); did := uuid.New()
	now := time.Now().UTC()
	store.CreateTrip(context.Background(), models.TransportTrip{ID:id,TripCode:"TR-test002",VehicleID:vid,DriverID:did,Status:models.TripScheduled,Country:"NG",OriginAddress:"Lagos",DestAddress:"Abuja",ScheduledPickup:now.Add(time.Hour),Currency:"USD",CreatedAt:now,UpdatedAt:now})
	body, _ := json.Marshal(models.UpdateTripStatusRequest{Status:models.TripEnRoute})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/trips/"+id.String()+"/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	trip, _ := store.GetTrip(context.Background(), id)
	if trip.ActualPickup == nil { t.Fatal("actual_pickup should be set when status is en_route") }
}

func TestUpdateStatus_Delayed(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); vid := uuid.New(); did := uuid.New()
	now := time.Now().UTC()
	store.CreateTrip(context.Background(), models.TransportTrip{ID:id,TripCode:"TR-test003",VehicleID:vid,DriverID:did,Status:models.TripEnRoute,Country:"TZ",OriginAddress:"Dar",DestAddress:"Dodoma",ScheduledPickup:now,Currency:"USD",CreatedAt:now,UpdatedAt:now})
	body, _ := json.Marshal(models.UpdateTripStatusRequest{Status:models.TripDelayed,DelayReasonCode:"road_block"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/trips/"+id.String()+"/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestUpdateGPS(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); vid := uuid.New(); did := uuid.New()
	now := time.Now().UTC()
	store.CreateTrip(context.Background(), models.TransportTrip{ID:id,TripCode:"TR-test004",VehicleID:vid,DriverID:did,Status:models.TripEnRoute,Country:"KE",OriginAddress:"Nairobi",DestAddress:"Mombasa",ScheduledPickup:now,Currency:"USD",CreatedAt:now,UpdatedAt:now})
	body, _ := json.Marshal(models.UpdateGPSRequest{Latitude:-2.0469,Longitude:37.9063,SpeedKph:80,Heading:135})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/trips/"+id.String()+"/gps", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestListTrips(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	for i := 0; i < 3; i++ {
		id := uuid.New(); vid := uuid.New(); did := uuid.New(); now := time.Now().UTC()
		store.CreateTrip(context.Background(), models.TransportTrip{ID:id,TripCode:"TR-"+uuid.New().String()[:6],VehicleID:vid,DriverID:did,Status:models.TripScheduled,Country:"SN",OriginAddress:"Dakar",DestAddress:"Thiès",ScheduledPickup:now.Add(time.Hour),Currency:"XOF",CreatedAt:now,UpdatedAt:now})
	}
	resp, _ := http.Get(srv.URL + "/api/v1/trips")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}
