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
	"github.com/klinova/kinara-os/driver-service/auth"
	"github.com/klinova/kinara-os/driver-service/crypto"
	"github.com/klinova/kinara-os/driver-service/db"
	"github.com/klinova/kinara-os/driver-service/handlers"
	"github.com/klinova/kinara-os/driver-service/middleware"
	"github.com/klinova/kinara-os/driver-service/models"
)

type memStore struct {
	mu      sync.RWMutex
	drivers map[uuid.UUID]*models.DriverRow
	trips   []models.DriverTrip
	audit   []models.DriverAuditLog
}

func newMemStore() *memStore { return &memStore{drivers: map[uuid.UUID]*models.DriverRow{}} }

func (s *memStore) CreateDriver(_ context.Context, d models.DriverRow) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.drivers[d.ID] = &d; return nil
}
func (s *memStore) GetDriver(_ context.Context, id uuid.UUID) (*models.DriverRow, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	d, ok := s.drivers[id]
	if !ok { return nil, errNotFound }
	cp := *d; return &cp, nil
}
func (s *memStore) ListDrivers(_ context.Context, p db.ListDriversParams) ([]models.DriverRow, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.DriverRow
	for _, d := range s.drivers {
		if p.Status != nil && d.Status != *p.Status { continue }
		if p.Country != nil && d.Country != *p.Country { continue }
		result = append(result, *d)
	}
	return result, nil
}
func (s *memStore) UpdateDriver(_ context.Context, id uuid.UUID, req models.UpdateDriverRequest, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	d, ok := s.drivers[id]
	if !ok { return errNotFound }
	if req.Status != nil { d.Status = *req.Status }
	d.UpdatedAt = now; return nil
}
func (s *memStore) LogTrip(_ context.Context, t models.DriverTrip) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.trips = append(s.trips, t); return nil
}
func (s *memStore) ListTrips(_ context.Context, driverID uuid.UUID) ([]models.DriverTrip, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.DriverTrip
	for _, t := range s.trips { if t.DriverID == driverID { result = append(result, t) } }
	return result, nil
}
func (s *memStore) IncrementTripStats(_ context.Context, id uuid.UUID, km float64, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	d, ok := s.drivers[id]
	if !ok { return nil }
	d.TotalTrips++; d.TotalKm += km; d.UpdatedAt = now; return nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.DriverAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}
type notFoundError struct{}
func (e *notFoundError) Error() string { return "not found" }

func testEncryptor(t *testing.T) *crypto.Encryptor {
	t.Helper()
	key := make([]byte, 32)
	enc, err := crypto.NewEncryptor(key)
	if err != nil { t.Fatal(err) }
	return enc
}

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	enc := testEncryptor(t)
	h := handlers.NewDriverHandlerWithStore(store, enc)
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

