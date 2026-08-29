package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/referral-service/auth"
	"github.com/klinova/kinara-os/referral-service/crypto"
	"github.com/klinova/kinara-os/referral-service/db"
	"github.com/klinova/kinara-os/referral-service/middleware"
	"github.com/klinova/kinara-os/referral-service/models"
)

type Handler struct {
	queries *db.Queries
	enc     *crypto.Encryptor
	logger  *slog.Logger
}

func New(queries *db.Queries, enc *crypto.Encryptor, logger *slog.Logger) *Handler {
	return &Handler{queries: queries, enc: enc, logger: logger}
}

func (h *Handler) Register(r *mux.Router, jwtMW func(http.Handler) http.Handler) {
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(jwtMW)

	api.HandleFunc("/referrals", h.createReferral).Methods(http.MethodPost)
	api.HandleFunc("/referrals", h.listReferrals).Methods(http.MethodGet)
	api.HandleFunc("/referrals/{id}", h.getReferral).Methods(http.MethodGet)
	api.HandleFunc("/referrals/{id}/status", h.updateStatus).Methods(http.MethodPut)
	api.HandleFunc("/referrals/{id}/notes", h.addNote).Methods(http.MethodPost)
	api.HandleFunc("/referrals/{id}/notes", h.listNotes).Methods(http.MethodGet)
	api.HandleFunc("/referrals/{id}/follow-up", h.scheduleFollowUp).Methods(http.MethodPost)
	api.HandleFunc("/referrals/{id}/history", h.listHistory).Methods(http.MethodGet)
}

// ─── Create referral ──────────────────────────────────────────────────────────

func (h *Handler) createReferral(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "doctor", "nurse", "clinician") {
		h.forbidden(w)
		return
	}

	var req models.CreateReferralRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.PatientID == "" || req.ToClinicID == "" || req.Reason == "" || req.PatientName == "" {
		h.badRequest(w, "patient_id, to_clinic_id, patient_name, and reason are required")
		return
	}
	if !validUrgency(req.Urgency) {
		req.Urgency = models.UrgencyRoutine
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		h.badRequest(w, "invalid patient_id")
		return
	}
	toClinicID, err := uuid.Parse(req.ToClinicID)
	if err != nil {
		h.badRequest(w, "invalid to_clinic_id")
		return
	}

	reasonEnc, err := h.enc.EncryptString(req.Reason)
	if err != nil {
		h.internalError(w, "encryption error")
		return
	}
	patientNameEnc, err := h.enc.EncryptString(req.PatientName)
	if err != nil {
		h.internalError(w, "encryption error")
		return
	}

	now := time.Now().UTC()
	fromClinicID := uuid.New() // In production, resolved from facility registry via claims.UserID
	row := models.ReferralRow{
		ID:             uuid.New(),
		PatientID:      patientID,
		FromClinicID:   fromClinicID,
		ToClinicID:     toClinicID,
		FromClinicianID: claims.UserID,
		ReasonEnc:      reasonEnc,
		PatientNameEnc: patientNameEnc,
		Urgency:        req.Urgency,
		Status:         models.ReferralPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := h.queries.CreateReferral(r.Context(), row); err != nil {
		h.logger.Error("create referral failed", "error", err)
		h.internalError(w, "failed to create referral")
		return
	}

	h.audit(r, claims, &row.ID, "create_referral", "referrals")

	ref := h.decryptReferral(row)
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: ref})
}

// ─── Get referral ─────────────────────────────────────────────────────────────

func (h *Handler) getReferral(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "doctor", "nurse", "clinician", "frontdesk") {
		h.forbidden(w)
		return
	}

	id, err := parseID(r)
	if err != nil {
		h.badRequest(w, "invalid referral id")
		return
	}

	row, err := h.queries.GetReferral(r.Context(), id)
	if err != nil {
		h.notFound(w, "referral not found")
		return
	}

	h.audit(r, claims, &id, "get_referral", "referrals")
	ref := h.decryptReferral(*row)
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: ref})
}

