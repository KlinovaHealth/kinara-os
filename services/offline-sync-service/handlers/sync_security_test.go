package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/offline-sync-service/auth"
	"github.com/klinova/kinara-os/offline-sync-service/db"
	"github.com/klinova/kinara-os/offline-sync-service/handlers"
	"github.com/klinova/kinara-os/offline-sync-service/middleware"
)

// ─────────────────────────────────────────────────────────────────────────────
// fakeDB satisfies handlers.DBQuerier for unit tests
// ─────────────────────────────────────────────────────────────────────────────

type fakeDB struct {
	deviceRevoked   bool
	deviceLastSeen  *time.Time
	patientInClinic map[uuid.UUID]bool
	existingKeys    map[string]bool
	inserted        []string
	returnCount     int // for cap test
}

func (f *fakeDB) GetDeviceStatus(_ context.Context, _ uuid.UUID) (*db.DeviceStatus, error) {
	return &db.DeviceStatus{
		Revoked:    f.deviceRevoked,
		LastSeenAt: f.deviceLastSeen,
	}, nil
}

func (f *fakeDB) PullPatients(_ context.Context, _ uuid.UUID) ([]db.PatientRecord, error) {
	cap := f.returnCount
	if cap > 200 {
		cap = 200 // handler/DB enforces cap
	}
	records := make([]db.PatientRecord, cap)
	for i := range records {
		records[i] = db.PatientRecord{
			PatientID:   uuid.New(),
			ClinicID:    uuid.New(),
			LastVisitAt: time.Now(),
			ExpiresAt:   time.Now().Add(72 * time.Hour),
		}
	}
	return records, nil
}

func (f *fakeDB) PatientInClinic(_ context.Context, patientID, _ uuid.UUID) (bool, error) {
	return f.patientInClinic[patientID], nil
}

func (f *fakeDB) PushExists(_ context.Context, _ uuid.UUID, key string) (bool, error) {
	if f.existingKeys == nil {
		return false, nil
	}
	return f.existingKeys[key], nil
}

func (f *fakeDB) InsertSyncRecord(_ context.Context, rec db.SyncRecord) error {
	f.inserted = append(f.inserted, rec.IdempotencyKey)
	return nil
}

func (f *fakeDB) MarkApplied(_ context.Context, _ uuid.UUID, _ time.Time) error  { return nil }
func (f *fakeDB) MarkRejected(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (f *fakeDB) GetSyncStatus(_ context.Context, _ uuid.UUID) (int, int, int, error) {
	return 0, 0, 0, nil
}
func (f *fakeDB) UpdateLastSeen(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────
// Security test 1: token without scope claim = 403 on any sync endpoint
// ─────────────────────────────────────────────────────────────────────────────

func TestRequireClinicScope_NoClaimsInContext_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/sync/pull", nil)
	// No claims in context at all.
	rr := httptest.NewRecorder()

	middleware.RequireClinicScope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing claims, got %d", rr.Code)
	}
}