func TestCreateDriver(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body := `{"full_name":"Kwame Mensah","phone":"+233501234567","national_id":"GHA-001-2020","license_no":"GH-DRV-001","license_class":"C","license_expiry":"2028-12-31","country":"GH","base_location":"Accra"}`
	resp, _ := http.Post(srv.URL+"/api/v1/drivers", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
}

func TestCreateDriver_MissingFields(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Post(srv.URL+"/api/v1/drivers", "application/json", bytes.NewBufferString(`{"full_name":"Only Name"}`))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetDriver_IncludesNationalID(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	enc := testEncryptor(t)
	id := uuid.New()
	nameEnc, _ := enc.EncryptString("Amina Diallo")
	phoneEnc, _ := enc.EncryptString("+221771234567")
	idEnc, _ := enc.EncryptString("SN-NAT-0042")
	store.CreateDriver(context.Background(), models.DriverRow{ID: id, FullNameEnc: nameEnc, PhoneEnc: phoneEnc, NationalIDEnc: idEnc, LicenseNo: "SN-001", LicenseClass: models.LicenseC, LicenseExpiry: time.Now().Add(365*24*time.Hour), Status: models.DriverAvailable, Country: "SN", Rating: 5.0, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/drivers/" + id.String())
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var raw map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&raw)
	data := raw["data"].(map[string]interface{})
	if data["national_id"] == "" { t.Fatal("national_id should be present on individual GET") }
}

func TestListDrivers_OmitsNationalID(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	enc := testEncryptor(t)
	id := uuid.New()
	nameEnc, _ := enc.EncryptString("John Doe")
	phoneEnc, _ := enc.EncryptString("+254711111111")
	idEnc, _ := enc.EncryptString("KE-ID-9999")
	store.CreateDriver(context.Background(), models.DriverRow{ID: id, FullNameEnc: nameEnc, PhoneEnc: phoneEnc, NationalIDEnc: idEnc, LicenseNo: "KE-001", LicenseClass: models.LicenseB, LicenseExpiry: time.Now().Add(365*24*time.Hour), Status: models.DriverAvailable, Country: "KE", Rating: 4.5, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	resp, _ := http.Get(srv.URL + "/api/v1/drivers")
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var raw map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&raw)
	items := raw["data"].([]interface{})
	if len(items) == 0 { t.Fatal("expected at least 1 driver") }
	d := items[0].(map[string]interface{})
	if d["national_id"] != nil && d["national_id"] != "" { t.Fatal("national_id must be omitted in list view") }
}

func TestUpdateDriver(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	enc := testEncryptor(t)
	id := uuid.New()
	nameEnc, _ := enc.EncryptString("Fatou Ba")
	phoneEnc, _ := enc.EncryptString("+221701234567")
	idEnc, _ := enc.EncryptString("SN-100")
	store.CreateDriver(context.Background(), models.DriverRow{ID: id, FullNameEnc: nameEnc, PhoneEnc: phoneEnc, NationalIDEnc: idEnc, LicenseNo: "SN-002", LicenseClass: models.LicenseC, LicenseExpiry: time.Now().Add(365*24*time.Hour), Status: models.DriverAvailable, Country: "SN", Rating: 5.0, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	status := models.DriverOnDuty
	body, _ := json.Marshal(models.UpdateDriverRequest{Status: &status})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/drivers/"+id.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestLogTrip(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	enc := testEncryptor(t)
	id := uuid.New()
	vid := uuid.New()
	nameEnc, _ := enc.EncryptString("Mamadou Coulibaly")
	phoneEnc, _ := enc.EncryptString("+225071234567")
	idEnc, _ := enc.EncryptString("CI-ID-0001")
	store.CreateDriver(context.Background(), models.DriverRow{ID: id, FullNameEnc: nameEnc, PhoneEnc: phoneEnc, NationalIDEnc: idEnc, LicenseNo: "CI-001", LicenseClass: models.LicenseC, LicenseExpiry: time.Now().Add(365*24*time.Hour), Status: models.DriverAvailable, Country: "CI", Rating: 5.0, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body, _ := json.Marshal(models.LogTripRequest{VehicleID: vid.String(), DistanceKm: 240.5})
	resp, _ := http.Post(srv.URL+"/api/v1/drivers/"+id.String()+"/trips", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 { t.Fatalf("expected 201, got %d", resp.StatusCode) }
}

func TestLogTrip_InvalidVehicleID(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	enc := testEncryptor(t)
	id := uuid.New()
	nameEnc, _ := enc.EncryptString("Test Driver")
	phoneEnc, _ := enc.EncryptString("+254700000000")
	idEnc, _ := enc.EncryptString("KE-0000")
	store.CreateDriver(context.Background(), models.DriverRow{ID: id, FullNameEnc: nameEnc, PhoneEnc: phoneEnc, NationalIDEnc: idEnc, LicenseNo: "KE-DRV-000", LicenseClass: models.LicenseB, LicenseExpiry: time.Now().Add(365*24*time.Hour), Status: models.DriverAvailable, Country: "KE", Rating: 5.0, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	body := `{"vehicle_id":"not-a-uuid","distance_km":100}`
	resp, _ := http.Post(srv.URL+"/api/v1/drivers/"+id.String()+"/trips", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 { t.Fatalf("expected 400, got %d", resp.StatusCode) }
}

func TestGetDriver_NotFound(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/drivers/" + uuid.New().String())
	if resp.StatusCode != 404 { t.Fatalf("expected 404, got %d", resp.StatusCode) }
}