// ─── List referrals ───────────────────────────────────────────────────────────

func (h *Handler) listReferrals(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "doctor", "nurse", "clinician", "frontdesk", "facility_admin") {
		h.forbidden(w)
		return
	}

	params := db.ListReferralsParams{
		Page:  pageParam(r, 1),
		Limit: limitParam(r, 50),
	}

	if pid := r.URL.Query().Get("patient_id"); pid != "" {
		id, err := uuid.Parse(pid)
		if err != nil {
			h.badRequest(w, "invalid patient_id")
			return
		}
		params.PatientID = &id
	}
	if cid := r.URL.Query().Get("clinic_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			h.badRequest(w, "invalid clinic_id")
			return
		}
		params.ClinicID = &id
	}
	if s := r.URL.Query().Get("status"); s != "" {
		st := models.ReferralStatus(s)
		params.Status = &st
	}

	rows, err := h.queries.ListReferrals(r.Context(), params)
	if err != nil {
		h.logger.Error("list referrals failed", "error", err)
		h.internalError(w, "failed to list referrals")
		return
	}
	total, _ := h.queries.CountReferrals(r.Context(), params)

	refs := make([]models.Referral, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, h.decryptReferral(row))
	}

	h.audit(r, claims, nil, "list_referrals", "referrals")
	totalPages := (total + params.Limit - 1) / params.Limit
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    refs,
		Meta:    &models.PageMeta{Page: params.Page, Limit: params.Limit, Total: total, TotalPages: totalPages},
	})
}

// ─── Update status ────────────────────────────────────────────────────────────

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "doctor", "clinician", "facility_admin") {
		h.forbidden(w)
		return
	}

	id, err := parseID(r)
	if err != nil {
		h.badRequest(w, "invalid referral id")
		return
	}

	var req models.UpdateReferralStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if !validStatus(req.Status) {
		h.badRequest(w, "invalid status")
		return
	}

	current, err := h.queries.GetReferral(r.Context(), id)
	if err != nil {
		h.notFound(w, "referral not found")
		return
	}
	if !validTransition(current.Status, req.Status) {
		h.badRequest(w, "invalid status transition from "+string(current.Status)+" to "+string(req.Status))
		return
	}

	now := time.Now().UTC()

	var toClinicianID *uuid.UUID
	if req.ToClinicianID != nil {
		uid, err := uuid.Parse(*req.ToClinicianID)
		if err != nil {
			h.badRequest(w, "invalid to_clinician_id")
			return
		}
		toClinicianID = &uid
	}

	var rejectionEnc *string
	if req.RejectionReason != nil {
		enc, err := h.enc.EncryptString(*req.RejectionReason)
		if err != nil {
			h.internalError(w, "encryption error")
			return
		}
		rejectionEnc = &enc
	}

	p := db.UpdateStatusParams{
		ID:              id,
		Status:          req.Status,
		ToClinicianID:   toClinicianID,
		RejectionReason: rejectionEnc,
		Now:             now,
	}
	if err := h.queries.UpdateReferralStatus(r.Context(), p); err != nil {
		h.logger.Error("update referral status failed", "error", err)
		h.internalError(w, "failed to update status")
		return
	}

	before := string(current.Status)
	h.queries.InsertHistory(r.Context(), models.ReferralHistory{
		ID:              uuid.New(),
		ReferralID:      id,
		StatusBefore:    &before,
		StatusAfter:     string(req.Status),
		ChangedByUserID: claims.UserID,
		ChangedByRole:   claims.Role,
		Notes:           req.Notes,
		CreatedAt:       now,
	})

	h.audit(r, claims, &id, "update_status", "referrals")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]string{"status": string(req.Status)}})
}

// ─── Add note ─────────────────────────────────────────────────────────────────

