package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/clinical-service/auth"
	"github.com/klinova/kinara-os/clinical-service/crypto"
	"github.com/klinova/kinara-os/clinical-service/db"
	"github.com/klinova/kinara-os/clinical-service/middleware"
	"github.com/klinova/kinara-os/clinical-service/models"
)

// Handler holds all dependencies for the clinical service HTTP layer.
type Handler struct {
	queries *db.Queries
	enc     *crypto.Encryptor
	logger  *slog.Logger
}

func New(queries *db.Queries, enc *crypto.Encryptor, logger *slog.Logger) *Handler {
	return &Handler{queries: queries, enc: enc, logger: logger}
}

// Register mounts all clinical routes on r.
func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/api/v1/consultations", h.createConsultation).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/consultations", h.listConsultations).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/consultations/{id}", h.getConsultation).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/consultations/{id}", h.updateConsultation).Methods(http.MethodPut)

	r.HandleFunc("/api/v1/consultations/{id}/diagnoses", h.createDiagnosis).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/consultations/{id}/diagnoses", h.listDiagnoses).Methods(http.MethodGet)

	r.HandleFunc("/api/v1/consultations/{id}/treatments", h.createTreatment).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/consultations/{id}/treatments", h.listTreatments).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/treatments/{id}/status", h.updateTreatmentStatus).Methods(http.MethodPut)

	r.HandleFunc("/api/v1/consultations/{id}/notes", h.createNote).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/consultations/{id}/notes", h.listNotes).Methods(http.MethodGet)

	r.HandleFunc("/api/v1/consultations/{id}/prescriptions", h.createPrescription).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/consultations/{id}/prescriptions", h.listPrescriptions).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prescriptions/{id}/status", h.updatePrescriptionStatus).Methods(http.MethodPut)

	r.HandleFunc("/api/v1/consultations/{id}/audit", h.getAuditLog).Methods(http.MethodGet)
}

// ─── Consultation handlers ────────────────────────────────────────────────────

func (h *Handler) createConsultation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor", "nurse", "frontdesk") {
		h.forbidden(w)
		return
	}

	var req models.CreateConsultationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	chiefEnc, err := h.enc.EncryptString(req.ChiefComplaint)
	if err != nil {
		h.internalError(w, err)
		return
	}

	var region *string
	if req.Region != "" {
		region = &req.Region
	}

	row, err := h.queries.CreateConsultation(r.Context(), db.CreateConsultationParams{
		PatientID:         req.PatientID,
		DoctorID:          req.DoctorID,
		ConsultationType:  req.ConsultationType,
		ChiefComplaintEnc: chiefEnc,
		ScheduledAt:       req.ScheduledAt,
		Country:           req.Country,
		Region:            region,
		FacilityID:        req.FacilityID,
		CreatedBy:         claims.UserID,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "consultation", row.ID, row.PatientID, models.AuditCreate, claims, nil)

	consultation, err := h.decryptConsultation(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: consultation})
}

func (h *Handler) listConsultations(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor", "nurse", "analyst", "government") {
		h.forbidden(w)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	params := db.ListConsultationsParams{
		Country: q.Get("country"),
		Page:    page,
		Limit:   limit,
	}
	if id := q.Get("patient_id"); id != "" {
		if uid, err := uuid.Parse(id); err == nil {
			params.PatientID = &uid
		}
	}
	if id := q.Get("doctor_id"); id != "" {
		if uid, err := uuid.Parse(id); err == nil {
			params.DoctorID = &uid
		}
	}
	if s := q.Get("status"); s != "" {
		status := models.ConsultationStatus(s)
		params.Status = &status
	}

	rows, err := h.queries.ListConsultations(r.Context(), params)
	if err != nil {
		h.internalError(w, err)
		return
	}
	total, _ := h.queries.CountConsultations(r.Context(), params)

	var results []*models.Consultation
	for _, row := range rows {
		c, err := h.decryptConsultation(row)
		if err != nil {
			h.internalError(w, err)
			return
		}
		results = append(results, c)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    results,
		Meta: &models.PageMeta{
			Page: page, Limit: limit, Total: total, TotalPages: totalPages,
		},
	})
}

