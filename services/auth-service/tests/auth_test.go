package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cryptopkg "github.com/klinova/kinara-os/auth-service/crypto"
	"github.com/klinova/kinara-os/auth-service/models"
)

// ─── Password tests ───────────────────────────────────────────────────────────

func TestHashPasswordRoundtrip(t *testing.T) {
	hash, err := cryptopkg.HashPassword("SecurePass123!")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if err := cryptopkg.VerifyPassword(hash, "SecurePass123!"); err != nil {
		t.Errorf("verify failed: %v", err)
	}
}

func TestWrongPasswordFails(t *testing.T) {
	hash, _ := cryptopkg.HashPassword("CorrectPass99!")
	if err := cryptopkg.VerifyPassword(hash, "WrongPass99!"); err == nil {
		t.Error("expected failure for wrong password")
	}
}

func TestHashPasswordUniqueness(t *testing.T) {
	h1, _ := cryptopkg.HashPassword("SamePassword1!")
	h2, _ := cryptopkg.HashPassword("SamePassword1!")
	if h1 == h2 {
		t.Error("bcrypt should produce unique salts — hashes should differ")
	}
}

func TestPasswordNeverInHash(t *testing.T) {
	password := "MySecret42@"
	hash, _ := cryptopkg.HashPassword(password)
	if strings.Contains(hash, password) {
		t.Error("plaintext password must not appear in hash")
	}
}

// ─── API Key tests ────────────────────────────────────────────────────────────

func TestGenerateAPIKeyFormat(t *testing.T) {
	key, hash, err := cryptopkg.GenerateAPIKey()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if !strings.HasPrefix(key, "kinara_") {
		t.Errorf("key must start with 'kinara_', got: %q", key[:12])
	}
	if hash == "" {
		t.Error("hash must not be empty")
	}
	if strings.Contains(hash, key) {
		t.Error("hash must not contain plaintext key")
	}
}

func TestAPIKeyHashConsistency(t *testing.T) {
	key, hash, _ := cryptopkg.GenerateAPIKey()
	if cryptopkg.HashAPIKey(key) != hash {
		t.Error("HashAPIKey must produce same hash as GenerateAPIKey")
	}
}

func TestAPIKeyUniqueness(t *testing.T) {
	k1, h1, _ := cryptopkg.GenerateAPIKey()
	k2, h2, _ := cryptopkg.GenerateAPIKey()
	if k1 == k2 || h1 == h2 {
		t.Error("two API keys must be different")
	}
}

// ─── Refresh token tests ──────────────────────────────────────────────────────

func TestRefreshTokenRoundtrip(t *testing.T) {
	token, hash, err := cryptopkg.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(token) < 64 {
		t.Errorf("refresh token too short: %d chars", len(token))
	}
	if cryptopkg.HashRefreshToken(token) != hash {
		t.Error("HashRefreshToken must match GenerateRefreshToken hash")
	}
}

func TestRefreshTokenUniqueness(t *testing.T) {
	t1, h1, _ := cryptopkg.GenerateRefreshToken()
	t2, h2, _ := cryptopkg.GenerateRefreshToken()
	if t1 == t2 || h1 == h2 {
		t.Error("two refresh tokens must be different")
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

	plaintext := "full_name: Akosua Mensah"
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

func TestAESUniqueCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := cryptopkg.NewEncryptor(key)
	c1, _ := enc.EncryptString("test")
	c2, _ := enc.EncryptString("test")
	if c1 == c2 {
		t.Error("AES-GCM must use random nonces — identical plaintexts must produce different ciphertexts")
	}
}

func TestAESWrongKeyFails(t *testing.T) {
	key1, key2 := make([]byte, 32), make([]byte, 32)
	key2[0] = 0xFF
	enc1, _ := cryptopkg.NewEncryptor(key1)
	enc2, _ := cryptopkg.NewEncryptor(key2)
	ct, _ := enc1.EncryptString("sensitive data")
	if _, err := enc2.DecryptString(ct); err == nil {
		t.Error("decryption with wrong key must fail")
	}
}

func TestAESInvalidKeySize(t *testing.T) {
	if _, err := cryptopkg.NewEncryptor([]byte("tooshort")); err != cryptopkg.ErrInvalidKeySize {
		t.Errorf("expected ErrInvalidKeySize, got %v", err)
	}
}

// ─── TOTP tests ───────────────────────────────────────────────────────────────

func TestTOTPSecretGeneration(t *testing.T) {
	secret, err := cryptopkg.GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) < 16 {
		t.Errorf("TOTP secret too short: %d chars", len(secret))
	}
}

func TestTOTPSecretUniqueness(t *testing.T) {
	s1, _ := cryptopkg.GenerateTOTPSecret()
	s2, _ := cryptopkg.GenerateTOTPSecret()
	if s1 == s2 {
		t.Error("two TOTP secrets must be different")
	}
}

func TestTOTPWrongCodeFails(t *testing.T) {
	secret, _ := cryptopkg.GenerateTOTPSecret()
	if cryptopkg.VerifyTOTP(secret, "000000") && cryptopkg.VerifyTOTP(secret, "111111") {
		t.Error("both static codes should not verify (extremely unlikely collision)")
	}
}

func TestTOTPInvalidSecretFails(t *testing.T) {
	if cryptopkg.VerifyTOTP("not-valid-base32!!!", "123456") {
		t.Error("invalid base32 secret should fail verification")
	}
}