func (h *Handler) addNote(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "doctor", "nurse", "clinician") {
		h.forbidden(w)
		return
	}

	id, err := parseID(r)
	if err != nil {
		h.badRequest(w, "invalid referral id")
		return
	}
	if _, err := h.queries.GetReferral(r.Context(), id); err != nil {
		h.notFound(w, "referral not found")
		return
	}

	var req models.AddNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Note == "" {
		h.badRequest(w, "note is required")
		return
	}

	noteEnc, err := h.enc.EncryptString(req.Note)
	if err != nil {
		h.internalError(w, "encryption error")
		return
	}

	now := time.Now().UTC()
	row := models.ReferralNoteRow{
		ID:              uuid.New(),
		ReferralID:      id,
		NoteEnc:         noteEnc,
		CreatedByUserID: claims.UserID,
		CreatedAt:       now,
	}
	if err := h.queries.CreateNote(r.Context(), row); err != nil {
		h.logger.Error("create note failed", "error", err)
		h.internalError(w, "failed to create note")
		return
	}

	h.audit(r, claims, &id, "add_note", "referral_notes")
	note := models.ReferralNote{
		ID:              row.ID,
		ReferralID:      row.ReferralID,
		Note:            req.Note,
		CreatedByUserID: row.CreatedByUserID,
		CreatedAt:       row.CreatedAt,
	}
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: note})
}

// ─── List notes ───────────────────────────────────────────────────────────────

func (h *Handler) listNotes(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "doctor", "nurse", "clinician", "frontdesk") {
		h.forbidden(w)
		return
	}

	id, err := parseID(r)
	if err != nil {
		h.badRequest(w, "invalid referral id")
		return
	}

	rows, err := h.queries.ListNotes(r.Context(), id)
	if err != nil {
		h.internalError(w, "failed to list notes")
		return
	}

	notes := make([]models.ReferralNote, 0, len(rows))
	for _, row := range rows {
		note, err := h.enc.DecryptString(row.NoteEnc)
		if err != nil {
			h.logger.Error("decrypt note failed", "id", row.ID)
			continue
		}
		notes = append(notes, models.ReferralNote{
			ID:              row.ID,
			ReferralID:      row.ReferralID,
			Note:            note,
			CreatedByUserID: row.CreatedByUserID,
			CreatedAt:       row.CreatedAt,
		})
	}

	h.audit(r, claims, &id, "list_notes", "referral_notes")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: notes})
}

// ─── Schedule follow-up ───────────────────────────────────────────────────────

func (h *Handler) scheduleFollowUp(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "doctor", "clinician") {
		h.forbidden(w)
		return
	}

	id, err := parseID(r)
	if err != nil {
		h.badRequest(w, "invalid referral id")
		return
	}
	if _, err := h.queries.GetReferral(r.Context(), id); err != nil {
		h.notFound(w, "referral not found")
		return
	}

	var req models.ScheduleFollowUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FollowUpDate == "" {
		h.badRequest(w, "follow_up_date is required")
		return
	}
	followUpDate, err := time.Parse(time.RFC3339, req.FollowUpDate)
	if err != nil {
		h.badRequest(w, "follow_up_date must be RFC3339 format")
		return
	}

	var notesEnc *string
	if req.Notes != nil {
		enc, err := h.enc.EncryptString(*req.Notes)
		if err != nil {
			h.internalError(w, "encryption error")
			return
		}
		notesEnc = &enc
	}

	p := db.FollowUpParams{
		ID:           id,
		FollowUpDate: followUpDate.UTC(),
		NotesEnc:     notesEnc,
		Now:          time.Now().UTC(),
	}
	if err := h.queries.ScheduleFollowUp(r.Context(), p); err != nil {
		h.logger.Error("schedule follow-up failed", "error", err)
		h.internalError(w, "failed to schedule follow-up")
		return
	}

	h.audit(r, claims, &id, "schedule_follow_up", "referrals")
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]string{"follow_up_date": followUpDate.UTC().Format(time.RFC3339)},
	})
}

// ─── List history ─────────────────────────────────────────────────────────────

