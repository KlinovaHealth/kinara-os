package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/klinova/kinara-os/appointment-service/auth"
	"github.com/klinova/kinara-os/appointment-service/handlers"
	"github.com/klinova/kinara-os/appointment-service/middleware"
	"github.com/klinova/kinara-os/appointment-service/models"
)

// ---------- mock store ----------

type mockStore struct {
	appointments map[uuid.UUID]*models.Appointment
	audits       []models.AuditEntry
	err          error
}

func newMockStore() *mockStore {
	return &mockStore{appointments: make(map[uuid.UUID]*models.Appointment)}
}

func (m *mockStore) CreateAppointment(_ context.Context, a models.Appointment) error {
	if m.err != nil {
		return m.err
	}
	cp := a
	m.appointments[a.ID] = &cp
	return nil
}

func (m *mockStore) GetAppointment(_ context.Context, id uuid.UUID) (*models.Appointment, error) {
	if m.err != nil {
		return nil, m.err
	}
	a, ok := m.appointments[id]
	if !ok {
		return nil, context.DeadlineExceeded // any non-nil error → 404
	}
	cp := *a
	return &cp, nil
}

func (m *mockStore) ListAppointments(_ context.Context, _, _ *uuid.UUID, _ string, _ int) ([]models.Appointment, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]models.Appointment, 0, len(m.appointments))
	for _, a := range m.appointments {
		out = append(out, *a)
	}
	return out, nil
}

func (m *mockStore) UpdateStatus(_ context.Context, id uuid.UUID, status models.AppointmentStatus, notes string, _ time.Time) error {
	if m.err != nil {
		return m.err
	}
	if a, ok := m.appointments[id]; ok {
		a.Status = status
		if notes != "" {
			a.Notes = notes
		}
	}
	return nil
}

func (m *mockStore) RescheduleAppointment(_ context.Context, id uuid.UUID, newTime time.Time, durationMin int, _ time.Time) error {
	if m.err != nil {
		return m.err
	}
	if a, ok := m.appointments[id]; ok {
		a.ScheduledAt = newTime
		a.DurationMin = durationMin
	}
	return nil
}

func (m *mockStore) CancelAppointment(_ context.Context, id uuid.UUID, reason, actorID string, _ time.Time) error {
	if m.err != nil {
		return m.err
	}
	if a, ok := m.appointments[id]; ok {
		a.Status = models.StatusCancelled
		a.Reason = reason
		a.CancelledBy = actorID
	}
	return nil
}

func (m *mockStore) CompleteAppointment(_ context.Context, id uuid.UUID, notes, actorID string, _ time.Time) error {
	if m.err != nil {
		return m.err
	}
	if a, ok := m.appointments[id]; ok {
		a.Status = models.StatusCompleted
		if notes != "" {
			a.Notes = notes
		}
		a.CompletedBy = actorID
	}
	return nil
}

