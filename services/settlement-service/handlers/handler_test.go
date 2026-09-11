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
	localauth "github.com/klinova/kinara-os/settlement-service/auth"
	"github.com/klinova/kinara-os/settlement-service/crypto"
	"github.com/klinova/kinara-os/settlement-service/db"
	"github.com/klinova/kinara-os/settlement-service/handlers"
	"github.com/klinova/kinara-os/settlement-service/middleware"
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

func (m *mockQuerier) List(_ context.Context, limit, offset int, tenantID uuid.UUID) ([]db.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var filtered []db.Record
	for _, r := range m.records {
		if r.TenantID == tenantID {
			filtered = append(filtered, r)
		}
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	if offset >= len(filtered) {
		return nil, nil
	}
	cp := make([]db.Record, end-offset)
	copy(cp, filtered[offset:end])
	return cp, nil
}

func (m *mockQuerier) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, r := range m.records {
		if r.ID == id && r.TenantID == tenantID {
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

func mintToken(t *testing.T, entityType string, tenantID uuid.UUID) string {
	t.Helper()
	claims := &localauth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:     uuid.New(),
		Role:       "worker",
		EntityType: entityType,
		TenantID:   tenantID,
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

	resp, err := http.Post(srv.URL+"/api/v1/settlements", "application/json", bytes.NewBufferString(`{}`))
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

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/settlements", bytes.NewBufferString(`{}`))
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

	tok := mintToken(t, "", uuid.New()) // empty entity_type triggers RequireTenantScope violation
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/settlements", bytes.NewBufferString(`{}`))
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

	tenantID := uuid.New()
	body := `{"name":"patient-alpha","note":"confidential"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/settlements", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+mintToken(t, "klinova", tenantID))
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
	if recs[0].TenantID != tenantID {
		t.Errorf("stored TenantID %v does not match token TenantID %v", recs[0].TenantID, tenantID)
	}
}

// TestCrossTenantList verifies that list returns only the requesting tenant's own records.
func TestCrossTenantList(t *testing.T) {
	store := &mockQuerier{}
	tenantA := uuid.New()
	tenantB := uuid.New()
	idA := uuid.New()
	idB := uuid.New()

	store.mu.Lock()
	store.records = []db.Record{
		{ID: idA, DataEnc: "enc-A", TenantID: tenantA, CreatedBy: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: idB, DataEnc: "enc-B", TenantID: tenantB, CreatedBy: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	store.mu.Unlock()

	srv := newTestServer(t, store)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/settlements", nil)
	req.Header.Set("Authorization", "Bearer "+mintToken(t, "klinova", tenantA))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var body struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if len(body.Items) != 1 {
		t.Errorf("tenantA should see 1 record, got %d", len(body.Items))
	}
	for _, item := range body.Items {
		if item.ID == idB.String() {
			t.Error("tenantA's list response must not contain tenantB's record")
		}
	}
}

// TestCrossTenantGet verifies cross-tenant get-by-id returns 404 (not 403) to avoid existence leak.
func TestCrossTenantGet(t *testing.T) {
	store := &mockQuerier{}
	tenantA := uuid.New()
	tenantB := uuid.New()
	idB := uuid.New()

	store.mu.Lock()
	store.records = []db.Record{
		{ID: idB, DataEnc: "enc-B", TenantID: tenantB, CreatedBy: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	store.mu.Unlock()

	srv := newTestServer(t, store)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/settlements/"+idB.String(), nil)
	req.Header.Set("Authorization", "Bearer "+mintToken(t, "klinova", tenantA))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	t.Logf("cross-tenant GET returns: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-tenant GET should return 404 (not %d) to avoid existence leak", resp.StatusCode)
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
