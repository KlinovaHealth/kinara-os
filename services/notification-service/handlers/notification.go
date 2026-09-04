package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/notification-service/auth"
	"github.com/klinova/kinara-os/notification-service/crypto"
	"github.com/klinova/kinara-os/notification-service/db"
	"github.com/klinova/kinara-os/notification-service/middleware"
	"github.com/klinova/kinara-os/notification-service/models"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
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
	api.Use(pkgauth.RequireTenantScope("notification-service", nil))

	api.HandleFunc("/notifications/send", h.sendNotification).Methods(http.MethodPost)
	api.HandleFunc("/notifications/schedule", h.scheduleNotification).Methods(http.MethodPost)
	api.HandleFunc("/notifications/bulk", h.bulkSend).Methods(http.MethodPost)
	api.HandleFunc("/notifications", h.listNotifications).Methods(http.MethodGet)
	api.HandleFunc("/notifications/{id}", h.getNotification).Methods(http.MethodGet)
	api.HandleFunc("/notifications/{id}/status", h.updateNotificationStatus).Methods(http.MethodPut)

	api.HandleFunc("/templates", h.listTemplates).Methods(http.MethodGet)
	api.HandleFunc("/templates/{id}", h.getTemplate).Methods(http.MethodGet)

	api.HandleFunc("/preferences", h.getPreferences).Methods(http.MethodGet)
	api.HandleFunc("/preferences", h.updatePreferences).Methods(http.MethodPut)
	api.HandleFunc("/preferences/{user_id}", h.getPreferencesForUser).Methods(http.MethodGet)
}

// ─── Send notification ────────────────────────────────────────────────────────

func (h *Handler) sendNotification(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "system", "clinician", "doctor", "nurse", "pharmacist",
		"farmer", "logistics", "fleet_operator", "port_operator", "facility_admin") {
		h.forbidden(w)
		return
	}

	var req models.SendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if err := validateSendRequest(req); err != nil {
		h.badRequest(w, err.Error())
		return
	}

	row, err := h.buildNotificationRow(req.UserID, req.Type, req.Channel, req.Priority,
		req.Subject, req.Message, req.Recipient, nil, nil)
	if err != nil {
		h.internalError(w, "failed to build notification")
		return
	}

	if err := h.queries.CreateNotification(r.Context(), *row); err != nil {
		h.logger.Error("create notification failed", "error", err)
		h.internalError(w, "failed to create notification")
		return
	}

	// Simulate dispatch to provider — in production this enqueues to Redis/worker
	now := time.Now().UTC()
	h.queries.UpdateNotificationStatus(r.Context(), row.ID, models.StatusSent, &now, nil, "", "simulated-"+row.ID.String(), now)

	notif, _ := h.queries.GetNotification(r.Context(), row.ID)
	h.audit(r, claims, &row.ID, "send_notification", "notifications")
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: h.decryptNotification(*notif)})
}

// ─── Schedule notification ────────────────────────────────────────────────────

func (h *Handler) scheduleNotification(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "system", "clinician", "doctor", "pharmacist", "facility_admin") {
		h.forbidden(w)
		return
	}

	var req models.ScheduleNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.UserID == "" || req.Message == "" || req.Recipient == "" || req.ScheduledAt == "" {
		h.badRequest(w, "user_id, message, recipient, and scheduled_at are required")
		return
	}
	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		h.badRequest(w, "scheduled_at must be RFC3339")
		return
	}
	if scheduledAt.Before(time.Now()) {
		h.badRequest(w, "scheduled_at must be in the future")
		return
	}
	scheduledAtUTC := scheduledAt.UTC()

	row, err := h.buildNotificationRow(req.UserID, req.Type, req.Channel, req.Priority,
		req.Subject, req.Message, req.Recipient, nil, &scheduledAtUTC)
	if err != nil {
		h.internalError(w, "failed to build notification")
		return
	}

	if err := h.queries.CreateNotification(r.Context(), *row); err != nil {
		h.logger.Error("schedule notification failed", "error", err)
		h.internalError(w, "failed to schedule notification")
		return
	}

	h.audit(r, claims, &row.ID, "schedule_notification", "notifications")
	notif, _ := h.queries.GetNotification(r.Context(), row.ID)
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: h.decryptNotification(*notif)})
}

// ─── Bulk send ────────────────────────────────────────────────────────────────