func (h *Handler) getConsultation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	row, err := h.queries.GetConsultationByID(r.Context(), id)
	if err != nil {
		h.notFound(w, "consultation not found")
		return
	}

	if claims.Role == "patient" && row.PatientID != claims.UserID {
		h.forbidden(w)
		return
	}

	h.audit(r, "consultation", row.ID, row.PatientID, models.AuditRead, claims, nil)

	consultation, err := h.decryptConsultation(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: consultation})
}

func (h *Handler) updateConsultation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor", "nurse") {
		h.forbidden(w)
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	var req models.UpdateConsultationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	params := db.UpdateConsultationParams{
		ID:        id,
		Status:    req.Status,
		StartedAt: req.StartedAt,
		EndedAt:   req.EndedAt,
	}

	if req.ChiefComplaint != nil {
		enc, err := h.enc.EncryptString(*req.ChiefComplaint)
		if err != nil {
			h.internalError(w, err)
			return
		}
		params.ChiefComplaint = &enc
	}

	row, err := h.queries.UpdateConsultation(r.Context(), params)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "consultation", row.ID, row.PatientID, models.AuditUpdate, claims, req)

	consultation, err := h.decryptConsultation(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: consultation})
}

// ─── Diagnosis handlers ───────────────────────────────────────────────────────

func (h *Handler) createDiagnosis(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor") {
		h.forbidden(w)
		return
	}

	consultationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	consRow, err := h.queries.GetConsultationByID(r.Context(), consultationID)
	if err != nil {
		h.notFound(w, "consultation not found")
		return
	}

	var req models.CreateDiagnosisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	notesEnc, err := h.enc.EncryptString(req.ClinicalNotes)
	if err != nil {
		h.internalError(w, err)
		return
	}

	row, err := h.queries.CreateDiagnosis(r.Context(), db.CreateDiagnosisParams{
		ConsultationID:   consultationID,
		PatientID:        consRow.PatientID,
		DoctorID:         claims.UserID,
		ICD10Code:        req.ICD10Code,
		ICD10Desc:        req.ICD10Desc,
		ClinicalNotesEnc: notesEnc,
		Severity:         req.Severity,
		IsPrimary:        req.IsPrimary,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "diagnosis", row.ID, row.PatientID, models.AuditCreate, claims, nil)

	diagnosis, err := h.decryptDiagnosis(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: diagnosis})
}

func (h *Handler) listDiagnoses(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	consultationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	rows, err := h.queries.ListDiagnoses(r.Context(), consultationID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	var results []*models.Diagnosis
	for _, row := range rows {
		if claims.Role == "patient" && row.PatientID != claims.UserID {
			continue
		}
		d, err := h.decryptDiagnosis(row)
		if err != nil {
			h.internalError(w, err)
			return
		}
		results = append(results, d)
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: results})
}

// ─── Treatment handlers ───────────────────────────────────────────────────────

func (h *Handler) createTreatment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor", "nurse") {
		h.forbidden(w)
		return
	}

	consultationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	consRow, err := h.queries.GetConsultationByID(r.Context(), consultationID)
	if err != nil {
		h.notFound(w, "consultation not found")
		return
	}

	var req models.CreateTreatmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	instrEnc, err := h.enc.EncryptString(req.Instructions)
	if err != nil {
		h.internalError(w, err)
		return
	}

	row, err := h.queries.CreateTreatment(r.Context(), db.CreateTreatmentParams{
		ConsultationID:  consultationID,
		PatientID:       consRow.PatientID,
		DoctorID:        claims.UserID,
		TreatmentType:   req.TreatmentType,
		InstructionsEnc: instrEnc,
		DurationDays:    req.DurationDays,
		FollowUpDate:    req.FollowUpDate,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "treatment", row.ID, row.PatientID, models.AuditCreate, claims, nil)

	treatment, err := h.decryptTreatment(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: treatment})
}

func (h *Handler) listTreatments(w http.ResponseWriter, r *http.Request) {
	consultationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	rows, err := h.queries.ListTreatments(r.Context(), consultationID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	var results []*models.Treatment
	for _, row := range rows {
		t, err := h.decryptTreatment(row)
		if err != nil {
			h.internalError(w, err)
			return
		}
		results = append(results, t)
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: results})
}

func (h *Handler) updateTreatmentStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor", "nurse") {
		h.forbidden(w)
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid treatment ID")
		return
	}

	var body struct {
		Status models.TreatmentStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	row, err := h.queries.UpdateTreatmentStatus(r.Context(), id, body.Status)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "treatment", row.ID, row.PatientID, models.AuditUpdate, claims, body)

	treatment, err := h.decryptTreatment(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: treatment})
}

