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
	"github.com/klinova/kinara-os/warehouse-service/auth"
	"github.com/klinova/kinara-os/warehouse-service/db"
	"github.com/klinova/kinara-os/warehouse-service/handlers"
	"github.com/klinova/kinara-os/warehouse-service/middleware"
	"github.com/klinova/kinara-os/warehouse-service/models"
)

type memStore struct {
	mu         sync.RWMutex
	warehouses map[uuid.UUID]*models.Warehouse
	stock      map[uuid.UUID]*models.StockItem
	movements  []models.StockMovement
	audit      []models.WarehouseAuditLog
}

func newMemStore() *memStore {
	return &memStore{warehouses: map[uuid.UUID]*models.Warehouse{}, stock: map[uuid.UUID]*models.StockItem{}}
}

func (s *memStore) CreateWarehouse(_ context.Context, w models.Warehouse) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.warehouses[w.ID] = &w; return nil
}
func (s *memStore) GetWarehouse(_ context.Context, id uuid.UUID) (*models.Warehouse, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	w, ok := s.warehouses[id]; if !ok { return nil, errNotFound }
	cp := *w; return &cp, nil
}
func (s *memStore) ListWarehouses(_ context.Context, p db.ListWarehouseParams) ([]models.Warehouse, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Warehouse
	for _, w := range s.warehouses {
		if p.Country != nil && w.Country != *p.Country { continue }
		result = append(result, *w)
	}
	return result, nil
}
func (s *memStore) CreateStockItem(_ context.Context, si models.StockItem) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.stock[si.ID] = &si; return nil
}
func (s *memStore) GetStockItem(_ context.Context, id uuid.UUID) (*models.StockItem, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	si, ok := s.stock[id]; if !ok { return nil, errNotFound }
	cp := *si; return &cp, nil
}
func (s *memStore) ListStockItems(_ context.Context, whID uuid.UUID) ([]models.StockItem, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.StockItem
	for _, si := range s.stock { if si.WarehouseID == whID { result = append(result, *si) } }
	return result, nil
}
func (s *memStore) ListLowStock(_ context.Context, whID uuid.UUID) ([]models.StockItem, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.StockItem
	for _, si := range s.stock { if si.WarehouseID == whID && si.QuantityOnHand <= si.ReorderLevel { result = append(result, *si) } }
	return result, nil
}
func (s *memStore) RecordMovement(_ context.Context, m models.StockMovement, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	si, ok := s.stock[m.StockItemID]; if !ok { return errNotFound }
	if m.MovementType == models.MovementReceive { si.QuantityOnHand += m.Quantity } else if m.MovementType == models.MovementDispatch { si.QuantityOnHand -= m.Quantity }
	s.movements = append(s.movements, m); return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.WarehouseAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewWarehouseHandlerWithStore(store)
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

func TestCreateWarehouse(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"name":"Accra Central Warehouse","country":"GH","address":"Tema Industrial Area","capacity_m3":5000,"manager_name":"Kweku Asante"}`
	resp, _ := http.Post(srv.URL+"/api/v1/warehouses", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestCreateWarehouse_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/warehouses", "application/json", bytes.NewBufferString(`{"name":"Incomplete"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetWarehouse(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateWarehouse(context.Background(), models.Warehouse{ID:id,Name:"Dakar Port Warehouse",Code:"WH-SN-001",Country:"SN",Address:"Port de Dakar",Status:models.WarehouseActive,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/warehouses/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestCreateStockItem(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	whID := uuid.New()
	store.CreateWarehouse(context.Background(), models.Warehouse{ID:whID,Name:"Kampala Warehouse",Code:"WH-UG-001",Country:"UG",Address:"Kampala Industrial",Status:models.WarehouseActive,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body := `{"sku":"MAIZE-50KG","product_name":"Maize Bags 50kg","category":"grain","unit":"bags","reorder_level":100}`
	resp, _ := http.Post(srv.URL+"/api/v1/warehouses/"+whID.String()+"/stock", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestRecordMovement_Receive(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	whID := uuid.New(); itemID := uuid.New()
	store.CreateWarehouse(context.Background(), models.Warehouse{ID:whID,Name:"Nairobi WH",Code:"WH-KE-001",Country:"KE",Address:"Industrial Area",Status:models.WarehouseActive,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.CreateStockItem(context.Background(), models.StockItem{ID:itemID,WarehouseID:whID,SKU:"BEANS-25KG",ProductName:"Beans 25kg",Unit:"bags",QuantityOnHand:0,ReorderLevel:50,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.StockMovementRequest{StockItemID:itemID.String(),MovementType:models.MovementReceive,Quantity:200})
	resp, _ := http.Post(srv.URL+"/api/v1/warehouses/"+whID.String()+"/movements", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	item, _ := store.GetStockItem(context.Background(), itemID)
	if item.QuantityOnHand != 200 { t.Fatalf("expected qty 200, got %f", item.QuantityOnHand) }
}

func TestListLowStock(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	whID := uuid.New(); itemID := uuid.New()
	store.CreateWarehouse(context.Background(), models.Warehouse{ID:whID,Name:"Lagos WH",Code:"WH-NG-001",Country:"NG",Address:"Apapa",Status:models.WarehouseActive,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.CreateStockItem(context.Background(), models.StockItem{ID:itemID,WarehouseID:whID,SKU:"RICE-50KG",ProductName:"Rice 50kg",Unit:"bags",QuantityOnHand:10,ReorderLevel:50,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/warehouses/" + whID.String() + "/stock/low")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 low-stock item, got %d", len(items)) }
}

func TestRecordMovement_MissingQty(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	whID := uuid.New(); itemID := uuid.New()
	store.CreateWarehouse(context.Background(), models.Warehouse{ID:whID,Name:"Abidjan WH",Code:"WH-CI-001",Country:"CI",Address:"Port Bouet",Status:models.WarehouseActive,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.CreateStockItem(context.Background(), models.StockItem{ID:itemID,WarehouseID:whID,SKU:"COFFEE",ProductName:"Coffee",Unit:"kg",QuantityOnHand:0,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.StockMovementRequest{StockItemID:itemID.String(),MovementType:models.MovementReceive,Quantity:0})
	resp, _ := http.Post(srv.URL+"/api/v1/warehouses/"+whID.String()+"/movements", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}