func (h *Handler) bulkSend(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "system", "ministry_official", "government", "facility_admin") {
		h.forbidden(w)
		return
	}

	var req models.BulkSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if len(req.UserIDs) == 0 || req.Message == "" {
		h.badRequest(w, "user_ids and message are required")
		return
	}
	if len(req.UserIDs) > 1000 {
		h.badRequest(w, "maximum 1000 recipients per bulk send")
		return
	}

	sent := 0
	failed := 0
	for _, userIDStr := range req.UserIDs {
		row, err := h.buildNotificationRow(userIDStr, req.Type, req.Channel, req.Priority,
			req.Subject, req.Message, "", nil, nil)
		if err != nil {
			failed++
			continue
		}
		if err := h.queries.CreateNotification(r.Context(), *row); err != nil {
			failed++
			continue
		}
		sent++
	}

	h.audit(r, claims, nil, "bulk_send", "notifications")
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]int{"sent": sent, "failed": failed, "total": len(req.UserIDs)},
	})
}

// ─── Get / List notifications ─────────────────────────────────────────────────

func (h *Handler) getNotification(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "system", "facility_admin") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	row, err := h.queries.GetNotification(r.Context(), id)
	if err != nil {
		h.notFound(w, "notification not found")
		return
	}
	h.audit(r, claims, &id, "get_notification", "notifications")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: h.decryptNotification(*row)})
}

func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "system", "facility_admin") {
		h.forbidden(w)
		return
	}

	params := db.ListNotificationsParams{
		Page:  pageParam(r, 1),
		Limit: limitParam(r, 50),
	}
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		id, err := uuid.Parse(uid)
		if err != nil {
			h.badRequest(w, "invalid user_id")
			return
		}
		params.UserID = &id
	}
	if t := r.URL.Query().Get("type"); t != "" {
		nt := models.NotificationType(t)
		params.Type = &nt
	}
	if c := r.URL.Query().Get("channel"); c != "" {
		nc := models.NotificationChannel(c)
		params.Channel = &nc
	}
	if s := r.URL.Query().Get("status"); s != "" {
		ns := models.NotificationStatus(s)
		params.Status = &ns
	}

	rows, err := h.queries.ListNotifications(r.Context(), params)
	if err != nil {
		h.internalError(w, "failed to list notifications")
		return
	}
	total, _ := h.queries.CountNotifications(r.Context(), params)

	notifs := make([]models.Notification, 0, len(rows))
	for _, row := range rows {
		notifs = append(notifs, h.decryptNotification(row))
	}

	totalPages := (total + params.Limit - 1) / params.Limit
	h.audit(r, claims, nil, "list_notifications", "notifications")
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true, Data: notifs,
		Meta: &models.PageMeta{Page: params.Page, Limit: params.Limit, Total: total, TotalPages: totalPages},
	})
}

