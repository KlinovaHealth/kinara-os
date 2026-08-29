package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/governance-service/auth"
	"github.com/klinova/kinara-os/governance-service/crypto"
	"github.com/klinova/kinara-os/governance-service/db"
	"github.com/klinova/kinara-os/governance-service/middleware"
	"github.com/klinova/kinara-os/governance-service/models"
)

type Handler struct {
	queries *db.Queries
	enc     *crypto.Encryptor
	logger  *slog.Logger
}

func New(queries *db.Queries, enc *crypto.Encryptor, logger *slog.Logger) *Handler {
	return &Handler{queries: queries, enc: enc, logger: logger}
}

func (h *Handler) Register(r *mux.Router) {
	// Compliance reports
	r.HandleFunc("/api/v1/compliance-reports", h.createComplianceReport).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/compliance-reports", h.listComplianceReports).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/compliance-reports/{id}", h.getComplianceReport).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/compliance-reports/{id}/review", h.reviewComplianceReport).Methods(http.MethodPut)

	// Epidemiology
	r.HandleFunc("/api/v1/epidemiology", h.createEpidemiologyRecord).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/epidemiology", h.listEpidemiologyRecords).Methods(http.MethodGet)

	// Coordination rules
	r.HandleFunc("/api/v1/rules", h.createRule).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/rules", h.listRules).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/rules/{id}/deactivate", h.deactivateRule).Methods(http.MethodPut)

	// Alerts
	r.HandleFunc("/api/v1/alerts", h.createAlert).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/alerts", h.listAlerts).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/alerts/{id}/status", h.updateAlertStatus).Methods(http.MethodPut)
}

// ─── Compliance Report handlers ───────────────────────────────────────────────

func (h *Handler) createComplianceReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official", "facility_admin") {
		h.forbidden(w)
		return
	}

	var req models.CreateComplianceReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	payloadJSON, err := json.Marshal(req.DataPayload)
	if err != nil {
		h.internalError(w, err)
		return
	}
	payloadEnc, err := h.enc.EncryptString(string(payloadJSON))
	if err != nil {
		h.internalError(w, err)
		return
	}

	var region *string
	if req.Region != "" {
		region = &req.Region
	}

	row, err := h.queries.CreateComplianceReport(r.Context(), db.CreateComplianceReportParams{
		FacilityID:     req.FacilityID,
		MinistryID:     req.MinistryID,
		ReportType:     req.ReportType,
		Frequency:      req.Frequency,
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		Country:        req.Country,
		Region:         region,
		Summary:        req.Summary,
		DataPayloadEnc: payloadEnc,
		SubmittedBy:    claims.UserID,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "compliance_report", row.ID, models.AuditCreate, claims, nil)

	report, err := h.decryptReport(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: report})
}

func (h *Handler) listComplianceReports(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official", "analyst", "facility_admin") {
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

	params := db.ListComplianceReportsParams{
		Country: q.Get("country"),
		Page:    page,
		Limit:   limit,
	}
	if s := q.Get("status"); s != "" {
		status := models.ComplianceStatus(s)
		params.Status = &status
	}
	if rt := q.Get("report_type"); rt != "" {
		rtype := models.ReportType(rt)
		params.ReportType = &rtype
	}

	rows, err := h.queries.ListComplianceReports(r.Context(), params)
	if err != nil {
		h.internalError(w, err)
		return
	}
	total, _ := h.queries.CountComplianceReports(r.Context(), params)

	var results []*models.ComplianceReport
	for _, row := range rows {
		rep, err := h.decryptReport(row)
		if err != nil {
			h.internalError(w, err)
			return
		}
		results = append(results, rep)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	h.json(w, http.StatusOK, models.APIResponse{
		Success: true, Data: results,
		Meta: &models.PageMeta{Page: page, Limit: limit, Total: total, TotalPages: totalPages},
	})
}

func (h *Handler) getComplianceReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid report ID")
		return
	}

	row, err := h.queries.GetComplianceReportByID(r.Context(), id)
	if err != nil {
		h.notFound(w, "compliance report not found")
		return
	}

	h.audit(r, "compliance_report", row.ID, models.AuditRead, claims, nil)

	report, err := h.decryptReport(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: report})
}

func (h *Handler) reviewComplianceReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official") {
		h.forbidden(w)
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid report ID")
		return
	}

	var req models.ReviewComplianceReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	notesEnc, err := h.enc.EncryptOptional(req.ViolationNotes)
	if err != nil {
		h.internalError(w, err)
		return
	}

	row, err := h.queries.ReviewComplianceReport(r.Context(), db.ReviewComplianceReportParams{
		ID:                id,
		Status:            req.Status,
		ReviewedBy:        claims.UserID,
		ViolationNotesEnc: notesEnc,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "compliance_report", row.ID, models.AuditUpdate, claims, req)

	report, err := h.decryptReport(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: report})
}