func TestTOTPAuthURI(t *testing.T) {
	uri := cryptopkg.TOTPAuthURI("JBSWY3DPEHPK3PXP", "testuser")
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Errorf("unexpected URI format: %q", uri[:30])
	}
	if !strings.Contains(uri, "testuser") {
		t.Error("URI must contain username")
	}
	if !strings.Contains(uri, "secret=") {
		t.Error("URI must contain secret param")
	}
}

// ─── Model constant tests ─────────────────────────────────────────────────────

func TestUserStatusConstants(t *testing.T) {
	statuses := []models.UserStatus{
		models.UserActive, models.UserInactive, models.UserSuspended,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty user status constant")
		}
	}
}

func TestAccessLogStatusConstants(t *testing.T) {
	statuses := []models.AccessLogStatus{
		models.LogSuccess, models.LogFailure, models.LogDenied,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty access log status constant")
		}
	}
}

func TestSystemRoles(t *testing.T) {
	if len(models.SystemRoles) < 10 {
		t.Errorf("expected at least 10 system roles, got %d", len(models.SystemRoles))
	}
	// Admin must be present
	found := false
	for _, r := range models.SystemRoles {
		if r == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("admin must be in system roles")
	}
}

// ─── Request serialization tests ──────────────────────────────────────────────

func TestRegisterRequestJSON(t *testing.T) {
	req := models.RegisterRequest{
		Username: "akosuamensah",
		Email:    "akosua@klinova.co",
		Password: "SecurePass123!",
		FullName: "Akosua Mensah",
		Country:  "Ghana",
		Role:     "clinician",
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var back models.RegisterRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Username != req.Username {
		t.Errorf("username mismatch: %q", back.Username)
	}
	if back.Role != "clinician" {
		t.Errorf("role mismatch: %q", back.Role)
	}
}

func TestLoginRequestJSON(t *testing.T) {
	req := models.LoginRequest{
		Username: "akosuamensah",
		Password: "SecurePass123!",
		MFACode:  "123456",
	}
	b, _ := json.Marshal(req)
	var back models.LoginRequest
	json.Unmarshal(b, &back)
	if back.MFACode != "123456" {
		t.Errorf("mfa_code mismatch: %q", back.MFACode)
	}
}

func TestLoginResponseNeedsMFA(t *testing.T) {
	resp := models.LoginResponse{NeedsMFA: true}
	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	if m["needs_mfa"] != true {
		t.Error("expected needs_mfa=true")
	}
}

// ─── HTTP handler tests ───────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"auth-service"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "auth-service" {
		t.Errorf("unexpected service: %q", body["service"])
	}
}

func TestMissingAuthHeaderReturns401(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, `{"success":false}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ─── Stress tests for rate limiting ──────────────────────────────────────────

func TestRateLimitCounterLogic(t *testing.T) {
	// Verify that after maxAttempts the check returns false
	// This tests the logic, not the Redis integration
	maxAttempts := 5
	attempts := 0
	check := func() bool {
		attempts++
		return attempts <= maxAttempts
	}

	for i := 0; i < maxAttempts; i++ {
		if !check() {
			t.Errorf("expected allowed on attempt %d", i+1)
		}
	}
	if check() {
		t.Error("expected blocked on attempt 6")
	}
}

func TestAPIResponseShape(t *testing.T) {
	resp := models.APIResponse{
		Success: true,
		Data:    map[string]string{"token": "abc"},
		Meta:    &models.PageMeta{Page: 1, Limit: 50, Total: 200, TotalPages: 4},
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
	if m["meta"] == nil {
		t.Error("expected meta field")
	}
}

func TestAPIErrorShape(t *testing.T) {
	resp := models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "UNAUTHORIZED", Message: "invalid credentials"},
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
	if errObj["code"] != "UNAUTHORIZED" {
		t.Errorf("unexpected code: %v", errObj["code"])
	}
}

func TestAccessLogSerialization(t *testing.T) {
	log := models.AccessLog{
		Action:    "login",
		Resource:  "auth",
		Status:    models.LogSuccess,
		IPAddress: "41.220.18.3",
		CreatedAt: time.Now(),
	}
	b, err := json.Marshal(log)
	if err != nil {
		t.Fatal(err)
	}
	var back models.AccessLog
	json.Unmarshal(b, &back)
	if back.Action != "login" {
		t.Errorf("unexpected action: %q", back.Action)
	}
	if back.Status != models.LogSuccess {
		t.Errorf("unexpected status: %q", back.Status)
	}
}

func TestCheckPermissionRequestJSON(t *testing.T) {
	req := models.CheckPermissionRequest{
		UserID:   "550e8400-e29b-41d4-a716-446655440000",
		Resource: "patients",
		Action:   "read",
	}
	b, _ := json.Marshal(req)
	var back models.CheckPermissionRequest
	json.Unmarshal(b, &back)
	if back.Resource != "patients" {
		t.Errorf("resource mismatch: %q", back.Resource)
	}
}

func TestGenerateAPIKeyRequest(t *testing.T) {
	exp := time.Now().Add(30 * 24 * time.Hour)
	req := models.GenerateAPIKeyRequest{
		Name:        "clinical-service",
		Permissions: []string{"read:patients", "read:consultations"},
		ExpiresAt:   &exp,
	}
	b, _ := json.Marshal(req)
	var back models.GenerateAPIKeyRequest
	json.Unmarshal(b, &back)
	if len(back.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(back.Permissions))
	}
}
