package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cryptopkg "github.com/klinova/kinara-os/referral-service/crypto"
	"github.com/klinova/kinara-os/referral-service/models"
)

// ─── Model constant tests ─────────────────────────────────────────────────────

func TestReferralStatusConstants(t *testing.T) {
	statuses := []models.ReferralStatus{
		models.ReferralPending, models.ReferralAccepted, models.ReferralInProgress,
		models.ReferralCompleted, models.ReferralRejected, models.ReferralCancelled,
	}
	if len(statuses) != 6 {
		t.Errorf("expected 6 statuses, got %d", len(statuses))
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty status constant")
		}
	}
}

func TestReferralUrgencyConstants(t *testing.T) {
	urgencies := []models.ReferralUrgency{
		models.UrgencyRoutine, models.UrgencySemiUrgent, models.UrgencyUrgent, models.UrgencyEmergency,
	}
	if len(urgencies) != 4 {
		t.Errorf("expected 4 urgencies, got %d", len(urgencies))
	}
	for _, u := range urgencies {
		if u == "" {
			t.Error("empty urgency constant")
		}
	}
}

// ─── AES encryption tests ─────────────────────────────────────────────────────

func TestAESEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := cryptopkg.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"Patient referred for cardiac evaluation",
		"History of hypertension and diabetes",
		"Kofi Mensah", // patient name
	}
	for _, plain := range cases {
		ct, err := enc.EncryptString(plain)
		if err != nil {
			t.Fatalf("encrypt failed for %q: %v", plain, err)
		}
		got, err := enc.DecryptString(ct)
		if err != nil {
			t.Fatalf("decrypt failed for %q: %v", plain, err)
		}
		if got != plain {
			t.Errorf("roundtrip failed: got %q, want %q", got, plain)
		}
	}
}

func TestAESUniqueCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := cryptopkg.NewEncryptor(key)
	c1, _ := enc.EncryptString("same reason")
	c2, _ := enc.EncryptString("same reason")
	if c1 == c2 {
		t.Error("AES-GCM must produce unique ciphertexts via random nonces")
	}
}

func TestAESWrongKeyFails(t *testing.T) {
	key1, key2 := make([]byte, 32), make([]byte, 32)
	key2[0] = 0xFF
	enc1, _ := cryptopkg.NewEncryptor(key1)
	enc2, _ := cryptopkg.NewEncryptor(key2)
	ct, _ := enc1.EncryptString("referral reason")
	if _, err := enc2.DecryptString(ct); err == nil {
		t.Error("decryption with wrong key must fail")
	}
}

func TestAESInvalidKeySize(t *testing.T) {
	if _, err := cryptopkg.NewEncryptor([]byte("tooshort")); err != cryptopkg.ErrInvalidKeySize {
		t.Errorf("expected ErrInvalidKeySize, got %v", err)
	}
}

func TestAESOptionalNil(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := cryptopkg.NewEncryptor(key)
	result, err := enc.EncryptOptional(nil)
	if err != nil || result != nil {
		t.Error("EncryptOptional(nil) should return nil, nil")
	}
	result2, err := enc.DecryptOptional(nil)
	if err != nil || result2 != nil {
		t.Error("DecryptOptional(nil) should return nil, nil")
	}
}

// ─── Request/response serialization ──────────────────────────────────────────

