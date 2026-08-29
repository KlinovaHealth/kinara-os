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
	"github.com/klinova/kinara-os/shipment-service/auth"
	"github.com/klinova/kinara-os/shipment-service/db"
	"github.com/klinova/kinara-os/shipment-service/handlers"
	"github.com/klinova/kinara-os/shipment-service/middleware"
	"github.com/klinova/kinara-os/shipment-service/models"
)

type memStore struct {
	mu        sync.RWMutex
	shipments map[uuid.UUID]*models.Shipment
	codes     map[string]uuid.UUID
	events    []models.ShipmentEvent
	audit     []models.ShipmentAuditLog
}

func newMemStore() *memStore {
	return &memStore{shipments: map[uuid.UUID]*models.Shipment{}, codes: map[string]uuid.UUID{}}
}

func (s *memStore) CreateShipment(_ context.Context, sh models.Shipment) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.shipments[sh.ID] = &sh; s.codes[sh.TrackingCode] = sh.ID; return nil
}
func (s *memStore) GetShipment(_ context.Context, id uuid.UUID) (*models.Shipment, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	sh, ok := s.shipments[id]; if !ok { return nil, errNotFound }
	cp := *sh; return &cp, nil
}
func (s *memStore) GetShipmentByTrackingCode(_ context.Context, code string) (*models.Shipment, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	id, ok := s.codes[code]; if !ok { return nil, errNotFound }
	sh := s.shipments[id]; cp := *sh; return &cp, nil
}
func (s *memStore) ListShipments(_ context.Context, p db.ListShipmentsParams) ([]models.Shipment, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Shipment
	for _, sh := range s.shipments {
		if p.Status != nil && sh.Status != *p.Status { continue }
		result = append(result, *sh)
	}
	return result, nil
}
func (s *memStore) UpdateShipmentStatus(_ context.Context, id uuid.UUID, status models.ShipmentStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	sh, ok := s.shipments[id]; if !ok { return errNotFound }
	sh.Status = status; sh.UpdatedAt = now
	if status == models.ShipmentPicked { sh.PickedAt = &now }
	if status == models.ShipmentDelivered { sh.DeliveredAt = &now }
	return nil
}
func (s *memStore) AddEvent(_ context.Context, e models.ShipmentEvent) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.events = append(s.events, e); return nil
}
func (s *memStore) ListEvents(_ context.Context, shipmentID uuid.UUID) ([]models.ShipmentEvent, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.ShipmentEvent
	for _, e := range s.events { if e.ShipmentID == shipmentID { result = append(result, e) } }
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.ShipmentAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewShipmentHandlerWithStore(store)
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

func TestCreateShipment(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateShipmentRequest{RecipientName:"Yaw Darko",RecipientPhone:"+233244100200",OriginAddress:"Kumasi Market",OriginCountry:"GH",DestAddress:"Accra",DestCountry:"GH",WeightKg:25,DeclaredValue:500,ServiceLevel:models.ServiceStandard})
	resp, _ := http.Post(srv.URL+"/api/v1/shipments", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreateShipment_MissingWeight(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"recipient_name":"Test","origin_address":"A","destination_address":"B"}`
	resp, _ := http.Post(srv.URL+"/api/v1/shipments", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetShipment(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateShipment(context.Background(), models.Shipment{ID:id,TrackingCode:"KN-testcode01",SenderID:uuid.New(),RecipientName:"Amara Bah",RecipientPhone:"+22175123456",OriginAddress:"Conakry",OriginCountry:"GN",DestAddress:"Dakar",DestCountry:"SN",WeightKg:10,Status:models.ShipmentCreated,Currency:"USD",ServiceLevel:models.ServiceStandard,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/shipments/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetByTrackingCode(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); code := "KN-abcdefghij"
	store.CreateShipment(context.Background(), models.Shipment{ID:id,TrackingCode:code,SenderID:uuid.New(),RecipientName:"Test",RecipientPhone:"+254700000000",OriginAddress:"Nairobi",OriginCountry:"KE",DestAddress:"Mombasa",DestCountry:"KE",WeightKg:5,Status:models.ShipmentCreated,Currency:"USD",ServiceLevel:models.ServiceExpress,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/shipments/track/" + code)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestUpdateStatus_Picked(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateShipment(context.Background(), models.Shipment{ID:id,TrackingCode:"KN-picked001",SenderID:uuid.New(),RecipientName:"Test",RecipientPhone:"+22677654321",OriginAddress:"Ouaga",OriginCountry:"BF",DestAddress:"Bobo",DestCountry:"BF",WeightKg:3,Status:models.ShipmentCreated,Currency:"XOF",ServiceLevel:models.ServiceStandard,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body := `{"status":"picked"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/shipments/"+id.String()+"/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	sh, _ := store.GetShipment(context.Background(), id)
	if sh.PickedAt == nil { t.Fatal("picked_at should be set") }
}

func TestAddEvent(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateShipment(context.Background(), models.Shipment{ID:id,TrackingCode:"KN-event001",SenderID:uuid.New(),RecipientName:"Test",RecipientPhone:"+255754321098",OriginAddress:"Dar es Salaam",OriginCountry:"TZ",DestAddress:"Arusha",DestCountry:"TZ",WeightKg:8,Status:models.ShipmentInTransit,Currency:"TZS",ServiceLevel:models.ServiceStandard,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.AddShipmentEventRequest{Status:models.ShipmentInTransit,Location:"Morogoro",Notes:"On track"})
	resp, _ := http.Post(srv.URL+"/api/v1/shipments/"+id.String()+"/events", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestGetShipment_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/shipments/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}
