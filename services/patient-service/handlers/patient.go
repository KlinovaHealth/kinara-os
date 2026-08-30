// Package handlers implements the REST API for the patient service.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/klinova/kinara-os/patient-service/crypto"
	"github.com/klinova/kinara-os/patient-service/db"
	"github.com/klinova/kinara-os/patient-service/middleware"
	"github.com/klinova/kinara-os/patient-service/models"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	queries   *db.Queries
	enc       *crypto.Encryptor
	logger    *slog.Logger
}

// New constructs a Handler. encryptionKey must be exactly 32 bytes.
func New(queries *db.Queries, encryptionKey []byte, logger *slog.Logger) *Handler {
	enc, err := crypto.New(encryptionKey)
	if err != nil {
		panic("handlers: invalid encryption key: " + err.Error())
	}
	return &Handler{queries: queries, enc: enc, logger: logger}
}

// Register mounts all patient routes under /api/v1/patients.
func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/patients").Subrouter()
	api.HandleFunc("", h.CreatePatient).Methods(http.MethodPost)
	api.HandleFunc("", h.ListPatients).Methods(http.MethodGet)
	api.HandleFunc("/{id}", h.GetPatient).Methods(http.MethodGet)
	api.HandleFunc("/{id}", h.UpdatePatient).Methods(http.MethodPut)
	api.HandleFunc("/{id}", h.DeletePatient).Methods(http.MethodDelete)
	api.HandleFunc("/{id}/audit", h.GetAuditLog).Methods(http.MethodGet)
}

// ─── POST /api/v1/patients ────────────────────────────────────────────────────

func (h *Handler) CreatePatient(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}
	if !claims.IsAllowedRole("admin", "nurse", "doctor", "frontdesk") {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "role not permitted to create patients")
		return
	}

	var req models.CreatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "request body is not valid JSON")
		return
	}

	if err := validateCreateRequest(&req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_DATE", "date_of_birth must be YYYY-MM-DD")
		return
	}

	// Encrypt all PHI before touching the database.
	nationalIDEnc, err := h.enc.EncryptString(req.NationalID)
	if err != nil {
		h.logger.Error("encrypt national_id failed", "error", err)
		respondError(w, http.StatusInternalServerError, "ENCRYPT_ERROR", "failed to encrypt patient data")
		return
	}
	fullNameEnc, err := h.enc.EncryptString(req.FullName)
	if err != nil {
		h.internalError(w, "encrypt full_name", err)
		return
	}
	dobEnc, err := h.enc.EncryptString(dob.Format(time.RFC3339))
	if err != nil {
		h.internalError(w, "encrypt dob", err)
		return
	}
	phoneEnc, err := h.enc.EncryptString(req.PhoneNumber)
	if err != nil {
		h.internalError(w, "encrypt phone", err)
		return
	}

	emailEnc, err := h.enc.EncryptOptional(req.Email)
	if err != nil {
		h.internalError(w, "encrypt email", err)
		return
	}
	addrEnc, err := h.enc.EncryptOptional(req.Address)
	if err != nil {
		h.internalError(w, "encrypt address", err)
		return
	}
	btEnc, err := h.enc.EncryptOptional(req.BloodType)
	if err != nil {
		h.internalError(w, "encrypt blood_type", err)
		return
	}

	var allergiesEnc, ecNameEnc, ecPhoneEnc, ecRelEnc *string

	if len(req.Allergies) > 0 {
		raw, _ := json.Marshal(req.Allergies)
		enc, err := h.enc.EncryptString(string(raw))
		if err != nil {
			h.internalError(w, "encrypt allergies", err)
			return
		}
		allergiesEnc = &enc
	}

	if req.EmergencyContact.Name != "" {
		enc, err := h.enc.EncryptString(req.EmergencyContact.Name)
		if err != nil {
			h.internalError(w, "encrypt ec_name", err)
			return
		}
		ecNameEnc = &enc
	}
	if req.EmergencyContact.Phone != "" {
		enc, err := h.enc.EncryptString(req.EmergencyContact.Phone)
		if err != nil {
			h.internalError(w, "encrypt ec_phone", err)
			return
		}
		ecPhoneEnc = &enc
	}
	if req.EmergencyContact.Relationship != "" {
		enc, err := h.enc.EncryptString(req.EmergencyContact.Relationship)
		if err != nil {
			h.internalError(w, "encrypt ec_rel", err)
			return
		}
		ecRelEnc = &enc
	}

	optStr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	optEncStr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}

	params := db.CreatePatientParams{
		NationalIDEnc:            nationalIDEnc,
		FullNameEnc:              fullNameEnc,
		DateOfBirthEnc:           dobEnc,
		Gender:                   req.Gender,
		PhoneNumberEnc:           phoneEnc,
		EmailEnc:                 optEncStr(emailEnc),
		AddressEnc:               optEncStr(addrEnc),
		Country:                  req.Country,
		Region:                   optStr(req.Region),
		BloodTypeEnc:             optEncStr(btEnc),
		AllergiesEnc:             allergiesEnc,
		EmergencyContactNameEnc:  ecNameEnc,
		EmergencyContactPhoneEnc: ecPhoneEnc,
		EmergencyContactRelEnc:   ecRelEnc,
		Status:                   models.StatusActive,
		CreatedBy:                claims.UserID,
	}

	row, err := h.queries.CreatePatient(r.Context(), params)
	if err != nil {
		h.internalError(w, "create patient", err)
		return
	}

	patient, err := h.decryptRow(row)
	if err != nil {
		h.internalError(w, "decrypt patient", err)
		return
	}

	// Immutable audit log entry.
	_ = h.queries.InsertAuditLog(r.Context(), db.InsertAuditLogParams{
		PatientID:    patient.ID,
		Action:       models.AuditCreate,
		AccessorID:   claims.UserID,
		AccessorRole: claims.Role,
		IPAddress:    r.RemoteAddr,
		RequestID:    requestID(r),
	})

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    patient,
	})
}

