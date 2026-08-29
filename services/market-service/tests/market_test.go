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
	"github.com/klinova/kinara-os/market-service/auth"
	"github.com/klinova/kinara-os/market-service/db"
	"github.com/klinova/kinara-os/market-service/handlers"
	"github.com/klinova/kinara-os/market-service/middleware"
	"github.com/klinova/kinara-os/market-service/models"
)

// ─── In-memory store for unit tests ──────────────────────────────────────────

type memStore struct {
	listings map[uuid.UUID]*models.MarketListing
	bids     map[uuid.UUID]*models.MarketBid
	prices   []models.PriceRecord
}

func newMemStore() *memStore {
	return &memStore{
		listings: map[uuid.UUID]*models.MarketListing{},
		bids:     map[uuid.UUID]*models.MarketBid{},
	}
}

func (m *memStore) CreateListing(_ context.Context, l models.MarketListing) error {
	m.listings[l.ID] = &l
	return nil
}
func (m *memStore) GetListing(_ context.Context, id uuid.UUID) (*models.MarketListing, error) {
	l, ok := m.listings[id]
	if !ok {
		return nil, errNotFound
	}
	return l, nil
}
func (m *memStore) ListListings(_ context.Context, p db.ListListingsParams) ([]models.MarketListing, error) {
	var result []models.MarketListing
	for _, l := range m.listings {
		if l.Status == models.ListingActive {
			result = append(result, *l)
		}
	}
	return result, nil
}
func (m *memStore) CountListings(_ context.Context, _ db.ListListingsParams) (int, error) {
	return len(m.listings), nil
}
func (m *memStore) UpdateListing(_ context.Context, id uuid.UUID, req models.UpdateListingRequest, now time.Time) error {
	l, ok := m.listings[id]
	if !ok {
		return errNotFound
	}
	if req.PricePerUnit != nil {
		l.PricePerUnit = *req.PricePerUnit
	}
	if req.QuantityAvail != nil {
		l.QuantityAvail = *req.QuantityAvail
	}
	if req.Status != nil {
		l.Status = *req.Status
	}
	l.UpdatedAt = now
	return nil
}
func (m *memStore) CreateBid(_ context.Context, b models.MarketBid) error {
	m.bids[b.ID] = &b
	return nil
}
func (m *memStore) GetBid(_ context.Context, id uuid.UUID) (*models.MarketBid, error) {
	b, ok := m.bids[id]
	if !ok {
		return nil, errNotFound
	}
	return b, nil
}
func (m *memStore) ListBidsForListing(_ context.Context, listingID uuid.UUID) ([]models.MarketBid, error) {
	var result []models.MarketBid
	for _, b := range m.bids {
		if b.ListingID == listingID {
			result = append(result, *b)
		}
	}
	return result, nil
}
func (m *memStore) UpdateBidStatus(_ context.Context, id uuid.UUID, status models.BidStatus, now time.Time) error {
	b, ok := m.bids[id]
	if !ok {
		return errNotFound
	}
	b.Status = status
	b.UpdatedAt = now
	return nil
}
func (m *memStore) RecordPrice(_ context.Context, r models.PriceRecord) error {
	m.prices = append(m.prices, r)
	return nil
}
func (m *memStore) GetPriceSummary(_ context.Context, cropType, market, country string, from, to time.Time) (models.PriceSummary, error) {
	return models.PriceSummary{CropType: cropType, Market: market, Country: country}, nil
}
func (m *memStore) ListPriceHistory(_ context.Context, cropType, country string, days int) ([]models.PriceRecord, error) {
	return m.prices, nil
}
func (m *memStore) InsertAuditLog(_ context.Context, _ models.MarketAuditLog) error { return nil }

var errNotFound = &notFoundErr{}

type notFoundErr struct{}

func (e *notFoundErr) Error() string { return "not found" }

// ─── Router setup ─────────────────────────────────────────────────────────────

func setupRouter(store handlers.Store) *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &auth.Claims{UserID: uuid.New(), Role: "farmer"}
			ctx := middleware.SetClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	h := handlers.NewMarketHandlerWithStore(store)
	h.RegisterRoutes(api)
	return r
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestCreateListing(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{
		"crop_type":     "maize",
		"quantity_kg":   500.0,
		"price_per_unit": 0.35,
		"currency":      "USD",
		"market":        "Nairobi Central",
		"region":        "Nairobi",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp models.APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestCreateListingMissingFields(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{"crop_type": "maize"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListListings(t *testing.T) {
	store := newMemStore()
	store.listings[uuid.New()] = &models.MarketListing{
		ID: uuid.New(), CropType: "beans", Status: models.ListingActive,
		FarmerID: uuid.New(), QuantityKg: 100, QuantityAvail: 100, PricePerUnit: 1.5,
		Country: "KE", Currency: "USD", PriceUnit: models.UnitKg,
		QualityGrade: "A", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	router := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/listings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetListingNotFound(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/listings/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestPlaceBid(t *testing.T) {
	store := newMemStore()
	listingID := uuid.New()
	farmerID := uuid.New()
	store.listings[listingID] = &models.MarketListing{
		ID: listingID, FarmerID: farmerID, CropType: "sorghum",
		QuantityKg: 200, QuantityAvail: 200, Status: models.ListingActive,
		PricePerUnit: 0.25, Currency: "USD", Country: "ET",
		PriceUnit: models.UnitKg, QualityGrade: "B",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	router := setupRouter(store)
	body := map[string]interface{}{
		"quantity_kg": 50.0,
		"bid_price":   0.28,
		"currency":    "USD",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings/"+listingID.String()+"/bids", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPlaceBidExceedsQuantity(t *testing.T) {
	store := newMemStore()
	listingID := uuid.New()
	store.listings[listingID] = &models.MarketListing{
		ID: listingID, FarmerID: uuid.New(), CropType: "millet",
		QuantityKg: 100, QuantityAvail: 50, Status: models.ListingActive,
		PricePerUnit: 0.30, Currency: "USD", Country: "GH",
		PriceUnit: models.UnitKg, QualityGrade: "B",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	router := setupRouter(store)
	body := map[string]interface{}{"quantity_kg": 200.0, "bid_price": 0.35, "currency": "USD"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings/"+listingID.String()+"/bids", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRecordPrice(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	body := map[string]interface{}{
		"crop_type":   "cassava",
		"market":      "Lagos Market",
		"country":     "NG",
		"price_per_kg": 0.15,
		"currency":    "USD",
		"source":      "reported",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/prices", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.prices) != 1 {
		t.Fatal("expected price to be recorded")
	}
}

func TestPriceSummaryMissingCropType(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/prices/summary", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetBidNotFound(t *testing.T) {
	store := newMemStore()
	router := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bids/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
