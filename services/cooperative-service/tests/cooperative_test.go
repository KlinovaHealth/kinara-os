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
	"github.com/klinova/kinara-os/cooperative-service/auth"
	"github.com/klinova/kinara-os/cooperative-service/db"
	"github.com/klinova/kinara-os/cooperative-service/handlers"
	"github.com/klinova/kinara-os/cooperative-service/middleware"
	"github.com/klinova/kinara-os/cooperative-service/models"
)

// ─── In-memory store ─────────────────────────────────────────────────────────

type memStore struct {
	coops         map[uuid.UUID]*models.Cooperative
	members       map[uuid.UUID]*models.CoopMember
	pools         map[uuid.UUID]*models.SellingPool
	contributions map[uuid.UUID]*models.PoolContribution
}

func newMemStore() *memStore {
	return &memStore{
		coops:         map[uuid.UUID]*models.Cooperative{},
		members:       map[uuid.UUID]*models.CoopMember{},
		pools:         map[uuid.UUID]*models.SellingPool{},
		contributions: map[uuid.UUID]*models.PoolContribution{},
	}
}

var errNotFound = &notFoundErr{}

type notFoundErr struct{}

func (e *notFoundErr) Error() string { return "not found" }

func (m *memStore) CreateCoop(_ context.Context, c models.Cooperative) error {
	m.coops[c.ID] = &c
	return nil
}
func (m *memStore) GetCoop(_ context.Context, id uuid.UUID) (*models.Cooperative, error) {
	c, ok := m.coops[id]
	if !ok {
		return nil, errNotFound
	}
	return c, nil
}
func (m *memStore) ListCoops(_ context.Context, _ db.ListCoopsParams) ([]models.Cooperative, error) {
	var result []models.Cooperative
	for _, c := range m.coops {
		result = append(result, *c)
	}
	return result, nil
}
func (m *memStore) CountCoops(_ context.Context, _ db.ListCoopsParams) (int, error) {
	return len(m.coops), nil
}
func (m *memStore) UpdateCoopStats(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }

func (m *memStore) AddMember(_ context.Context, mem models.CoopMember) error {
	m.members[mem.ID] = &mem
	return nil
}
func (m *memStore) GetMember(_ context.Context, id uuid.UUID) (*models.CoopMember, error) {
	mem, ok := m.members[id]
	if !ok {
		return nil, errNotFound
	}
	return mem, nil
}
func (m *memStore) GetMemberByFarmer(_ context.Context, _, _ uuid.UUID) (*models.CoopMember, error) {
	return nil, errNotFound
}
func (m *memStore) ListMembers(_ context.Context, coopID uuid.UUID, _, _ int) ([]models.CoopMember, error) {
	var result []models.CoopMember
	for _, mem := range m.members {
		if mem.CoopID == coopID {
			result = append(result, *mem)
		}
	}
	return result, nil
}
func (m *memStore) UpdateMember(_ context.Context, id uuid.UUID, req models.UpdateMemberRequest, now time.Time) error {
	mem, ok := m.members[id]
	if !ok {
		return errNotFound
	}
	if req.Role != nil {
		mem.Role = *req.Role
	}
	if req.Status != nil {
		mem.Status = *req.Status
	}
	mem.UpdatedAt = now
	return nil
}

func (m *memStore) CreatePool(_ context.Context, p models.SellingPool) error {
	m.pools[p.ID] = &p
	return nil
}
func (m *memStore) GetPool(_ context.Context, id uuid.UUID) (*models.SellingPool, error) {
	p, ok := m.pools[id]
	if !ok {
		return nil, errNotFound
	}
	return p, nil
}
func (m *memStore) ListPools(_ context.Context, coopID uuid.UUID, _, _ int) ([]models.SellingPool, error) {
	var result []models.SellingPool
	for _, p := range m.pools {
		if p.CoopID == coopID {
			result = append(result, *p)
		}
	}
	return result, nil
}
func (m *memStore) ClosePool(_ context.Context, id uuid.UUID, now time.Time) error {
	p, ok := m.pools[id]
	if !ok {
		return errNotFound
	}
	p.Status = models.PoolClosed
	p.UpdatedAt = now
	return nil
}
func (m *memStore) RecordSale(_ context.Context, id uuid.UUID, pricePerKg, totalRevenue float64, now time.Time) error {
	p, ok := m.pools[id]
	if !ok {
		return errNotFound
	}
	p.Status = models.PoolSold
	p.PricePerKg = pricePerKg
	p.TotalRevenue = totalRevenue
	p.UpdatedAt = now
	return nil
}
func (m *memStore) AddPoolQuantity(_ context.Context, id uuid.UUID, qty float64, _ time.Time) error {
	p, ok := m.pools[id]
	if !ok {
		return errNotFound
	}
	p.CollectedQtyKg += qty
	return nil
}