// ─── GET /api/v1/patients/:id ─────────────────────────────────────────────────

func (h *Handler) GetPatient(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}

	id, err := parseUUID(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "patient id must be a valid UUID")
		return
	}

	row, err := h.queries.GetPatientByID(r.Context(), id)
	if err != nil {
		h.internalError(w, "get patient", err)
		return
	}
	if row == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("patient %s not found", id))
		return
	}

	// Patients can only see themselves; privileged roles see all.
	if claims.Role == "patient" && row.CreatedBy != claims.UserID {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "patients may only access their own record")
		return
	}

	patient, err := h.decryptRow(row)
	if err != nil {
		h.internalError(w, "decrypt patient", err)
		return
	}

	_ = h.queries.InsertAuditLog(r.Context(), db.InsertAuditLogParams{
		PatientID:    id,
		Action:       models.AuditRead,
		AccessorID:   claims.UserID,
		AccessorRole: claims.Role,
		IPAddress:    r.RemoteAddr,
		RequestID:    requestID(r),
	})

	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: patient})
}

// ─── GET /api/v1/patients ─────────────────────────────────────────────────────

func (h *Handler) ListPatients(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}
	if !claims.IsAllowedRole("admin", "doctor", "nurse", "analyst", "government") {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "role not permitted to list patients")
		return
	}

	q := r.URL.Query()
	page := queryInt(q.Get("page"), 1)
	limit := queryInt(q.Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	params := db.ListPatientsParams{
		Country: q.Get("country"),
		Region:  q.Get("region"),
		Status:  q.Get("status"),
		Limit:   limit,
		Offset:  offset,
	}

	rows, err := h.queries.ListPatients(r.Context(), params)
	if err != nil {
		h.internalError(w, "list patients", err)
		return
	}
	total, err := h.queries.CountPatients(r.Context(), params)
	if err != nil {
		h.internalError(w, "count patients", err)
		return
	}

	patients := make([]*models.Patient, 0, len(rows))
	for _, row := range rows {
		p, err := h.decryptRow(row)
		if err != nil {
			h.internalError(w, "decrypt patient list item", err)
			return
		}
		patients = append(patients, p)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    patients,
		Meta: &models.PageMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// ─── PUT /api/v1/patients/:id ─────────────────────────────────────────────────

func (h *Handler) UpdatePatient(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}
	if !claims.IsAllowedRole("admin", "nurse", "doctor") {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "role not permitted to update patients")
		return
	}

	id, err := parseUUID(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "patient id must be a valid UUID")
		return
	}

	var req models.UpdatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "request body is not valid JSON")
		return
	}

	params := db.UpdatePatientParams{ID: id}

	if req.PhoneNumber != nil {
		enc, err := h.enc.EncryptString(*req.PhoneNumber)
		if err != nil {
			h.internalError(w, "encrypt phone", err)
			return
		}
		params.PhoneNumberEnc = &enc
	}
	if req.Email != nil {
		enc, err := h.enc.EncryptOptional(*req.Email)
		if err != nil {
			h.internalError(w, "encrypt email", err)
			return
		}
		params.EmailEnc = &enc
	}
	if req.Address != nil {
		enc, err := h.enc.EncryptOptional(*req.Address)
		if err != nil {
			h.internalError(w, "encrypt address", err)
			return
		}
		params.AddressEnc = &enc
	}
	if req.Region != nil {
		params.Region = req.Region
	}
	if req.BloodType != nil {
		enc, err := h.enc.EncryptOptional(*req.BloodType)
		if err != nil {
			h.internalError(w, "encrypt blood_type", err)
			return
		}
		params.BloodTypeEnc = &enc
	}
	if len(req.Allergies) > 0 {
		raw, _ := json.Marshal(req.Allergies)
		enc, err := h.enc.EncryptString(string(raw))
		if err != nil {
			h.internalError(w, "encrypt allergies", err)
			return
		}
		params.AllergiesEnc = &enc
	}
	if req.EmergencyContact != nil {
		if req.EmergencyContact.Name != "" {
			enc, _ := h.enc.EncryptString(req.EmergencyContact.Name)
			params.EmergencyContactNameEnc = &enc
		}
		if req.EmergencyContact.Phone != "" {
			enc, _ := h.enc.EncryptString(req.EmergencyContact.Phone)
			params.EmergencyContactPhoneEnc = &enc
		}
		if req.EmergencyContact.Relationship != "" {
			enc, _ := h.enc.EncryptString(req.EmergencyContact.Relationship)
			params.EmergencyContactRelEnc = &enc
		}
	}
	if req.Status != nil {
		s := string(*req.Status)
		params.Status = &s
	}

	row, err := h.queries.UpdatePatient(r.Context(), params)
	if err != nil {
		h.internalError(w, "update patient", err)
		return
	}
	if row == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("patient %s not found", id))
		return
	}

	patient, err := h.decryptRow(row)
	if err != nil {
		h.internalError(w, "decrypt updated patient", err)
		return
	}

	_ = h.queries.InsertAuditLog(r.Context(), db.InsertAuditLogParams{
		PatientID:    id,
		Action:       models.AuditUpdate,
		AccessorID:   claims.UserID,
		AccessorRole: claims.Role,
		IPAddress:    r.RemoteAddr,
		RequestID:    requestID(r),
	})

	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: patient})
}

