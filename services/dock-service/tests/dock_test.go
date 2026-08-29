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
	"github.com/klinova/kinara-os/dock-service/auth"
	"github.com/klinova/kinara-os/dock-service/db"
	"github.com/klinova/kinara-os/dock-service/handlers"
	"github.com/klinova/kinara-os/dock-service/middleware"
	"github.com/klinova/kinara-os/dock-service/models"
)

type memStore struct {
	mu         sync.RWMutex
	operations map[uuid.UUID]*models.DockOperation
	equipment  map[uuid.UUID]*models.Equipment
	safety     []models.SafetyEvent
	audit      []models.DockAuditLog
}

func newMemStore() *memStore {
	return &memStore{operations: map[uuid.UUID]*models.DockOperation{}, equipment: map[uuid.UUID]*models.Equipment{}}
}

func (s *memStore) CreateOperation(_ context.Context, op models.DockOperation) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.operations[op.ID] = &op; return nil
}
func (s *memStore) GetOperation(_ context.Context, id uuid.UUID) (*models.DockOperation, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	op, ok := s.operations[id]; if !ok { return nil, errNotFound }
	cp := *op; return &cp, nil
}
func (s *memStore) ListOperations(_ context.Context, p db.ListOperationsParams) ([]models.DockOperation, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.DockOperation
	for _, op := range s.operations {
		if p.PortID != nil && op.PortID != *p.PortID { continue }
		if p.VesselID != nil && op.VesselID != *p.VesselID { continue }
		result = append(result, *op)
	}
	return result, nil
}
func (s *memStore) StartOperation(_ context.Context, id uuid.UUID, startedAt time.Time, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	op, ok := s.operations[id]; if !ok { return errNotFound }
	if op.StartedAt == nil { op.StartedAt = &startedAt }
	op.UpdatedAt = now; return nil
}
func (s *memStore) CompleteOperation(_ context.Context, id uuid.UUID, completedAt time.Time, duration float64, safetyIncident bool, details string, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	op, ok := s.operations[id]; if !ok { return errNotFound }
	if op.CompletedAt == nil { op.CompletedAt = &completedAt }
	op.ActualDuration = duration; op.SafetyIncident = safetyIncident; op.IncidentDetails = details; op.UpdatedAt = now; return nil
}
func (s *memStore) CreateEquipment(_ context.Context, e models.Equipment) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.equipment[e.ID] = &e; return nil
}
func (s *memStore) GetEquipment(_ context.Context, id uuid.UUID) (*models.Equipment, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	e, ok := s.equipment[id]; if !ok { return nil, errNotFound }
	cp := *e; return &cp, nil
}
func (s *memStore) ListEquipment(_ context.Context, portID uuid.UUID) ([]models.Equipment, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Equipment
	for _, e := range s.equipment { if e.PortID == portID { result = append(result, *e) } }
	return result, nil
}
func (s *memStore) UpdateEquipmentStatus(_ context.Context, id uuid.UUID, status models.EquipmentStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	e, ok := s.equipment[id]; if !ok { return errNotFound }
	e.Status = status; e.UpdatedAt = now; return nil
}
func (s *memStore) ReportSafetyEvent(_ context.Context, e models.SafetyEvent) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.safety = append(s.safety, e); return nil
}
func (s *memStore) ListSafetyEvents(_ context.Context, portID uuid.UUID) ([]models.SafetyEvent, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.SafetyEvent
	for _, e := range s.safety { if e.PortID == portID { result = append(result, e) } }
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.DockAuditLog) error {
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
			claims := &auth.Claims{UserID: uuid.New(), Role: "dock_supervisor"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestCreateOperation(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateOperationRequest{PortID:uuid.New().String(),BerthID:uuid.New().String(),VesselID:uuid.New().String(),OperationType:string(models.OpUnloading),CargoType:"containers",TonnageT:35000,UnitCount:1200,StevedoreTeam:"Alpha Team",PlannedDuration:24,BillingAmount:45000,Currency:"USD"})
	resp, _ := http.Post(srv.URL+"/api/v1/operations", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreateOperation_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"vessel_id":"` + uuid.New().String() + `"}`
	resp, _ := http.Post(srv.URL+"/api/v1/operations", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestStartOperation(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	portID := uuid.New(); vesselID := uuid.New(); opID := uuid.New()
	store.CreateOperation(context.Background(), models.DockOperation{ID:opID,PortID:portID,BerthID:uuid.New(),VesselID:vesselID,OperationType:models.OpLoading,CargoType:"bulk grain",TonnageT:20000,UnitCount:0,PlannedDuration:18,BillingAmount:30000,Currency:"USD",SafetyIncident:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/operations/"+opID.String()+"/start", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	op, _ := store.GetOperation(context.Background(), opID)
	if op.StartedAt == nil { t.Fatal("started_at should be set") }
}

func TestCompleteOperation(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	portID := uuid.New(); opID := uuid.New()
	now := time.Now()
	store.CreateOperation(context.Background(), models.DockOperation{ID:opID,PortID:portID,BerthID:uuid.New(),VesselID:uuid.New(),OperationType:models.OpUnloading,CargoType:"vehicles",TonnageT:8000,UnitCount:400,StartedAt:&now,PlannedDuration:12,BillingAmount:20000,Currency:"USD",SafetyIncident:false,CreatedAt:now,UpdatedAt:now})
	body, _ := json.Marshal(models.CompleteOperationRequest{ActualDuration:11.5,SafetyIncident:false})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/operations/"+opID.String()+"/complete", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	op, _ := store.GetOperation(context.Background(), opID)
	if op.CompletedAt == nil { t.Fatal("completed_at should be set") }
	if op.ActualDuration != 11.5 { t.Fatalf("expected 11.5, got %f", op.ActualDuration) }
}

func TestReportSafetyEvent(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	portID := uuid.New(); opID := uuid.New()
	store.CreateOperation(context.Background(), models.DockOperation{ID:opID,PortID:portID,BerthID:uuid.New(),VesselID:uuid.New(),OperationType:models.OpLoading,CargoType:"chemicals",TonnageT:5000,UnitCount:0,PlannedDuration:8,BillingAmount:15000,Currency:"USD",SafetyIncident:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.ReportSafetyEventRequest{EventType:"spill",Severity:"moderate",Description:"Chemical spill during loading — contained immediately",Injured:0})
	resp, _ := http.Post(srv.URL+"/api/v1/operations/"+opID.String()+"/safety", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestCreateEquipment(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	portID := uuid.New()
	body, _ := json.Marshal(models.CreateEquipmentRequest{EquipmentType:string(models.EquipCrane),Model:"Liebherr LHM 550",CapacityT:124})
	resp, _ := http.Post(srv.URL+"/api/v1/ports/"+portID.String()+"/equipment", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["status"] != string(models.EquipAvailable) { t.Fatal("expected status available") }
}

func TestUpdateEquipmentStatus(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	portID := uuid.New(); equipID := uuid.New()
	store.CreateEquipment(context.Background(), models.Equipment{ID:equipID,PortID:portID,EquipmentCode:"EQ-TEST001",EquipmentType:models.EquipForklift,Model:"Toyota 8FGU25",Status:models.EquipAvailable,CapacityT:2.5,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body := `{"status":"in_use"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/equipment/"+equipID.String()+"/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	e, _ := store.GetEquipment(context.Background(), equipID)
	if e.Status != models.EquipInUse { t.Fatal("expected status in_use") }
}

func TestGetOperation_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/operations/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}
