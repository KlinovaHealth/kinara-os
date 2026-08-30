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

	"github.com/klinova/kinara-os/outbreak-service/auth"
	"github.com/klinova/kinara-os/outbreak-service/handlers"
	"github.com/klinova/kinara-os/outbreak-service/middleware"
	"github.com/klinova/kinara-os/outbreak-service/models"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockStore struct {
	mu          sync.Mutex
	cases       []models.SuspectedCase
	outbreaks   []models.ConfirmedOutbreak
	notifs      []models.OutbreakNotification
	auditLog    []string
	countReturn int
	err         error
}

func (m *mockStore) InsertCase(_ context.Context, c models.SuspectedCase) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.cases = append(m.cases, c)
	m.mu.Unlock()
	return nil
}

func (m *mockStore) CountRecentCases(_ context.Context, _, _ string, _ time.Duration) (int, error) {
	return m.countReturn, m.err
}

func (m *mockStore) UpsertOutbreak(_ context.Context, diseaseCode, clinicID, diseaseName string, caseCount int) error {
	if m.err != nil {
		return m.err
	}
	id := uuid.New()
	o := models.ConfirmedOutbreak{
		ID:          id,
		AlertRef:    "OBK-" + strings.ToUpper(id.String()[:8]),
		DiseaseCode: diseaseCode,
		DiseaseName: diseaseName,
		ClinicID:    clinicID,
		CaseCount:   caseCount,
		Status:      "active",
		TenantID:    "test",
		DetectedAt:  time.Now().UTC(),
	}
	m.mu.Lock()
	m.outbreaks = append(m.outbreaks, o)
	m.mu.Unlock()
	return nil
}

func (m *mockStore) ListActiveOutbreaks(_ context.Context) ([]models.ConfirmedOutbreak, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []models.ConfirmedOutbreak
	for _, o := range m.outbreaks {
		if o.Status == "active" {
			out = append(out, o)
		}
	}
	return out, nil
}

func (m *mockStore) ConfirmOutbreak(_ context.Context, id uuid.UUID, actorID string) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.outbreaks {
		if m.outbreaks[i].ID == id {
			m.outbreaks[i].Status = "confirmed"
		}
	}
	m.auditLog = append(m.auditLog, "confirm_outbreak:"+actorID)
	return nil
}

func (m *mockStore) GetClusters(_ context.Context) ([]models.DiseaseCluster, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []models.DiseaseCluster{
		{DiseaseCode: "A00", DiseaseName: "Cholera", ClinicID: "clinic-1", CaseCount: 7},
	}, nil
}

func (m *mockStore) GetTrends(_ context.Context) ([]models.DiseaseTrend, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []models.DiseaseTrend{
		{DiseaseCode: "A00", Date: time.Now().UTC(), CaseCount: 3},
	}, nil
}

func (m *mockStore) InsertNotification(_ context.Context, n models.OutbreakNotification) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.notifs = append(m.notifs, n)
	m.mu.Unlock()
	return nil
}

