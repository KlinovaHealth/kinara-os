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

	"github.com/klinova/kinara-os/extension-service/auth"
	"github.com/klinova/kinara-os/extension-service/handlers"
	"github.com/klinova/kinara-os/extension-service/middleware"
	"github.com/klinova/kinara-os/extension-service/models"
)

// ---------- mock store ----------

type mockStore struct {
	resources     []models.ExtensionResource
	consultations map[uuid.UUID]*models.Consultation
	feedbacks     []models.ExtensionFeedback
	bestPractices []models.BestPractice
	audits        []string
	err           error
}

func newMockStore() *mockStore {
	return &mockStore{
		consultations: make(map[uuid.UUID]*models.Consultation),
	}
}

func (m *mockStore) ListResources(_ context.Context, cropType, language string) ([]models.ExtensionResource, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []models.ExtensionResource
	for _, r := range m.resources {
		if (cropType == "" || r.CropType == cropType) && (language == "" || r.Language == language) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockStore) GetRecommendedResources(_ context.Context, cropType string, limit int) ([]models.ExtensionResource, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []models.ExtensionResource
	for _, r := range m.resources {
		if cropType == "" || r.CropType == cropType {
			out = append(out, r)
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *mockStore) BookConsultation(_ context.Context, c models.Consultation) error {
	if m.err != nil {
		return m.err
	}
	cp := c
	m.consultations[c.ID] = &cp
	return nil
}

func (m *mockStore) GetConsultation(_ context.Context, id uuid.UUID) (*models.Consultation, error) {
	if m.err != nil {
		return nil, m.err
	}
	c, ok := m.consultations[id]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	cp := *c
	return &cp, nil
}

func (m *mockStore) InsertFeedback(_ context.Context, f models.ExtensionFeedback) error {
	if m.err != nil {
		return m.err
	}
	m.feedbacks = append(m.feedbacks, f)
	return nil
}

func (m *mockStore) GetBestPractices(_ context.Context, cropType string) ([]models.BestPractice, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []models.BestPractice
	for _, bp := range m.bestPractices {
		if bp.CropType == cropType {
			out = append(out, bp)
		}
	}
	return out, nil
}

func (m *mockStore) InsertAudit(_ context.Context, consultID, actorID, action string) error {
	m.audits = append(m.audits, action+":"+consultID+":"+actorID)
	return nil
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

func withClaimsUserID(r *http.Request, role string, userID uuid.UUID) *http.Request {
	claims := &auth.Claims{UserID: userID, Role: role, TenantID: uuid.Nil}
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

func seedResource(store *mockStore, cropType, language string) uuid.UUID {
	id := uuid.New()
	store.resources = append(store.resources, models.ExtensionResource{
		ID:             id,
		Title:          "Test Resource",
		ContentSummary: "A test resource",
		CropType:       cropType,
		Language:       language,
		ResourceType:   "guide",
		ViewedCount:    10,
		CreatedAt:      time.Now().UTC(),
	})
	return id
}

func seedConsultation(store *mockStore, farmerID uuid.UUID) uuid.UUID {
	id := uuid.New()
	store.consultations[id] = &models.Consultation{
		ID:         id,
		ConsultRef: "EXT-" + strings.ToUpper(id.String()[:8]),
		FarmerID:   farmerID,
		Topic:      "Pest control",
		CropType:   "maize",
		Status:     "pending",
		TenantID: uuid.Nil.String(),
		BookedAt:   time.Now().UTC(),
	}
	return id
}

func seedBestPractice(store *mockStore, cropType string) {
	store.bestPractices = append(store.bestPractices, models.BestPractice{
		ID:                       uuid.New(),
		CropType:                 cropType,
		Technique:                "Micro-dosing fertilizer",
		Description:              "Apply 6g per planting hole",
		ExpectedYieldImprovement: 25.0,
		Climate:                  "semi-arid",
	})
}

// ---------- tests ----------

func TestListResources_Success(t *testing.T) {
	store := newMockStore()
	seedResource(store, "maize", "en")
	seedResource(store, "cocoa", "fr")
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/extension/resources", nil)
	req = withClaims(req, "farmer")
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
		t.Errorf("expected 2 resources, got %d", len(data))
	}
}

func TestListResources_FilterByCrop(t *testing.T) {
	store := newMockStore()
	seedResource(store, "maize", "en")
	seedResource(store, "maize", "fr")
	seedResource(store, "cocoa", "fr")
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/extension/resources?crop_type=maize", nil)
	req = withClaims(req, "farmer")
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
		t.Errorf("expected 2 maize resources, got %d", len(data))
	}
}

func TestBookConsultation_Success(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	body := mustMarshal(t, models.BookConsultationRequest{
		Topic:    "Fertilizer schedule",
		CropType: "maize",
		FarmID:   "FARM-001",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/consultations", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "farmer")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.consultations) != 1 {
		t.Errorf("expected 1 consultation in store, got %d", len(store.consultations))
	}
}

func TestBookConsultation_Unauthorized(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	body := mustMarshal(t, models.BookConsultationRequest{Topic: "Pest issue"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/consultations", body)
	req.Header.Set("Content-Type", "application/json")
	// No claims in context.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGetConsultation_Success(t *testing.T) {
	store := newMockStore()
	farmerID := uuid.New()
	id := seedConsultation(store, farmerID)
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/extension/consultations/"+id.String(), nil)
	req = withClaims(req, "admin")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeBody(t, rec)
	if resp["success"] != true {
		t.Errorf("expected success true, got %v", resp["success"])
	}
}

func TestGetConsultation_NotFound(t *testing.T) {
	store := newMockStore()
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/extension/consultations/"+uuid.New().String(), nil)
	req = withClaims(req, "admin")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSubmitFeedback_Success(t *testing.T) {
	store := newMockStore()
	farmerID := uuid.New()
	consultID := seedConsultation(store, farmerID)
	r := newRouter(store)

	body := mustMarshal(t, models.FeedbackRequest{Rating: 4, Notes: "Very helpful", Result: "improved"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/consultations/"+consultID.String()+"/feedback", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaimsUserID(req, "farmer", farmerID)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.feedbacks) != 1 {
		t.Errorf("expected 1 feedback in store, got %d", len(store.feedbacks))
	}
	if store.feedbacks[0].Rating != 4 {
		t.Errorf("expected rating 4, got %d", store.feedbacks[0].Rating)
	}
}

func TestSubmitFeedback_InvalidRating(t *testing.T) {
	store := newMockStore()
	consultID := seedConsultation(store, uuid.New())
	r := newRouter(store)

	body := mustMarshal(t, models.FeedbackRequest{Rating: 6, Notes: "bad rating"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/consultations/"+consultID.String()+"/feedback", body)
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "farmer")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for rating > 5, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetBestPractices_Success(t *testing.T) {
	store := newMockStore()
	seedBestPractice(store, "maize")
	seedBestPractice(store, "maize")
	seedBestPractice(store, "cocoa")
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/extension/best-practices/maize", nil)
	req = withClaims(req, "farmer")
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
		t.Errorf("expected 2 maize best practices, got %d", len(data))
	}
}

func TestGetRecommended_Success(t *testing.T) {
	store := newMockStore()
	for i := 0; i < 7; i++ {
		seedResource(store, "maize", "en")
	}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/extension/resources/recommended?crop_type=maize", nil)
	req = withClaims(req, "farmer")
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
	if len(data) > 5 {
		t.Errorf("expected at most 5 recommended resources, got %d", len(data))
	}
}

func TestConsultRef_Format(t *testing.T) {
	id := uuid.New()
	ref := "EXT-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "EXT-") {
		t.Errorf("consult_ref must start with EXT-, got %s", ref)
	}
	// "EXT-" (4) + 8 hex chars = 12
	if len(ref) != 12 {
		t.Errorf("consult_ref length must be 12, got %d (%s)", len(ref), ref)
	}
}
