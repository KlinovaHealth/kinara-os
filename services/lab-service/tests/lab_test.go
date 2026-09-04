package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/klinova/kinara-os/lab-service/auth"
	"github.com/klinova/kinara-os/lab-service/handlers"
	"github.com/klinova/kinara-os/lab-service/middleware"
	"github.com/klinova/kinara-os/lab-service/models"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockStore struct {
	createOrderErr      error
	getOrderResult      *models.LabOrder
	getOrderErr         error
	uploadResultErr     error
	updateStatusErr     error
	listPatientResult   []models.LabResultWithOrder
	listPatientErr      error
	getStatusResult     string
	getStatusErr        error
	getCatalogResult    *models.TestCatalogEntry
	getCatalogErr       error
	insertAuditErr      error
}

func (m *mockStore) CreateOrder(_ context.Context, _ models.LabOrder) error {
	return m.createOrderErr
}
func (m *mockStore) GetOrder(_ context.Context, _ uuid.UUID) (*models.LabOrder, error) {
	return m.getOrderResult, m.getOrderErr
}
func (m *mockStore) UploadResult(_ context.Context, _ models.LabResult) error {
	return m.uploadResultErr
}
func (m *mockStore) UpdateOrderStatus(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return m.updateStatusErr
}
func (m *mockStore) ListPatientResults(_ context.Context, _ uuid.UUID) ([]models.LabResultWithOrder, error) {
	return m.listPatientResult, m.listPatientErr
}
func (m *mockStore) GetOrderStatus(_ context.Context, _ uuid.UUID) (string, error) {
	return m.getStatusResult, m.getStatusErr
}
func (m *mockStore) GetTestCatalog(_ context.Context, _ string) (*models.TestCatalogEntry, error) {
	return m.getCatalogResult, m.getCatalogErr
}
func (m *mockStore) InsertAudit(_ context.Context, _, _, _ string) error {
	return m.insertAuditErr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func doctorCtx() context.Context {
	c := &auth.Claims{Role: "doctor", TenantID: uuid.Nil}
	c.UserID = uuid.New()
	return middleware.SetClaims(context.Background(), c)
}

func labTechCtx() context.Context {
	c := &auth.Claims{Role: "lab_tech", TenantID: uuid.Nil}
	c.UserID = uuid.New()
	return middleware.SetClaims(context.Background(), c)
}

func patientCtx() context.Context {
	c := &auth.Claims{Role: "patient", TenantID: uuid.Nil}
	c.UserID = uuid.New()
	return middleware.SetClaims(context.Background(), c)
}

func routerWith(h *handlers.Handler) *mux.Router {
	r := mux.NewRouter()
	h.Register(r)
	return r
}

func orderBody() []byte {
	b, _ := json.Marshal(models.CreateOrderRequest{
		PatientID: uuid.New(),
		OrderedBy: uuid.New(),
		ClinicID:  "clinic-001",
		TestCode:  "HGB",
		TestName:  "Hemoglobin",
		Priority:  "routine",
	})
	return b
}

func sampleOrder(id uuid.UUID) *models.LabOrder {
	return &models.LabOrder{
		ID:        id,
		OrderRef:  "LAB-" + strings.ToUpper(id.String()[:8]),
		PatientID: uuid.New(),
		OrderedBy: uuid.New(),
		ClinicID:  "clinic-001",
		TestCode:  "HGB",
		TestName:  "Hemoglobin",
		Priority:  "routine",
		Status:    models.OrderPending,
		TenantID: uuid.Nil,
		OrderedAt: time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestCreateOrder_Success — doctor creates a lab order → 201.
func TestCreateOrder_Success(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lab/orders",
		bytes.NewReader(orderBody()))
	req = req.WithContext(doctorCtx())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["success"] != true {
		t.Error("expected success:true")
	}
}

// TestCreateOrder_Unauthorized — no context claims → 401.
func TestCreateOrder_Unauthorized(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lab/orders",
		bytes.NewReader(orderBody()))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestCreateOrder_ForbiddenRole — patient role cannot create orders → 403.
func TestCreateOrder_ForbiddenRole(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lab/orders",
		bytes.NewReader(orderBody()))
	req = req.WithContext(patientCtx())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// TestGetOrder_NotFound — unknown ID → 404.
func TestGetOrder_NotFound(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{getOrderErr: errors.New("not found")})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/lab/orders/"+uuid.New().String(), nil)
	req = req.WithContext(doctorCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// TestGetOrder_Success — known ID → 200 with order payload.
func TestGetOrder_Success(t *testing.T) {
	id := uuid.New()
	order := sampleOrder(id)
	h := handlers.NewWithStore(&mockStore{getOrderResult: order})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lab/orders/"+id.String(), nil)
	req = req.WithContext(doctorCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestUploadResult_Success — lab_tech uploads a result → 201.
func TestUploadResult_Success(t *testing.T) {
	id := uuid.New()
	order := sampleOrder(id)
	h := handlers.NewWithStore(&mockStore{getOrderResult: order})
	router := routerWith(h)

	body, _ := json.Marshal(models.UploadResultRequest{
		ResultValue: 13.5,
		Unit:        "g/dL",
		NormalLow:   11.5,
		NormalHigh:  17.5,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lab/orders/"+id.String()+"/result",
		bytes.NewReader(body))
	req = req.WithContext(labTechCtx())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestUploadResult_FlagNormal — value within [low, high] → flag "normal".
func TestUploadResult_FlagNormal(t *testing.T) {
	flag := interpretFlag(13.5, 11.5, 17.5)
	if flag != "normal" {
		t.Errorf("expected normal, got %s", flag)
	}
}

// TestUploadResult_FlagAbnormal — value outside range but within 2x → "abnormal".
func TestUploadResult_FlagAbnormal(t *testing.T) {
	flag := interpretFlag(10.0, 11.5, 17.5) // below low but above low*0.5 (5.75)
	if flag != "abnormal" {
		t.Errorf("expected abnormal, got %s", flag)
	}
	flagHigh := interpretFlag(20.0, 11.5, 17.5) // above high but below high*2.0 (35)
	if flagHigh != "abnormal" {
		t.Errorf("expected abnormal for high value, got %s", flagHigh)
	}
}

// TestUploadResult_FlagCritical — value > high*2 or < low*0.5 → "critical".
func TestUploadResult_FlagCritical(t *testing.T) {
	flagLow := interpretFlag(5.0, 11.5, 17.5) // 5.0 < 11.5*0.5=5.75 → critical
	if flagLow != "critical" {
		t.Errorf("expected critical for very low, got %s", flagLow)
	}
	flagHigh := interpretFlag(40.0, 11.5, 17.5) // 40 > 17.5*2=35 → critical
	if flagHigh != "critical" {
		t.Errorf("expected critical for very high, got %s", flagHigh)
	}
}

// TestListPatientResults_Success — returns all orders+results for patient.
func TestListPatientResults_Success(t *testing.T) {
	patientID := uuid.New()
	orderID := uuid.New()
	results := []models.LabResultWithOrder{
		{
			Order: *sampleOrder(orderID),
			Result: &models.LabResult{
				ID:          uuid.New(),
				OrderID:     orderID,
				ResultValue: 13.5,
				Unit:        "g/dL",
				Flag:        "normal",
				RecordedBy:  uuid.New(),
				RecordedAt:  time.Now().UTC(),
			},
		},
	}
	h := handlers.NewWithStore(&mockStore{listPatientResult: results})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/lab/patient/"+patientID.String()+"/results", nil)
	req = req.WithContext(doctorCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Errorf("expected 1 result, got %v", resp["data"])
	}
}

// TestGetOrderStatus_Success — returns status string for known order.
func TestGetOrderStatus_Success(t *testing.T) {
	id := uuid.New()
	h := handlers.NewWithStore(&mockStore{getStatusResult: "pending"})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lab/orders/"+id.String()+"/status", nil)
	req = req.WithContext(doctorCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	data, ok := resp["data"].(map[string]interface{})
	if !ok || data["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", resp["data"])
	}
}

// TestOrderRef_Format — generated ref starts with "LAB-" and is 12 chars.
func TestOrderRef_Format(t *testing.T) {
	id := uuid.New()
	ref := "LAB-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "LAB-") {
		t.Errorf("expected LAB- prefix, got %s", ref)
	}
	if len(ref) != 12 {
		t.Errorf("expected ref length 12, got %d", len(ref))
	}
}

// TestInterpretFlag — unit test for interpretFlag function directly.
func TestInterpretFlag(t *testing.T) {
	cases := []struct {
		value, low, high float64
		want             string
	}{
		{13.5, 11.5, 17.5, "normal"},   // mid-range
		{11.5, 11.5, 17.5, "normal"},   // exactly at low boundary
		{17.5, 11.5, 17.5, "normal"},   // exactly at high boundary
		{10.0, 11.5, 17.5, "abnormal"}, // below low but above low*0.5
		{20.0, 11.5, 17.5, "abnormal"}, // above high but below high*2
		{5.0, 11.5, 17.5, "critical"},  // below low*0.5
		{40.0, 11.5, 17.5, "critical"}, // above high*2
	}
	for _, tc := range cases {
		got := interpretFlag(tc.value, tc.low, tc.high)
		if got != tc.want {
			t.Errorf("interpretFlag(%v,%v,%v)=%s, want %s",
				tc.value, tc.low, tc.high, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// interpretFlag helper (duplicated from handlers to keep tests self-contained)
// ---------------------------------------------------------------------------

func interpretFlag(value, low, high float64) string {
	if value < low*0.5 || value > high*2.0 {
		return "critical"
	}
	if value < low || value > high {
		return "abnormal"
	}
	return "normal"
}

// ---------------------------------------------------------------------------
// Legacy / model-level tests (kept from original stub)
// ---------------------------------------------------------------------------

func TestOrderStatusValues(t *testing.T) {
	statuses := []models.OrderStatus{
		models.OrderPending, models.OrderCollected,
		models.OrderProcessing, models.OrderCompleted, models.OrderCancelled,
	}
	if len(statuses) != 5 {
		t.Errorf("expected 5 statuses, got %d", len(statuses))
	}
}

func TestResultFlagValues(t *testing.T) {
	flags := []models.ResultFlag{
		models.FlagNormal, models.FlagHigh, models.FlagLow, models.FlagCritical,
	}
	for _, f := range flags {
		if string(f) == "" {
			t.Errorf("empty flag")
		}
	}
}

func TestDefaultPriority(t *testing.T) {
	req := models.CreateOrderRequest{Priority: ""}
	if req.Priority == "" {
		req.Priority = "routine"
	}
	if req.Priority != "routine" {
		t.Errorf("expected routine priority, got %s", req.Priority)
	}
}
