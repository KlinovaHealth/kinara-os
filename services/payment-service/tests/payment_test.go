package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/payment-service/auth"
	"github.com/klinova/kinara-os/payment-service/handlers"
	"github.com/klinova/kinara-os/payment-service/middleware"
	"github.com/klinova/kinara-os/payment-service/models"
)

type memStore struct {
	mu          sync.RWMutex
	wallets     map[uuid.UUID]*models.Wallet
	txns        []models.Transaction
	conversions []models.CurrencyConversion
	settlements map[uuid.UUID]*models.Settlement
	audit       []models.PaymentAuditLog
}

func newMemStore() *memStore {
	return &memStore{
		wallets:     map[uuid.UUID]*models.Wallet{},
		settlements: map[uuid.UUID]*models.Settlement{},
	}
}

func (s *memStore) CreateWallet(_ context.Context, w models.Wallet) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.wallets[w.ID] = &w; return nil
}
func (s *memStore) GetWallet(_ context.Context, id uuid.UUID) (*models.Wallet, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	w, ok := s.wallets[id]; if !ok { return nil, errNotFound }
	cp := *w; return &cp, nil
}
func (s *memStore) GetWalletByOwner(_ context.Context, ownerID uuid.UUID, currency string) (*models.Wallet, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	for _, w := range s.wallets { if w.OwnerID == ownerID && w.Currency == currency { cp := *w; return &cp, nil } }
	return nil, errNotFound
}
func (s *memStore) CreditWallet(_ context.Context, walletID uuid.UUID, amount float64, txn models.Transaction) error {
	s.mu.Lock(); defer s.mu.Unlock()
	w, ok := s.wallets[walletID]; if !ok { return errNotFound }
	w.Balance += amount; s.txns = append(s.txns, txn); return nil
}
func (s *memStore) DebitWallet(_ context.Context, walletID uuid.UUID, amount float64, txn models.Transaction) error {
	s.mu.Lock(); defer s.mu.Unlock()
	w, ok := s.wallets[walletID]; if !ok { return errNotFound }
	if w.Balance < amount { return errInsufficientFunds }
	w.Balance -= amount; s.txns = append(s.txns, txn); return nil
}
func (s *memStore) ListTransactions(_ context.Context, walletID uuid.UUID) ([]models.Transaction, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Transaction
	for _, t := range s.txns { if t.WalletID == walletID { result = append(result, t) } }
	return result, nil
}
func (s *memStore) CreateConversion(_ context.Context, c models.CurrencyConversion) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.conversions = append(s.conversions, c); return nil
}
func (s *memStore) CreateSettlement(_ context.Context, st models.Settlement) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.settlements[st.ID] = &st; return nil
}
func (s *memStore) ConfirmSettlement(_ context.Context, id uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	st, ok := s.settlements[id]; if !ok { return errNotFound }
	st.Status = models.TxnCompleted
	if st.SettledAt == nil { st.SettledAt = &now }
	st.UpdatedAt = now; return nil
}
func (s *memStore) GetSettlement(_ context.Context, id uuid.UUID) (*models.Settlement, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	st, ok := s.settlements[id]; if !ok { return nil, errNotFound }
	cp := *st; return &cp, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.PaymentAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
var errInsufficientFunds = &insufficientError{}
type notFoundError struct{}
type insufficientError struct{}
func (e *notFoundError) Error() string { return "not found" }
func (e *insufficientError) Error() string { return "insufficient funds" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewHandlerWithStore(store)
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := &auth.Claims{UserID: uuid.New().String(), Role: "finance_admin"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestCreateWallet(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	ownerID := uuid.New()
	body, _ := json.Marshal(models.CreateWalletRequest{OwnerID: ownerID.String(), OwnerType: "farmer", Currency: "KES"})
	resp, _ := http.Post(srv.URL+"/api/v1/wallets", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreateWallet_UnsupportedCurrency(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateWalletRequest{OwnerID: uuid.New().String(), Currency: "JPY"})
	resp, _ := http.Post(srv.URL+"/api/v1/wallets", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestCreditAndDebit(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	walletID := uuid.New(); now := time.Now().UTC()
	store.CreateWallet(context.Background(), models.Wallet{ID: walletID, OwnerID: uuid.New(), OwnerType: "driver", Currency: "USD", Balance: 0, Status: models.WalletActive, CreatedAt: now, UpdatedAt: now})

	// Credit 5000
	body, _ := json.Marshal(models.CreditWalletRequest{Amount: 5000, Description: "Initial funding", ReferenceType: "shipment", ReferenceID: "SHP-001"})
	resp, _ := http.Post(srv.URL+"/api/v1/wallets/"+walletID.String()+"/credit", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("credit: expected 201, got %d", resp.StatusCode) }
	w, _ := store.GetWallet(context.Background(), walletID)
	if w.Balance != 5000 { t.Fatalf("expected balance 5000, got %.4f", w.Balance) }

	// Debit 1500
	body2, _ := json.Marshal(models.DebitWalletRequest{Amount: 1500, Description: "Service fee"})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/wallets/"+walletID.String()+"/debit", bytes.NewBuffer(body2))
	req.Header.Set("Content-Type", "application/json")
	resp2, _ := http.DefaultClient.Do(req)
	if resp2.StatusCode != 201 { t.Fatalf("debit: expected 201, got %d", resp2.StatusCode) }
	w2, _ := store.GetWallet(context.Background(), walletID)
	if w2.Balance != 3500 { t.Fatalf("expected balance 3500, got %.4f", w2.Balance) }
}

func TestDebit_InsufficientFunds(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	walletID := uuid.New(); now := time.Now().UTC()
	store.CreateWallet(context.Background(), models.Wallet{ID: walletID, OwnerID: uuid.New(), OwnerType: "operator", Currency: "GHS", Balance: 100, Status: models.WalletActive, CreatedAt: now, UpdatedAt: now})
	body, _ := json.Marshal(models.DebitWalletRequest{Amount: 9999, Description: "Overdraft attempt"})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/wallets/"+walletID.String()+"/debit", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 422 { t.Fatalf("expected 422, got %d", resp.StatusCode) }
}

func TestCurrencyConversion_USD_to_KES(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	now := time.Now().UTC()
	fromID := uuid.New(); toID := uuid.New()
	store.CreateWallet(context.Background(), models.Wallet{ID: fromID, OwnerID: uuid.New(), OwnerType: "org", Currency: "USD", Balance: 1000, Status: models.WalletActive, CreatedAt: now, UpdatedAt: now})
	store.CreateWallet(context.Background(), models.Wallet{ID: toID, OwnerID: uuid.New(), OwnerType: "org", Currency: "KES", Balance: 0, Status: models.WalletActive, CreatedAt: now, UpdatedAt: now})

	body, _ := json.Marshal(models.ConvertCurrencyRequest{FromWalletID: fromID.String(), ToWalletID: toID.String(), FromAmount: 100})
	resp, _ := http.Post(srv.URL+"/api/v1/convert", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	// 100 USD * 130 KES/USD = 13000 KES gross, 0.5% fee = 65, net = 12935
	toAmount := data["to_amount"].(float64)
	if toAmount < 12900 || toAmount > 13000 { t.Fatalf("expected ~12935 KES, got %.4f", toAmount) }
	// Check from wallet debited
	w, _ := store.GetWallet(context.Background(), fromID)
	if w.Balance != 900 { t.Fatalf("from wallet expected 900, got %.4f", w.Balance) }
}

func TestSettlement_CreateAndConfirm(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	walletID := uuid.New(); now := time.Now().UTC()
	store.CreateWallet(context.Background(), models.Wallet{ID: walletID, OwnerID: uuid.New(), OwnerType: "farmer", Currency: "XOF", Balance: 50000, Status: models.WalletActive, CreatedAt: now, UpdatedAt: now})

	type settlementReq struct {
		WalletID      string  `json:"wallet_id"`
		Amount        float64 `json:"amount"`
		Provider      string  `json:"provider"`
		MobileMoneyNo string  `json:"mobile_money_no"`
	}
	body, _ := json.Marshal(settlementReq{WalletID: walletID.String(), Amount: 25000, Provider: "Orange Money", MobileMoneyNo: "+22170123456"})
	resp, _ := http.Post(srv.URL+"/api/v1/settlements", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("create settlement: expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["status"].(string) != "pending" { t.Fatal("settlement should start as pending") }
	if !strings.HasPrefix(data["settlement_ref"].(string), "ST-") { t.Fatal("settlement_ref must start with ST-") }

	stID, _ := uuid.Parse(data["id"].(string))

	// Confirm
	req2, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/settlements/"+stID.String()+"/confirm", nil)
	resp2, _ := http.DefaultClient.Do(req2)
	if resp2.StatusCode != 200 { t.Fatalf("confirm settlement: expected 200, got %d", resp2.StatusCode) }
	st, _ := store.GetSettlement(context.Background(), stID)
	if st.Status != models.TxnCompleted { t.Fatal("expected status completed") }
	if st.SettledAt == nil { t.Fatal("settled_at must be set") }
}

func TestListTransactions(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	walletID := uuid.New(); now := time.Now().UTC()
	store.CreateWallet(context.Background(), models.Wallet{ID: walletID, OwnerID: uuid.New(), OwnerType: "driver", Currency: "NGN", Balance: 200000, Status: models.WalletActive, CreatedAt: now, UpdatedAt: now})
	// Credit twice
	store.CreditWallet(context.Background(), walletID, 10000, models.Transaction{ID: uuid.New(), TxnRef: "TX-A001", WalletID: walletID, TxnType: models.TxnCredit, Amount: 10000, Currency: "NGN", BalanceBefore: 200000, BalanceAfter: 210000, Status: models.TxnCompleted, CreatedAt: now})
	store.CreditWallet(context.Background(), walletID, 5000, models.Transaction{ID: uuid.New(), TxnRef: "TX-A002", WalletID: walletID, TxnType: models.TxnCredit, Amount: 5000, Currency: "NGN", BalanceBefore: 210000, BalanceAfter: 215000, Status: models.TxnCompleted, CreatedAt: now})
	resp, _ := http.Get(srv.URL + "/api/v1/wallets/" + walletID.String() + "/transactions")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 2 { t.Fatalf("expected 2 transactions, got %d", len(items)) }
}
