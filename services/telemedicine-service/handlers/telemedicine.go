package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/telemedicine-service/db"
	"github.com/klinova/kinara-os/telemedicine-service/models"
)

type Store interface {
	RegisterDoctor(ctx context.Context, d models.Doctor) error
	ListAvailableDoctors(ctx context.Context) ([]models.Doctor, error)
	SetDoctorAvailability(ctx context.Context, doctorID uuid.UUID, available bool) error
	CreateConsultation(ctx context.Context, c models.Consultation) error
	GetConsultation(ctx context.Context, id uuid.UUID) (*models.Consultation, error)
	ListConsultations(ctx context.Context, patientID *uuid.UUID, doctorID *uuid.UUID, limit int) ([]models.Consultation, error)
	StartConsultation(ctx context.Context, id uuid.UUID, now time.Time) error
	CompleteConsultation(ctx context.Context, id uuid.UUID, durationMinutes int, now time.Time) error
	IssuePrescription(ctx context.Context, p models.Prescription, instructionsEnc string) error
	GetPrescription(ctx context.Context, consultationID uuid.UUID) (*models.Prescription, string, error)
	SaveRecording(ctx context.Context, r models.RecordingMetadata) error
	GetRecording(ctx context.Context, consultationID uuid.UUID) (*models.RecordingMetadata, error)
	InsertAuditLog(ctx context.Context, l models.TelemedicineAuditLog) error
}

type Handler struct {
	store      Store
	jwtSecret  []byte
	logger     *slog.Logger
}

func NewHandler(q *db.Queries, jwtSecret []byte, logger *slog.Logger) *Handler {
	return &Handler{store: q, jwtSecret: jwtSecret, logger: logger}
}

func NewHandlerWithStore(s Store) *Handler {
	return &Handler{store: s, jwtSecret: []byte("test-secret-32-bytes-for-testing"), logger: slog.Default()}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.health).Methods(http.MethodGet)
	api := r.PathPrefix("/api/v1/telemedicine").Subrouter()
	api.HandleFunc("/doctors", h.registerDoctor).Methods(http.MethodPost)
	api.HandleFunc("/doctors/available", h.listAvailableDoctors).Methods(http.MethodGet)
	api.HandleFunc("/doctors/{id}/availability", h.setDoctorAvailability).Methods(http.MethodPut)
	api.HandleFunc("/consultations", h.bookConsultation).Methods(http.MethodPost)
	api.HandleFunc("/consultations", h.listConsultations).Methods(http.MethodGet)
	api.HandleFunc("/consultations/{id}", h.getConsultation).Methods(http.MethodGet)
	api.HandleFunc("/consultations/{id}/start", h.startConsultation).Methods(http.MethodPut)
	api.HandleFunc("/consultations/{id}/complete", h.completeConsultation).Methods(http.MethodPut)
	api.HandleFunc("/consultations/{id}/video-token", h.getVideoToken).Methods(http.MethodGet)
	api.HandleFunc("/consultations/{id}/prescription", h.issuePrescription).Methods(http.MethodPost)
	api.HandleFunc("/consultations/{id}/prescription", h.getPrescription).Methods(http.MethodGet)
	api.HandleFunc("/consultations/{id}/recording", h.getRecording).Methods(http.MethodGet)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "telemedicine-service"})
}

