package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/klinova/kinara-os/input-service/auth"
	"github.com/klinova/kinara-os/input-service/handlers"
	"github.com/klinova/kinara-os/input-service/middleware"
	"github.com/klinova/kinara-os/input-service/models"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockStore struct {
	mu          sync.Mutex
	forms       map[string]*models.Form
	submissions map[uuid.UUID]*models.FormSubmission
	auditLog    []string
	err         error
}

func newMockStore() *mockStore {
	schema := json.RawMessage(`{"required":["full_name","date_of_birth","sex"],"properties":{"full_name":{"type":"string"},"date_of_birth":{"type":"string","format":"date"},"sex":{"type":"string","enum":["M","F"]}}}`)
	return &mockStore{
		forms: map[string]*models.Form{
			"patient-intake": {
				ID:        uuid.New(),
				FormType:  "patient-intake",
				Title:     "Patient Intake Form",
				Schema:    schema,
				Version:   1,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
		submissions: make(map[uuid.UUID]*models.FormSubmission),
	}
}

func (m *mockStore) GetForm(_ context.Context, formType string) (*models.Form, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.forms[formType]
	if !ok {
		return nil, errors.New("not found")
	}
	return f, nil
}

func (m *mockStore) CreateSubmission(_ context.Context, s models.FormSubmission) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.submissions[s.ID] = &s
	m.mu.Unlock()
	return nil
}

func (m *mockStore) GetSubmission(_ context.Context, id uuid.UUID) (*models.FormSubmission, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.submissions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}

func (m *mockStore) ListByPatient(_ context.Context, patientID uuid.UUID) ([]models.FormSubmission, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []models.FormSubmission
	for _, s := range m.submissions {
		if s.PatientID == patientID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *mockStore) UpdateSubmission(_ context.Context, id uuid.UUID, data []byte, updatedAt time.Time) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.submissions[id]
	if !ok {
		return errors.New("not found")
	}
	s.Data = data
	s.UpdatedAt = updatedAt
	return nil
}

func (m *mockStore) InsertAudit(_ context.Context, submissionID, actorID, action string) error {
	m.mu.Lock()
	m.auditLog = append(m.auditLog, action+":"+actorID+":"+submissionID)
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
		TenantID: uuid.Nil,
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

func TestGetForm_PatientIntake(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	rr := doRequest(t, router, http.MethodGet, "/api/v1/input/forms/patient-intake", nil, claimsCtx("nurse"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	if data["form_type"] != "patient-intake" {
		t.Errorf("expected form_type=patient-intake, got %v", data["form_type"])
	}
	if data["schema"] == nil {
		t.Error("expected schema to be present")
	}
}

func TestGetForm_UnknownType(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	rr := doRequest(t, router, http.MethodGet, "/api/v1/input/forms/nonexistent-form", nil, claimsCtx("admin"))
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSubmitForm_Success(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	body := models.SubmitRequest{
		PatientID: uuid.New(),
		FormType:  "patient-intake",
		Data:      json.RawMessage(`{"full_name":"Amara Kone","date_of_birth":"1990-05-15","sex":"M"}`),
	}
	rr := doRequest(t, router, http.MethodPost, "/api/v1/input/submissions", body, claimsCtx("nurse"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.submissions) != 1 {
		t.Errorf("expected 1 submission in store, got %d", len(store.submissions))
	}
}

func TestSubmitForm_Unauthorized(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	body := models.SubmitRequest{PatientID: uuid.New(), FormType: "patient-intake"}
	rr := doRequest(t, router, http.MethodPost, "/api/v1/input/submissions", body, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestSubmitForm_MissingPatientID(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	body := models.SubmitRequest{
		// PatientID left as zero value (uuid.Nil)
		FormType: "patient-intake",
		Data:     json.RawMessage(`{}`),
	}
	rr := doRequest(t, router, http.MethodPost, "/api/v1/input/submissions", body, claimsCtx("doctor"))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetSubmission_Success(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	// Create a submission directly in the mock store.
	id := uuid.New()
	now := time.Now().UTC()
	store.submissions[id] = &models.FormSubmission{
		ID:            id,
		SubmissionRef: "SUB-" + strings.ToUpper(id.String()[:8]),
		PatientID:     uuid.New(),
		FormType:      "patient-intake",
		FormVersion:   1,
		Data:          json.RawMessage(`{"full_name":"Koffi Adu"}`),
		SubmittedBy:   uuid.New(),
		TenantID: uuid.Nil,
		SubmittedAt:   now,
		UpdatedAt:     now,
	}

	rr := doRequest(t, router, http.MethodGet, "/api/v1/input/submissions/"+id.String(), nil, claimsCtx("admin"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	if data["id"] != id.String() {
		t.Errorf("id mismatch: got %v", data["id"])
	}
}

func TestGetSubmission_NotFound(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	rr := doRequest(t, router, http.MethodGet, "/api/v1/input/submissions/"+uuid.New().String(), nil, claimsCtx("nurse"))
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestListByPatient_Success(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	patientID := uuid.New()
	otherID := uuid.New()
	now := time.Now().UTC()

	for i, pid := range []uuid.UUID{patientID, patientID, otherID} {
		id := uuid.New()
		store.submissions[id] = &models.FormSubmission{
			ID:          id,
			PatientID:   pid,
			FormType:    "patient-intake",
			FormVersion: 1,
			Data:        json.RawMessage(`{}`),
			SubmittedBy: uuid.New(),
			TenantID: uuid.Nil,
			SubmittedAt: now.Add(time.Duration(i) * time.Minute),
			UpdatedAt:   now,
		}
	}

	rr := doRequest(t, router, http.MethodGet, "/api/v1/input/submissions/patient/"+patientID.String(), nil, claimsCtx("nurse"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("expected 2 submissions for patient, got %d", len(data))
	}
}

func TestUpdateSubmission_Success(t *testing.T) {
	store := newMockStore()
	router := newRouter(store)

	id := uuid.New()
	now := time.Now().UTC()
	store.submissions[id] = &models.FormSubmission{
		ID:          id,
		PatientID:   uuid.New(),
		FormType:    "patient-intake",
		FormVersion: 1,
		Data:        json.RawMessage(`{"full_name":"Old Name"}`),
		SubmittedBy: uuid.New(),
		TenantID: uuid.Nil,
		SubmittedAt: now,
		UpdatedAt:   now,
	}

	body := models.UpdateRequest{
		Data: json.RawMessage(`{"full_name":"New Name","date_of_birth":"1985-03-10","sex":"F"}`),
	}
	rr := doRequest(t, router, http.MethodPut, "/api/v1/input/submissions/"+id.String(), body, claimsCtx("doctor"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Give the audit goroutine a moment.
	time.Sleep(20 * time.Millisecond)

	store.mu.Lock()
	defer store.mu.Unlock()
	updated := store.submissions[id]
	if string(updated.Data) != `{"full_name":"New Name","date_of_birth":"1985-03-10","sex":"F"}` {
		t.Errorf("data not updated: %s", updated.Data)
	}
}

func TestSubmissionRef_Format(t *testing.T) {
	id := uuid.New()
	ref := "SUB-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "SUB-") {
		t.Errorf("expected SUB- prefix, got %s", ref)
	}
	if len(ref) != 12 { // "SUB-" (4) + 8 chars
		t.Errorf("unexpected ref length: %d for %s", len(ref), ref)
	}
}

func TestFormSchema_HasRequired(t *testing.T) {
	store := newMockStore()

	form, err := store.GetForm(context.Background(), "patient-intake")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(form.Schema, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	required, ok := schema["required"].([]interface{})
	if !ok || len(required) == 0 {
		t.Error("expected non-empty 'required' array in schema")
	}
	// Verify the three mandatory fields.
	requiredSet := make(map[string]bool)
	for _, f := range required {
		requiredSet[f.(string)] = true
	}
	for _, field := range []string{"full_name", "date_of_birth", "sex"} {
		if !requiredSet[field] {
			t.Errorf("expected %q in required fields", field)
		}
	}
}