func (m *mockStore) ListByClinic(_ context.Context, clinicID, _ string, _ *time.Time, _ *string, _ int) ([]models.Appointment, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []models.Appointment
	for _, a := range m.appointments {
		if a.ClinicID == clinicID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (m *mockStore) ListByPatient(_ context.Context, patientID uuid.UUID, _ string, _ int) ([]models.Appointment, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []models.Appointment
	for _, a := range m.appointments {
		if a.PatientID == patientID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (m *mockStore) InsertAudit(_ context.Context, entry models.AuditEntry) error {
	m.audits = append(m.audits, entry)
	return nil
}

func (m *mockStore) GetAuditHistory(_ context.Context, apptID uuid.UUID) ([]models.AuditEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []models.AuditEntry
	for _, e := range m.audits {
		if e.AppointmentID == apptID.String() {
			out = append(out, e)
		}
	}
	return out, nil
}

// ---------- helpers ----------

func newRouter(store handlers.Store) *mux.Router {
	r := mux.NewRouter()
	handlers.NewWithStore(store).Register(r)
	return r
}

func withClaims(r *http.Request, role string) *http.Request {
	claims := &auth.Claims{UserID: uuid.New(), Role: role, TenantID: uuid.Nil}
	ctx := middleware.SetClaims(r.Context(), claims)
	return r.WithContext(ctx)
}

func mustMarshal(t *testing.T, v interface{}) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

// seedAppointment creates an appointment in the mock store and returns its ID.
func seedAppointment(store *mockStore, patientID uuid.UUID, clinicID string) uuid.UUID {
	id := uuid.New()
	store.appointments[id] = &models.Appointment{
		ID:             id,
		AppointmentRef: "APT-" + strings.ToUpper(id.String()[:8]),
		PatientID:      patientID,
		DoctorID:       uuid.New(),
		ClinicID:       clinicID,
		ScheduledAt:    time.Now().UTC().Add(24 * time.Hour),
		DurationMin:    30,
		Type:           models.TypeConsultation,
		Status:         models.StatusScheduled,
		TenantID: uuid.Nil.String(),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	return id
}

// ---------- tests ----------

func TestCreateAppointment_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	body := mustMarshal(t, models.CreateAppointmentRequest{
		PatientID:   uuid.New(),
		DoctorID:    uuid.New(),
		ClinicID:    "CLINIC-1",
		ScheduledAt: time.Now().UTC().Add(time.Hour),
		DurationMin: 45,
		Type:        models.TypeConsultation,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/appointments", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "nurse")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	if resp["success"] != true {
		t.Error("expected success=true")
	}
	if len(store.appointments) != 1 {
		t.Errorf("expected 1 appointment in store, got %d", len(store.appointments))
	}
	if len(store.audits) != 1 || store.audits[0].Action != "created" {
		t.Error("expected audit entry with action=created")
	}
}

func TestCreateAppointment_MissingPatientID(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	body := mustMarshal(t, models.CreateAppointmentRequest{
		DoctorID: uuid.New(),
		// PatientID intentionally zero
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/appointments", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "admin")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateAppointment_DefaultDuration(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	body := mustMarshal(t, models.CreateAppointmentRequest{
		PatientID:   uuid.New(),
		DoctorID:    uuid.New(),
		ClinicID:    "C1",
		ScheduledAt: time.Now().UTC().Add(time.Hour),
		DurationMin: 0, // should default to 30
		Type:        models.TypeFollowUp,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/appointments", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "doctor")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	for _, a := range store.appointments {
		if a.DurationMin != 30 {
			t.Errorf("expected default duration 30, got %d", a.DurationMin)
		}
	}
}

func TestGetAppointment_NotFound(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/appointments/"+uuid.New().String(), nil)
	req = withClaims(req, "doctor")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetAppointment_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	id := seedAppointment(store, uuid.New(), "CLINIC-X")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/appointments/"+id.String(), nil)
	req = withClaims(req, "nurse")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	if resp["success"] != true {
		t.Error("expected success=true")
	}
}

func TestRescheduleAppointment_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	id := seedAppointment(store, uuid.New(), "CLINIC-Y")

	newTime := time.Now().UTC().Add(48 * time.Hour)
	body := mustMarshal(t, models.RescheduleRequest{
		ScheduledAt: newTime,
		DurationMin: 60,
		Reason:      "patient request",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/appointments/"+id.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "frontdesk")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.appointments[id].DurationMin != 60 {
		t.Errorf("expected duration 60, got %d", store.appointments[id].DurationMin)
	}
	if len(store.audits) != 1 || store.audits[0].Action != "rescheduled" {
		t.Error("expected audit entry with action=rescheduled")
	}
}

func TestCancelAppointment_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	patientID := uuid.New()
	id := seedAppointment(store, patientID, "CLINIC-Z")

	body := mustMarshal(t, models.CancelRequest{Reason: "doctor unavailable"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/appointments/"+id.String()+"/cancel", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "admin")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.appointments[id].Status != models.StatusCancelled {
		t.Errorf("expected status cancelled, got %s", store.appointments[id].Status)
	}
	if len(store.audits) != 1 || store.audits[0].Action != "cancelled" {
		t.Error("expected audit entry with action=cancelled")
	}
	if store.audits[0].OldStatus != string(models.StatusScheduled) {
		t.Errorf("expected old_status=scheduled, got %s", store.audits[0].OldStatus)
	}
	if store.audits[0].NewStatus != string(models.StatusCancelled) {
		t.Errorf("expected new_status=cancelled, got %s", store.audits[0].NewStatus)
	}
}

func TestCompleteAppointment_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	id := seedAppointment(store, uuid.New(), "CLINIC-A")

	body := mustMarshal(t, models.CompleteRequest{Notes: "all good"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/appointments/"+id.String()+"/complete", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "doctor")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.appointments[id].Status != models.StatusCompleted {
		t.Errorf("expected status completed, got %s", store.appointments[id].Status)
	}
	if len(store.audits) != 1 || store.audits[0].Action != "completed" {
		t.Error("expected audit entry with action=completed")
	}
}

func TestListByClinic_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	clinicID := "CLINIC-LIST"
	seedAppointment(store, uuid.New(), clinicID)
	seedAppointment(store, uuid.New(), clinicID)
	seedAppointment(store, uuid.New(), "OTHER-CLINIC")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/appointments/clinic/"+clinicID, nil)
	req = withClaims(req, "nurse")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array, got %T", resp["data"])
	}
	if len(data) != 2 {
		t.Errorf("expected 2 appointments for clinic, got %d", len(data))
	}
}

func TestListByPatient_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	patientID := uuid.New()
	seedAppointment(store, patientID, "C1")
	seedAppointment(store, patientID, "C2")
	seedAppointment(store, uuid.New(), "C3") // different patient

	req := httptest.NewRequest(http.MethodGet, "/api/v1/appointments/patient/"+patientID.String(), nil)
	req = withClaims(req, "doctor")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array, got %T", resp["data"])
	}
	if len(data) != 2 {
		t.Errorf("expected 2 appointments for patient, got %d", len(data))
	}
}

func TestAuditHistory_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	id := seedAppointment(store, uuid.New(), "CLINIC-H")

	// Seed two audit entries manually.
	store.audits = append(store.audits,
		models.AuditEntry{AppointmentID: id.String(), Action: "created", ActorID: "u1", NewStatus: "scheduled", OccurredAt: time.Now().UTC()},
		models.AuditEntry{AppointmentID: id.String(), Action: "rescheduled", ActorID: "u1", OccurredAt: time.Now().UTC()},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/appointments/"+id.String()+"/history", nil)
	req = withClaims(req, "admin")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array, got %T", resp["data"])
	}
	if len(data) != 2 {
		t.Errorf("expected 2 audit entries, got %d", len(data))
	}
}

func TestRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "APT-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "APT-") {
		t.Errorf("ref must start with APT-, got %s", ref)
	}
	// "APT-" (4) + 8 hex chars = 12
	if len(ref) != 12 {
		t.Errorf("ref length must be 12, got %d (%s)", len(ref), ref)
	}
}

