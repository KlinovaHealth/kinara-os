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
	"github.com/klinova/kinara-os/cargo-maritime-service/auth"
	"github.com/klinova/kinara-os/cargo-maritime-service/handlers"
	"github.com/klinova/kinara-os/cargo-maritime-service/middleware"
	"github.com/klinova/kinara-os/cargo-maritime-service/models"
)

type memStore struct {
	mu         sync.RWMutex
	containers map[uuid.UUID]*models.Container
	manifests  map[uuid.UUID]*models.CargoManifest
	mcLinks    []models.ManifestContainer
	damage     []models.DamageReport
	audit      []models.CargoMaritimeAuditLog
}

func newMemStore() *memStore {
	return &memStore{containers: map[uuid.UUID]*models.Container{}, manifests: map[uuid.UUID]*models.CargoManifest{}}
}

func (s *memStore) RegisterContainer(_ context.Context, c models.Container) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.containers[c.ID] = &c; return nil
}
func (s *memStore) GetContainer(_ context.Context, id uuid.UUID) (*models.Container, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	c, ok := s.containers[id]; if !ok { return nil, errNotFound }
	cp := *c; return &cp, nil
}
func (s *memStore) ListContainers(_ context.Context, status *models.ContainerStatus, vesselID *uuid.UUID) ([]models.Container, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Container
	for _, c := range s.containers {
		if status != nil && c.Status != *status { continue }
		if vesselID != nil && (c.VesselID == nil || *c.VesselID != *vesselID) { continue }
		result = append(result, *c)
	}
	return result, nil
}
func (s *memStore) UpdateContainerStatus(_ context.Context, id uuid.UUID, status models.ContainerStatus, sealNo string, portID, vesselID *uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	c, ok := s.containers[id]; if !ok { return errNotFound }
	c.Status = status; c.UpdatedAt = now
	if sealNo != "" { c.SealNo = sealNo }
	if portID != nil { c.CurrentPortID = portID }
	if vesselID != nil { c.VesselID = vesselID }
	return nil
}
func (s *memStore) CreateManifest(_ context.Context, m models.CargoManifest) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.manifests[m.ID] = &m; return nil
}
func (s *memStore) GetManifest(_ context.Context, id uuid.UUID) (*models.CargoManifest, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	m, ok := s.manifests[id]; if !ok { return nil, errNotFound }
	cp := *m; return &cp, nil
}
func (s *memStore) AddContainerToManifest(_ context.Context, mc models.ManifestContainer, weightKg float64) error {
	s.mu.Lock(); defer s.mu.Unlock()
	s.mcLinks = append(s.mcLinks, mc)
	m, ok := s.manifests[mc.ManifestID]; if !ok { return errNotFound }
	m.TotalContainers++; m.TotalWeightKg += weightKg; return nil
}
func (s *memStore) FinalizeManifest(_ context.Context, id uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	m, ok := s.manifests[id]; if !ok { return errNotFound }
	m.IsFinalized = true; m.UpdatedAt = now; return nil
}
func (s *memStore) ReportDamage(_ context.Context, d models.DamageReport) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.damage = append(s.damage, d); return nil
}
func (s *memStore) ListDamageReports(_ context.Context, containerID uuid.UUID) ([]models.DamageReport, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.DamageReport
	for _, d := range s.damage { if d.ContainerID == containerID { result = append(result, d) } }
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.CargoMaritimeAuditLog) error {
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
			claims := &auth.Claims{UserID: uuid.New().String(), Role: "cargo_officer"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestRegisterContainer(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.RegisterContainerRequest{ContainerNo:"MSCU1234567",ContainerType:string(models.Container40HC),TareWeightKg:3900,IsHazmat:false})
	resp, _ := http.Post(srv.URL+"/api/v1/containers", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["status"] != string(models.StatusEmpty) { t.Fatal("expected status empty") }
}

func TestRegisterContainer_MissingNo(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/containers", "application/json", bytes.NewBufferString(`{}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetContainer(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.RegisterContainer(context.Background(), models.Container{ID:id,ContainerNo:"MAEU7654321",ContainerType:models.Container20Dry,OwnerID:uuid.New(),Status:models.StatusAtPort,WeightKg:15000,TareWeightKg:2200,IsHazmat:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/containers/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestUpdateContainerStatus_Loaded(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); portID := uuid.New()
	store.RegisterContainer(context.Background(), models.Container{ID:id,ContainerNo:"HLCU9999001",ContainerType:models.Container40Dry,OwnerID:uuid.New(),Status:models.StatusEmpty,WeightKg:2300,TareWeightKg:2300,IsHazmat:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.UpdateContainerStatusRequest{Status:string(models.StatusLoaded),SealNo:"KE-SEAL-001",PortID:portID.String()})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/containers/"+id.String()+"/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestCreateManifestAndAddContainer(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	containerID := uuid.New()
	store.RegisterContainer(context.Background(), models.Container{ID:containerID,ContainerNo:"OOLU1234001",ContainerType:models.Container20Reefer,OwnerID:uuid.New(),Status:models.StatusLoaded,WeightKg:18000,TareWeightKg:2500,IsHazmat:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	manifestBody, _ := json.Marshal(models.CreateManifestRequest{VesselID:uuid.New().String(),VoyageID:uuid.New().String(),PortOfLoading:uuid.New().String(),PortOfDischarge:uuid.New().String(),ShipperName:"Ghana Agri Exports Ltd",ConsigneeName:"Rotterdam Foods BV",Commodity:"Fresh Pineapples"})
	resp, _ := http.Post(srv.URL+"/api/v1/manifests", "application/json", bytes.NewBuffer(manifestBody))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	manifestData := out.Data.(map[string]interface{})
	manifestID := manifestData["id"].(string)
	addBody, _ := json.Marshal(models.AddContainerToManifestRequest{ContainerID:containerID.String()})
	resp2, _ := http.Post(srv.URL+"/api/v1/manifests/"+manifestID+"/containers", "application/json", bytes.NewBuffer(addBody))
	if resp2.StatusCode != 201 { t.Fatalf("expected 201 adding container, got %d", resp2.StatusCode) }
	mID, _ := uuid.Parse(manifestID)
	m, _ := store.GetManifest(context.Background(), mID)
	if m.TotalContainers != 1 { t.Fatalf("expected 1 container, got %d", m.TotalContainers) }
}

func TestFinalizeManifest(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	mID := uuid.New()
	store.CreateManifest(context.Background(), models.CargoManifest{ID:mID,ManifestNo:"MF-TEST01234",VoyageID:uuid.New(),VesselID:uuid.New(),PortOfLoading:uuid.New(),PortOfDischarge:uuid.New(),ShipperName:"Dakar Exports",ConsigneeName:"Marseille Imports",Commodity:"Groundnuts",IsFinalized:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/manifests/"+mID.String()+"/finalize", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	m, _ := store.GetManifest(context.Background(), mID)
	if !m.IsFinalized { t.Fatal("manifest should be finalized") }
}

func TestReportDamage(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	cID := uuid.New()
	store.RegisterContainer(context.Background(), models.Container{ID:cID,ContainerNo:"CMAU0987654",ContainerType:models.Container20Dry,OwnerID:uuid.New(),Status:models.StatusAtPort,WeightKg:12000,TareWeightKg:2200,IsHazmat:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.ReportDamageRequest{DamageLevel:string(models.DamageMinor),Description:"Corner dent on rear panel, no structural compromise",EstimatedCost:1500,Currency:"USD",PortID:uuid.New().String()})
	resp, _ := http.Post(srv.URL+"/api/v1/containers/"+cID.String()+"/damage", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestListContainers_ByStatus(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	store.RegisterContainer(context.Background(), models.Container{ID:uuid.New(),ContainerNo:"TCKU1111111",ContainerType:models.Container20Dry,OwnerID:uuid.New(),Status:models.StatusInTransit,WeightKg:15000,TareWeightKg:2200,IsHazmat:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	store.RegisterContainer(context.Background(), models.Container{ID:uuid.New(),ContainerNo:"TCKU2222222",ContainerType:models.Container40Dry,OwnerID:uuid.New(),Status:models.StatusEmpty,WeightKg:3900,TareWeightKg:3900,IsHazmat:false,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/containers?status=in_transit")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 in_transit container, got %d", len(items)) }
}