func (h *Handler) listHistory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "doctor", "nurse", "clinician", "frontdesk", "facility_admin") {
		h.forbidden(w)
		return
	}

	id, err := parseID(r)
	if err != nil {
		h.badRequest(w, "invalid referral id")
		return
	}

	history, err := h.queries.ListHistory(r.Context(), id)
	if err != nil {
		h.internalError(w, "failed to list history")
		return
	}

	h.audit(r, claims, &id, "list_history", "referral_history")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: history})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) decryptReferral(row models.ReferralRow) models.Referral {
	reason, _ := h.enc.DecryptString(row.ReasonEnc)
	patientName, _ := h.enc.DecryptString(row.PatientNameEnc)
	var followUpNotes *string
	if row.FollowUpNotesEnc != nil {
		dec, err := h.enc.DecryptString(*row.FollowUpNotesEnc)
		if err == nil {
			followUpNotes = &dec
		}
	}
	var rejectionReason *string
	if row.RejectionReasonEnc != nil {
		dec, err := h.enc.DecryptString(*row.RejectionReasonEnc)
		if err == nil {
			rejectionReason = &dec
		}
	}
	return models.Referral{
		ID:              row.ID,
		PatientID:       row.PatientID,
		FromClinicID:    row.FromClinicID,
		ToClinicID:      row.ToClinicID,
		FromClinicianID: row.FromClinicianID,
		ToClinicianID:   row.ToClinicianID,
		Reason:          reason,
		PatientName:     patientName,
		Urgency:         row.Urgency,
		Status:          row.Status,
		FollowUpDate:    row.FollowUpDate,
		FollowUpNotes:   followUpNotes,
		AcceptedAt:      row.AcceptedAt,
		CompletedAt:     row.CompletedAt,
		RejectedAt:      row.RejectedAt,
		RejectionReason: rejectionReason,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (h *Handler) audit(r *http.Request, claims *auth.Claims, referralID *uuid.UUID, action, resource string) {
	h.queries.InsertAuditLog(r.Context(), models.ReferralAuditLog{
		ID:         uuid.New(),
		ReferralID: referralID,
		UserID:     claims.UserID,
		Action:     action,
		Resource:   resource,
		IPAddress:  remoteIP(r),
		CreatedAt:  time.Now().UTC(),
	})
}

func (h *Handler) json(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
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

func (h *Handler) internalError(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusInternalServerError, models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "INTERNAL_ERROR", Message: msg},
	})
}

func roleAllowed(claims *auth.Claims, roles ...string) bool {
	if claims == nil {
		return false
	}
	for _, r := range roles {
		if claims.Role == r {
			return true
		}
	}
	return false
}

func parseID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(mux.Vars(r)["id"])
}

func pageParam(r *http.Request, def int) int {
	p, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || p < 1 {
		return def
	}
	return p
}

func limitParam(r *http.Request, def int) int {
	l, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || l < 1 || l > 200 {
		return def
	}
	return l
}

func remoteIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}

func validUrgency(u models.ReferralUrgency) bool {
	switch u {
	case models.UrgencyRoutine, models.UrgencySemiUrgent, models.UrgencyUrgent, models.UrgencyEmergency:
		return true
	}
	return false
}

func validStatus(s models.ReferralStatus) bool {
	switch s {
	case models.ReferralPending, models.ReferralAccepted, models.ReferralInProgress,
		models.ReferralCompleted, models.ReferralRejected, models.ReferralCancelled:
		return true
	}
	return false
}

// validTransition enforces allowed status state machine transitions.
func validTransition(from, to models.ReferralStatus) bool {
	allowed := map[models.ReferralStatus][]models.ReferralStatus{
		models.ReferralPending:    {models.ReferralAccepted, models.ReferralRejected, models.ReferralCancelled},
		models.ReferralAccepted:   {models.ReferralInProgress, models.ReferralCancelled},
		models.ReferralInProgress: {models.ReferralCompleted, models.ReferralCancelled},
	}
	for _, a := range allowed[from] {
		if a == to {
			return true
		}
	}
	return false
}