func TestStatusConstants(t *testing.T) {
	statuses := []models.AppointmentStatus{
		models.StatusScheduled,
		models.StatusConfirmed,
		models.StatusCompleted,
		models.StatusCancelled,
		models.StatusNoShow,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("status constant must not be empty")
		}
	}
	if models.StatusScheduled != "scheduled" {
		t.Errorf("unexpected value for StatusScheduled: %s", models.StatusScheduled)
	}
	if models.StatusCancelled != "cancelled" {
		t.Errorf("unexpected value for StatusCancelled: %s", models.StatusCancelled)
	}
	if models.StatusCompleted != "completed" {
		t.Errorf("unexpected value for StatusCompleted: %s", models.StatusCompleted)
	}
}

func TestUnauthorized_NoToken(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	// No claims in context → handler should return 401.
	body := mustMarshal(t, models.CreateAppointmentRequest{PatientID: uuid.New(), DoctorID: uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/appointments", body)
	req.Header.Set("Content-Type", "application/json")
	// Deliberately NOT calling withClaims.

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestForbidden_WrongRole(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	// "patient" role is not allowed to create appointments.
	body := mustMarshal(t, models.CreateAppointmentRequest{PatientID: uuid.New(), DoctorID: uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/appointments", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "patient")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for patient role on create, got %d", rec.Code)
	}
}

func TestCompleteForbidden_FrontdeskRole(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	id := seedAppointment(store, uuid.New(), "CLINIC-F")

	body := mustMarshal(t, models.CompleteRequest{Notes: "done"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/appointments/"+id.String()+"/complete", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "frontdesk") // not allowed to complete

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for frontdesk on complete, got %d", rec.Code)
	}
}

func TestRescheduleDefaultDuration(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)
	id := seedAppointment(store, uuid.New(), "CLINIC-D")

	body := mustMarshal(t, models.RescheduleRequest{
		ScheduledAt: time.Now().UTC().Add(72 * time.Hour),
		DurationMin: 0, // should default to 30
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/appointments/"+id.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "nurse")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.appointments[id].DurationMin != 30 {
		t.Errorf("expected default duration 30 after reschedule, got %d", store.appointments[id].DurationMin)
	}
}
