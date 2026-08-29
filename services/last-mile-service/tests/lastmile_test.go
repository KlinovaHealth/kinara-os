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
	"github.com/klinova/kinara-os/last-mile-service/auth"
	"github.com/klinova/kinara-os/last-mile-service/db"
	"github.com/klinova/kinara-os/last-mile-service/handlers"
	"github.com/klinova/kinara-os/last-mile-service/middleware"
	"github.com/klinova/kinara-os/last-mile-service/models"
)

type memStore struct {
	mu         sync.RWMutex
	deliveries map[uuid.UUID]*models.Delivery
	audit      []models.LastMileAuditLog
}

func newMemStore() *memStore { return &memStore{deliveries: map[uuid.UUID]*models.Delivery{}} }

func (s *memStore) CreateDelivery(_ context.Context, d models.Delivery) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.deliveries[d.ID] = &d; return nil
}
func (s *memStore) GetDelivery(_ context.Context, id uuid.UUID) (*models.Delivery, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	d, ok := s.deliveries[id]; if !ok { return nil, errNotFound }
	cp := *d; return &cp, nil
}
func (s *memStore) ListDeliveries(_ context.Context, p db.ListDeliveryParams) ([]models.Delivery, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Delivery
	for _, d := range s.deliveries {
		if p.Status != nil && d.Status != *p.Status { continue }
		result = append(result, *d)
	}
	return result, nil
}
func (s *memStore) AssignDriver(_ context.Context, id, driverID uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	d, ok := s.deliveries[id]; if !ok { return errNotFound }
	d.DriverID = &driverID; d.Status = models.DeliveryAssigned; d.UpdatedAt = now; return nil
}
func (s *memStore) RecordDelivered(_ context.Context, id uuid.UUID, photoURL, sigURL, notes string, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	d, ok := s.deliveries[id]; if !ok { return errNotFound }
	d.Status = models.DeliveryDelivered; d.DeliveredAt = &now; d.ProofPhotoURL = photoURL; d.SignatureURL = sigURL; d.Notes = notes; d.UpdatedAt = now; return nil
}
func (s *memStore) RecordFailure(_ context.Context, id uuid.UUID, reason models.FailureReason, nextAt *time.Time, notes string, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	d, ok := s.deliveries[id]; if !ok { return errNotFound }
	d.Status = models.DeliveryFailed; d.FailureReason = reason; d.NextAttemptAt = nextAt; d.AttemptCount++; d.Notes = notes; d.UpdatedAt = now; return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.LastMileAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewLastMileHandlerWithStore(store)
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := &auth.Claims{UserID: uuid.New().String(), Role: "admin"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestCreateDelivery(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"recipient_name":"Ama Sarpong","recipient_phone":"+233501234567","delivery_address":"22 Ring Road, Accra","country":"GH"}`
	resp, _ := http.Post(srv.URL+"/api/v1/deliveries", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestCreateDelivery_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/deliveries", "application/json", bytes.NewBufferString(`{"recipient_name":"No Address"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestAssignDriver(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); did := uuid.New()
	store.CreateDelivery(context.Background(), models.Delivery{ID:id,DeliveryCode:"DL-test001",RecipientName:"Kofi Boateng",RecipientPhone:"+233244567890",DeliveryAddress:"Tema",Status:models.DeliveryPending,Country:"GH",CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.AssignDriverRequest{DriverID:did.String()})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/deliveries/"+id.String()+"/assign", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	d, _ := store.GetDelivery(context.Background(), id)
	if d.Status != models.DeliveryAssigned { t.Fatal("expected status assigned") }
}

func TestRecordDelivered(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateDelivery(context.Background(), models.Delivery{ID:id,DeliveryCode:"DL-test002",RecipientName:"Fatima Al-Hassan",RecipientPhone:"+234806543210",DeliveryAddress:"Kano",Status:models.DeliveryEnRoute,Country:"NG",CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.RecordDeliveryRequest{ProofPhotoURL:"https://cdn.kinara.io/proof/abc.jpg",Notes:"Left at gate"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/deliveries/"+id.String()+"/delivered", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	d, _ := store.GetDelivery(context.Background(), id)
	if d.Status != models.DeliveryDelivered { t.Fatal("expected status delivered") }
	if d.DeliveredAt == nil { t.Fatal("delivered_at should be set") }
}

func TestRecordFailed(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateDelivery(context.Background(), models.Delivery{ID:id,DeliveryCode:"DL-test003",RecipientName:"Moussa Diallo",RecipientPhone:"+221777654321",DeliveryAddress:"Dakar Medina",Status:models.DeliveryEnRoute,Country:"SN",AttemptCount:0,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.RecordFailureRequest{FailureReason:models.FailureNotHome,NextAttemptAt:"2026-09-02T10:00:00Z",Notes:"Will retry tomorrow"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/deliveries/"+id.String()+"/failed", bytes.NewBuffer(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	d, _ := store.GetDelivery(context.Background(), id)
	if d.Status != models.DeliveryFailed { t.Fatal("expected status failed") }
	if d.AttemptCount != 1 { t.Fatalf("expected attempt_count 1, got %d", d.AttemptCount) }
}

func TestGetDelivery(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New()
	store.CreateDelivery(context.Background(), models.Delivery{ID:id,DeliveryCode:"DL-test004",RecipientName:"Test User",RecipientPhone:"+254700000000",DeliveryAddress:"Nairobi",Status:models.DeliveryPending,Country:"KE",CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/deliveries/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetDelivery_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/deliveries/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}

func TestListDeliveries(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	store.CreateDelivery(context.Background(), models.Delivery{ID:uuid.New(),DeliveryCode:"DL-list001",RecipientName:"A",RecipientPhone:"+254711111111",DeliveryAddress:"Mombasa",Status:models.DeliveryPending,Country:"KE",CreatedAt:time.Now(),UpdatedAt:time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/deliveries")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}