// ─── Note handlers ────────────────────────────────────────────────────────────

func (h *Handler) createNote(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor", "nurse") {
		h.forbidden(w)
		return
	}

	consultationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	consRow, err := h.queries.GetConsultationByID(r.Context(), consultationID)
	if err != nil {
		h.notFound(w, "consultation not found")
		return
	}

	var req models.CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	contentEnc, err := h.enc.EncryptString(req.Content)
	if err != nil {
		h.internalError(w, err)
		return
	}

	row, err := h.queries.CreateNote(r.Context(), db.CreateNoteParams{
		ConsultationID: consultationID,
		PatientID:      consRow.PatientID,
		AuthorID:       claims.UserID,
		NoteType:       req.NoteType,
		ContentEnc:     contentEnc,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "clinical_note", row.ID, row.PatientID, models.AuditCreate, claims, nil)

	note, err := h.decryptNote(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: note})
}

func (h *Handler) listNotes(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor", "nurse", "analyst") {
		h.forbidden(w)
		return
	}

	consultationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	rows, err := h.queries.ListNotes(r.Context(), consultationID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	var results []*models.ClinicalNote
	for _, row := range rows {
		n, err := h.decryptNote(row)
		if err != nil {
			h.internalError(w, err)
			return
		}
		results = append(results, n)
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: results})
}

// ─── Prescription handlers ────────────────────────────────────────────────────

func (h *Handler) createPrescription(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor") {
		h.forbidden(w)
		return
	}

	consultationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	consRow, err := h.queries.GetConsultationByID(r.Context(), consultationID)
	if err != nil {
		h.notFound(w, "consultation not found")
		return
	}

	var req models.CreatePrescriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	medsJSON, err := json.Marshal(req.Medications)
	if err != nil {
		h.internalError(w, err)
		return
	}
	medsEnc, err := h.enc.EncryptString(string(medsJSON))
	if err != nil {
		h.internalError(w, err)
		return
	}

	var notesEnc *string
	if req.Notes != "" {
		enc, err := h.enc.EncryptString(req.Notes)
		if err != nil {
			h.internalError(w, err)
			return
		}
		notesEnc = &enc
	}

	row, err := h.queries.CreatePrescription(r.Context(), db.CreatePrescriptionParams{
		ConsultationID: consultationID,
		PatientID:      consRow.PatientID,
		DoctorID:       claims.UserID,
		PharmacyID:     req.PharmacyID,
		MedicationsEnc: medsEnc,
		NotesEnc:       notesEnc,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "prescription", row.ID, row.PatientID, models.AuditCreate, claims, nil)

	prescription, err := h.decryptPrescription(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: prescription})
}

func (h *Handler) listPrescriptions(w http.ResponseWriter, r *http.Request) {
	consultationID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	rows, err := h.queries.ListPrescriptions(r.Context(), consultationID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	var results []*models.Prescription
	for _, row := range rows {
		p, err := h.decryptPrescription(row)
		if err != nil {
			h.internalError(w, err)
			return
		}
		results = append(results, p)
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: results})
}

func (h *Handler) updatePrescriptionStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "doctor", "pharmacist") {
		h.forbidden(w)
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid prescription ID")
		return
	}

	var body struct {
		Status models.PrescriptionStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	row, err := h.queries.UpdatePrescriptionStatus(r.Context(), id, body.Status)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "prescription", row.ID, row.PatientID, models.AuditUpdate, claims, body)

	prescription, err := h.decryptPrescription(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: prescription})
}

// ─── Audit log handler ────────────────────────────────────────────────────────

func (h *Handler) getAuditLog(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "analyst") {
		h.forbidden(w)
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid consultation ID")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	logs, err := h.queries.GetAuditLog(r.Context(), id, limit, (page-1)*limit)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: logs})
}

// ─── Decrypt helpers ──────────────────────────────────────────────────────────

