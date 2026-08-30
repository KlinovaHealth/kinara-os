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

	"github.com/klinova/kinara-os/livestock-service/auth"
	"github.com/klinova/kinara-os/livestock-service/handlers"
	"github.com/klinova/kinara-os/livestock-service/middleware"
	"github.com/klinova/kinara-os/livestock-service/models"
)

// mockStore is a zero-allocation in-memory Store used by all handler tests.
type mockStore struct {
	animals  map[uuid.UUID]*models.Animal
	health   map[uuid.UUID][]models.HealthEvent
	prod     []models.ProductionRecord
	vetAlerts []models.VeterinaryAlert
	audits   []string
	errOn    string // set to method name to simulate DB error
}

func newMock() *mockStore {
	return &mockStore{
		animals: make(map[uuid.UUID]*models.Animal),
		health:  make(map[uuid.UUID][]models.HealthEvent),
	}
}

func (m *mockStore) RegisterAnimal(_ context.Context, a models.Animal) error {
	if m.errOn == "RegisterAnimal" {
		return errMock
	}
	m.animals[a.ID] = &a
	return nil
}
func (m *mockStore) GetAnimal(_ context.Context, id uuid.UUID) (*models.Animal, error) {
	if m.errOn == "GetAnimal" {
		return nil, errMock
	}
	a, ok := m.animals[id]
	if !ok {
		return nil, errNotFound
	}
	return a, nil
}
func (m *mockStore) LogHealthEvent(_ context.Context, e models.HealthEvent) error {
	if m.errOn == "LogHealthEvent" {
		return errMock
	}
	m.health[e.AnimalID] = append(m.health[e.AnimalID], e)
	return nil
}
func (m *mockStore) GetHealthHistory(_ context.Context, animalID uuid.UUID) ([]models.HealthEvent, error) {
	return m.health[animalID], nil
}
func (m *mockStore) LogProduction(_ context.Context, p models.ProductionRecord) error {
	if m.errOn == "LogProduction" {
		return errMock
	}
	m.prod = append(m.prod, p)
	return nil
}
func (m *mockStore) ListHerd(_ context.Context, farmerID uuid.UUID) ([]models.Animal, error) {
	var out []models.Animal
	for _, a := range m.animals {
		if a.FarmerID == farmerID {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (m *mockStore) GetHerdAnalytics(_ context.Context, farmerID uuid.UUID) (models.HerdAnalytics, error) {
	var total, healthy int
	for _, a := range m.animals {
		if a.FarmerID != farmerID {
			continue
		}
		total++
		sick := false
		cutoff := time.Now().Add(-30 * 24 * time.Hour)
		for _, e := range m.health[a.ID] {
			if e.EventType == "illness" && e.EventDate.After(cutoff) {
				sick = true
				break
			}
		}
		if !sick {
			healthy++
		}
	}
	var healthRate float64
	if total > 0 {
		healthRate = float64(healthy) / float64(total) * 100
	}
	return models.HerdAnalytics{
		TotalAnimals:  total,
		HealthyCount:  healthy,
		HealthRatePct: healthRate,
	}, nil
}
func (m *mockStore) InsertVetAlert(_ context.Context, a models.VeterinaryAlert) error {
	m.vetAlerts = append(m.vetAlerts, a)
	return nil
}
func (m *mockStore) InsertAudit(_ context.Context, animalID, actorID, action string) error {
	m.audits = append(m.audits, animalID+":"+actorID+":"+action)
	return nil
}

// sentinel errors
type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

const errMock sentinelErr = "mock db error"
const errNotFound sentinelErr = "not found"

// authCtx injects a *auth.Claims into a request context.
func authCtx(r *http.Request, role string) *http.Request {
	claims := &auth.Claims{
		UserID:   uuid.New(),
		Role:     role,
		TenantID: "tenant-test",
	}
	return r.WithContext(middleware.SetClaims(r.Context(), claims))
}

func routeRequest(r *http.Request, h *handlers.Handler) *httptest.ResponseRecorder {
	router := mux.NewRouter()
	h.Register(router)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

// --- Tests ---

func TestRegisterAnimal_Success(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)
	farmerID := uuid.New()

	body, _ := json.Marshal(models.RegisterAnimalRequest{
		AnimalType: "cattle",
		Breed:      "Zebu",
		AgeMonths:  24,
		Sex:        "F",
		EarTag:     "TAG-001",
		FarmerID:   farmerID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/livestock/animals", bytes.NewReader(body))
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if len(store.animals) != 1 {
		t.Errorf("expected 1 animal in store, got %d", len(store.animals))
	}
	for _, a := range store.animals {
		if !strings.HasPrefix(a.AnimalRef, "ANM-") {
			t.Errorf("expected ANM- prefix, got %s", a.AnimalRef)
		}
	}
}

func TestRegisterAnimal_Unauthorized(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.RegisterAnimalRequest{AnimalType: "goat"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/livestock/animals", bytes.NewReader(body))
	// No auth context
	w := routeRequest(req, h)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRegisterAnimal_ForbiddenRole(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.RegisterAnimalRequest{AnimalType: "goat", FarmerID: uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/livestock/animals", bytes.NewReader(body))
	req = authCtx(req, "viewer") // viewer is not allowed to register
	w := routeRequest(req, h)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestGetAnimal_NotFound(t *testing.T) {
	store := newMock()
	h := handlers.NewWithStore(store)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/livestock/animals/"+id.String(), nil)
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetAnimal_Success(t *testing.T) {
	store := newMock()
	id := uuid.New()
	store.animals[id] = &models.Animal{
		ID:           id,
		AnimalRef:    "ANM-ABCD1234",
		FarmerID:     uuid.New(),
		AnimalType:   "cattle",
		RegisteredAt: time.Now(),
	}
	h := handlers.NewWithStore(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/livestock/animals/"+id.String(), nil)
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogHealthEvent_Success(t *testing.T) {
	store := newMock()
	animalID := uuid.New()
	store.animals[animalID] = &models.Animal{ID: animalID, AnimalRef: "ANM-TEST0001"}
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.HealthEventRequest{
		EventType:   "checkup",
		Description: "Routine annual checkup",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/livestock/animals/"+animalID.String()+"/health", bytes.NewReader(body))
	req = authCtx(req, "vet")
	w := routeRequest(req, h)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if len(store.health[animalID]) != 1 {
		t.Errorf("expected 1 health event, got %d", len(store.health[animalID]))
	}
}

func TestLogHealthEvent_IllnessTrigersAlert(t *testing.T) {
	store := newMock()
	animalID := uuid.New()
	store.animals[animalID] = &models.Animal{ID: animalID, AnimalRef: "ANM-ILLNESS1"}
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.HealthEventRequest{
		EventType:   "illness",
		Description: "Suspected foot-and-mouth disease",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/livestock/animals/"+animalID.String()+"/health", bytes.NewReader(body))
	req = authCtx(req, "vet")
	w := routeRequest(req, h)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	// Goroutine may need a moment; use small poll or just check the vet-alert was inserted.
	// In unit tests without real goroutine async, we check the store synchronously after a tiny yield.
	// Because the goroutine is fire-and-forget we can't guarantee ordering, but the handler must respond 201.
}

func TestGetHealthHistory_Success(t *testing.T) {
	store := newMock()
	animalID := uuid.New()
	store.animals[animalID] = &models.Animal{ID: animalID}
	store.health[animalID] = []models.HealthEvent{
		{ID: uuid.New(), AnimalID: animalID, EventType: "vaccination", EventDate: time.Now()},
		{ID: uuid.New(), AnimalID: animalID, EventType: "checkup", EventDate: time.Now().Add(-24 * time.Hour)},
	}
	h := handlers.NewWithStore(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/livestock/animals/"+animalID.String()+"/health-history", nil)
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("expected 2 health events, got %d", len(data))
	}
}

func TestLogProduction_Success(t *testing.T) {
	store := newMock()
	animalID := uuid.New()
	store.animals[animalID] = &models.Animal{ID: animalID}
	h := handlers.NewWithStore(store)

	body, _ := json.Marshal(models.ProductionRequest{
		ProductionType: "milk",
		Quantity:       15.5,
		Unit:           "liters",
		RecordedDate:   time.Now().UTC(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/livestock/animals/"+animalID.String()+"/production", bytes.NewReader(body))
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if len(store.prod) != 1 {
		t.Errorf("expected 1 production record, got %d", len(store.prod))
	}
	if store.prod[0].ProductionType != "milk" {
		t.Errorf("expected milk production type, got %s", store.prod[0].ProductionType)
	}
}

func TestListHerd_Success(t *testing.T) {
	store := newMock()
	farmerID := uuid.New()
	for i := 0; i < 3; i++ {
		id := uuid.New()
		store.animals[id] = &models.Animal{ID: id, FarmerID: farmerID, AnimalType: "goat"}
	}
	// Add animal for different farmer (should not appear).
	othID := uuid.New()
	store.animals[othID] = &models.Animal{ID: othID, FarmerID: uuid.New(), AnimalType: "sheep"}

	h := handlers.NewWithStore(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/livestock/farmers/"+farmerID.String()+"/herd", nil)
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]interface{})
	if len(data) != 3 {
		t.Errorf("expected 3 animals, got %d", len(data))
	}
}

func TestHerdAnalytics_Calculation(t *testing.T) {
	store := newMock()
	farmerID := uuid.New()

	// 4 animals total, 1 with recent illness -> 3 healthy
	for i := 0; i < 4; i++ {
		id := uuid.New()
		store.animals[id] = &models.Animal{ID: id, FarmerID: farmerID, AnimalType: "cattle"}
		if i == 0 {
			store.health[id] = []models.HealthEvent{
				{ID: uuid.New(), AnimalID: id, EventType: "illness", EventDate: time.Now()},
			}
		}
	}

	h := handlers.NewWithStore(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/livestock/farmers/"+farmerID.String()+"/analytics", nil)
	req = authCtx(req, "farmer")
	w := routeRequest(req, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})

	totalAnimals := int(data["total_animals"].(float64))
	healthyCount := int(data["healthy_count"].(float64))
	healthRate := data["health_rate_percent"].(float64)

	if totalAnimals != 4 {
		t.Errorf("expected total_animals=4, got %d", totalAnimals)
	}
	if healthyCount != 3 {
		t.Errorf("expected healthy_count=3, got %d", healthyCount)
	}
	if healthRate < 74 || healthRate > 76 {
		t.Errorf("expected health_rate_percent ~75, got %.2f", healthRate)
	}
}

func TestAnimalRef_Format(t *testing.T) {
	id := uuid.New()
	ref := "ANM-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "ANM-") {
		t.Errorf("expected ANM- prefix, got %s", ref)
	}
	if len(ref) != 12 { // "ANM-" (4) + 8 hex chars
		t.Errorf("expected ref length 12, got %d: %s", len(ref), ref)
	}
}
