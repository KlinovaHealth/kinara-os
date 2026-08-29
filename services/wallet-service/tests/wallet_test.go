package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/wallet-service/handlers"
	"github.com/klinova/kinara-os/wallet-service/models"
)

type memStore struct {
	mu      sync.RWMutex
	balances map[string]models.WalletBalance
	logs    []models.ReconciliationLog
}

func newMemStore() *memStore {
	return &memStore{balances: map[string]models.WalletBalance{}}
}

func key(userID uuid.UUID, currency models.Currency) string {
	return fmt.Sprintf("%s:%s", userID, currency)
}

func (s *memStore) UpsertBalance(_ context.Context, userID uuid.UUID, currency models.Currency, balance float64) error {
	s.mu.Lock(); defer s.mu.Unlock()
	s.balances[key(userID, currency)] = models.WalletBalance{UserID: userID, Currency: currency, Balance: balance, UpdatedAt: time.Now()}
	return nil
}
func (s *memStore) GetBalance(_ context.Context, userID uuid.UUID, currency models.Currency) (float64, time.Time, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	wb, ok := s.balances[key(userID, currency)]
	if !ok { return 0, time.Time{}, errNotFound }
	return wb.Balance, wb.UpdatedAt, nil
}
func (s *memStore) GetAllBalances(_ context.Context, userID uuid.UUID) ([]models.WalletBalance, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.WalletBalance
	for _, wb := range s.balances {
		if wb.UserID == userID { result = append(result, wb) }
	}
	return result, nil
}
func (s *memStore) SaveReconciliationLog(_ context.Context, l models.ReconciliationLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.logs = append(s.logs, l); return nil
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

func TestUpsertAndGetBalance(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	uid := uuid.New()
	body := `{"balance":600000}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/wallets/"+uid.String()+"/balance/XOF", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("upsert: expected 200, got %d", resp.StatusCode) }

	resp2, _ := http.Get(srv.URL + "/api/v1/wallets/" + uid.String() + "/balance/XOF")
	if resp2.StatusCode != 200 { t.Fatalf("get: expected 200, got %d", resp2.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp2.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["balance"].(float64) != 600000 { t.Fatalf("expected 600000, got %v", data["balance"]) }
}

func TestUnsupportedCurrency(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	uid := uuid.New()
	body := `{"balance":100}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/wallets/"+uid.String()+"/balance/INVALID", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetAllBalances(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	uid := uuid.New()
	store.UpsertBalance(context.Background(), uid, models.CurrencyXOF, 300000)
	store.UpsertBalance(context.Background(), uid, models.CurrencyUSD, 500)
	resp, _ := http.Get(srv.URL + "/api/v1/wallets/" + uid.String() + "/balances")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	totalUSD := data["total_usd"].(float64)
	// 300000 XOF / 600 + 500 USD = 500 + 500 = 1000 USD
	if totalUSD != 1000.0 { t.Fatalf("expected total_usd=1000, got %f", totalUSD) }
}

func TestReconcile_Balanced(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	uid := uuid.New()
	store.UpsertBalance(context.Background(), uid, models.CurrencyXOF, 600000)
	store.UpsertBalance(context.Background(), uid, models.CurrencyGHS, 1400)
	resp, _ := http.Post(srv.URL+"/api/v1/wallets/"+uid.String()+"/reconcile", "application/json", nil)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if !data["is_balanced"].(bool) { t.Fatal("expected balanced wallet") }
	// Both balances positive, so discrepancy should be 0
	if data["discrepancy_usd"].(float64) != 0 { t.Fatalf("expected 0 discrepancy, got %v", data["discrepancy_usd"]) }
}

func TestReconciliationReport(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	uid := uuid.New()
	store.UpsertBalance(context.Background(), uid, models.CurrencyKES, 130000)
	resp, _ := http.Get(srv.URL + "/api/v1/wallets/" + uid.String() + "/reconciliation-report")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}
