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
	"github.com/klinova/kinara-os/shipping-service/auth"
	"github.com/klinova/kinara-os/shipping-service/handlers"
	"github.com/klinova/kinara-os/shipping-service/middleware"
	"github.com/klinova/kinara-os/shipping-service/models"
)

type memStore struct {
	mu        sync.RWMutex
	bookings  map[uuid.UUID]*models.FreightBooking
	bols      map[uuid.UUID]*models.BillOfLading
	demurrage []models.DemurrageRecord
	audit     []models.ShippingAuditLog
}

func newMemStore() *memStore {
	return &memStore{bookings: map[uuid.UUID]*models.FreightBooking{}, bols: map[uuid.UUID]*models.BillOfLading{}}
}

func (s *memStore) CreateBooking(_ context.Context, b models.FreightBooking) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.bookings[b.ID] = &b; return nil
}
func (s *memStore) GetBooking(_ context.Context, id uuid.UUID) (*models.FreightBooking, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	b, ok := s.bookings[id]; if !ok { return nil, errNotFound }
	cp := *b; return &cp, nil
}
func (s *memStore) ListBookings(_ context.Context, shipperID *uuid.UUID, status *models.FreightStatus) ([]models.FreightBooking, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.FreightBooking
	for _, b := range s.bookings {
		if shipperID != nil && b.ShipperID != *shipperID { continue }
		if status != nil && b.Status != *status { continue }
		result = append(result, *b)
	}
	return result, nil
}
func (s *memStore) UpdateBookingStatus(_ context.Context, id uuid.UUID, status models.FreightStatus, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	b, ok := s.bookings[id]; if !ok { return errNotFound }
	b.Status = status; b.UpdatedAt = now; return nil
}
func (s *memStore) IssueBOL(_ context.Context, bol models.BillOfLading) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.bols[bol.ID] = &bol; return nil
}
func (s *memStore) GetBOL(_ context.Context, id uuid.UUID) (*models.BillOfLading, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	bol, ok := s.bols[id]; if !ok { return nil, errNotFound }
	cp := *bol; return &cp, nil
}
func (s *memStore) SurrenderBOL(_ context.Context, id uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	bol, ok := s.bols[id]; if !ok { return errNotFound }
	bol.Status = models.BOLSurrendered; bol.UpdatedAt = now
	if bol.SurrenderedAt == nil { bol.SurrenderedAt = &now }
	return nil
}
func (s *memStore) RecordDemurrage(_ context.Context, d models.DemurrageRecord) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.demurrage = append(s.demurrage, d); return nil
}
func (s *memStore) ListDemurrage(_ context.Context, bookingID uuid.UUID) ([]models.DemurrageRecord, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.DemurrageRecord
	for _, d := range s.demurrage { if d.BookingID == bookingID { result = append(result, d) } }
	return result, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.ShippingAuditLog) error {
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
			claims := &auth.Claims{UserID: uuid.New(), Role: "shipping_agent"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestCreateBooking(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateBookingRequest{ShipperName:"Kumasi Cocoa Board",ConsigneeName:"Amsterdam Traders BV",ShipmentType:"fcl",PortOfLoading:uuid.New().String(),PortOfDischarge:uuid.New().String(),CommodityDesc:"Cocoa Beans",ContainerCount:5,WeightKg:125000,FreightRate:3500,DeclaredValue:750000,InsurancePct:0.5,Currency:"USD"})
	resp, _ := http.Post(srv.URL+"/api/v1/bookings", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["status"] != string(models.FreightPending) { t.Fatal("expected status pending") }
}

func TestInsuranceCalculation(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateBookingRequest{ShipperName:"Dakar Phosphates SA",ConsigneeName:"Hamburg Chemicals GmbH",ShipmentType:"fcl",CommodityDesc:"Phosphate Rock",ContainerCount:10,WeightKg:300000,FreightRate:2800,DeclaredValue:1000000,InsurancePct:0.75,Currency:"USD"})
	resp, _ := http.Post(srv.URL+"/api/v1/bookings", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["insurance_amount"].(float64) != 7500.0 { t.Fatalf("expected insurance 7500, got %f", data["insurance_amount"].(float64)) }
}

func TestIssueBOL(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	bookingID := uuid.New()
	store.CreateBooking(context.Background(), models.FreightBooking{ID:bookingID,BookingRef:"SB-TEST000001",ShipperID:uuid.New(),ShipperName:"Lagos Steel Ltd",ConsigneeName:"Antwerp Metals",ShipmentType:models.ShipFCL,PortOfLoading:uuid.New(),PortOfDischarge:uuid.New(),CommodityDesc:"Steel Coils",ContainerCount:3,WeightKg:90000,FreightRate:4200,TotalFreight:12600,Currency:"USD",Status:models.FreightBooked,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.IssueBOLRequest{VesselName:"MV West Africa",VoyageNo:"WA-2026-08",POL:"Apapa, Lagos",POD:"Port of Antwerp",FreightPrepaid:true})
	resp, _ := http.Post(srv.URL+"/api/v1/bookings/"+bookingID.String()+"/bol", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["issued_at"] == nil { t.Fatal("issued_at should be set") }
}

func TestSurrenderBOL(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	bookingID := uuid.New(); bolID := uuid.New()
	now := time.Now()
	store.CreateBooking(context.Background(), models.FreightBooking{ID:bookingID,BookingRef:"SB-TEST000002",ShipperID:uuid.New(),ShipperName:"Mombasa Shipping",ConsigneeName:"Rotterdam Port Authority",ShipmentType:models.ShipLCL,PortOfLoading:uuid.New(),PortOfDischarge:uuid.New(),CommodityDesc:"Mixed Goods",ContainerCount:1,WeightKg:12000,TotalFreight:5000,Currency:"USD",Status:models.FreightInTransit,CreatedAt:now,UpdatedAt:now})
	store.IssueBOL(context.Background(), models.BillOfLading{ID:bolID,BOLNumber:"BL-TEST000001",BookingID:bookingID,VesselName:"MV Indian Ocean",VoyageNo:"IO-2026-01",ShipperName:"Mombasa Shipping",ConsigneeName:"Rotterdam Port Authority",POL:"Mombasa",POD:"Rotterdam",CommodityDesc:"Mixed Goods",ContainerCount:1,GrossWeightKg:12000,FreightPrepaid:true,Status:models.BOLIssued,IssuedAt:&now,CreatedAt:now,UpdatedAt:now})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/bol/"+bolID.String()+"/surrender", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	bol, _ := store.GetBOL(context.Background(), bolID)
	if bol.Status != models.BOLSurrendered { t.Fatal("expected BOL surrendered") }
	if bol.SurrenderedAt == nil { t.Fatal("surrendered_at should be set") }
}

func TestRecordDemurrage(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	bookingID := uuid.New()
	store.CreateBooking(context.Background(), models.FreightBooking{ID:bookingID,BookingRef:"SB-TEST000003",ShipperID:uuid.New(),ShipperName:"Abidjan Cashews",ConsigneeName:"Paris Nuts",ShipmentType:models.ShipFCL,PortOfLoading:uuid.New(),PortOfDischarge:uuid.New(),CommodityDesc:"Cashew Nuts",ContainerCount:2,WeightKg:50000,TotalFreight:8000,Currency:"USD",Status:models.FreightDelivered,CreatedAt:time.Now(),UpdatedAt:time.Now()})
	body, _ := json.Marshal(models.RecordDemurrageRequest{ContainerNo:"GAOU1234567",FreeDays:5,UsedDays:12,DailyRate:150,PortID:uuid.New().String(),Currency:"USD"})
	resp, _ := http.Post(srv.URL+"/api/v1/bookings/"+bookingID.String()+"/demurrage", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["total_charge"].(float64) != 1050.0 { t.Fatalf("expected 1050 demurrage (7 days * 150), got %f", data["total_charge"].(float64)) }
}

func TestCreateBooking_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/bookings", "application/json", bytes.NewBufferString(`{"consignee_name":"only"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetBooking_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/bookings/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}
