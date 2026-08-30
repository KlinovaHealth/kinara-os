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
	"github.com/klinova/kinara-os/vessel-service/auth"
	"github.com/klinova/kinara-os/vessel-service/handlers"
	"github.com/klinova/kinara-os/vessel-service/middleware"
	"github.com/klinova/kinara-os/vessel-service/models"
)

type memStore struct {
	mu          sync.RWMutex
	vessels     map[uuid.UUID]*models.Vessel
	voyages     map[uuid.UUID]*models.VoyageRecord
	maintenance []models.MaintenanceRecord
	audit       []models.VesselAuditLog
}

func newMemStore() *memStore {
	return &memStore{vessels: map[uuid.UUID]*models.Vessel{}, voyages: map[uuid.UUID]*models.VoyageRecord{}}
}

func (s *memStore) RegisterVessel(_ context.Context, v models.Vessel) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.vessels[v.ID] = &v; return nil
}
func (s *memStore) GetVessel(_ context.Context, id uuid.UUID) (*models.Vessel, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	v, ok := s.vessels[id]; if !ok { return nil, errNotFound }
	cp := *v; return &cp, nil
}
func (s *memStore) ListVessels(_ context.Context, flag *string, activeOnly bool) ([]models.Vessel, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Vessel
	for _, v := range s.vessels {
		if flag != nil && v.Flag != *flag { continue }
		if activeOnly && !v.IsActive { continue }
		result = append(result, *v)
	}
	return result, nil
}
func (s *memStore) UpdateVesselCondition(_ context.Context, id uuid.UUID, condition models.VesselCondition, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	v, ok := s.vessels[id]; if !ok { return errNotFound }
	v.Condition = condition; v.UpdatedAt = now; return nil
}
func (s *memStore) LogVoyage(_ context.Context, v models.VoyageRecord) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.voyages[v.ID] = &v; return nil
}
func (s *memStore) GetVoyage(_ context.Context, id uuid.UUID) (*models.VoyageRecord, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	v, ok := s.voyages[id]; if !ok { return nil, errNotFound }
	cp := *v; return &cp, nil
}
func (s *memStore) ListVoyages(_ context.Context, vesselID uuid.UUID) ([]models.VoyageRecord, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.VoyageRecord
	for _, v := range s.voyages { if v.VesselID == vesselID { result = append(result, *v) } }
	return result, nil
}
func (s *memStore) LogMaintenance(_ context.Context, m models.MaintenanceRecord) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.maintenance = append(s.maintenance, m); return nil
}
func (s *memStore) ListMaintenance(_ context.Context, vesselID uuid.UUID) ([]models.MaintenanceRecord, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.MaintenanceRecord
	for _, m := range s.maintenance { if m.VesselID == vesselID { result = append(result, m) } }
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.VesselAuditLog) error {
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
			claims := &auth.Claims{UserID: uuid.New(), Role: "fleet_manager"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestRegisterVessel(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.RegisterVesselRequest{IMONumber:"IMO9876543",Name:"MV Kinara Star",VesselType:string(models.VesselContainerShip),Flag:"KE",Owner:"KinaraShipping Ltd",YearBuilt:2018,GrossTonnage:45000,DeadweightT:62000,LengthM:230,BeamM:32,MaxDraftM:12,MaxSpeed:20})
	resp, _ := http.Post(srv.URL+"/api/v1/vessels", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestRegisterVessel_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"name":"Incomplete Vessel"}`
	resp, _ := http.Post(srv.URL+"/api/v1/vessels", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetVessel(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.RegisterVessel(context.Background(), models.Vessel{ID:id,IMONumber:"IMO1234567",Name:"MV Africa Pride",VesselType:models.VesselBulkCarrier,Flag:"GH",Owner:"Ghana Shipping",OperatorID:uuid.New(),YearBuilt:2015,GrossTonnage:35000,Condition:models.ConditionGood,IsActive:true,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/vessels/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetVessel_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/vessels/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}

func TestUpdateVesselCondition(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.RegisterVessel(context.Background(), models.Vessel{ID:id,IMONumber:"IMO5550001",Name:"MV Sahel Wind",VesselType:models.VesselGeneral,Flag:"SN",Owner:"Dakar Lines",OperatorID:uuid.New(),YearBuilt:2010,GrossTonnage:12000,Condition:models.ConditionGood,IsActive:true,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body := `{"condition":"poor"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/vessels/"+id.String()+"/condition", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	v, _ := store.GetVessel(context.Background(), id)
	if v.Condition != models.ConditionPoor { t.Fatal("expected condition poor") }
}

func TestLogVoyage(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.RegisterVessel(context.Background(), models.Vessel{ID:id,IMONumber:"IMO7778889",Name:"MV Lagos Express",VesselType:models.VesselContainerShip,Flag:"NG",Owner:"Nigeria Cargo",OperatorID:uuid.New(),YearBuilt:2020,GrossTonnage:55000,Condition:models.ConditionExcellent,IsActive:true,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.LogVoyageRequest{DeparturePortID:uuid.New().String(),ArrivalPortID:uuid.New().String(),DepartedAt:"2026-08-01T08:00:00Z",DistanceNM:1500,CargoTonnage:48000})
	resp, _ := http.Post(srv.URL+"/api/v1/vessels/"+id.String()+"/voyages", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if !strings.HasPrefix(data["voyage_code"].(string), "VO-") { t.Fatal("voyage_code must start with VO-") }
}

func TestLogMaintenance(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.RegisterVessel(context.Background(), models.Vessel{ID:id,IMONumber:"IMO3334445",Name:"MV Tema Bay",VesselType:models.VesselTanker,Flag:"GH",Owner:"Tema Shipping",OperatorID:uuid.New(),YearBuilt:2012,GrossTonnage:22000,Condition:models.ConditionFair,IsActive:true,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.LogMaintenanceRequest{MaintenanceType:string(models.MaintDryDock),Description:"Annual dry dock inspection and hull cleaning",StartDate:"2026-09-01T00:00:00Z",Cost:85000,Currency:"USD",Vendor:"Mombasa Drydock Ltd"})
	resp, _ := http.Post(srv.URL+"/api/v1/vessels/"+id.String()+"/maintenance", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestListVessels_ByFlag(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	store.RegisterVessel(context.Background(), models.Vessel{ID:uuid.New(),IMONumber:"IMO1111111",Name:"MV KE Vessel",VesselType:models.VesselFerry,Flag:"KE",Owner:"Kenya Ferry",OperatorID:uuid.New(),YearBuilt:2019,GrossTonnage:5000,Condition:models.ConditionGood,IsActive:true,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.RegisterVessel(context.Background(), models.Vessel{ID:uuid.New(),IMONumber:"IMO2222222",Name:"MV TZ Vessel",VesselType:models.VesselFerry,Flag:"TZ",Owner:"Tanzania Ferry",OperatorID:uuid.New(),YearBuilt:2018,GrossTonnage:4500,Condition:models.ConditionGood,IsActive:true,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/vessels?flag=KE")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 KE vessel, got %d", len(items)) }
}