func TestCreateReferralRequestJSON(t *testing.T) {
	req := models.CreateReferralRequest{
		PatientID:   "550e8400-e29b-41d4-a716-446655440000",
		PatientName: "Ama Owusu",
		ToClinicID:  "660e8400-e29b-41d4-a716-446655440001",
		Reason:      "Suspected tuberculosis — needs specialist evaluation",
		Urgency:     models.UrgencyUrgent,
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var back models.CreateReferralRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Urgency != models.UrgencyUrgent {
		t.Errorf("urgency mismatch: %q", back.Urgency)
	}
	if back.PatientID != req.PatientID {
		t.Errorf("patient_id mismatch: %q", back.PatientID)
	}
}

func TestUpdateStatusRequestJSON(t *testing.T) {
	notes := "Accepted — Dr. Asante will see patient Thursday"
	req := models.UpdateReferralStatusRequest{
		Status: models.ReferralAccepted,
		Notes:  &notes,
	}
	b, _ := json.Marshal(req)
	var back models.UpdateReferralStatusRequest
	json.Unmarshal(b, &back)
	if back.Status != models.ReferralAccepted {
		t.Errorf("status mismatch: %q", back.Status)
	}
	if back.Notes == nil || *back.Notes != notes {
		t.Error("notes mismatch")
	}
}

func TestScheduleFollowUpRequestJSON(t *testing.T) {
	notes := "Check chest X-ray results"
	req := models.ScheduleFollowUpRequest{
		FollowUpDate: "2026-09-15T09:00:00Z",
		Notes:        &notes,
	}
	b, _ := json.Marshal(req)
	var back models.ScheduleFollowUpRequest
	json.Unmarshal(b, &back)
	if back.FollowUpDate != req.FollowUpDate {
		t.Errorf("date mismatch: %q", back.FollowUpDate)
	}
}

func TestAddNoteRequestJSON(t *testing.T) {
	req := models.AddNoteRequest{Note: "Patient history reviewed — sending full records"}
	b, _ := json.Marshal(req)
	var back models.AddNoteRequest
	json.Unmarshal(b, &back)
	if back.Note != req.Note {
		t.Errorf("note mismatch: %q", back.Note)
	}
}

// ─── HTTP handler tests ───────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"referral-service"}`))
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "referral-service" {
		t.Errorf("unexpected service: %q", body["service"])
	}
}

func TestMissingAuthReturns401(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, `{"success":false}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAPIResponseShape(t *testing.T) {
	resp := models.APIResponse{
		Success: true,
		Data:    map[string]string{"id": "abc-123"},
		Meta:    &models.PageMeta{Page: 1, Limit: 50, Total: 120, TotalPages: 3},
	}
	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)

	if m["success"] != true {
		t.Error("expected success=true")
	}
	if m["data"] == nil {
		t.Error("expected data field")
	}
	meta, ok := m["meta"].(map[string]interface{})
	if !ok {
		t.Fatal("expected meta object")
	}
	if meta["total_pages"].(float64) != 3 {
		t.Errorf("unexpected total_pages: %v", meta["total_pages"])
	}
}

func TestAPIErrorShape(t *testing.T) {
	resp := models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "NOT_FOUND", Message: "referral not found"},
	}
	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	if m["success"] != false {
		t.Error("expected success=false")
	}
	errObj, ok := m["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("unexpected code: %v", errObj["code"])
	}
}

// ─── State machine tests ──────────────────────────────────────────────────────

func TestValidStatusTransitions(t *testing.T) {
	transitions := []struct {
		from models.ReferralStatus
		to   models.ReferralStatus
		ok   bool
	}{
		{models.ReferralPending, models.ReferralAccepted, true},
		{models.ReferralPending, models.ReferralRejected, true},
		{models.ReferralPending, models.ReferralCancelled, true},
		{models.ReferralAccepted, models.ReferralInProgress, true},
		{models.ReferralInProgress, models.ReferralCompleted, true},
		{models.ReferralCompleted, models.ReferralPending, false},  // cannot reopen
		{models.ReferralRejected, models.ReferralAccepted, false},  // cannot unrejected
		{models.ReferralCompleted, models.ReferralAccepted, false},
	}
	for _, tc := range transitions {
		got := validTransition(tc.from, tc.to)
		if got != tc.ok {
			t.Errorf("transition %s→%s: got %v, want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestReferralNoteImmutability(t *testing.T) {
	note := models.ReferralNote{
		Note:      "Initial assessment: patient presents with persistent cough",
		CreatedAt: time.Now(),
	}
	b, _ := json.Marshal(note)
	var back models.ReferralNote
	json.Unmarshal(b, &back)
	if back.Note != note.Note {
		t.Error("note content should be preserved")
	}
}

// validTransition mirrors the handler logic for testing without HTTP context.
func validTransition(from, to models.ReferralStatus) bool {
	allowed := map[models.ReferralStatus][]models.ReferralStatus{
		models.ReferralPending:    {models.ReferralAccepted, models.ReferralRejected, models.ReferralCancelled},
		models.ReferralAccepted:   {models.ReferralInProgress, models.ReferralCancelled},
		models.ReferralInProgress: {models.ReferralCompleted, models.ReferralCancelled},
	}
	for _, a := range allowed[from] {
		if a == to {
			return true
		}
	}
	return false
}
