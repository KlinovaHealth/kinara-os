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
	"github.com/klinova/kinara-os/trade-finance-service/auth"
	"github.com/klinova/kinara-os/trade-finance-service/handlers"
	"github.com/klinova/kinara-os/trade-finance-service/middleware"
	"github.com/klinova/kinara-os/trade-finance-service/models"
)

type memStore struct {
	mu         sync.RWMutex
	lcs        map[uuid.UUID]*models.LetterOfCredit
	financings map[uuid.UUID]*models.FinancingRequest
	audit      []models.TradeFinanceAuditLog
}

func newMemStore() *memStore {
	return &memStore{lcs: map[uuid.UUID]*models.LetterOfCredit{}, financings: map[uuid.UUID]*models.FinancingRequest{}}
}

func (s *memStore) CreateLC(_ context.Context, lc models.LetterOfCredit) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.lcs[lc.ID] = &lc; return nil
}
func (s *memStore) GetLC(_ context.Context, id uuid.UUID) (*models.LetterOfCredit, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	lc, ok := s.lcs[id]; if !ok { return nil, errNotFound }
	cp := *lc; return &cp, nil
}
func (s *memStore) ListLCs(_ context.Context, applicantID *uuid.UUID, status *models.LCStatus) ([]models.LetterOfCredit, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.LetterOfCredit
	for _, lc := range s.lcs {
		if applicantID != nil && lc.ApplicantID != *applicantID { continue }
		if status != nil && lc.Status != *status { continue }
		result = append(result, *lc)
	}
	return result, nil
}
func (s *memStore) UpdateLCStatus(_ context.Context, id uuid.UUID, status models.LCStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	lc, ok := s.lcs[id]; if !ok { return errNotFound }
	lc.Status = status; lc.UpdatedAt = now
	if status == models.LCIssued && lc.IssuedAt == nil { lc.IssuedAt = &now }
	if status == models.LCRealized && lc.RealizedAt == nil { lc.RealizedAt = &now }
	return nil
}
func (s *memStore) CreateFinancing(_ context.Context, f models.FinancingRequest) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.financings[f.ID] = &f; return nil
}
func (s *memStore) GetFinancing(_ context.Context, id uuid.UUID) (*models.FinancingRequest, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	f, ok := s.financings[id]; if !ok { return nil, errNotFound }
	cp := *f; return &cp, nil
}
func (s *memStore) ApproveFinancing(_ context.Context, id uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	f, ok := s.financings[id]; if !ok { return errNotFound }
	f.Status = "approved"; f.UpdatedAt = now
	if f.ApprovedAt == nil { f.ApprovedAt = &now }
	return nil
}
func (s *memStore) DisburseFinancing(_ context.Context, id uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	f, ok := s.financings[id]; if !ok { return errNotFound }
	f.Status = "disbursed"; f.UpdatedAt = now
	if f.DisbursedAt == nil { f.DisbursedAt = &now }
	return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.TradeFinanceAuditLog) error {
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
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := &auth.Claims{UserID: uuid.New(), Role: "trade_finance_officer"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestCreateLC(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateLCRequest{LCType:"documentary",ApplicantName:"Ghana Cocoa Board",BeneficiaryName:"Cargill Amsterdam BV",IssuingBank:"GCB Bank Ghana",Amount:5000000,Currency:"USD",ExpiryDate:"2026-12-31T23:59:59Z",ShipmentPOL:"Port of Tema, Ghana",ShipmentPOD:"Port of Amsterdam",GoodsDescription:"Cocoa Beans Grade A",DocumentsRequired:[]string{"bill_of_lading","commercial_invoice","phytosanitary_certificate"}})
	resp, _ := http.Post(srv.URL+"/api/v1/lc", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["status"] != string(models.LCDraft) { t.Fatal("expected status draft") }
}

func TestCreateLC_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/lc", "application/json", bytes.NewBufferString(`{"applicant_name":"only"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestUpdateLCStatus_Issued(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); expiry := time.Now().AddDate(0, 6, 0)
	store.CreateLC(context.Background(), models.LetterOfCredit{ID:id,LCNumber:"LC-TEST000001",LCType:models.LCStandard,ApplicantID:uuid.New(),ApplicantName:"Lagos Steel Ltd",BeneficiaryName:"China Steel Corp",IssuingBank:"Zenith Bank Nigeria",Amount:2000000,Currency:"USD",ExpiryDate:expiry,ShipmentPOL:"Apapa Lagos",ShipmentPOD:"Port of Tianjin",GoodsDescription:"Hot Rolled Steel Coils",DocumentsRequired:[]string{"bol","invoice"},Status:models.LCDraft,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body := `{"status":"issued"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/lc/"+id.String()+"/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	lc, _ := store.GetLC(context.Background(), id)
	if lc.Status != models.LCIssued { t.Fatal("expected status issued") }
	if lc.IssuedAt == nil { t.Fatal("issued_at should be set") }
}

func TestCreateFinancing(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateFinancingRequest{RequestedAmount:500000,Currency:"USD",PaymentTerms:string(models.PayNet60),InterestRatePct:4.5})
	resp, _ := http.Post(srv.URL+"/api/v1/financing", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["interest_amount"].(float64) != 22500.0 { t.Fatalf("expected interest 22500, got %f", data["interest_amount"].(float64)) }
}

func TestApproveAndDisburseFinancing(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateFinancing(context.Background(), models.FinancingRequest{ID:id,RefNo:"TF-TEST000001",ApplicantID:uuid.New(),RequestedAmount:250000,Currency:"USD",PaymentTerms:models.PayNet30,InterestRatePct:3.5,InterestAmount:8750,TotalRepayable:258750,Status:"pending",CreatedAt:time.Now(),UpdatedAt:time.Now()})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/financing/"+id.String()+"/approve", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200 on approve, got %d", resp.StatusCode) }
	f, _ := store.GetFinancing(context.Background(), id)
	if f.Status != "approved" { t.Fatal("expected status approved") }
	if f.ApprovedAt == nil { t.Fatal("approved_at should be set") }

	req2, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/financing/"+id.String()+"/disburse", nil)
	resp2, _ := http.DefaultClient.Do(req2)
	if resp2.StatusCode != 200 { t.Fatalf("expected 200 on disburse, got %d", resp2.StatusCode) }
	f2, _ := store.GetFinancing(context.Background(), id)
	if f2.Status != "disbursed" { t.Fatal("expected status disbursed") }
	if f2.DisbursedAt == nil { t.Fatal("disbursed_at should be set") }
}

func TestStandbyLC(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); expiry := time.Now().AddDate(1, 0, 0)
	store.CreateLC(context.Background(), models.LetterOfCredit{ID:id,LCNumber:"LC-TEST000002",LCType:models.LCStandby,ApplicantID:uuid.New(),ApplicantName:"Nairobi Infrastructure Ltd",BeneficiaryName:"KfW Development Bank",IssuingBank:"Equity Bank Kenya",Amount:10000000,Currency:"USD",ExpiryDate:expiry,ShipmentPOL:"N/A",ShipmentPOD:"N/A",GoodsDescription:"Performance guarantee - road construction",DocumentsRequired:[]string{"demand_certificate"},Status:models.LCDraft,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/lc/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["lc_type"] != string(models.LCStandby) { t.Fatal("expected lc_type standby") }
}
