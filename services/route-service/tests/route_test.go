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
	"github.com/klinova/kinara-os/route-service/auth"
	"github.com/klinova/kinara-os/route-service/db"
	"github.com/klinova/kinara-os/route-service/handlers"
	"github.com/klinova/kinara-os/route-service/middleware"
	"github.com/klinova/kinara-os/route-service/models"
)

type memStore struct {
	mu        sync.RWMutex
	routes    map[uuid.UUID]*models.Route
	schedules map[uuid.UUID]*models.RouteSchedule
	audit     []models.RouteAuditLog
}

func newMemStore() *memStore {
	return &memStore{routes: map[uuid.UUID]*models.Route{}, schedules: map[uuid.UUID]*models.RouteSchedule{}}
}

func (s *memStore) CreateRoute(_ context.Context, r models.Route) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.routes[r.ID] = &r; return nil
}
func (s *memStore) GetRoute(_ context.Context, id uuid.UUID) (*models.Route, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	r, ok := s.routes[id]
	if !ok { return nil, errNotFound }
	cp := *r; return &cp, nil
}
func (s *memStore) ListRoutes(_ context.Context, p db.ListRoutesParams) ([]models.Route, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Route
	for _, r := range s.routes {
		if r.Status != models.RouteActive { continue }
		if p.Country != nil && r.Country != *p.Country { continue }
		result = append(result, *r)
	}
	return result, nil
}
func (s *memStore) ScheduleRoute(_ context.Context, sched models.RouteSchedule) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.schedules[sched.ID] = &sched; return nil
}
func (s *memStore) ListSchedules(_ context.Context, routeID uuid.UUID) ([]models.RouteSchedule, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.RouteSchedule
	for _, s := range s.schedules { if s.RouteID == routeID { result = append(result, *s) } }
	return result, nil
}
func (s *memStore) UpdateScheduleStatus(_ context.Context, id uuid.UUID, status string, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	sched, ok := s.schedules[id]
	if !ok { return errNotFound }
	sched.Status = status; sched.UpdatedAt = now; return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.RouteAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewRouteHandlerWithStore(store)
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

func TestCreateRoute(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"name":"Nairobi-Mombasa Highway","route_type":"fixed","country":"KE","origin_name":"Nairobi CBD","origin_lat":-1.2921,"origin_lng":36.8219,"destination_name":"Mombasa Port","destination_lat":-4.0435,"destination_lng":39.6682,"distance_km":480,"estimated_hours":8}`
	resp, _ := http.Post(srv.URL+"/api/v1/routes", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreateRoute_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/routes", "application/json", bytes.NewBufferString(`{"name":"Incomplete Route"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetRoute(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateRoute(context.Background(), models.Route{ID: id, Name: "Accra-Kumasi", RouteCode: "RT-GH-001", RouteType: models.RouteFixed, Status: models.RouteActive, Country: "GH", OriginName: "Accra", DestName: "Kumasi", DistanceKm: 250, EstHours: 4, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/routes/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetRoute_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/routes/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}

func TestListRoutes_OnlyActive(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	store.CreateRoute(context.Background(), models.Route{ID: uuid.New(), Name: "Active Route", RouteCode: "RT-001", RouteType: models.RouteFixed, Status: models.RouteActive, Country: "TZ", OriginName: "Dar es Salaam", DestName: "Dodoma", DistanceKm: 450, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	store.CreateRoute(context.Background(), models.Route{ID: uuid.New(), Name: "Archived Route", RouteCode: "RT-002", RouteType: models.RouteFixed, Status: models.RouteArchived, Country: "TZ", OriginName: "Arusha", DestName: "Moshi", DistanceKm: 80, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/routes")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 active route, got %d", len(items)) }
}

func TestScheduleRoute(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateRoute(context.Background(), models.Route{ID: id, Name: "Lagos-Abuja", RouteCode: "RT-NG-001", RouteType: models.RouteFixed, Status: models.RouteActive, Country: "NG", OriginName: "Lagos", DestName: "Abuja", DistanceKm: 700, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body := `{"departure_time":"2026-09-01T08:00:00Z","notes":"Morning cargo run"}`
	resp, _ := http.Post(srv.URL+"/api/v1/routes/"+id.String()+"/schedule", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestScheduleRoute_MissingDeparture(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateRoute(context.Background(), models.Route{ID: id, Name: "Dakar-Thiès", RouteCode: "RT-SN-001", RouteType: models.RouteFixed, Status: models.RouteActive, Country: "SN", OriginName: "Dakar", DestName: "Thiès", DistanceKm: 70, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Post(srv.URL+"/api/v1/routes/"+id.String()+"/schedule", "application/json", bytes.NewBufferString(`{}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestListSchedules(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateRoute(context.Background(), models.Route{ID: id, Name: "Kigali-Musanze", RouteCode: "RT-RW-001", RouteType: models.RouteFixed, Status: models.RouteActive, Country: "RW", OriginName: "Kigali", DestName: "Musanze", DistanceKm: 110, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	store.ScheduleRoute(context.Background(), models.RouteSchedule{ID: uuid.New(), RouteID: id, DepartureTime: time.Now().Add(2 * time.Hour), Status: "scheduled", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/routes/" + id.String() + "/schedules")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestCreateRoute_WithWaypoints(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"name":"Multi-stop Route","country":"CI","origin_name":"Abidjan","destination_name":"Bouaké","distance_km":340,"estimated_hours":5,"waypoints":[{"sequence":1,"name":"Yamoussoukro","lat":6.8276,"lng":-5.2893}]}`
	resp, _ := http.Post(srv.URL+"/api/v1/routes", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	waypoints := data["waypoints"].([]interface{})
	if len(waypoints) != 1 { t.Fatalf("expected 1 waypoint, got %d", len(waypoints)) }
}
