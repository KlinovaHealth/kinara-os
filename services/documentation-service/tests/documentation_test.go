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
	"github.com/klinova/kinara-os/documentation-service/auth"
	"github.com/klinova/kinara-os/documentation-service/handlers"
	"github.com/klinova/kinara-os/documentation-service/middleware"
	"github.com/klinova/kinara-os/documentation-service/models"
)

type memStore struct {
	mu    sync.RWMutex
	docs  map[uuid.UUID]*models.TradeDocument
	audit []models.DocumentAuditLog
}

func newMemStore() *memStore { return &memStore{docs: map[uuid.UUID]*models.TradeDocument{}} }

func (s *memStore) CreateDocument(_ context.Context, d models.TradeDocument) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.docs[d.ID] = &d; return nil
}
func (s *memStore) GetDocument(_ context.Context, id uuid.UUID) (*models.TradeDocument, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	d, ok := s.docs[id]; if !ok { return nil, errNotFound }
	cp := *d; return &cp, nil
}
func (s *memStore) ListDocuments(_ context.Context, docType *models.DocType, bookingRef *string) ([]models.TradeDocument, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.TradeDocument
	for _, d := range s.docs {
		if docType != nil && d.DocType != *docType { continue }
		if bookingRef != nil && d.BookingRef != *bookingRef { continue }
		result = append(result, *d)
	}
	return result, nil
}
func (s *memStore) IssueDocument(_ context.Context, id uuid.UUID, fileURL string, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	d, ok := s.docs[id]; if !ok { return errNotFound }
	d.Status = models.DocIssued
	if d.IssuedAt == nil { d.IssuedAt = &now }
	if fileURL != "" { d.FileURL = fileURL }
	d.UpdatedAt = now; return nil
}
func (s *memStore) RevokeDocument(_ context.Context, id uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	d, ok := s.docs[id]; if !ok { return errNotFound }
	d.Status = models.DocRevoked; d.UpdatedAt = now; return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.DocumentAuditLog) error {
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
			claims := &auth.Claims{UserID: uuid.New().String(), Role: "trade_officer"}
			next.ServeHTTP(w, req.WithContext(middleware.SetClaims(req.Context(), claims)))
		})
	})
	h.RegisterRoutes(api)
	return httptest.NewServer(r), store
}

func TestCreateDocument(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(models.CreateDocumentRequest{DocType:string(models.DocCommercialInvoice),ShipperName:"Kinara Exports Ltd",ConsigneeName:"Dakar Imports SARL",IssuingCountry:"KE",IssuingAuthority:"Kenya Revenue Authority",GoodsDescription:"Processed Coffee 500kg",Value:15000,Currency:"USD",WeightKg:500,NetWeightKg:480,Packages:10})
	resp, _ := http.Post(srv.URL+"/api/v1/documents", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
	data := out.Data.(map[string]interface{})
	if !strings.HasPrefix(data["document_ref"].(string), "TD-") { t.Fatal("document_ref must start with TD-") }
	if data["status"].(string) != "draft" { t.Fatal("initial status must be draft") }
}

func TestCreateDocument_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"shipper_name":"Shipper Only"}`
	resp, _ := http.Post(srv.URL+"/api/v1/documents", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetDocument(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); now := time.Now().UTC()
	store.CreateDocument(context.Background(), models.TradeDocument{ID:id,DocumentRef:"TD-CERT001",DocType:models.DocCertOrigin,ShipperName:"Ghana Cocoa Board",ConsigneeName:"Swiss Trader AG",IssuingCountry:"GH",IssuingAuthority:"Ghana Standards Authority",GoodsDescription:"Cocoa Beans Grade A",Value:80000,Currency:"USD",WeightKg:2000,NetWeightKg:1950,Packages:40,Status:models.DocDraft,CreatedAt:now,UpdatedAt:now})
	resp, _ := http.Get(srv.URL + "/api/v1/documents/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestGetDocument_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/documents/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}

func TestIssueDocument(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); now := time.Now().UTC()
	store.CreateDocument(context.Background(), models.TradeDocument{ID:id,DocumentRef:"TD-INS001",DocType:models.DocInsuranceCert,ShipperName:"Lagos Shipping Co",ConsigneeName:"Accra Receivers Ltd",IssuingCountry:"NG",IssuingAuthority:"NAICOM",GoodsDescription:"Electronics",Value:50000,Currency:"USD",WeightKg:300,NetWeightKg:285,Packages:15,Status:models.DocDraft,CreatedAt:now,UpdatedAt:now})
	body := `{"file_url":"https://cdn.kinara.io/docs/insurance-001.pdf"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/documents/"+id.String()+"/issue", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	d, _ := store.GetDocument(context.Background(), id)
	if d.Status != models.DocIssued { t.Fatal("expected status issued") }
	if d.IssuedAt == nil { t.Fatal("issued_at must be set") }
	if d.FileURL == "" { t.Fatal("file_url must be set") }
}