func (h *Handler) updateNotificationStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "system") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}

	var body struct {
		Status        models.NotificationStatus `json:"status"`
		ExternalID    string                    `json:"external_id,omitempty"`
		FailureReason string                    `json:"failure_reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}

	now := time.Now().UTC()
	var sentAt, deliveredAt *time.Time
	if body.Status == models.StatusSent || body.Status == models.StatusDelivered {
		sentAt = &now
	}
	if body.Status == models.StatusDelivered {
		deliveredAt = &now
	}

	if err := h.queries.UpdateNotificationStatus(r.Context(), id, body.Status, sentAt, deliveredAt,
		body.FailureReason, body.ExternalID, now); err != nil {
		h.internalError(w, "failed to update status")
		return
	}

	h.audit(r, claims, &id, "update_notification_status", "notifications")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]string{"status": string(body.Status)}})
}

// ─── Templates ────────────────────────────────────────────────────────────────

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "system", "facility_admin") {
		h.forbidden(w)
		return
	}

	var notifType *models.NotificationType
	var channel *models.NotificationChannel
	if t := r.URL.Query().Get("type"); t != "" {
		nt := models.NotificationType(t)
		notifType = &nt
	}
	if c := r.URL.Query().Get("channel"); c != "" {
		nc := models.NotificationChannel(c)
		channel = &nc
	}

	templates, err := h.queries.ListTemplates(r.Context(), notifType, channel)
	if err != nil {
		h.internalError(w, "failed to list templates")
		return
	}
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: templates})
}

func (h *Handler) getTemplate(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "system", "facility_admin") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	tpl, err := h.queries.GetTemplate(r.Context(), id)
	if err != nil {
		h.notFound(w, "template not found")
		return
	}
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: tpl})
}

// ─── Preferences ──────────────────────────────────────────────────────────────

func (h *Handler) getPreferences(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	prefs, err := h.queries.GetOrCreatePreferences(r.Context(), claims.UserID)
	if err != nil {
		h.internalError(w, "failed to get preferences")
		return
	}
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: prefs})
}

func (h *Handler) getPreferencesForUser(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "facility_admin") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "user_id")
	if err != nil {
		h.badRequest(w, "invalid user_id")
		return
	}
	prefs, err := h.queries.GetOrCreatePreferences(r.Context(), id)
	if err != nil {
		h.internalError(w, "failed to get preferences")
		return
	}
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: prefs})
}

func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req models.UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}

	p := db.UpdatePreferencesParams{
		UserID:          claims.UserID,
		SMSEnabled:      req.SMSEnabled,
		PushEnabled:     req.PushEnabled,
		WhatsAppEnabled: req.WhatsAppEnabled,
		EmailEnabled:    req.EmailEnabled,
		InAppEnabled:    req.InAppEnabled,
		QuietHoursStart: req.QuietHoursStart,
		QuietHoursEnd:   req.QuietHoursEnd,
		TimeZone:        req.TimeZone,
		Language:        req.Language,
		Now:             time.Now().UTC(),
	}
	if err := h.queries.UpdatePreferences(r.Context(), p); err != nil {
		h.internalError(w, "failed to update preferences")
		return
	}

	prefs, _ := h.queries.GetOrCreatePreferences(r.Context(), claims.UserID)
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: prefs})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) buildNotificationRow(
	userIDStr string,
	notifType models.NotificationType,
	channel models.NotificationChannel,
	priority models.NotificationPriority,
	subject, message, recipient string,
	templateID *uuid.UUID,
	scheduledAt *time.Time,
) (*models.NotificationRow, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}
	if priority == "" {
		priority = models.PriorityNormal
	}
	if channel == "" {
		channel = models.ChannelSMS
	}

	messageEnc, err := h.enc.EncryptString(message)
	if err != nil {
		return nil, err
	}
	subjectEnc := ""
	if subject != "" {
		subjectEnc, err = h.enc.EncryptString(subject)
		if err != nil {
			return nil, err
		}
	}
	recipientEnc := ""
	if recipient != "" {
		recipientEnc, err = h.enc.EncryptString(recipient)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	status := models.StatusPending
	if scheduledAt == nil {
		status = models.StatusQueued
	}

	return &models.NotificationRow{
		ID:           uuid.New(),
		UserID:       userID,
		Type:         notifType,
		Channel:      channel,
		Priority:     priority,
		MessageEnc:   messageEnc,
		SubjectEnc:   subjectEnc,
		RecipientEnc: recipientEnc,
		TemplateID:   templateID,
		Status:       status,
		ScheduledAt:  scheduledAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (h *Handler) decryptNotification(row models.NotificationRow) models.Notification {
	message, _ := h.enc.DecryptString(row.MessageEnc)
	subject := ""
	if row.SubjectEnc != "" {
		subject, _ = h.enc.DecryptString(row.SubjectEnc)
	}
	// Recipient omitted from API response for privacy
	return models.Notification{
		ID:            row.ID,
		UserID:        row.UserID,
		Type:          row.Type,
		Channel:       row.Channel,
		Priority:      row.Priority,
		Message:       message,
		Subject:       subject,
		TemplateID:    row.TemplateID,
		Status:        row.Status,
		RetryCount:    row.RetryCount,
		ScheduledAt:   row.ScheduledAt,
		SentAt:        row.SentAt,
		DeliveredAt:   row.DeliveredAt,
		FailureReason: row.FailureReason,
		CreatedAt:     row.CreatedAt,
	}
}

func (h *Handler) audit(r *http.Request, claims *auth.Claims, notifID *uuid.UUID, action, resource string) {
	h.queries.InsertAuditLog(r.Context(), models.NotificationAuditLog{
		ID:             uuid.New(),
		NotificationID: notifID,
		UserID:         claims.UserID,
		Action:         action,
		Resource:       resource,
		IPAddress:      remoteIP(r),
		CreatedAt:      time.Now().UTC(),
	})
}

func validateSendRequest(req models.SendNotificationRequest) error {
	if req.UserID == "" {
		return errMsg("user_id is required")
	}
	if req.Message == "" {
		return errMsg("message is required")
	}
	if req.Recipient == "" {
		return errMsg("recipient is required")
	}
	return nil
}

type validationError struct{ msg string }

func (e validationError) Error() string { return e.msg }
func errMsg(msg string) error           { return validationError{msg: msg} }

func (h *Handler) json(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
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
func (h *Handler) internalError(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusInternalServerError, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "INTERNAL_ERROR", Message: msg},
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

func parseID(r *http.Request, key string) (uuid.UUID, error) {
	return uuid.Parse(mux.Vars(r)[key])
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