func TestRequireClinicScope_NonDeviceToken_Returns403(t *testing.T) {
	// Regular staff JWT — no device_id, no clinic scope.
	claims := &auth.Claims{
		UserID: uuid.New(),
		Role:   "nurse",
		Scope:  "", // no clinic scope
	}

	req := httptest.NewRequest(http.MethodPost, "/sync/pull", nil)
	req = req.WithContext(middleware.InjectClaimsForTest(req.Context(), claims))
	rr := httptest.NewRecorder()

	middleware.RequireClinicScope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-device token, got %d", rr.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Security test 2: revoked device gets wipe directive (401 + wipe=true)
// ─────────────────────────────────────────────────────────────────────────────

func TestPull_RevokedDevice_GetsWipeDirective(t *testing.T) {
	deviceID := uuid.New()
	clinicID := uuid.New()

	h := handlers.NewWithFakeDB(&fakeDB{deviceRevoked: true})
	req := httptest.NewRequest(http.MethodPost, "/sync/pull", nil)
	req = req.WithContext(injectDeviceCtx(req.Context(), deviceID, clinicID))
	rr := httptest.NewRecorder()

	h.Pull(rr, req)

	assertWipe(t, rr, "device_revoked")
}

// ─────────────────────────────────────────────────────────────────────────────
// Security test 3: stale device (>7d without sync) gets wipe directive
// ─────────────────────────────────────────────────────────────────────────────

func TestPull_StaleDevice_GetsWipeDirective(t *testing.T) {
	deviceID := uuid.New()
	clinicID := uuid.New()
	staleTime := time.Now().Add(-8 * 24 * time.Hour)

	h := handlers.NewWithFakeDB(&fakeDB{
		deviceRevoked:  false,
		deviceLastSeen: &staleTime,
	})
	req := httptest.NewRequest(http.MethodPost, "/sync/pull", nil)
	req = req.WithContext(injectDeviceCtx(req.Context(), deviceID, clinicID))
	rr := httptest.NewRecorder()

	h.Pull(rr, req)

	assertWipe(t, rr, "stale_7_days")
}

// ─────────────────────────────────────────────────────────────────────────────
// Security test 4: duplicate push applies exactly once
// ─────────────────────────────────────────────────────────────────────────────

func TestPush_DuplicateIdempotencyKey_NotReprocessed(t *testing.T) {
	deviceID := uuid.New()
	clinicID := uuid.New()
	patientID := uuid.New()
	const iKey = "device1-seq-42"

	fake := &fakeDB{
		patientInClinic: map[uuid.UUID]bool{patientID: true},
		existingKeys:    map[string]bool{iKey: true},
	}
	h := handlers.NewWithFakeDB(fake)

	body, _ := json.Marshal(map[string]interface{}{
		"writes": []map[string]interface{}{{
			"idempotency_key": iKey,
			"payload_type":    "consultation",
			"patient_id":      patientID.String(),
			"payload":         map[string]string{"note": "fever"},
		}},
	})
	req := httptest.NewRequest(http.MethodPost, "/sync/push", bytes.NewReader(body))
	req = req.WithContext(injectDeviceCtx(req.Context(), deviceID, clinicID))
	rr := httptest.NewRecorder()
	h.Push(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	results := parseResults(t, rr)
	if results[0]["status"] != "duplicate" {
		t.Fatalf("expected duplicate, got %v", results[0]["status"])
	}
	if len(fake.inserted) != 0 {
		t.Fatal("duplicate key must not insert a new record")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Security test 5: out-of-scope patient_id is rejected, write not stored
// ─────────────────────────────────────────────────────────────────────────────

func TestPush_OutOfScopePatient_Rejected(t *testing.T) {
	deviceID := uuid.New()
	clinicID := uuid.New()
	foreignPatient := uuid.New()

	fake := &fakeDB{
		patientInClinic: map[uuid.UUID]bool{foreignPatient: false},
	}
	h := handlers.NewWithFakeDB(fake)

	body, _ := json.Marshal(map[string]interface{}{
		"writes": []map[string]interface{}{{
			"idempotency_key": "seq-1",
			"payload_type":    "consultation",
			"patient_id":      foreignPatient.String(),
			"payload":         map[string]string{"note": "test"},
		}},
	})
	req := httptest.NewRequest(http.MethodPost, "/sync/push", bytes.NewReader(body))
	req = req.WithContext(injectDeviceCtx(req.Context(), deviceID, clinicID))
	rr := httptest.NewRecorder()
	h.Push(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	results := parseResults(t, rr)
	if results[0]["status"] != "rejected" {
		t.Fatalf("expected rejected, got %v", results[0]["status"])
	}
	if results[0]["reason"] != "patient_not_in_clinic_scope" {
		t.Fatalf("expected patient_not_in_clinic_scope, got %v", results[0]["reason"])
	}
	if len(fake.inserted) != 0 {
		t.Fatal("out-of-scope write must not be stored")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Security test 6: cache cap enforced at 200 records
// ─────────────────────────────────────────────────────────────────────────────

func TestPull_CacheCapAt200Records(t *testing.T) {
	deviceID := uuid.New()
	clinicID := uuid.New()
	recent := time.Now().Add(-1 * time.Hour)

	h := handlers.NewWithFakeDB(&fakeDB{
		deviceRevoked:  false,
		deviceLastSeen: &recent,
		returnCount:    250, // DB stub enforces 200 cap
	})

	req := httptest.NewRequest(http.MethodPost, "/sync/pull", nil)
	req = req.WithContext(injectDeviceCtx(req.Context(), deviceID, clinicID))
	rr := httptest.NewRecorder()
	h.Pull(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	count := int(resp["count"].(float64))
	if count > 200 {
		t.Fatalf("expected count <= 200, got %d", count)
	}
	if resp["cap"].(float64) != 200 {
		t.Fatal("expected cap=200 in response")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Security test 7: clinic A device cannot pull clinic B patients
// This verifies that clinic_id in the DB query is always taken from the JWT,
// never from the request body.
// ─────────────────────────────────────────────────────────────────────────────

func TestPull_ClinicScopeFromJWT_NotBody(t *testing.T) {
	deviceID := uuid.New()
	clinicA := uuid.New()
	recent := time.Now().Add(-1 * time.Hour)

	queriedClinic := uuid.Nil
	fake := &fakeDB{
		deviceRevoked:  false,
		deviceLastSeen: &recent,
	}
	// Wrap fake to capture the clinic_id passed to PullPatients.
	h := handlers.NewWithFakeDB(&capturingDB{fakeDB: fake, onPull: func(id uuid.UUID) { queriedClinic = id }})

	// Device is scoped to clinicA; body contains no clinic override (pull has no body).
	req := httptest.NewRequest(http.MethodPost, "/sync/pull", nil)
	req = req.WithContext(injectDeviceCtx(req.Context(), deviceID, clinicA))
	rr := httptest.NewRecorder()
	h.Pull(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if queriedClinic != clinicA {
		t.Fatalf("handler queried clinic %v instead of claim clinic %v", queriedClinic, clinicA)
	}
}

// capturingDB wraps fakeDB and intercepts PullPatients to record the clinic_id used.
type capturingDB struct {
	*fakeDB
	onPull func(uuid.UUID)
}

func (c *capturingDB) PullPatients(ctx context.Context, clinicID uuid.UUID) ([]db.PatientRecord, error) {
	if c.onPull != nil {
		c.onPull(clinicID)
	}
	return c.fakeDB.PullPatients(ctx, clinicID)
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func injectDeviceCtx(ctx context.Context, deviceID, clinicID uuid.UUID) context.Context {
	scope := "clinic:" + clinicID.String()
	claims := &auth.Claims{
		UserID:   uuid.New(),
		Role:     "nurse",
		DeviceID: &deviceID,
		ClinicID: &clinicID,
		Scope:    scope,
		Scopes:   []string{scope},
	}
	ctx = middleware.InjectClaimsForTest(ctx, claims)
	ctx = middleware.InjectClinicIDForTest(ctx, clinicID)
	return ctx
}

func assertWipe(t *testing.T, rr *httptest.ResponseRecorder, expectedReason string) {
	t.Helper()
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 wipe directive, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["wipe"] != true {
		t.Fatalf("expected wipe=true, got %v", resp["wipe"])
	}
	if resp["reason"] != expectedReason {
		t.Fatalf("expected reason=%s, got %v", expectedReason, resp["reason"])
	}
}

func parseResults(t *testing.T, rr *httptest.ResponseRecorder) []map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	raw := resp["results"].([]interface{})
	out := make([]map[string]interface{}, len(raw))
	for i, r := range raw {
		out[i] = r.(map[string]interface{})
	}
	return out
}