func (m *memStore) AddContribution(_ context.Context, c models.PoolContribution) error {
	m.contributions[c.ID] = &c
	return nil
}
func (m *memStore) GetContribution(_ context.Context, id uuid.UUID) (*models.PoolContribution, error) {
	c, ok := m.contributions[id]
	if !ok {
		return nil, errNotFound
	}
	return c, nil
}
func (m *memStore) ListContributions(_ context.Context, poolID uuid.UUID) ([]models.PoolContribution, error) {
	var result []models.PoolContribution
	for _, c := range m.contributions {
		if c.PoolID == poolID {
			result = append(result, *c)
		}
	}
	return result, nil
}
func (m *memStore) DistributePayouts(_ context.Context, _ uuid.UUID, _ float64, _ time.Time) error {
	return nil
}
func (m *memStore) MarkPayoutPaid(_ context.Context, id uuid.UUID, now time.Time) error {
	c, ok := m.contributions[id]
	if !ok {
		return errNotFound
	}
	c.PayoutPaid = true
	c.PaidAt = &now
	return nil
}
func (m *memStore) InsertAuditLog(_ context.Context, _ models.CoopAuditLog) error { return nil }

// ─── Router ───────────────────────────────────────────────────────────────────

func setupRouter(store handlers.Store) *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &auth.Claims{UserID: uuid.New().String(), Role: "farmer", FacilityID: "KE"}
			ctx := middleware.SetClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	h := handlers.NewCoopHandlerWithStore(store)
	h.RegisterRoutes(api)
	return r
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestCreateCoop(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{
		"name":          "Nakuru Farmers Coop",
		"coop_type":     "marketing",
		"country":       "KE",
		"region":        "Nakuru",
		"contact_phone": "+254700000001",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cooperatives", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.coops) != 1 {
		t.Fatal("expected coop to be stored")
	}
}

func TestCreateCoopMissingFields(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{"name": "x"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cooperatives", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetCoopNotFound(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cooperatives/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAddMember(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	coopID := uuid.New()
	store.coops[coopID] = &models.Cooperative{
		ID: coopID, Name: "Test Coop", Status: models.CoopActive,
		Country: "GH", ContactPhone: "+233200000001",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	body := map[string]interface{}{
		"farmer_id":   uuid.New().String(),
		"role":        "member",
		"shares_held": 5,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cooperatives/"+coopID.String()+"/members", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreatePool(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	coopID := uuid.New()
	store.coops[coopID] = &models.Cooperative{
		ID: coopID, Name: "Test Coop", Status: models.CoopActive,
		Country: "TZ", ContactPhone: "+255700000001",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	body := map[string]interface{}{
		"crop_type":           "sunflower",
		"target_quantity_kg":  5000.0,
		"price_per_kg":        0.40,
		"currency":            "USD",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cooperatives/"+coopID.String()+"/pools", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContributeToPool(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	coopID := uuid.New()
	poolID := uuid.New()
	store.coops[coopID] = &models.Cooperative{
		ID: coopID, Name: "Coop", Status: models.CoopActive,
		Country: "UG", ContactPhone: "+256700000001",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	store.pools[poolID] = &models.SellingPool{
		ID: poolID, CoopID: coopID, CropType: "groundnuts",
		TargetQtyKg: 2000, CollectedQtyKg: 0, Status: models.PoolOpen,
		Currency: "USD", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	body := map[string]interface{}{
		"farmer_id":   uuid.New().String(),
		"quantity_kg": 300.0,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pools/"+poolID.String()+"/contribute", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.pools[poolID].CollectedQtyKg != 300 {
		t.Fatalf("expected collected_qty=300, got %f", store.pools[poolID].CollectedQtyKg)
	}
}

func TestContributeToClosedPool(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	poolID := uuid.New()
	store.pools[poolID] = &models.SellingPool{
		ID: poolID, CoopID: uuid.New(), CropType: "wheat",
		TargetQtyKg: 1000, CollectedQtyKg: 1000, Status: models.PoolClosed,
		Currency: "USD", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	body := map[string]interface{}{"quantity_kg": 50.0}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pools/"+poolID.String()+"/contribute", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestRecordSale(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	poolID := uuid.New()
	store.pools[poolID] = &models.SellingPool{
		ID: poolID, CoopID: uuid.New(), CropType: "sesame",
		TargetQtyKg: 1000, CollectedQtyKg: 950, Status: models.PoolClosed,
		Currency: "USD", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	body := map[string]interface{}{
		"price_per_kg":   0.55,
		"total_revenue":  522.5,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pools/"+poolID.String()+"/sale", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.pools[poolID].Status != models.PoolSold {
		t.Fatal("expected pool status to be sold")
	}
}

func TestListMembers(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	coopID := uuid.New()
	store.coops[coopID] = &models.Cooperative{
		ID: coopID, Name: "Coop", Status: models.CoopActive,
		Country: "RW", ContactPhone: "+250700000001",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	for i := 0; i < 3; i++ {
		mid := uuid.New()
		store.members[mid] = &models.CoopMember{
			ID: mid, CoopID: coopID, FarmerID: uuid.New(),
			Role: models.RoleMember, Status: models.MemberActive,
			SharesHeld: 1, JoinedAt: time.Now(), UpdatedAt: time.Now(),
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cooperatives/"+coopID.String()+"/members", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