func (h *Handler) registerDoctor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClinicID       string `json:"clinic_id"`
		FullName       string `json:"full_name"`
		Specialization string `json:"specialization"`
		LicenseNumber  string `json:"license_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.FullName == "" || req.LicenseNumber == "" || req.ClinicID == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "full_name, license_number, and clinic_id are required")
		return
	}
	clinicID, err := uuid.Parse(req.ClinicID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid clinic_id")
		return
	}
	spec := req.Specialization
	if spec == "" {
		spec = "general"
	}
	now := time.Now().UTC()
	doc := models.Doctor{
		ID:             uuid.New(),
		ClinicID:       clinicID,
		FullName:       req.FullName,
		Specialization: spec,
		LicenseNumber:  req.LicenseNumber,
		IsAvailable:    false,
		CreatedAt:      now,
	}
	if err := h.store.RegisterDoctor(r.Context(), doc); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to register doctor")
		return
	}
	h.audit(r.Context(), uuid.Nil, uuid.Nil, "register_doctor", "doctor: "+doc.ID.String())
	respond(w, http.StatusCreated, doc)
}

func (h *Handler) listAvailableDoctors(w http.ResponseWriter, r *http.Request) {
	docs, err := h.store.ListAvailableDoctors(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list doctors")
		return
	}
	if docs == nil {
		docs = []models.Doctor{}
	}
	respond(w, http.StatusOK, docs)
}

func (h *Handler) setDoctorAvailability(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid doctor id")
		return
	}
	var req struct {
		Available bool `json:"is_available"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if err := h.store.SetDoctorAvailability(r.Context(), id, req.Available); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to update availability")
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"doctor_id": id, "is_available": req.Available})
}

func (h *Handler) bookConsultation(w http.ResponseWriter, r *http.Request) {
	var req models.BookConsultationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.PatientID == "" || req.DoctorID == "" || req.ClinicID == "" || req.ChiefComplaint == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "patient_id, doctor_id, clinic_id, chief_complaint are required")
		return
	}
	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid patient_id")
		return
	}
	doctorID, err := uuid.Parse(req.DoctorID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid doctor_id")
		return
	}
	clinicID, err := uuid.Parse(req.ClinicID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid clinic_id")
		return
	}
	consultType := models.ConsultationType(req.Type)
	if consultType == "" {
		consultType = models.TypeVideo
	}
	scheduledAt := time.Now().UTC().Add(30 * time.Minute)
	if req.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduledAt = t
		}
	}
	now := time.Now().UTC()
	id := uuid.New()
	ref := "TC-" + strings.ToUpper(id.String()[:8])
	c := models.Consultation{
		ID:             id,
		ConsultRef:     ref,
		PatientID:      patientID,
		DoctorID:       doctorID,
		ClinicID:       clinicID,
		Type:           consultType,
		Status:         models.StatusScheduled,
		ChiefComplaint: req.ChiefComplaint,
		ScheduledAt:    scheduledAt,
		CostUSD:        5.00,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := h.store.CreateConsultation(r.Context(), c); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to book consultation")
		return
	}
	h.audit(r.Context(), id, patientID, "book_consultation", fmt.Sprintf("ref:%s type:%s", ref, consultType))
	respond(w, http.StatusCreated, c)
}

func (h *Handler) getConsultation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid consultation id")
		return
	}
	c, err := h.store.GetConsultation(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "consultation not found")
		return
	}
	respond(w, http.StatusOK, c)
}

func (h *Handler) listConsultations(w http.ResponseWriter, r *http.Request) {
	var patientID, doctorID *uuid.UUID
	if p := r.URL.Query().Get("patient_id"); p != "" {
		if pid, err := uuid.Parse(p); err == nil {
			patientID = &pid
		}
	}
	if d := r.URL.Query().Get("doctor_id"); d != "" {
		if did, err := uuid.Parse(d); err == nil {
			doctorID = &did
		}
	}
	cs, err := h.store.ListConsultations(r.Context(), patientID, doctorID, 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list consultations")
		return
	}
	if cs == nil {
		cs = []models.Consultation{}
	}
	respond(w, http.StatusOK, cs)
}

func (h *Handler) startConsultation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid consultation id")
		return
	}
	now := time.Now().UTC()
	if err := h.store.StartConsultation(r.Context(), id, now); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to start consultation")
		return
	}
	h.audit(r.Context(), id, uuid.Nil, "start_consultation", "")
	respond(w, http.StatusOK, map[string]interface{}{"consultation_id": id, "status": "in_progress", "started_at": now})
}