func (h *Handler) decryptConsultation(row *models.ConsultationRow) (*models.Consultation, error) {
	complaint, err := h.enc.DecryptString(row.ChiefComplaintEnc)
	if err != nil {
		return nil, err
	}
	c := &models.Consultation{
		ID:               row.ID,
		PatientID:        row.PatientID,
		DoctorID:         row.DoctorID,
		Status:           row.Status,
		ConsultationType: row.ConsultationType,
		ChiefComplaint:   complaint,
		ScheduledAt:      row.ScheduledAt,
		StartedAt:        row.StartedAt,
		EndedAt:          row.EndedAt,
		Country:          row.Country,
		FacilityID:       row.FacilityID,
		CreatedBy:        row.CreatedBy,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
	if row.Region != nil {
		c.Region = *row.Region
	}
	return c, nil
}

func (h *Handler) decryptDiagnosis(row *models.DiagnosisRow) (*models.Diagnosis, error) {
	notes, err := h.enc.DecryptString(row.ClinicalNotesEnc)
	if err != nil {
		return nil, err
	}
	return &models.Diagnosis{
		ID:             row.ID,
		ConsultationID: row.ConsultationID,
		PatientID:      row.PatientID,
		DoctorID:       row.DoctorID,
		ICD10Code:      row.ICD10Code,
		ICD10Desc:      row.ICD10Desc,
		ClinicalNotes:  notes,
		Severity:       row.Severity,
		IsPrimary:      row.IsPrimary,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

func (h *Handler) decryptTreatment(row *models.TreatmentRow) (*models.Treatment, error) {
	instructions, err := h.enc.DecryptString(row.InstructionsEnc)
	if err != nil {
		return nil, err
	}
	return &models.Treatment{
		ID:             row.ID,
		ConsultationID: row.ConsultationID,
		PatientID:      row.PatientID,
		DoctorID:       row.DoctorID,
		TreatmentType:  row.TreatmentType,
		Instructions:   instructions,
		DurationDays:   row.DurationDays,
		FollowUpDate:   row.FollowUpDate,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

func (h *Handler) decryptNote(row *models.ClinicalNoteRow) (*models.ClinicalNote, error) {
	content, err := h.enc.DecryptString(row.ContentEnc)
	if err != nil {
		return nil, err
	}
	return &models.ClinicalNote{
		ID:             row.ID,
		ConsultationID: row.ConsultationID,
		PatientID:      row.PatientID,
		AuthorID:       row.AuthorID,
		NoteType:       row.NoteType,
		Content:        content,
		CreatedAt:      row.CreatedAt,
	}, nil
}

func (h *Handler) decryptPrescription(row *models.PrescriptionRow) (*models.Prescription, error) {
	medsJSON, err := h.enc.DecryptString(row.MedicationsEnc)
	if err != nil {
		return nil, err
	}
	var meds []models.Medication
	if err := json.Unmarshal([]byte(medsJSON), &meds); err != nil {
		return nil, err
	}

	p := &models.Prescription{
		ID:             row.ID,
		ConsultationID: row.ConsultationID,
		PatientID:      row.PatientID,
		DoctorID:       row.DoctorID,
		PharmacyID:     row.PharmacyID,
		Medications:    meds,
		Status:         row.Status,
		DispensedAt:    row.DispensedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}

	if row.NotesEnc != nil {
		notes, err := h.enc.DecryptString(*row.NotesEnc)
		if err != nil {
			return nil, err
		}
		p.Notes = notes
	}

	return p, nil
}

// ─── Response helpers ─────────────────────────────────────────────────────────

func (h *Handler) json(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func (h *Handler) badRequest(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusBadRequest, models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "BAD_REQUEST", Message: msg},
	})
}

func (h *Handler) forbidden(w http.ResponseWriter) {
	h.json(w, http.StatusForbidden, models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "FORBIDDEN", Message: "insufficient role"},
	})
}

func (h *Handler) notFound(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusNotFound, models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "NOT_FOUND", Message: msg},
	})
}

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	h.logger.Error("internal error", "error", err)
	h.json(w, http.StatusInternalServerError, models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "INTERNAL_ERROR", Message: "an internal error occurred"},
	})
}

// ─── Audit helper ─────────────────────────────────────────────────────────────

func (h *Handler) audit(
	r *http.Request,
	resourceType string,
	resourceID uuid.UUID,
	patientID uuid.UUID,
	action models.AuditAction,
	claims *auth.Claims,
	changes interface{},
) {
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	_ = h.queries.InsertAuditLog(r.Context(), db.InsertAuditParams{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		PatientID:    patientID,
		Action:       action,
		AccessorID:   claims.UserID,
		AccessorRole: claims.Role,
		IPAddress:    ip,
		RequestID:    r.Header.Get("X-Request-ID"),
		Changes:      changes,
	})
}