// ─── Epidemiology handlers ────────────────────────────────────────────────────

func (h *Handler) createEpidemiologyRecord(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official", "analyst", "doctor") {
		h.forbidden(w)
		return
	}

	var req models.CreateEpidemiologyRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	params := db.CreateEpidemiologyParams{
		ICD10Code:      req.ICD10Code,
		ICD10Desc:      req.ICD10Desc,
		Country:        req.Country,
		CaseCount:      req.CaseCount,
		DeathCount:     req.DeathCount,
		RecoveredCount: req.RecoveredCount,
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		FacilityID:     req.FacilityID,
		ReportedBy:     claims.UserID,
	}
	if req.Region != "" {
		params.Region = &req.Region
	}
	if req.District != "" {
		params.District = &req.District
	}
	if req.AgeGroup != "" {
		params.AgeGroup = &req.AgeGroup
	}
	if req.Gender != "" {
		params.Gender = &req.Gender
	}

	record, err := h.queries.CreateEpidemiologyRecord(r.Context(), params)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "epidemiology_record", record.ID, models.AuditCreate, claims, nil)
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: record})
}

func (h *Handler) listEpidemiologyRecords(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official", "analyst", "doctor") {
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

	records, err := h.queries.ListEpidemiologyRecords(r.Context(), db.ListEpidemiologyParams{
		Country:   q.Get("country"),
		ICD10Code: q.Get("icd10_code"),
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: records})
}

// ─── Coordination Rule handlers ───────────────────────────────────────────────

func (h *Handler) createRule(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government") {
		h.forbidden(w)
		return
	}

	var req models.CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	paramsJSON, err := json.Marshal(req.Parameters)
	if err != nil {
		h.internalError(w, err)
		return
	}
	paramsEnc, err := h.enc.EncryptString(string(paramsJSON))
	if err != nil {
		h.internalError(w, err)
		return
	}

	var region *string
	if req.Region != "" {
		region = &req.Region
	}

	row, err := h.queries.CreateRule(r.Context(), db.CreateRuleParams{
		RuleType:       req.RuleType,
		Name:           req.Name,
		Description:    req.Description,
		Country:        req.Country,
		Region:         region,
		ParametersEnc:  paramsEnc,
		EffectiveFrom:  req.EffectiveFrom,
		EffectiveUntil: req.EffectiveUntil,
		CreatedBy:      claims.UserID,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "coordination_rule", row.ID, models.AuditCreate, claims, nil)

	rule, err := h.decryptRule(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: rule})
}

func (h *Handler) listRules(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official", "analyst") {
		h.forbidden(w)
		return
	}

	q := r.URL.Query()
	activeOnly := q.Get("active") != "false"

	rows, err := h.queries.ListRules(r.Context(), q.Get("country"), activeOnly)
	if err != nil {
		h.internalError(w, err)
		return
	}

	var results []*models.CoordinationRule
	for _, row := range rows {
		rule, err := h.decryptRule(row)
		if err != nil {
			h.internalError(w, err)
			return
		}
		results = append(results, rule)
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: results})
}

func (h *Handler) deactivateRule(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government") {
		h.forbidden(w)
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid rule ID")
		return
	}

	if err := h.queries.DeactivateRule(r.Context(), id); err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "coordination_rule", id, models.AuditUpdate, claims, map[string]bool{"is_active": false})
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]string{"id": id.String(), "status": "deactivated"}})
}

// ─── Alert handlers ───────────────────────────────────────────────────────────

func (h *Handler) createAlert(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official", "analyst", "system") {
		h.forbidden(w)
		return
	}

	var req models.CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	var metadataEnc *string
	if req.Metadata != nil {
		metaJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			h.internalError(w, err)
			return
		}
		enc, err := h.enc.EncryptString(string(metaJSON))
		if err != nil {
			h.internalError(w, err)
			return
		}
		metadataEnc = &enc
	}

	var region *string
	if req.Region != "" {
		region = &req.Region
	}

	row, err := h.queries.CreateAlert(r.Context(), db.CreateAlertParams{
		RuleID:      req.RuleID,
		Severity:    req.Severity,
		Title:       req.Title,
		Description: req.Description,
		Country:     req.Country,
		Region:      region,
		MetadataEnc: metadataEnc,
		RaisedBy:    claims.UserID,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "governance_alert", row.ID, models.AuditCreate, claims, nil)

	alert, err := h.decryptAlert(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: alert})
}

func (h *Handler) listAlerts(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official", "analyst") {
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

	params := db.ListAlertsParams{
		Country: q.Get("country"),
		Page:    page,
		Limit:   limit,
	}
	if s := q.Get("severity"); s != "" {
		sv := models.AlertSeverity(s)
		params.Severity = &sv
	}
	if s := q.Get("status"); s != "" {
		st := models.AlertStatus(s)
		params.Status = &st
	}

	rows, err := h.queries.ListAlerts(r.Context(), params)
	if err != nil {
		h.internalError(w, err)
		return
	}

	var results []*models.GovernanceAlert
	for _, row := range rows {
		alert, err := h.decryptAlert(row)
		if err != nil {
			h.internalError(w, err)
			return
		}
		results = append(results, alert)
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: results})
}

