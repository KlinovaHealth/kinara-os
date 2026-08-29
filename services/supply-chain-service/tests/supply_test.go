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
	"github.com/klinova/kinara-os/supply-chain-service/db"
	"github.com/klinova/kinara-os/supply-chain-service/handlers"
	"github.com/klinova/kinara-os/supply-chain-service/models"
)

type memStore struct {
	mu        sync.RWMutex
	shipments map[uuid.UUID]*models.Shipment
	tracking  map[uuid.UUID][]models.TrackingEvent
	audit     []models.SupplyAuditLog
}

func newMemStore() *memStore {
	return &memStore{
		shipments: map[uuid.UUID]*models.Shipment{},
		tracking:  map[uuid.UUID][]models.TrackingEvent{},
	}
}

func (s *memStore) CreateShipment(_ context.Context, sh models.Shipment) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.shipments[sh.ID] = &sh; return nil
}
func (s *memStore) GetShipment(_ context.Context, id uuid.UUID) (*models.Shipment, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	sh, ok := s.shipments[id]
	if !ok { return nil, errNotFound }
	cp := *sh; return &cp, nil
}
func (s *memStore) ListShipments(_ context.Context, p db.ListShipmentsParams) ([]models.Shipment, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Shipment
	for _, sh := range s.shipments {
		if p.FarmerID != nil && sh.FarmerID != *p.FarmerID { continue }
		if p.Status != nil && string(sh.Status) != *p.Status { continue }
		result = append(result, *sh)
	}
	return result, nil
}
func (s *memStore) UpdateShipmentStatus(_ context.Context, id uuid.UUID, status models.ShipmentStatus, actualCost *float64, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if sh, ok := s.shipments[id]; ok {
		sh.Status = status
		if actualCost != nil { sh.ActualCostUSD = actualCost }
		sh.UpdatedAt = now
	}
	return nil
}
func (s *memStore) AddTrackingEvent(_ context.Context, e models.TrackingEvent) error {
	s.mu.Lock(); defer s.mu.Unlock()
	s.tracking[e.ShipmentID] = append(s.tracking[e.ShipmentID], e)
	return nil
}
func (s *memStore) ListTrackingEvents(_ context.Context, shipmentID uuid.UUID) ([]models.TrackingEvent, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	return s.tracking[shipmentID], nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.SupplyAuditLog) error {
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

func TestCreateShipment(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateShipmentRequest{
		FarmerID:      uuid.New().String(),
		CommodityName: "maize",
		QuantityKg:    500,
		OriginLocation: "Kpalimé, Togo",
		DestLocation:   "Lomé Market, Togo",
	})
	resp, _ := http.Post(srv.URL+"/api/v1/supply-chain/shipments", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	ref := data["shipment_ref"].(string)
	if len(ref) < 3 { t.Fatalf("expected shipment_ref, got %q", ref) }
}

func TestCreateShipment_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"commodity_name":"rice"}`
	resp, _ := http.Post(srv.URL+"/api/v1/supply-chain/shipments", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetShipment(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	fid := uuid.New()
	now := time.Now()
	store.CreateShipment(context.Background(), models.Shipment{
		ID: id, ShipmentRef: "SC-TEST001", FarmerID: fid,
		CommodityName: "cassava", QuantityKg: 200,
		OriginLocation: "Sokodé, Togo", DestLocation: "Lomé, Togo",
		Status: models.ShipmentPending, PillarHandoff: models.HandoffAgriToLogistics,
		EstimatedCostUSD: 15.0, CreatedAt: now, UpdatedAt: now,
	})
	resp, _ := http.Get(srv.URL + "/api/v1/supply-chain/shipments/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestUpdateStatus_Tracking(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	now := time.Now()
	store.CreateShipment(context.Background(), models.Shipment{
		ID: id, ShipmentRef: "SC-TEST002", FarmerID: uuid.New(),
		CommodityName: "sorghum", QuantityKg: 1000,
		OriginLocation: "Dapaong, Togo", DestLocation: "Accra, Ghana",
		Status: models.ShipmentPending, PillarHandoff: models.HandoffAgriToLogistics,
		EstimatedCostUSD: 80.0, CreatedAt: now, UpdatedAt: now,
	})
	body := `{"status":"picked_up","location":"Dapaong Cooperative","note":"Loaded onto truck"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/supply-chain/shipments/"+id.String()+"/status",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }

	sh, _ := store.GetShipment(context.Background(), id)
	if sh.Status != models.ShipmentPickedUp { t.Fatalf("expected picked_up, got %s", sh.Status) }

	// Check tracking event added
	store.mu.RLock()
	events := store.tracking[id]
	store.mu.RUnlock()
	if len(events) == 0 { t.Fatal("expected tracking event to be recorded") }
}

func TestGetTracking(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.AddTrackingEvent(context.Background(), models.TrackingEvent{
		ID: uuid.New(), ShipmentID: id, Status: models.ShipmentInTransit,
		Location: "Border Checkpoint", Note: "Crossed TG-GH border", RecordedAt: time.Now(),
	})
	resp, _ := http.Get(srv.URL + "/api/v1/supply-chain/shipments/" + id.String() + "/tracking")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestEstimateCost(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/supply-chain/cost?origin=Lomé&destination=Accra&quantity_kg=1000")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["estimated_cost_usd"].(float64) <= 0 { t.Fatal("expected positive cost estimate") }
}

func TestListShipments_ByFarmer(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	fid := uuid.New()
	now := time.Now()
	store.CreateShipment(context.Background(), models.Shipment{
		ID: uuid.New(), ShipmentRef: "SC-F01", FarmerID: fid,
		CommodityName: "yam", QuantityKg: 300,
		OriginLocation: "Atakpamé", DestLocation: "Lomé",
		Status: models.ShipmentPending, PillarHandoff: models.HandoffAgriToLogistics,
		EstimatedCostUSD: 20, CreatedAt: now, UpdatedAt: now,
	})
	resp, _ := http.Get(srv.URL + "/api/v1/supply-chain/shipments?farmer_id=" + fid.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 shipment for farmer, got %d", len(items)) }
}