// ─── DELETE /api/v1/patients/:id ──────────────────────────────────────────────

func (h *Handler) DeletePatient(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}
	if !claims.IsAllowedRole("admin") {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "only admins may delete patient records")
		return
	}

	id, err := parseUUID(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "patient id must be a valid UUID")
		return
	}

	deletedID, err := h.queries.SoftDeletePatient(r.Context(), id)
	if err != nil {
		h.internalError(w, "delete patient", err)
		return
	}
	if deletedID == uuid.Nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("patient %s not found", id))
		return
	}

	_ = h.queries.InsertAuditLog(r.Context(), db.InsertAuditLogParams{
		PatientID:    id,
		Action:       models.AuditDelete,
		AccessorID:   claims.UserID,
		AccessorRole: claims.Role,
		IPAddress:    r.RemoteAddr,
		RequestID:    requestID(r),
	})

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]string{"id": id.String(), "status": "deleted"},
	})
}

// ─── GET /api/v1/patients/:id/audit ──────────────────────────────────────────

func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}
	if !claims.IsAllowedRole("admin", "analyst") {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "only admins and analysts may view audit logs")
		return
	}

	id, err := parseUUID(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "patient id must be a valid UUID")
		return
	}

	page := queryInt(r.URL.Query().Get("page"), 1)
	limit := queryInt(r.URL.Query().Get("limit"), 50)
	if limit > 200 {
		limit = 200
	}

	logs, err := h.queries.GetAuditLog(r.Context(), id, limit, (page-1)*limit)
	if err != nil {
		h.internalError(w, "get audit log", err)
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: logs})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// decryptRow converts a PatientRow (all encrypted) to a Patient (plaintext).
func (h *Handler) decryptRow(row *models.PatientRow) (*models.Patient, error) {
	fullName, err := h.enc.DecryptString(row.FullNameEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt full_name: %w", err)
	}
	nationalID, err := h.enc.DecryptString(row.NationalIDEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt national_id: %w", err)
	}
	dobStr, err := h.enc.DecryptString(row.DateOfBirthEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt dob: %w", err)
	}
	dob, _ := time.Parse(time.RFC3339, dobStr)

	phone, err := h.enc.DecryptString(row.PhoneNumberEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt phone: %w", err)
	}

	email, _ := h.enc.DecryptOptional(row.EmailEnc)
	address, _ := h.enc.DecryptOptional(row.AddressEnc)
	bloodType, _ := h.enc.DecryptOptional(row.BloodTypeEnc)

	var allergies []string
	if row.AllergiesEnc != nil {
		plain, err := h.enc.DecryptString(*row.AllergiesEnc)
		if err == nil {
			_ = json.Unmarshal([]byte(plain), &allergies)
		}
	}

	var ec models.EmergencyContact
	if row.EmergencyContactNameEnc != nil {
		ec.Name, _ = h.enc.DecryptString(*row.EmergencyContactNameEnc)
	}
	if row.EmergencyContactPhoneEnc != nil {
		ec.Phone, _ = h.enc.DecryptString(*row.EmergencyContactPhoneEnc)
	}
	if row.EmergencyContactRelEnc != nil {
		ec.Relationship, _ = h.enc.DecryptString(*row.EmergencyContactRelEnc)
	}

	region := ""
	if row.Region != nil {
		region = *row.Region
	}

	return &models.Patient{
		ID:               row.ID,
		NationalID:       nationalID,
		FullName:         fullName,
		DateOfBirth:      dob,
		Gender:           row.Gender,
		PhoneNumber:      phone,
		Email:            email,
		Address:          address,
		Country:          row.Country,
		Region:           region,
		BloodType:        bloodType,
		Allergies:        allergies,
		EmergencyContact: ec,
		Status:           row.Status,
		CreatedBy:        row.CreatedBy,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (h *Handler) internalError(w http.ResponseWriter, op string, err error) {
	h.logger.Error("internal error", "op", op, "error", err)
	respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
}

func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: code, Message: message},
	})
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func queryInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	return n
}

func requestID(r *http.Request) string {
	return r.Header.Get("X-Request-ID")
}

func validateCreateRequest(req *models.CreatePatientRequest) error {
	if req.NationalID == "" {
		return fmt.Errorf("national_id is required")
	}
	if req.FullName == "" {
		return fmt.Errorf("full_name is required")
	}
	if req.DateOfBirth == "" {
		return fmt.Errorf("date_of_birth is required")
	}
	if req.PhoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}
	if req.Country == "" {
		return fmt.Errorf("country is required")
	}
	switch req.Gender {
	case models.GenderMale, models.GenderFemale, models.GenderOther, models.GenderPreferNotSay:
	default:
		return fmt.Errorf("gender must be one of: male, female, other, prefer_not_to_say")
	}
	return nil
}
