package handlers_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	localauth "github.com/klinova/kinara-os/mental-health-service/auth"
	"github.com/klinova/kinara-os/mental-health-service/crypto"
	"github.com/klinova/kinara-os/mental-health-service/db"
	"github.com/klinova/kinara-os/mental-health-service/handlers"
	"github.com/klinova/kinara-os/mental-health-service/middleware"
)

// mockQuerier is an in-memory Querier for tests.
type mockQuerier struct {
	mu      sync.Mutex
	records []db.Record
}

func (m *mockQuerier) Create(_ context.Context, r db.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, r)
	return nil
}

func (m *mockQuerier) Get(_ context.Context, id uuid.UUID) (*db.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.records {
		if r.ID == id {
			cp := r
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockQuerier) List(_ context.Context, limit, offset int) ([]db.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	end := offset + limit
	if end > len(m.records) {
		end = len(m.records)
	}
	if offset >= len(m.records) {
		return nil, nil
	}
	cp := make([]db.Record, end-offset)
	copy(cp, m.records[offset:end])
	return cp, nil
}

func (m *mockQuerier) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, r := range m.records {
		if r.ID == id {
			m.records = append(m.records[:i], m.records[i+1:]...)
			return nil
		}
	}
	return nil
}

var (
	testPrivKey *rsa.PrivateKey
	testJWTMW   func(http.Handler) http.Handler
)

func TestMain(m *testing.M) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	testPrivKey = key

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		panic(err)
	}
	f, err := os.CreateTemp("", "kinara-test-jwt-pub-*.pem")
	if err != nil {
		panic(err)
	}
	if err := pem.Encode(f, &pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}); err != nil {
		panic(err)
	}
	f.Close()

	v, err := localauth.NewValidator(f.Name())
	if err != nil {
		panic(err)
	}
	os.Remove(f.Name())

	testJWTMW = middleware.JWT(v)
	os.Exit(m.Run())
}

func mintToken(t *testing.T, entityType string) string {
	t.Helper()
	claims := &localauth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:     uuid.New(),
		Role:       "worker",
		EntityType: entityType,
		TenantID:   uuid.New(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(testPrivKey)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func newTestServer(t *testing.T, store db.Querier) *httptest.Server {
	t.Helper()
	enc, err := crypto.NewEncryptor(bytes.Repeat([]byte("k"), 32))
	if err != nil {
		t.Fatal(err)
	}
	h := handlers.New(store, enc, slog.Default())
	r := mux.NewRouter()
	h.Register(r, testJWTMW)
	return httptest.NewServer(r)
}

// TestNoAuth_Returns401 verifies the JWT middleware rejects requests with no token.
func TestNoAuth_Returns401(t *testing.T) {
	srv := newTestServer(t, &mockQuerier{})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/mental-healths", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", resp.StatusCode)
	}
}

// TestInvalidToken_Returns401 verifies that a malformed bearer token is rejected.
func TestInvalidToken_Returns401(t *testing.T) {
	srv := newTestServer(t, &mockQuerier{})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/mental-healths", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", resp.StatusCode)
	}
}

// TestEmptyEntityType_Returns403 verifies that enforce mode blocks tokens with no entity_type.
func TestEmptyEntityType_Returns403(t *testing.T) {
	t.Setenv("TENANT_SCOPE_MODE", "enforce")

	// Build a fresh server so RequireTenantScope picks up the env at construction time.
	enc, err := crypto.NewEncryptor(bytes.Repeat([]byte("k"), 32))
	if err != nil {
		t.Fatal(err)
	}
	h := handlers.New(&mockQuerier{}, enc, slog.Default())
	r := mux.NewRouter()
	h.Register(r, testJWTMW)
	srv := httptest.NewServer(r)
	defer srv.Close()

	tok := mintToken(t, "") // empty entity_type triggers RequireTenantScope violation
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/mental-healths", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("want 403, got %d", resp.StatusCode)
	}
}

// TestCreate_StoresCiphertext verifies that a valid create request stores encrypted (not plaintext) data.
func TestCreate_StoresCiphertext(t *testing.T) {
	store := &mockQuerier{}
	srv := newTestServer(t, store)
	defer srv.Close()

	body := `{"name":"patient-alpha","note":"confidential"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/mental-healths", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+mintToken(t, "klinova"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}

	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["id"] == "" {
		t.Fatal("response missing id")
	}

	store.mu.Lock()
	recs := store.records
	store.mu.Unlock()

	if len(recs) != 1 {
		t.Fatalf("expected 1 record in store, got %d", len(recs))
	}
	if recs[0].DataEnc == body {
		t.Error("DataEnc must not equal the plaintext payload")
	}
	if recs[0].DataEnc == "" {
		t.Error("DataEnc must not be empty")
	}
}

// TestEncryptionRoundtrip verifies AES-GCM encrypt/decrypt symmetry without HTTP overhead.
func TestEncryptionRoundtrip(t *testing.T) {
	enc, err := crypto.NewEncryptor(bytes.Repeat([]byte("x"), 32))
	if err != nil {
		t.Fatal(err)
	}
	plaintext := `{"field":"sensitive value"}`
	ciphertext, err := enc.EncryptString(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext must differ from plaintext")
	}
	recovered, err := enc.DecryptString(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != plaintext {
		t.Errorf("roundtrip mismatch: got %q, want %q", recovered, plaintext)
	}
}
