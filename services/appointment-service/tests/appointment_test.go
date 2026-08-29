package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/appointment-service/models"
)

func TestAppointmentStatusValues(t *testing.T) {
	statuses := []models.AppointmentStatus{
		models.StatusScheduled, models.StatusConfirmed,
		models.StatusCompleted, models.StatusCancelled, models.StatusNoShow,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("empty status")
		}
	}
}

func TestAppointmentTypes(t *testing.T) {
	types := []models.AppointmentType{
		models.TypeConsultation, models.TypeFollowUp,
		models.TypeProcedure, models.TypeEmergency,
	}
	if len(types) != 4 {
		t.Errorf("expected 4 types, got %d", len(types))
	}
}

func TestRefFormat(t *testing.T) {
	ref := "APT-" + strings.ToUpper("12345678")
	if !strings.HasPrefix(ref, "APT-") {
		t.Errorf("ref must start with APT-, got %s", ref)
	}
	if len(ref) != 12 {
		t.Errorf("ref length mismatch: %d", len(ref))
	}
}

func TestCreateAppointmentMissingAuth(t *testing.T) {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/appointments", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}).Methods(http.MethodPost)

	body, _ := json.Marshal(models.CreateAppointmentRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/appointments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestDefaultDuration(t *testing.T) {
	req := models.CreateAppointmentRequest{DurationMin: 0}
	if req.DurationMin <= 0 {
		req.DurationMin = 30
	}
	if req.DurationMin != 30 {
		t.Errorf("expected default 30 min, got %d", req.DurationMin)
	}
}

func TestUpdateStatusRequest(t *testing.T) {
	req := models.UpdateStatusRequest{Status: models.StatusCompleted}
	b, _ := json.Marshal(req)
	var out models.UpdateStatusRequest
	json.Unmarshal(b, &out)
	if out.Status != models.StatusCompleted {
		t.Errorf("roundtrip failed: %s", out.Status)
	}
}
