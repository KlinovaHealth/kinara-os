package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/telemedicine-service/handlers"
	"github.com/klinova/kinara-os/telemedicine-service/models"
)

type memStore struct {
	mu            sync.RWMutex
	doctors       map[uuid.UUID]*models.Doctor
	consultations map[uuid.UUID]*models.Consultation
	prescriptions map[uuid.UUID]*models.Prescription
	recordings    map[uuid.UUID]*models.RecordingMetadata
	audit         []models.TelemedicineAuditLog
}

func newMemStore() *memStore {
	return &memStore{
		doctors:       map[uuid.UUID]*models.Doctor{},
		consultations: map[uuid.UUID]*models.Consultation{},
		prescriptions: map[uuid.UUID]*models.Prescription{},
		recordings:    map[uuid.UUID]*models.RecordingMetadata{},
	}
}

func (s *memStore) RegisterDoctor(_ context.Context, d models.Doctor) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.doctors[d.ID] = &d; return nil
}
func (s *memStore) ListAvailableDoctors(_ context.Context) ([]models.Doctor, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Doctor
	for _, d := range s.doctors {
		if d.IsAvailable {
			result = append(result, *d)
		}
	}
	return result, nil
}
func (s *memStore) SetDoctorAvailability(_ context.Context, id uuid.UUID, available bool) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if d, ok := s.doctors[id]; ok {
		d.IsAvailable = available
	}
	return nil
}
func (s *memStore) CreateConsultation(_ context.Context, c models.Consultation) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.consultations[c.ID] = &c; return nil
}
func (s *memStore) GetConsultation(_ context.Context, id uuid.UUID) (*models.Consultation, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	c, ok := s.consultations[id]
	if !ok {
		return nil, errNotFound
	}
	cp := *c; return &cp, nil
}
func (s *memStore) ListConsultations(_ context.Context, patientID *uuid.UUID, doctorID *uuid.UUID, limit int) ([]models.Consultation, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var result []models.Consultation
	for _, c := range s.consultations {
		if patientID != nil && c.PatientID != *patientID {
			continue
		}
		if doctorID != nil && c.DoctorID != *doctorID {
			continue
		}
		result = append(result, *c)
	}
	return result, nil
}
func (s *memStore) StartConsultation(_ context.Context, id uuid.UUID, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if c, ok := s.consultations[id]; ok {
		c.Status = models.StatusInProgress
		if c.StartedAt == nil {
			c.StartedAt = &now
		}
	}
	return nil
}
func (s *memStore) CompleteConsultation(_ context.Context, id uuid.UUID, durationMinutes int, now time.Time) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if c, ok := s.consultations[id]; ok {
		c.Status = models.StatusCompleted
		if c.CompletedAt == nil {
			c.CompletedAt = &now
		}
		if c.DurationMinutes == nil {
			c.DurationMinutes = &durationMinutes
		}
	}
	return nil
}
func (s *memStore) IssuePrescription(_ context.Context, p models.Prescription, _ string) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.prescriptions[p.ConsultationID] = &p; return nil
}
func (s *memStore) GetPrescription(_ context.Context, consultationID uuid.UUID) (*models.Prescription, string, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	p, ok := s.prescriptions[consultationID]
	if !ok {
		return nil, "", errNotFound
	}
	cp := *p; return &cp, "encrypted", nil
}
func (s *memStore) SaveRecording(_ context.Context, r models.RecordingMetadata) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.recordings[r.ConsultationID] = &r; return nil
}
func (s *memStore) GetRecording(_ context.Context, consultationID uuid.UUID) (*models.RecordingMetadata, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	r, ok := s.recordings[consultationID]
	if !ok {
		return nil, errNotFound
	}
	cp := *r; return &cp, nil
}
func (s *memStore) InsertAuditLog(_ context.Context, l models.TelemedicineAuditLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.audit = append(s.audit, l); return nil
}

var errNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "not found" }

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := newMemStore()
	h := handlers.NewHandlerWithStore(store)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return httptest.NewServer(r), store
}