func (h *Handler) completeConsultation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid consultation id")
		return
	}
	var req struct {
		DurationMinutes int `json:"duration_minutes"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 30
	}
	now := time.Now().UTC()
	if err := h.store.CompleteConsultation(r.Context(), id, req.DurationMinutes, now); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to complete consultation")
		return
	}
	h.audit(r.Context(), id, uuid.Nil, "complete_consultation", fmt.Sprintf("duration:%d min", req.DurationMinutes))
	respond(w, http.StatusOK, map[string]interface{}{"consultation_id": id, "status": "completed", "completed_at": now})
}

func (h *Handler) getVideoToken(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid consultation id")
		return
	}
	c, err := h.store.GetConsultation(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "consultation not found")
		return
	}
	if c.Status == models.StatusCancelled || c.Status == models.StatusCompleted {
		respondError(w, http.StatusUnprocessableEntity, "INVALID_STATUS", "consultation is not active")
		return
	}
	expiresAt := time.Now().UTC().Add(2 * time.Hour)
	claims := jwt.MapClaims{
		"consultation_id": id.String(),
		"patient_id":      c.PatientID.String(),
		"doctor_id":       c.DoctorID.String(),
		"room_id":         "room-" + id.String()[:8],
		"exp":             expiresAt.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(h.jwtSecret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "failed to generate video token")
		return
	}
	h.audit(r.Context(), id, c.PatientID, "get_video_token", "")
	respond(w, http.StatusOK, models.VideoToken{
		Token:     signed,
		ExpiresAt: expiresAt,
		RoomID:    "room-" + id.String()[:8],
	})
}

func (h *Handler) issuePrescription(w http.ResponseWriter, r *http.Request) {
	cid, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid consultation id")
		return
	}
	c, err := h.store.GetConsultation(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "consultation not found")
		return
	}
	var req models.IssuePrescriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Medication == "" || req.Dosage == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "medication_name and dosage are required")
		return
	}
	if req.FrequencyDays <= 0 {
		req.FrequencyDays = 7
	}
	if req.Instructions == "" {
		req.Instructions = "Take as directed by your doctor."
	}
	now := time.Now().UTC()
	p := models.Prescription{
		ID:             uuid.New(),
		ConsultationID: cid,
		PatientID:      c.PatientID,
		DoctorID:       c.DoctorID,
		Medication:     req.Medication,
		Dosage:         req.Dosage,
		FrequencyDays:  req.FrequencyDays,
		Instructions:   req.Instructions,
		IssuedAt:       now,
	}
	if err := h.store.IssuePrescription(r.Context(), p, req.Instructions); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to issue prescription")
		return
	}
	h.audit(r.Context(), cid, c.PatientID, "issue_prescription", "medication: "+req.Medication)
	respond(w, http.StatusCreated, p)
}

func (h *Handler) getPrescription(w http.ResponseWriter, r *http.Request) {
	cid, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid consultation id")
		return
	}
	p, _, err := h.store.GetPrescription(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "prescription not found")
		return
	}
	respond(w, http.StatusOK, p)
}

func (h *Handler) getRecording(w http.ResponseWriter, r *http.Request) {
	cid, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid consultation id")
		return
	}
	rec, err := h.store.GetRecording(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "no recording found for this consultation")
		return
	}
	h.audit(r.Context(), cid, uuid.Nil, "access_recording", "")
	respond(w, http.StatusOK, rec)
}

func (h *Handler) audit(ctx context.Context, consultID, actorID uuid.UUID, action, detail string) {
	_ = h.store.InsertAuditLog(ctx, models.TelemedicineAuditLog{
		ID:             uuid.New(),
		ConsultationID: consultID,
		ActorID:        actorID,
		Action:         action,
		Detail:         detail,
		CreatedAt:      time.Now().UTC(),
	})
}

func respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: data})
}

func respondError(w http.ResponseWriter, code int, errCode, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: errCode, Message: msg},
	})
}
