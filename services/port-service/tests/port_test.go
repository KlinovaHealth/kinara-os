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
	"github.com/klinova/kinara-os/port-service/auth"
	"github.com/klinova/kinara-os/port-service/db"
	"github.com/klinova/kinara-os/port-service/handlers"
	"github.com/klinova/kinara-os/port-service/middleware"
	"github.com/klinova/kinara-os/port-service/models"
)

type memStore struct {
	mu        sync.RWMutex
	ports     map[uuid.UUID]*models.Port
	berths    map[uuid.UUID]*models.Berth
	schedules map[uuid.UUID]*models.BerthSchedule
	alerts    []models.CongestionAlert
	audit     []models.PortAuditLog
}

func newMemStore() *memStore {
	return &memStore{ports: map[uuid.UUID]*models.Port{}, berths: map[uuid.UUID]*models.Berth{}, schedules: map[uuid.UUID]*models.BerthSchedule{}}
}

func (s *memStore) CreatePort(_ context.Context, p models.Port) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.ports[p.ID] = &p; return nil
}
func (s *memStore) GetPort(_ context.Context, id uuid.UUID) (*models.Port, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	p, ok := s.ports[id]; if !ok { return nil, errNotFound }
	cp := *p; return &cp, nil
}
func (s *memStore) ListPorts(_ context.Context, country *string) ([]models.Port, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Port
	for _, p := range s.ports {
		if country != nil && p.Country != *country { continue }
		result = append(result, *p)
	}
	return result, nil
}
func (s *memStore) CreateBerth(_ context.Context, b models.Berth) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.berths[b.ID] = &b; return nil
}
func (s *memStore) GetBerth(_ context.Context, id uuid.UUID) (*models.Berth, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	b, ok := s.berths[id]; if !ok { return nil, errNotFound }
	cp := *b; return &cp, nil
}
func (s *memStore) ListBerths(_ context.Context, p db.ListBerthsParams) ([]models.Berth, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Berth
	for _, b := range s.berths {
		if p.PortID != nil && b.PortID != *p.PortID { continue }
		if p.Status != nil && b.Status != *p.Status { continue }
		result = append(result, *b)
	}
	return result, nil
}
func (s *memStore) UpdateBerthStatus(_ context.Context, id uuid.UUID, status models.BerthStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	b, ok := s.berths[id]; if !ok { return errNotFound }
	b.Status = status; b.UpdatedAt = now; return nil
}
func (s *memStore) CreateSchedule(_ context.Context, sc models.BerthSchedule) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.schedules[sc.ID] = &sc; return nil
}
func (s *memStore) GetSchedule(_ context.Context, id uuid.UUID) (*models.BerthSchedule, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	sc, ok := s.schedules[id]; if !ok { return nil, errNotFound }
	cp := *sc; return &cp, nil
}
func (s *memStore) ListSchedulesByBerth(_ context.Context, berthID uuid.UUID) ([]models.BerthSchedule, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.BerthSchedule
	for _, sc := range s.schedules { if sc.BerthID == berthID { result = append(result, *sc) } }
	return result, nil
}
func (s *memStore) UpdateScheduleStatus(_ context.Context, id uuid.UUID, status models.VesselStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	sc, ok := s.schedules[id]; if !ok { return errNotFound }
	sc.Status = status; sc.UpdatedAt = now
	if status == models.VesselArrived { sc.ActualArrival = &now }
	if status == models.VesselDeparted { sc.ActualDeparture = &now }
	return nil
}
func (s *memStore) CreateCongestionAlert(_ context.Context, a models.CongestionAlert) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.alerts = append(s.alerts, a); return nil
}
func (s *memStore) ListAlerts(_ context.Context, portID uuid.UUID) ([]models.CongestionAlert, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.CongestionAlert
	for _, a := range s.alerts { if a.PortID == portID { result = append(result, a) } }
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.PortAuditLog) error {
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
			claims := &auth.Claims{UserID: uuid.New(), Role: "port_admin"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestCreatePort(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreatePortRequest{Name:"Port of Mombasa",Country:"KE",City:"Mombasa",Latitude:-4.0435,Longitude:39.6682,MaxDraft:12.5,TotalBerths:20})
	resp, _ := http.Post(srv.URL+"/api/v1/ports", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreatePort_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"name":"Incomplete Port"}`
	resp, _ := http.Post(srv.URL+"/api/v1/ports", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetPort(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreatePort(context.Background(), models.Port{ID:id,Name:"Port of Dar es Salaam",Code:"PT-DARES",Country:"TZ",City:"Dar es Salaam",AlertLevel:models.AlertNormal,TotalBerths:15,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/ports/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetPort_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/ports/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}

func TestCreateBerthAndSchedule(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	portID := uuid.New()
	store.CreatePort(context.Background(), models.Port{ID:portID,Name:"Port of Lagos",Code:"PT-LAGOS",Country:"NG",City:"Lagos",AlertLevel:models.AlertNormal,TotalBerths:30,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	berthBody, _ := json.Marshal(models.CreateBerthRequest{BerthNumber:"B-001",MaxLengthM:200,MaxDraftM:10,MaxTonnage:50000})
	resp, _ := http.Post(srv.URL+"/api/v1/ports/"+portID.String()+"/berths", "application/json", bytes.NewBuffer(berthBody))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	berthData := out.Data.(map[string]interface{})
	berthID := berthData["id"].(string)
	schedBody, _ := json.Marshal(models.ScheduleBerthRequest{VesselID:uuid.New().String(),VesselName:"MV Kinara Star",ETA:"2026-09-15T06:00:00Z",ETD:"2026-09-17T18:00:00Z",CargoType:"containers",TonnageT:35000})
	resp2, _ := http.Post(srv.URL+"/api/v1/berths/"+berthID+"/schedules", "application/json", bytes.NewBuffer(schedBody))
	if resp2.StatusCode != 201 { t.Fatalf("expected 201 for schedule, got %d", resp2.StatusCode) }
}

func TestUpdateScheduleStatus_Arrived(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	portID := uuid.New(); berthID := uuid.New(); schedID := uuid.New()
	store.CreatePort(context.Background(), models.Port{ID:portID,Name:"Port of Abidjan",Code:"PT-ABJ",Country:"CI",City:"Abidjan",AlertLevel:models.AlertNormal,TotalBerths:10,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.CreateBerth(context.Background(), models.Berth{ID:berthID,PortID:portID,BerthNumber:"C-003",Status:models.BerthReserved,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	eta := time.Now().Add(24 * time.Hour)
	store.CreateSchedule(context.Background(), models.BerthSchedule{ID:schedID,BerthID:berthID,VesselID:uuid.New(),VesselName:"MV Atlas",Status:models.VesselExpected,ETA:eta,ETD:eta.Add(48*time.Hour),CargoType:"bulk",TonnageT:20000,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body := `{"status":"arrived"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/schedules/"+schedID.String()+"/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	sc, _ := store.GetSchedule(context.Background(), schedID)
	if sc.ActualArrival == nil { t.Fatal("actual_arrival should be set") }
}

func TestCreateCongestionAlert(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	portID := uuid.New(); berthID := uuid.New()
	store.CreatePort(context.Background(), models.Port{ID:portID,Name:"Port of Dakar",Code:"PT-DKR",Country:"SN",City:"Dakar",AlertLevel:models.AlertNormal,TotalBerths:8,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.CreateBerth(context.Background(), models.Berth{ID:berthID,PortID:portID,BerthNumber:"D-001",Status:models.BerthOccupied,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.CreateAlertRequest{AlertLevel:string(models.AlertHigh),Message:"Port at 85% capacity — vessel queuing expected"})
	resp, _ := http.Post(srv.URL+"/api/v1/ports/"+portID.String()+"/alerts", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestListBerths_ByStatus(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	portID := uuid.New()
	store.CreatePort(context.Background(), models.Port{ID:portID,Name:"Port of Tema",Code:"PT-TEMA",Country:"GH",City:"Tema",AlertLevel:models.AlertNormal,TotalBerths:5,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.CreateBerth(context.Background(), models.Berth{ID:uuid.New(),PortID:portID,BerthNumber:"E-001",Status:models.BerthAvailable,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.CreateBerth(context.Background(), models.Berth{ID:uuid.New(),PortID:portID,BerthNumber:"E-002",Status:models.BerthOccupied,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/ports/" + portID.String() + "/berths?status=available")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 available berth, got %d", len(items)) }
}
