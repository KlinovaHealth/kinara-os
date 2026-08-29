package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/klinova/kinara-os/clinical-service/crypto"
	"github.com/klinova/kinara-os/clinical-service/models"
)

// ─── Crypto tests ─────────────────────────────────────────────────────────────

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := "Patient chief complaint: fever and headache for 3 days"
	ciphertext, err := enc.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	got, err := enc.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if got != plaintext {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := crypto.NewEncryptor(key)

	plaintext := "Malaria diagnosis"
	c1, _ := enc.EncryptString(plaintext)
	c2, _ := enc.EncryptString(plaintext)

	if c1 == c2 {
		t.Error("two encryptions of same plaintext produced identical ciphertext — nonce not random")
	}
}

func TestWrongKeyFails(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	enc1, _ := crypto.NewEncryptor(key1)
	enc2, _ := crypto.NewEncryptor(key2)

	ct, _ := enc1.EncryptString("ICD10: A09")
	_, err := enc2.DecryptString(ct)
	if err == nil {
		t.Error("expected decryption with wrong key to fail")
	}
}

func TestInvalidKeySize(t *testing.T) {
	_, err := crypto.NewEncryptor([]byte("tooshort"))
	if err != crypto.ErrInvalidKeySize {
		t.Errorf("expected ErrInvalidKeySize, got %v", err)
	}
}

func TestEncryptOptionalNilOnEmpty(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := crypto.NewEncryptor(key)

	result, err := enc.EncryptOptional("")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", result)
	}
}

// ─── Model constant tests ─────────────────────────────────────────────────────

func TestConsultationStatusConstants(t *testing.T) {
	cases := []models.ConsultationStatus{
		models.ConsultScheduled,
		models.ConsultInProgress,
		models.ConsultCompleted,
		models.ConsultCancelled,
		models.ConsultNoShow,
	}
	for _, c := range cases {
		if c == "" {
			t.Errorf("empty consultation status constant")
		}
	}
}

func TestSeverityConstants(t *testing.T) {
	cases := []models.Severity{
		models.SeverityMild,
		models.SeverityModerate,
		models.SeveritySevere,
		models.SeverityCritical,
	}
	for _, s := range cases {
		if s == "" {
			t.Errorf("empty severity constant")
		}
	}
}

func TestPrescriptionStatusConstants(t *testing.T) {
	cases := []models.PrescriptionStatus{
		models.PrescriptionPending,
		models.PrescriptionSent,
		models.PrescriptionDispensed,
		models.PrescriptionCancelled,
	}
	for _, s := range cases {
		if s == "" {
			t.Errorf("empty prescription status constant")
		}
	}
}

// ─── Request serialization tests ──────────────────────────────────────────────

func TestCreateConsultationRequestJSON(t *testing.T) {
	scheduled := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	req := models.CreateConsultationRequest{
		ConsultationType: models.TypeInPerson,
		ChiefComplaint:   "abdominal pain",
		ScheduledAt:      &scheduled,
		Country:          "Nigeria",
		Region:           "Lagos",
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var back models.CreateConsultationRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if back.ChiefComplaint != req.ChiefComplaint {
		t.Errorf("chief_complaint mismatch: got %q", back.ChiefComplaint)
	}
	if back.Country != req.Country {
		t.Errorf("country mismatch: got %q", back.Country)
	}
}

func TestMedicationSerialization(t *testing.T) {
	meds := []models.Medication{
		{Name: "Amoxicillin", Dosage: "500mg", Frequency: "3x daily", Duration: "7 days", Route: "oral"},
		{Name: "Paracetamol", Dosage: "1g", Frequency: "4x daily", Duration: "5 days", Route: "oral"},
	}

	b, err := json.Marshal(meds)
	if err != nil {
		t.Fatal(err)
	}

	var back []models.Medication
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if len(back) != len(meds) {
		t.Errorf("expected %d medications, got %d", len(meds), len(back))
	}
	if back[0].Name != "Amoxicillin" {
		t.Errorf("medication name mismatch: %q", back[0].Name)
	}
}

// ─── API response shape tests ─────────────────────────────────────────────────

func TestAPIResponseSuccessShape(t *testing.T) {
	resp := models.APIResponse{
		Success: true,
		Data:    map[string]string{"id": "abc"},
		Meta:    &models.PageMeta{Page: 1, Limit: 20, Total: 100, TotalPages: 5},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	json.Unmarshal(b, &m)

	if m["success"] != true {
		t.Error("expected success=true")
	}
	if m["data"] == nil {
		t.Error("expected data field")
	}
	if m["meta"] == nil {
		t.Error("expected meta field")
	}
}

func TestAPIResponseErrorShape(t *testing.T) {
	resp := models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "NOT_FOUND", Message: "consultation not found"},
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
		t.Errorf("unexpected error code: %v", errObj["code"])
	}
}

// ─── HTTP handler tests ───────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"clinical-service"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "clinical-service" {
		t.Errorf("unexpected service name: %q", body["service"])
	}
}

func TestMissingAuthHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED"}}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/consultations", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuditLogSerialization(t *testing.T) {
	log := models.ClinicalAuditLog{
		ResourceType: "consultation",
		Action:       models.AuditCreate,
		AccessorRole: "doctor",
		IPAddress:    "10.0.0.1",
		RequestID:    "req-123",
		CreatedAt:    time.Now(),
	}

	b, err := json.Marshal(log)
	if err != nil {
		t.Fatal(err)
	}

	var back models.ClinicalAuditLog
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if back.Action != models.AuditCreate {
		t.Errorf("unexpected action: %q", back.Action)
	}
	if back.ResourceType != "consultation" {
		t.Errorf("unexpected resource type: %q", back.ResourceType)
	}
}

func TestContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(5 * time.Millisecond)

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
		}
	default:
		t.Error("context should have expired")
	}
}

func TestCreatePrescriptionRequestJSON(t *testing.T) {
	req := models.CreatePrescriptionRequest{
		Medications: []models.Medication{
			{Name: "Metformin", Dosage: "500mg", Frequency: "2x daily", Duration: "30 days", Route: "oral"},
		},
		Notes: "Take with food",
	}

	body, _ := json.Marshal(req)
	reqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/consultations/x/prescriptions",
		bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")

	var decoded models.CreatePrescriptionRequest
	if err := json.NewDecoder(reqHTTP.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if len(decoded.Medications) != 1 {
		t.Errorf("expected 1 medication, got %d", len(decoded.Medications))
	}
	if decoded.Medications[0].Name != "Metformin" {
		t.Errorf("unexpected medication name: %q", decoded.Medications[0].Name)
	}
	if decoded.Notes != "Take with food" {
		t.Errorf("unexpected notes: %q", decoded.Notes)
	}
}
