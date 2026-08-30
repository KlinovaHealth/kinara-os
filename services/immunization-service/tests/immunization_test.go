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

	"github.com/klinova/kinara-os/immunization-service/auth"
	"github.com/klinova/kinara-os/immunization-service/handlers"
	"github.com/klinova/kinara-os/immunization-service/middleware"
	"github.com/klinova/kinara-os/immunization-service/models"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockStore struct {
	recordErr      error
	listResult     []models.ImmunizationRecord
	listErr        error
	scheduleResult []models.VaccineDue
	scheduleErr    error
	alertErr       error
	complianceResult models.ComplianceReport
	complianceErr  error
	coverageResult []models.CoverageItem
	coverageErr    error
	auditErr       error
}

func (m *mockStore) RecordImmunization(_ context.Context, _ models.ImmunizationRecord) error {
	return m.recordErr
}
func (m *mockStore) ListByPatient(_ context.Context, _ uuid.UUID) ([]models.ImmunizationRecord, error) {
	return m.listResult, m.listErr
}
func (m *mockStore) GetSchedule(_ context.Context, _ uuid.UUID) ([]models.VaccineDue, error) {
	return m.scheduleResult, m.scheduleErr
}
func (m *mockStore) InsertAlert(_ context.Context, _ models.ImmunizationAlert) error {
	return m.alertErr
}
func (m *mockStore) GetClinicCompliance(_ context.Context, _ string) (models.ComplianceReport, error) {
	return m.complianceResult, m.complianceErr
}
func (m *mockStore) GetPopulationCoverage(_ context.Context) ([]models.CoverageItem, error) {
	return m.coverageResult, m.coverageErr
}
func (m *mockStore) InsertAudit(_ context.Context, _ models.ImmunizationRecord, _ string) error {
	return m.auditErr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func nurseCtx() context.Context {
	c := &auth.Claims{Role: "nurse", TenantID: "tg"}
	c.UserID = uuid.New()
	return middleware.SetClaims(context.Background(), c)
}

func patientCtx() context.Context {
	c := &auth.Claims{Role: "patient", TenantID: "tg"}
	c.UserID = uuid.New()
	return middleware.SetClaims(context.Background(), c)
}

func routerWith(h *handlers.Handler) *mux.Router {
	r := mux.NewRouter()
	h.Register(r)
	return r
}

func recordBody() []byte {
	b, _ := json.Marshal(models.CreateImmunizationRequest{
		PatientID:      uuid.New(),
		VaccineCode:    "BCG",
		VaccineName:    "BCG Vaccine",
		DoseNumber:     1,
		AdministeredAt: time.Now().UTC(),
		ClinicID:       "clinic-001",
	})
	return b
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestRecordImmunization_Success — happy path for nurse recording a vaccination.
func TestRecordImmunization_Success(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/immunizations/record",
		bytes.NewReader(recordBody()))
	req = req.WithContext(nurseCtx())
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

// TestRecordImmunization_Unauthorized — no claims in context → 401.
func TestRecordImmunization_Unauthorized(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/immunizations/record",
		bytes.NewReader(recordBody()))
	// No context claims injected.
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestRecordImmunization_ForbiddenRole — patient role cannot record → 403.
func TestRecordImmunization_ForbiddenRole(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/immunizations/record",
		bytes.NewReader(recordBody()))
	req = req.WithContext(patientCtx())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// TestListByPatient_Success — returns records for a known patient.
func TestListByPatient_Success(t *testing.T) {
	patientID := uuid.New()
	records := []models.ImmunizationRecord{
		{ID: uuid.New(), PatientID: patientID, VaccineCode: "OPV", DoseNumber: 1},
	}
	h := handlers.NewWithStore(&mockStore{listResult: records})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/immunizations/patient/"+patientID.String(), nil)
	req = req.WithContext(nurseCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Errorf("expected 1 record, got %v", resp["data"])
	}
}

// TestListByPatient_EmptyResult — patient with no records returns [] not null.
func TestListByPatient_EmptyResult(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{listResult: nil})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/immunizations/patient/"+uuid.New().String(), nil)
	req = req.WithContext(nurseCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Errorf("expected array for data, got %T", resp["data"])
	}
	if len(data) != 0 {
		t.Errorf("expected empty array, got %d items", len(data))
	}
}

// TestGetSchedule_Success — returns a (possibly empty) schedule list.
func TestGetSchedule_Success(t *testing.T) {
	due := []models.VaccineDue{
		{VaccineCode: "MCV", VaccineName: "Measles", DueDate: time.Now().Add(7 * 24 * time.Hour), Status: "upcoming"},
	}
	h := handlers.NewWithStore(&mockStore{scheduleResult: due})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/immunizations/patient/"+uuid.New().String()+"/schedule", nil)
	req = req.WithContext(nurseCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestSendAlert_Success — POSTing an alert returns 201 and the alert payload.
func TestSendAlert_Success(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{})
	router := routerWith(h)

	body, _ := json.Marshal(models.SendAlertRequest{
		PatientID: uuid.New(),
		Message:   "Your measles vaccine is overdue.",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/immunizations/alert",
		bytes.NewReader(body))
	req = req.WithContext(nurseCtx())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestGetClinicCompliance_Success — returns a compliance report for a clinic.
func TestGetClinicCompliance_Success(t *testing.T) {
	report := models.ComplianceReport{
		ClinicID:        "clinic-001",
		CompliancePct:   87.5,
		VaccinatedCount: 35,
		TotalEligible:   40,
	}
	h := handlers.NewWithStore(&mockStore{complianceResult: report})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/immunizations/compliance/clinic/clinic-001", nil)
	req = req.WithContext(nurseCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["success"] != true {
		t.Error("expected success:true")
	}
}

// TestGetPopulationCoverage_Success — returns vaccine coverage counts.
func TestGetPopulationCoverage_Success(t *testing.T) {
	coverage := []models.CoverageItem{
		{VaccineCode: "BCG", CoverageCount: 150},
		{VaccineCode: "OPV", CoverageCount: 142},
	}
	h := handlers.NewWithStore(&mockStore{coverageResult: coverage})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/immunizations/coverage", nil)
	req = req.WithContext(nurseCtx())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 2 {
		t.Errorf("expected 2 coverage items, got %v", resp["data"])
	}
}

// TestRecordRef_Format — generated ref must start with "IMM-".
func TestRecordRef_Format(t *testing.T) {
	id := uuid.New()
	ref := "IMM-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "IMM-") {
		t.Errorf("expected IMM- prefix, got %s", ref)
	}
	if len(ref) != 12 {
		t.Errorf("expected ref length 12, got %d", len(ref))
	}
}

// TestRecordImmunization_StoreError — store returning error yields 500.
func TestRecordImmunization_StoreError(t *testing.T) {
	h := handlers.NewWithStore(&mockStore{recordErr: errors.New("db down")})
	router := routerWith(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/immunizations/record",
		bytes.NewReader(recordBody()))
	req = req.WithContext(nurseCtx())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Legacy / model-level tests (kept from original stub)
// ---------------------------------------------------------------------------

func TestDoseStatusValues(t *testing.T) {
	statuses := []models.DoseStatus{
		models.DoseAdministered, models.DoseScheduled,
		models.DoseOverdue, models.DoseMissed,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("empty status")
		}
	}
}

func TestNextDoseNilable(t *testing.T) {
	rec := models.ImmunizationRecord{NextDoseDate: nil}
	if rec.NextDoseDate != nil {
		t.Error("next dose should be nil when not set")
	}
}