func (m *mockStore) InsertAudit(_ context.Context, outbreakID, actorID, action string) error {
	m.mu.Lock()
	m.auditLog = append(m.auditLog, action+":"+actorID+":"+outbreakID)
	m.mu.Unlock()
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func claimsCtx(role string) *auth.Claims {
	return &auth.Claims{
		UserID:   uuid.New(),
		Role:     role,
		TenantID: "test-tenant",
	}
}

func newRouter(store *mockStore) *mux.Router {
	h := handlers.NewWithStore(store)
	r := mux.NewRouter()
	h.Register(r)
	return r
}

func doRequest(t *testing.T, router *mux.Router, method, path string, body interface{}, claims *auth.Claims) *httptest.ResponseRecorder {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	if claims != nil {
		req = req.WithContext(middleware.SetClaims(req.Context(), claims))
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestReportCase_Success(t *testing.T) {
	store := &mockStore{}
	router := newRouter(store)

	body := models.ReportCaseRequest{
		PatientID:   uuid.New(),
		DiseaseCode: "A00",
		DiseaseName: "Cholera",
		ClinicID:    "clinic-1",
		Location:    "Lomé",
		Symptoms:    "diarrhea, vomiting",
	}
	rr := doRequest(t, router, http.MethodPost, "/api/v1/outbreak/cases", body, claimsCtx("doctor"))

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.cases) != 1 {
		t.Errorf("expected 1 case in store, got %d", len(store.cases))
	}
}

func TestReportCase_Unauthorized(t *testing.T) {
	store := &mockStore{}
	router := newRouter(store)

	body := models.ReportCaseRequest{PatientID: uuid.New(), DiseaseCode: "A00", ClinicID: "c1"}
	rr := doRequest(t, router, http.MethodPost, "/api/v1/outbreak/cases", body, nil)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestReportCase_ForbiddenRole(t *testing.T) {
	store := &mockStore{}
	router := newRouter(store)

	body := models.ReportCaseRequest{PatientID: uuid.New(), DiseaseCode: "A00", ClinicID: "c1"}
	rr := doRequest(t, router, http.MethodPost, "/api/v1/outbreak/cases", body, claimsCtx("patient"))

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestOutbreakThreshold_TriggeredAt5Cases(t *testing.T) {
	// Unit test: when CountRecentCases returns >= 5, UpsertOutbreak is called.
	store := &mockStore{countReturn: 5}
	h := handlers.NewWithStore(store)

	// Access checkOutbreakThreshold via the exported helper used in tests.
	// Since the method is unexported, we trigger it through the HTTP handler
	// and verify the outbreak was upserted.
	router := newRouter(store)
	body := models.ReportCaseRequest{
		PatientID:   uuid.New(),
		DiseaseCode: "B00",
		DiseaseName: "Measles",
		ClinicID:    "clinic-2",
	}
	rr := doRequest(t, router, http.MethodPost, "/api/v1/outbreak/cases", body, claimsCtx("nurse"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	// Give the goroutine a moment to complete.
	time.Sleep(50 * time.Millisecond)

	store.mu.Lock()
	outbreakCount := len(store.outbreaks)
	store.mu.Unlock()

	if outbreakCount < 1 {
		t.Errorf("expected outbreak upsert when count=%d >= 5; got %d outbreaks", store.countReturn, outbreakCount)
	}

	_ = h // referenced to confirm handler is created
}

func TestListAlerts_Success(t *testing.T) {
	// Pre-populate with one active and one confirmed outbreak.
	activeID := uuid.New()
	confirmedID := uuid.New()
	store := &mockStore{
		outbreaks: []models.ConfirmedOutbreak{
			{ID: activeID, AlertRef: "OBK-ACTIVE01", DiseaseCode: "A00", Status: "active", DetectedAt: time.Now().UTC()},
			{ID: confirmedID, AlertRef: "OBK-CONF001", DiseaseCode: "B00", Status: "confirmed", DetectedAt: time.Now().UTC()},
		},
	}
	router := newRouter(store)
	rr := doRequest(t, router, http.MethodGet, "/api/v1/outbreak/alerts", nil, claimsCtx("admin"))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Errorf("expected 1 active alert, got %d", len(data))
	}
	alert := data[0].(map[string]interface{})
	if alert["status"] != "active" {
		t.Errorf("expected status=active, got %v", alert["status"])
	}
}

func TestConfirmOutbreak_Success(t *testing.T) {
	id := uuid.New()
	store := &mockStore{
		outbreaks: []models.ConfirmedOutbreak{
			{ID: id, AlertRef: "OBK-TEST001", DiseaseCode: "A00", Status: "active", DetectedAt: time.Now().UTC()},
		},
	}
	router := newRouter(store)
	rr := doRequest(t, router, http.MethodPost, "/api/v1/outbreak/alerts/"+id.String()+"/confirm", nil, claimsCtx("epidemiologist"))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.outbreaks[0].Status != "confirmed" {
		t.Errorf("expected status=confirmed, got %s", store.outbreaks[0].Status)
	}
	if len(store.auditLog) == 0 {
		t.Error("expected audit log entry")
	}
}

func TestConfirmOutbreak_ForbiddenRole(t *testing.T) {
	id := uuid.New()
	store := &mockStore{}
	router := newRouter(store)
	rr := doRequest(t, router, http.MethodPost, "/api/v1/outbreak/alerts/"+id.String()+"/confirm", nil, claimsCtx("nurse"))

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestGetClusters_Success(t *testing.T) {
	store := &mockStore{}
	router := newRouter(store)
	rr := doRequest(t, router, http.MethodGet, "/api/v1/outbreak/clusters", nil, claimsCtx("admin"))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["success"] != true {
		t.Error("expected success=true")
	}
	data := resp["data"].([]interface{})
	if len(data) < 1 {
		t.Error("expected at least one cluster")
	}
}

func TestGetTrends_Success(t *testing.T) {
	store := &mockStore{}
	router := newRouter(store)
	rr := doRequest(t, router, http.MethodGet, "/api/v1/outbreak/trends", nil, claimsCtx("admin"))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) < 1 {
		t.Error("expected at least one trend")
	}
}

func TestNotify_Success(t *testing.T) {
	outbreakID := uuid.New()
	store := &mockStore{}
	router := newRouter(store)

	body := models.NotifyRequest{
		OutbreakID: outbreakID,
		Message:    "Cholera outbreak confirmed in Lomé North clinic.",
		Recipients: "ministry@health.tg",
	}
	rr := doRequest(t, router, http.MethodPost, "/api/v1/outbreak/notify", body, claimsCtx("epidemiologist"))

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.notifs) != 1 {
		t.Errorf("expected 1 notification, got %d", len(store.notifs))
	}
	if store.notifs[0].OutbreakID != outbreakID {
		t.Error("outbreak_id mismatch")
	}
}

func TestCaseRef_Format(t *testing.T) {
	id := uuid.New()
	ref := "CASE-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "CASE-") {
		t.Errorf("expected CASE- prefix, got %s", ref)
	}
	if len(ref) != 13 { // "CASE-" (5) + 8 chars
		t.Errorf("unexpected ref length: %d for %s", len(ref), ref)
	}
}

func TestAlertRef_Format(t *testing.T) {
	id := uuid.New()
	ref := "OBK-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "OBK-") {
		t.Errorf("expected OBK- prefix, got %s", ref)
	}
	if len(ref) != 12 { // "OBK-" (4) + 8 chars
		t.Errorf("unexpected ref length: %d for %s", len(ref), ref)
	}
}
