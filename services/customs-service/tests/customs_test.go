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
	"github.com/klinova/kinara-os/customs-service/auth"
	"github.com/klinova/kinara-os/customs-service/handlers"
	"github.com/klinova/kinara-os/customs-service/middleware"
	"github.com/klinova/kinara-os/customs-service/models"
)

type memStore struct {
	mu         sync.RWMutex
	tariffs    map[string]*models.TariffCode
	clearances map[uuid.UUID]*models.ClearanceRequest
	audit      []models.CustomsAuditLog
}

func newMemStore() *memStore {
	return &memStore{tariffs: map[string]*models.TariffCode{}, clearances: map[uuid.UUID]*models.ClearanceRequest{}}
}

func (s *memStore) CreateTariff(_ context.Context, t models.TariffCode) error {
	s.mu.Lock(); defer s.mu.Unlock(); key := t.HSCode + "|" + t.Country; s.tariffs[key] = &t; return nil
}
func (s *memStore) LookupTariff(_ context.Context, hsCode, country string) (*models.TariffCode, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	key := hsCode + "|" + country
	t, ok := s.tariffs[key]; if !ok { return nil, errNotFound }
	cp := *t; return &cp, nil
}
func (s *memStore) ListTariffs(_ context.Context, country *string) ([]models.TariffCode, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.TariffCode
	for _, t := range s.tariffs {
		if country != nil && t.Country != *country { continue }
		result = append(result, *t)
	}
	return result, nil
}
func (s *memStore) CreateClearance(_ context.Context, c models.ClearanceRequest) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.clearances[c.ID] = &c; return nil
}
func (s *memStore) GetClearance(_ context.Context, id uuid.UUID) (*models.ClearanceRequest, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	c, ok := s.clearances[id]; if !ok { return nil, errNotFound }
	cp := *c; return &cp, nil
}
func (s *memStore) ListClearances(_ context.Context, portID *uuid.UUID, status *models.ClearanceStatus) ([]models.ClearanceRequest, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.ClearanceRequest
	for _, c := range s.clearances {
		if portID != nil && c.PortID != *portID { continue }
		if status != nil && c.Status != *status { continue }
		result = append(result, *c)
	}
	return result, nil
}
func (s *memStore) UpdateClearanceStatus(_ context.Context, id uuid.UUID, status models.ClearanceStatus, reviewerID, rejectionReason string, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	c, ok := s.clearances[id]; if !ok { return errNotFound }
	c.Status = status; c.ReviewedBy = reviewerID; c.UpdatedAt = now
	if rejectionReason != "" { c.RejectionReason = rejectionReason }
	if c.ReviewedAt == nil { c.ReviewedAt = &now }
	return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.CustomsAuditLog) error {
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
			claims := &auth.Claims{UserID: uuid.New(), Role: "customs_officer"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestCreateTariff(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateTariffRequest{HSCode:"0901",Description:"Coffee, whether or not roasted",Category:string(models.TariffAg),DutyRate:5.0,VATRate:16.0,Country:"KE",IsRestricted:false})
	resp, _ := http.Post(srv.URL+"/api/v1/tariffs", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestCreateTariff_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/tariffs", "application/json", bytes.NewBufferString(`{"description":"no code"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestLookupTariff(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	store.CreateTariff(context.Background(), models.TariffCode{ID:uuid.New(),HSCode:"3926",Description:"Plastic goods",Category:models.TariffManuf,DutyRate:10.0,VATRate:18.0,Country:"GH",IsRestricted:false,CreatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/tariffs/lookup?hs_code=3926&country=GH")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["duty_rate_pct"].(float64) != 10.0 { t.Fatal("expected duty rate 10.0") }
}

func TestCreateClearance(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateClearanceRequest{ImporterName:"Lagos Traders Ltd",ImporterID:"RC1234567",ManifestID:uuid.New().String(),VesselID:uuid.New().String(),PortID:uuid.New().String(),HSCode:"8471",GoodsDescription:"Laptop computers",DeclaredValue:250000,Currency:"USD",WeightKg:5000})
	resp, _ := http.Post(srv.URL+"/api/v1/clearances", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["status"] != string(models.ClearancePending) { t.Fatal("expected status pending") }
}

func TestUpdateClearanceStatus_Approved(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateClearance(context.Background(), models.ClearanceRequest{ID:id,ReferenceNo:"CR-TEST000001",ImporterName:"Accra Foods",ImporterID:"GH-9999",ManifestID:uuid.New(),VesselID:uuid.New(),PortID:uuid.New(),HSCode:"1905",GoodsDescription:"Bread and bakery",DeclaredValue:50000,Currency:"USD",WeightKg:8000,Status:models.ClearancePending,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body := `{"status":"approved"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/clearances/"+id.String()+"/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	c, _ := store.GetClearance(context.Background(), id)
	if c.Status != models.ClearanceApproved { t.Fatal("expected status approved") }
	if c.ReviewedAt == nil { t.Fatal("reviewed_at should be set") }
}

func TestUpdateClearanceStatus_Rejected(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateClearance(context.Background(), models.ClearanceRequest{ID:id,ReferenceNo:"CR-TEST000002",ImporterName:"Nairobi Tech",ImporterID:"KE-1111",ManifestID:uuid.New(),VesselID:uuid.New(),PortID:uuid.New(),HSCode:"9013",GoodsDescription:"Laser equipment",DeclaredValue:500000,Currency:"USD",WeightKg:2000,Status:models.ClearancePending,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.UpdateClearanceStatusRequest{Status:"rejected",RejectionReason:"Dual-use technology requires additional permits"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/clearances/"+id.String()+"/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	c, _ := store.GetClearance(context.Background(), id)
	if c.Status != models.ClearanceRejected { t.Fatal("expected status rejected") }
	if c.RejectionReason == "" { t.Fatal("rejection reason should be set") }
}

func TestDutyCalculation(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	store.CreateTariff(context.Background(), models.TariffCode{ID:uuid.New(),HSCode:"2709",Description:"Crude petroleum oils",Category:models.TariffEnergy,DutyRate:5.0,VATRate:7.5,Country:"NG",IsRestricted:false,CreatedAt:time.Now()})
	body, _ := json.Marshal(models.CreateClearanceRequest{ImporterName:"Warri Refinery",ImporterID:"NG-5432",HSCode:"2709",GoodsDescription:"Crude oil 10000 barrels",DeclaredValue:1000000,Currency:"USD",WeightKg:1500000})
	resp, _ := http.Post(srv.URL+"/api/v1/clearances", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["duty_amount"].(float64) != 50000.0 { t.Fatalf("expected duty 50000, got %f", data["duty_amount"].(float64)) }
}