func (h *Handler) updateAlertStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "government", "ministry_official") {
		h.forbidden(w)
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid alert ID")
		return
	}

	var body struct {
		Status models.AlertStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	row, err := h.queries.UpdateAlertStatus(r.Context(), id, body.Status, claims.UserID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.audit(r, "governance_alert", row.ID, models.AuditUpdate, claims, body)

	alert, err := h.decryptAlert(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: alert})
}

// ─── Decrypt helpers ──────────────────────────────────────────────────────────

func (h *Handler) decryptReport(row *models.ComplianceReportRow) (*models.ComplianceReport, error) {
	payloadJSON, err := h.enc.DecryptString(row.DataPayloadEnc)
	if err != nil {
		return nil, err
	}
	var payload map[string]interface{}
	json.Unmarshal([]byte(payloadJSON), &payload)

	r := &models.ComplianceReport{
		ID:          row.ID,
		FacilityID:  row.FacilityID,
		MinistryID:  row.MinistryID,
		ReportType:  row.ReportType,
		Frequency:   row.Frequency,
		PeriodStart: row.PeriodStart,
		PeriodEnd:   row.PeriodEnd,
		Status:      row.Status,
		Country:     row.Country,
		Summary:     row.Summary,
		DataPayload: payload,
		SubmittedBy: row.SubmittedBy,
		SubmittedAt: row.SubmittedAt,
		ReviewedBy:  row.ReviewedBy,
		ReviewedAt:  row.ReviewedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.Region != nil {
		r.Region = *row.Region
	}
	if row.ViolationNotesEnc != nil {
		notes, err := h.enc.DecryptString(*row.ViolationNotesEnc)
		if err != nil {
			return nil, err
		}
		r.ViolationNotes = notes
	}
	return r, nil
}

func (h *Handler) decryptRule(row *models.CoordinationRuleRow) (*models.CoordinationRule, error) {
	paramsJSON, err := h.enc.DecryptString(row.ParametersEnc)
	if err != nil {
		return nil, err
	}
	var params map[string]interface{}
	json.Unmarshal([]byte(paramsJSON), &params)

	r := &models.CoordinationRule{
		ID:             row.ID,
		RuleType:       row.RuleType,
		Name:           row.Name,
		Description:    row.Description,
		Country:        row.Country,
		Parameters:     params,
		IsActive:       row.IsActive,
		EffectiveFrom:  row.EffectiveFrom,
		EffectiveUntil: row.EffectiveUntil,
		CreatedBy:      row.CreatedBy,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.Region != nil {
		r.Region = *row.Region
	}
	return r, nil
}

func (h *Handler) decryptAlert(row *models.GovernanceAlertRow) (*models.GovernanceAlert, error) {
	a := &models.GovernanceAlert{
		ID:             row.ID,
		RuleID:         row.RuleID,
		Severity:       row.Severity,
		Status:         row.Status,
		Title:          row.Title,
		Description:    row.Description,
		Country:        row.Country,
		RaisedBy:       row.RaisedBy,
		AcknowledgedBy: row.AcknowledgedBy,
		ResolvedBy:     row.ResolvedBy,
		ResolvedAt:     row.ResolvedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.Region != nil {
		a.Region = *row.Region
	}
	if row.MetadataEnc != nil {
		metaJSON, err := h.enc.DecryptString(*row.MetadataEnc)
		if err != nil {
			return nil, err
		}
		var meta map[string]interface{}
		json.Unmarshal([]byte(metaJSON), &meta)
		a.Metadata = meta
	}
	return a, nil
}

// ─── Response helpers ─────────────────────────────────────────────────────────

func (h *Handler) json(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func (h *Handler) badRequest(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusBadRequest, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "BAD_REQUEST", Message: msg},
	})
}

func (h *Handler) forbidden(w http.ResponseWriter) {
	h.json(w, http.StatusForbidden, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "FORBIDDEN", Message: "insufficient role"},
	})
}

func (h *Handler) notFound(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusNotFound, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "NOT_FOUND", Message: msg},
	})
}

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	h.logger.Error("internal error", "error", err)
	h.json(w, http.StatusInternalServerError, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "INTERNAL_ERROR", Message: "an internal error occurred"},
	})
}

func (h *Handler) audit(
	r *http.Request,
	resourceType string,
	resourceID uuid.UUID,
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
		Action:       action,
		AccessorID:   claims.UserID,
		AccessorRole: claims.Role,
		IPAddress:    ip,
		RequestID:    r.Header.Get("X-Request-ID"),
		Changes:      changes,
	})
}
