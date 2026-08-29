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
	"github.com/klinova/kinara-os/compliance-service/auth"
	"github.com/klinova/kinara-os/compliance-service/db"
	"github.com/klinova/kinara-os/compliance-service/handlers"
	"github.com/klinova/kinara-os/compliance-service/middleware"
	"github.com/klinova/kinara-os/compliance-service/models"
)

type memStore struct {
	mu       sync.RWMutex
	permits  map[uuid.UUID]*models.TransitPermit
	crossings []models.BorderCrossing
	checks   []models.WeightCheck
	audit    []models.ComplianceAuditLog
}

func newMemStore() *memStore { return &memStore{permits: map[uuid.UUID]*models.TransitPermit{}} }

func (s *memStore) CreatePermit(_ context.Context, p models.TransitPermit) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.permits[p.ID] = &p; return nil
}
func (s *memStore) GetPermit(_ context.Context, id uuid.UUID) (*models.TransitPermit, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	p, ok := s.permits[id]; if !ok { return nil, errNotFound }
	cp := *p; return &cp, nil
}
func (s *memStore) ListPermits(_ context.Context, p db.ListPermitsParams) ([]models.TransitPermit, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.TransitPermit
	for _, permit := range s.permits {
		if p.Status != nil && permit.Status != *p.Status { continue }
		result = append(result, *permit)
	}
	return result, nil
}
func (s *memStore) UpdatePermitStatus(_ context.Context, id uuid.UUID, status models.PermitStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	p, ok := s.permits[id]; if !ok { return errNotFound }
	p.Status = status; p.UpdatedAt = now; return nil
}
func (s *memStore) CreateBorderCrossing(_ context.Context, b models.BorderCrossing) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.crossings = append(s.crossings, b); return nil
}
func (s *memStore) ListBorderCrossings(_ context.Context, vehicleID uuid.UUID) ([]models.BorderCrossing, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.BorderCrossing
	for _, b := range s.crossings { if b.VehicleID == vehicleID { result = append(result, b) } }
	return result, nil
}
func (s *memStore) CreateWeightCheck(_ context.Context, w models.WeightCheck) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.checks = append(s.checks, w); return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.ComplianceAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewComplianceHandlerWithStore(store)
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

func TestCreatePermit(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	vid := uuid.New()
	body, _ := json.Marshal(models.CreatePermitRequest{VehicleID:vid.String(),PermitType:models.PermitTransit,IssuedBy:"Kenya Transport Authority",Country:"KE",MaxWeightKg:30000,ValidFrom:"2026-09-01T00:00:00Z",ValidUntil:"2026-12-31T23:59:59Z"})
	resp, _ := http.Post(srv.URL+"/api/v1/permits", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreatePermit_MissingVehicle(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"issued_by":"Authority","country":"GH","valid_from":"2026-09-01T00:00:00Z","valid_until":"2026-12-31T23:59:59Z"}`
	resp, _ := http.Post(srv.URL+"/api/v1/permits", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetPermit(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); vid := uuid.New()
	now := time.Now().UTC()
	store.CreatePermit(context.Background(), models.TransitPermit{ID:id,PermitNo:"PM-test001",VehicleID:vid,PermitType:models.PermitTransit,Status:models.PermitActive,IssuedBy:"NTSA",Country:"KE",MaxWeightKg:25000,ValidFrom:now,ValidUntil:now.Add(90*24*time.Hour),CreatedAt:now,UpdatedAt:now})
	resp, _ := http.Get(srv.URL + "/api/v1/permits/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestUpdatePermitStatus(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); vid := uuid.New()
	now := time.Now().UTC()
	store.CreatePermit(context.Background(), models.TransitPermit{ID:id,PermitNo:"PM-test002",VehicleID:vid,PermitType:models.PermitBorderCross,Status:models.PermitActive,IssuedBy:"Ghana DVLA",Country:"GH",ValidFrom:now,ValidUntil:now.Add(60*24*time.Hour),CreatedAt:now,UpdatedAt:now})
	body := `{"status":"revoked"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/permits/"+id.String()+"/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	p, _ := store.GetPermit(context.Background(), id)
	if p.Status != models.PermitRevoked { t.Fatal("expected status revoked") }
}

func TestCreateBorderCrossing(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	vid := uuid.New(); did := uuid.New()
	body, _ := json.Marshal(models.CreateBorderCrossingRequest{VehicleID:vid.String(),DriverID:did.String(),FromCountry:"KE",ToCountry:"TZ",BorderPost:"Namanga",CargoDesc:"Maize 10 tons",GrossWeightKg:15000,ExitPermitNo:"KE-EXIT-001",EntryPermitNo:"TZ-ENTRY-001"})
	resp, _ := http.Post(srv.URL+"/api/v1/border-crossings", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestCreateWeightCheck_Compliant(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	vid := uuid.New()
	body, _ := json.Marshal(models.CreateWeightCheckRequest{VehicleID:vid.String(),Country:"NG",CheckStation:"Sagamu Toll Gate",GrossWeightKg:20000,LegalLimitKg:30000})
	resp, _ := http.Post(srv.URL+"/api/v1/weight-checks", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["is_compliant"] != true { t.Fatal("expected compliant") }
}

func TestCreateWeightCheck_NonCompliant(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	vid := uuid.New()
	body, _ := json.Marshal(models.CreateWeightCheckRequest{VehicleID:vid.String(),Country:"SN",CheckStation:"Mbour Weigh Station",GrossWeightKg:35000,LegalLimitKg:30000,FineAmount:500,Currency:"XOF"})
	resp, _ := http.Post(srv.URL+"/api/v1/weight-checks", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["is_compliant"] != false { t.Fatal("expected non-compliant") }
}
