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
	"github.com/klinova/kinara-os/farmer-finance-service/handlers"
	"github.com/klinova/kinara-os/farmer-finance-service/models"
)

type memStore struct {
	mu       sync.RWMutex
	income   []models.IncomeRecord
	loans    map[uuid.UUID]*models.Loan
	savings  map[uuid.UUID]*models.SavingsAccount
}

func newMemStore() *memStore {
	return &memStore{
		loans:   map[uuid.UUID]*models.Loan{},
		savings: map[uuid.UUID]*models.SavingsAccount{},
	}
}

func (s *memStore) RecordIncome(_ context.Context, r models.IncomeRecord) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.income = append(s.income, r); return nil
}
func (s *memStore) ListIncome(_ context.Context, farmerID uuid.UUID, limit int) ([]models.IncomeRecord, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.IncomeRecord
	for _, r := range s.income {
		if r.FarmerID == farmerID {
			result = append(result, r)
		}
	}
	return result, nil
}
func (s *memStore) SumIncome(_ context.Context, farmerID uuid.UUID, since time.Time) (float64, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var total float64
	for _, r := range s.income {
		if r.FarmerID == farmerID && r.RecordedAt.After(since) {
			total += r.Amount
		}
	}
	return total, nil
}
func (s *memStore) CreateLoan(_ context.Context, l models.Loan) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.loans[l.ID] = &l; return nil
}
func (s *memStore) GetLoan(_ context.Context, id uuid.UUID) (*models.Loan, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	l, ok := s.loans[id]
	if !ok { return nil, errNotFound }
	cp := *l; return &cp, nil
}
func (s *memStore) ListLoans(_ context.Context, farmerID uuid.UUID) ([]models.Loan, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Loan
	for _, l := range s.loans {
		if l.FarmerID == farmerID { result = append(result, *l) }
	}
	return result, nil
}
func (s *memStore) UpdateLoanStatus(_ context.Context, id uuid.UUID, status models.LoanStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if l, ok := s.loans[id]; ok { l.Status = status }
	return nil
}
func (s *memStore) GetOrCreateSavings(_ context.Context, farmerID uuid.UUID, currency models.Currency) (*models.SavingsAccount, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	if acc, ok := s.savings[farmerID]; ok { cp := *acc; return &cp, nil }
	acc := &models.SavingsAccount{ID: uuid.New(), FarmerID: farmerID, Currency: currency, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	s.savings[farmerID] = acc
	cp := *acc; return &cp, nil
}
func (s *memStore) AddSavings(_ context.Context, farmerID uuid.UUID, amount float64) (*models.SavingsAccount, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	acc, ok := s.savings[farmerID]
	if !ok { return nil, errNotFound }
	acc.Balance += amount
	acc.TotalSaved += amount
	acc.UpdatedAt = time.Now()
	cp := *acc; return &cp, nil
}
func (s *memStore) GetSavings(_ context.Context, farmerID uuid.UUID) (*models.SavingsAccount, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	acc, ok := s.savings[farmerID]
	if !ok { return nil, errNotFound }
	cp := *acc; return &cp, nil
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

func TestRecordIncome(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	fid := uuid.New()
	body := `{"source":"crop_sale","amount":125000,"currency":"XOF","description":"Sold 500kg maize at 250 XOF/kg"}`
	resp, _ := http.Post(srv.URL+"/api/v1/finance/"+fid.String()+"/income", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestRecordIncome_MissingAmount(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	fid := uuid.New()
	body := `{"source":"crop_sale"}`
	resp, _ := http.Post(srv.URL+"/api/v1/finance/"+fid.String()+"/income", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetIncome(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	fid := uuid.New()
	store.RecordIncome(context.Background(), models.IncomeRecord{
		ID: uuid.New(), FarmerID: fid, Source: "crop_sale", Amount: 80000, Currency: models.CurrencyXOF, RecordedAt: time.Now(), CreatedAt: time.Now(),
	})
	resp, _ := http.Get(srv.URL + "/api/v1/finance/" + fid.String() + "/income")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestLoanEligibility_Eligible(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	fid := uuid.New()
	// Add 3 months of income (avg 50,000 XOF/month)
	for i := 0; i < 3; i++ {
		store.RecordIncome(context.Background(), models.IncomeRecord{
			ID: uuid.New(), FarmerID: fid, Source: "crop_sale", Amount: 50000, Currency: models.CurrencyXOF,
			RecordedAt: time.Now().AddDate(0, -i, 0), CreatedAt: time.Now(),
		})
	}
	resp, _ := http.Get(srv.URL + "/api/v1/finance/" + fid.String() + "/loan-eligibility")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if !data["is_eligible"].(bool) { t.Fatal("expected eligible farmer (avg 50k XOF/month)") }
}

func TestLoanEligibility_Ineligible(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	fid := uuid.New()
	resp, _ := http.Get(srv.URL + "/api/v1/finance/" + fid.String() + "/loan-eligibility")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["is_eligible"].(bool) { t.Fatal("expected ineligible farmer (no income history)") }
}

func TestRequestLoan(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	fid := uuid.New()
	body := `{"principal_amount":150000,"currency":"XOF","due_date":"2027-02-28T00:00:00Z"}`
	resp, _ := http.Post(srv.URL+"/api/v1/finance/"+fid.String()+"/loan", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["loan_ref"] == nil { t.Fatal("expected loan_ref") }
}

func TestSavingsFlow(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	fid := uuid.New()
	// Create savings account
	store.GetOrCreateSavings(context.Background(), fid, models.CurrencyXOF)
	// Add savings
	body := `{"amount":25000,"currency":"XOF"}`
	resp, _ := http.Post(srv.URL+"/api/v1/finance/"+fid.String()+"/save", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 200 { t.Fatalf("add savings: expected 200, got %d", resp.StatusCode) }
	// Get savings
	resp2, _ := http.Get(srv.URL + "/api/v1/finance/" + fid.String() + "/savings")
	if resp2.StatusCode != 200 { t.Fatalf("get savings: expected 200, got %d", resp2.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp2.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["balance"].(float64) != 25000 { t.Fatalf("expected balance 25000, got %v", data["balance"]) }
}