func TestIssueDocument_Idempotent(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); now := time.Now().UTC()
	issuedAt := now.Add(-1 * time.Hour)
	store.CreateDocument(context.Background(), models.TradeDocument{ID:id,DocumentRef:"TD-IDEM001",DocType:models.DocPackingList,ShipperName:"Kinara",ConsigneeName:"Partner",IssuingCountry:"TZ",IssuingAuthority:"TBS",GoodsDescription:"Tea",Value:5000,Currency:"USD",WeightKg:100,Status:models.DocIssued,IssuedAt:&issuedAt,CreatedAt:now,UpdatedAt:now})
	body := `{"file_url":"https://new-file.pdf"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/documents/"+id.String()+"/issue", bytes.NewBufferString(body))
	req.Header.Set("Content-Type","application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	d, _ := store.GetDocument(context.Background(), id)
	if !d.IssuedAt.Equal(issuedAt) { t.Fatal("issued_at must not be overwritten (COALESCE)") }
}

func TestRevokeDocument(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	id := uuid.New(); now := time.Now().UTC(); issuedAt := now
	store.CreateDocument(context.Background(), models.TradeDocument{ID:id,DocumentRef:"TD-REV001",DocType:models.DocBillOfLading,ShipperName:"Mombasa Lines",ConsigneeName:"Dar Port Authority",IssuingCountry:"KE",IssuingAuthority:"KPA",GoodsDescription:"Container cargo",Value:200000,Currency:"USD",WeightKg:20000,Status:models.DocIssued,IssuedAt:&issuedAt,CreatedAt:now,UpdatedAt:now})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/documents/"+id.String()+"/revoke", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	d, _ := store.GetDocument(context.Background(), id)
	if d.Status != models.DocRevoked { t.Fatal("expected status revoked") }
}

func TestListDocuments_ByType(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	now := time.Now().UTC()
	store.CreateDocument(context.Background(), models.TradeDocument{ID:uuid.New(),DocumentRef:"TD-A001",DocType:models.DocCommercialInvoice,ShipperName:"A",ConsigneeName:"B",IssuingCountry:"KE",IssuingAuthority:"Auth",GoodsDescription:"Goods",Currency:"USD",Status:models.DocDraft,CreatedAt:now,UpdatedAt:now})
	store.CreateDocument(context.Background(), models.TradeDocument{ID:uuid.New(),DocumentRef:"TD-B001",DocType:models.DocCertOrigin,ShipperName:"C",ConsigneeName:"D",IssuingCountry:"GH",IssuingAuthority:"Auth",GoodsDescription:"Other",Currency:"USD",Status:models.DocDraft,CreatedAt:now,UpdatedAt:now})
	resp, _ := http.Get(srv.URL + "/api/v1/documents?document_type=commercial_invoice")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 { t.Fatalf("expected 1 commercial_invoice, got %d", len(items)) }
}
