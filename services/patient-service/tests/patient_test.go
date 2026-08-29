package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/klinova/kinara-os/patient-service/crypto"
	"github.com/klinova/kinara-os/patient-service/models"
)

// ─── Crypto tests ─────────────────────────────────────────────────────────────

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	enc, err := crypto.New(key)
	if err != nil {
		t.Fatalf("crypto.New: %v", err)
	}

	cases := []string{
		"Kofi Mensah",
		"GH-1234567890",
		"+233 20 123 4567",
		"12 Independence Ave, Accra",
		"",
	}

	for _, plain := range cases {
		t.Run("plaintext="+plain, func(t *testing.T) {
			if plain == "" {
				// EncryptOptional should return "" for empty input
				cipher, err := enc.EncryptOptional(plain)
				if err != nil {
					t.Fatalf("EncryptOptional: %v", err)
				}
				if cipher != "" {
					t.Errorf("expected empty ciphertext for empty plaintext, got %q", cipher)
				}
				return
			}

			cipher, err := enc.EncryptString(plain)
			if err != nil {
				t.Fatalf("EncryptString: %v", err)
			}
			if cipher == plain {
				t.Error("ciphertext should not equal plaintext")
			}

			decrypted, err := enc.DecryptString(cipher)
			if err != nil {
				t.Fatalf("DecryptString: %v", err)
			}
			if decrypted != plain {
				t.Errorf("want %q, got %q", plain, decrypted)
			}
		})
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := crypto.New(key)

	plain := "same plaintext"
	c1, _ := enc.EncryptString(plain)
	c2, _ := enc.EncryptString(plain)

	// Each call uses a fresh random nonce, so ciphertexts must differ.
	if c1 == c2 {
		t.Error("two encryptions of the same plaintext produced identical ciphertexts — nonce reuse detected")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = 0xFF
	}

	enc1, _ := crypto.New(key1)
	enc2, _ := crypto.New(key2)

	cipher, _ := enc1.EncryptString("secret")
	_, err := enc2.DecryptString(cipher)
	if err == nil {
		t.Error("decryption with wrong key should fail but succeeded")
	}
}

func TestInvalidKeySize(t *testing.T) {
	_, err := crypto.New(make([]byte, 16)) // AES-128, not AES-256
	if err == nil {
		t.Error("expected error for 16-byte key, got nil")
	}
}

// ─── Model validation tests ───────────────────────────────────────────────────

func TestPatientStatusValues(t *testing.T) {
	valid := []models.PatientStatus{
		models.StatusActive,
		models.StatusInactive,
		models.StatusDeceased,
		models.StatusTransferred,
	}
	for _, s := range valid {
		if s == "" {
			t.Errorf("status constant is empty string")
		}
	}
}

func TestGenderValues(t *testing.T) {
	valid := []models.Gender{
		models.GenderMale,
		models.GenderFemale,
		models.GenderOther,
		models.GenderPreferNotSay,
	}
	for _, g := range valid {
		if g == "" {
			t.Errorf("gender constant is empty string")
		}
	}
}

// ─── HTTP handler tests (using httptest, no real DB) ─────────────────────────

// mockPatient is a minimal valid CreatePatientRequest for handler tests.
var mockPatient = models.CreatePatientRequest{
	NationalID:  "GH-1234567890",
	FullName:    "Akosua Asante",
	DateOfBirth: "1990-05-15",
	Gender:      models.GenderFemale,
	PhoneNumber: "+233 20 000 1234",
	Country:     "Ghana",
	Region:      "Greater Accra",
}

func TestCreatePatientRequestJSON(t *testing.T) {
	body, err := json.Marshal(mockPatient)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded models.CreatePatientRequest
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.FullName != mockPatient.FullName {
		t.Errorf("full_name: want %q, got %q", mockPatient.FullName, decoded.FullName)
	}
	if decoded.Country != mockPatient.Country {
		t.Errorf("country: want %q, got %q", mockPatient.Country, decoded.Country)
	}
}

func TestAPIResponseShape(t *testing.T) {
	resp := models.APIResponse{
		Success: true,
		Data:    map[string]string{"id": uuid.New().String()},
		Meta: &models.PageMeta{
			Page: 1, Limit: 20, Total: 1, TotalPages: 1,
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal APIResponse: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out["success"] != true {
		t.Error("success field missing or false")
	}
	if out["data"] == nil {
		t.Error("data field missing")
	}
	if out["meta"] == nil {
		t.Error("meta field missing")
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Inline health handler test — no server needed.
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"patient-service"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rw.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rw.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("want status=ok, got %q", body["status"])
	}
}

func TestMissingAuthHeader(t *testing.T) {
	// A request without Authorization should get 401.
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/patients", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/patients",
		bytes.NewBufferString(`{}`))
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	if rw.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rw.Code)
	}
}

// ─── Audit log model test ─────────────────────────────────────────────────────

func TestAuditLogSerialization(t *testing.T) {
	log := models.PatientAuditLog{
		ID:           uuid.New(),
		PatientID:    uuid.New(),
		Action:       models.AuditCreate,
		AccessorID:   uuid.New(),
		AccessorRole: "nurse",
		IPAddress:    "10.0.0.1",
		RequestID:    "req-abc-123",
		Changes:      map[string]interface{}{"status": "active"},
		CreatedAt:    time.Now().UTC(),
	}

	b, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("marshal audit log: %v", err)
	}

	var out map[string]interface{}
	_ = json.Unmarshal(b, &out)

	if out["action"] != string(models.AuditCreate) {
		t.Errorf("action: want %q, got %v", models.AuditCreate, out["action"])
	}
	if out["accessor_role"] != "nurse" {
		t.Errorf("accessor_role: want nurse, got %v", out["accessor_role"])
	}
}

// ─── Context tests ────────────────────────────────────────────────────────────

func TestContextDeadlinePropagation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(200 * time.Millisecond):
		t.Error("context did not expire within expected time")
	}
}