func TestBookConsultation(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	body, _ := json.Marshal(models.BookConsultationRequest{
		PatientID:      uuid.New().String(),
		DoctorID:       uuid.New().String(),
		ClinicID:       uuid.New().String(),
		Type:           "video",
		ChiefComplaint: "Persistent fever for 3 days",
		ScheduledAt:    time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
	})
	resp, _ := http.Post(srv.URL+"/api/v1/telemedicine/consultations", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success {
		t.Fatal("expected success")
	}
}

func TestBookConsultation_MissingFields(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	body := `{"chief_complaint":"Fever"}`
	resp, _ := http.Post(srv.URL+"/api/v1/telemedicine/consultations", "application/json", bytes.NewBufferString(body))
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetVideoToken(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	cid := uuid.New()
	store.CreateConsultation(context.Background(), models.Consultation{
		ID:             cid,
		ConsultRef:     "TC-TEST001",
		PatientID:      uuid.New(),
		DoctorID:       uuid.New(),
		ClinicID:       uuid.New(),
		Type:           models.TypeVideo,
		Status:         models.StatusScheduled,
		ChiefComplaint: "Test",
		ScheduledAt:    time.Now().Add(time.Hour),
		CostUSD:        5.0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})
	resp, _ := http.Get(srv.URL + "/api/v1/telemedicine/consultations/" + cid.String() + "/video-token")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if data["token"] == "" {
		t.Fatal("expected video token in response")
	}
	if data["room_id"] == "" {
		t.Fatal("expected room_id in response")
	}
}

func TestIssuePrescription(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	cid := uuid.New()
	store.CreateConsultation(context.Background(), models.Consultation{
		ID:             cid,
		ConsultRef:     "TC-TEST002",
		PatientID:      uuid.New(),
		DoctorID:       uuid.New(),
		ClinicID:       uuid.New(),
		Type:           models.TypeVideo,
		Status:         models.StatusInProgress,
		ChiefComplaint: "Malaria",
		ScheduledAt:    time.Now(),
		CostUSD:        5.0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})
	body, _ := json.Marshal(models.IssuePrescriptionRequest{
		Medication:    "Artemether-Lumefantrine",
		Dosage:        "80mg/480mg",
		FrequencyDays: 3,
		Instructions:  "Take 4 tablets twice daily for 3 days with food.",
	})
	resp, _ := http.Post(srv.URL+"/api/v1/telemedicine/consultations/"+cid.String()+"/prescription",
		"application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestStartAndCompleteConsultation(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	cid := uuid.New()
	now := time.Now().UTC()
	store.CreateConsultation(context.Background(), models.Consultation{
		ID:             cid,
		ConsultRef:     "TC-TEST003",
		PatientID:      uuid.New(),
		DoctorID:       uuid.New(),
		ClinicID:       uuid.New(),
		Type:           models.TypeVideo,
		Status:         models.StatusScheduled,
		ChiefComplaint: "Headache",
		ScheduledAt:    now,
		CostUSD:        5.0,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	// Start
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/telemedicine/consultations/"+cid.String()+"/start", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("start: expected 200, got %d", resp.StatusCode)
	}
	c, _ := store.GetConsultation(context.Background(), cid)
	if c.Status != models.StatusInProgress {
		t.Fatalf("expected in_progress, got %s", c.Status)
	}
	// Complete
	body := `{"duration_minutes":45}`
	req2, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/telemedicine/consultations/"+cid.String()+"/complete",
		bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := http.DefaultClient.Do(req2)
	if resp2.StatusCode != 200 {
		t.Fatalf("complete: expected 200, got %d", resp2.StatusCode)
	}
	c2, _ := store.GetConsultation(context.Background(), cid)
	if c2.Status != models.StatusCompleted {
		t.Fatalf("expected completed, got %s", c2.Status)
	}
}

func TestListAvailableDoctors(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	store.RegisterDoctor(context.Background(), models.Doctor{
		ID:             uuid.New(),
		ClinicID:       uuid.New(),
		FullName:       "Dr. Kwame Asante",
		Specialization: "internal_medicine",
		LicenseNumber:  "GH-DOC-001",
		IsAvailable:    true,
		CreatedAt:      time.Now(),
	})
	store.RegisterDoctor(context.Background(), models.Doctor{
		ID:             uuid.New(),
		ClinicID:       uuid.New(),
		FullName:       "Dr. Amina Diallo",
		Specialization: "pediatrics",
		LicenseNumber:  "SN-DOC-002",
		IsAvailable:    false,
		CreatedAt:      time.Now(),
	})
	resp, _ := http.Get(srv.URL + "/api/v1/telemedicine/doctors/available")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	items := out.Data.([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 available doctor, got %d", len(items))
	}
}

func TestGetConsultation_NotFound(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/v1/telemedicine/consultations/" + uuid.New().String())
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAuditLogged(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	body, _ := json.Marshal(models.BookConsultationRequest{
		PatientID:      uuid.New().String(),
		DoctorID:       uuid.New().String(),
		ClinicID:       uuid.New().String(),
		Type:           "audio",
		ChiefComplaint: "Cough",
	})
	http.Post(srv.URL+"/api/v1/telemedicine/consultations", "application/json", bytes.NewBuffer(body))
	store.mu.RLock()
	defer store.mu.RUnlock()
	if len(store.audit) == 0 {
		t.Fatal("expected audit log entry after booking")
	}
}
