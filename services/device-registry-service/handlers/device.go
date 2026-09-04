package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/device-registry-service/auth"
	"github.com/klinova/kinara-os/device-registry-service/db"
	"github.com/klinova/kinara-os/device-registry-service/middleware"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
	"golang.org/x/crypto/argon2"
)

const (
	maxPINAttempts   = 10
	staleWipeDays    = 7
	maxCachedRecords = 200
)

type Handler struct {
	queries *db.Queries
	logger  *slog.Logger
}

func New(q *db.Queries, logger *slog.Logger) *Handler {
	return &Handler{queries: q, logger: logger}
}

func (h *Handler) Register(r *mux.Router, jwtMW func(http.Handler) http.Handler) {
	// Device enrollment — admin only
	admin := r.PathPrefix("/devices").Subrouter()
	admin.Use(jwtMW)
	admin.Use(pkgauth.RequireTenantScope("device-registry-service", nil))
	admin.HandleFunc("", h.listDevices).Methods(http.MethodGet)
	admin.HandleFunc("/enroll", h.enrollDevice).Methods(http.MethodPost)
	admin.HandleFunc("/{id}", h.getDevice).Methods(http.MethodGet)
	admin.HandleFunc("/{id}/revoke", h.revokeDevice).Methods(http.MethodPost)

	// Heartbeat — device token (scoped)
	r.Handle("/devices/{id}/heartbeat", jwtMW(http.HandlerFunc(h.heartbeat))).Methods(http.MethodPost)
}

// enrollDevice issues a new device secret and registers the device.
// The secret is returned ONCE and stored only as an argon2 hash.
// POST /devices/enroll
func (h *Handler) enrollDevice(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "clinic_admin") {
		h.forbidden(w)
		return
	}

	var req struct {
		DeviceName      string    `json:"device_name"`
		ClinicID        uuid.UUID `json:"clinic_id"`
		AssignedStaffID uuid.UUID `json:"assigned_staff_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.DeviceName == "" || req.ClinicID == uuid.Nil {
		h.badRequest(w, "device_name and clinic_id are required")
		return
	}

	// Generate 32-byte random secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		h.internalError(w, err)
		return
	}
	secret := hex.EncodeToString(secretBytes)

	// Hash with argon2id
	salt := make([]byte, 16)
	rand.Read(salt)
	hash := argon2.IDKey(secretBytes, salt, 1, 64*1024, 4, 32)
	hashHex := fmt.Sprintf("argon2id$%x$%x", salt, hash)

	var staffID *uuid.UUID
	if req.AssignedStaffID != uuid.Nil {
		staffID = &req.AssignedStaffID
	}

	device := db.Device{
		ID:               uuid.New(),
		DeviceName:       req.DeviceName,
		ClinicID:         req.ClinicID,
		AssignedStaffID:  staffID,
		DeviceSecretHash: hashHex,
		EnrolledAt:       time.Now().UTC(),
	}
	if err := h.queries.EnrollDevice(r.Context(), device); err != nil {
		h.internalError(w, err)
		return
	}

	h.queries.InsertAuditLog(r.Context(), device.ID, "enrolled", &claims.UserID, remoteIP(r))
	h.logger.Info("device enrolled",
		"device_id", device.ID,
		"clinic_id", req.ClinicID,
		"enrolled_by", claims.UserID)

	// Secret returned ONCE — not stored in plaintext
	h.json(w, http.StatusCreated, map[string]interface{}{
		"device_id":     device.ID,
		"device_secret": secret,
		"note":          "Store this secret securely. It will not be shown again.",
		"cache_scope": map[string]interface{}{
			"clinic_id":          req.ClinicID,
			"max_records":        maxCachedRecords,
			"cache_window_hours": 72,
			"stale_wipe_days":    staleWipeDays,
		},
	})
}

// getDevice returns device detail + last_seen.
// GET /devices/:id
func (h *Handler) getDevice(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid device id")
		return
	}
	device, err := h.queries.GetDevice(r.Context(), id)
	if err != nil {
		h.notFound(w)
		return
	}
	stale, _ := h.queries.IsStale(r.Context(), id)
	h.json(w, http.StatusOK, map[string]interface{}{
		"id":                device.ID,
		"device_name":       device.DeviceName,
		"clinic_id":         device.ClinicID,
		"assigned_staff_id": device.AssignedStaffID,
		"enrolled_at":       device.EnrolledAt,
		"last_seen_at":      device.LastSeenAt,
		"revoked_at":        device.RevokedAt,
		"revoked_reason":    device.RevokedReason,
		"is_stale":          stale,
	})
}

// listDevices returns all devices (optionally filtered by clinic).
// GET /devices?clinic_id=<uuid>
func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	var clinicID *uuid.UUID
	if raw := r.URL.Query().Get("clinic_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			h.badRequest(w, "invalid clinic_id")
			return
		}
		clinicID = &id
	}
	devices, err := h.queries.ListDevices(r.Context(), clinicID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.json(w, http.StatusOK, map[string]interface{}{"devices": devices, "total": len(devices)})
}

// revokeDevice revokes a device credential; next sync gets wipe directive.
// POST /devices/:id/revoke
func (h *Handler) revokeDevice(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "clinic_admin") {
		h.forbidden(w)
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid device id")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "admin_revoked"
	}

	if err := h.queries.RevokeDevice(r.Context(), id, req.Reason, time.Now().UTC()); err != nil {
		h.internalError(w, err)
		return
	}
	h.queries.InsertAuditLog(r.Context(), id, "revoked", &claims.UserID, remoteIP(r))
	h.logger.Warn("device revoked", "device_id", id, "reason", req.Reason, "by", claims.UserID)
	h.json(w, http.StatusOK, map[string]string{"status": "revoked", "device_id": id.String()})
}

// heartbeat updates last_seen and returns a wipe directive if the device is
// revoked or stale (>7 days since last sync).
// POST /devices/:id/heartbeat
func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid device id")
		return
	}

	now := time.Now().UTC()

	revoked, err := h.queries.IsRevoked(r.Context(), id)
	if err != nil {
		h.internalError(w, err)
		return
	}
	if revoked {
		h.queries.InsertAuditLog(r.Context(), id, "wipe_triggered", nil, remoteIP(r))
		h.json(w, http.StatusOK, map[string]interface{}{
			"wipe":   true,
			"reason": "device_revoked",
		})
		return
	}

	stale, _ := h.queries.IsStale(r.Context(), id)
	if stale {
		h.queries.InsertAuditLog(r.Context(), id, "wipe_triggered", nil, remoteIP(r))
		h.json(w, http.StatusOK, map[string]interface{}{
			"wipe":   true,
			"reason": "stale_7_days",
		})
		return
	}

	h.queries.UpdateLastSeen(r.Context(), id, now)
	h.queries.InsertAuditLog(r.Context(), id, "heartbeat", nil, remoteIP(r))
	h.json(w, http.StatusOK, map[string]interface{}{
		"wipe":          false,
		"last_seen_at":  now,
		"cache_expires": now.Add(db.StalenessThreshold),
	})
}

// ── helpers ────────────────────────────────────────────────────────────────────

func (h *Handler) json(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func (h *Handler) badRequest(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `{"success":false,"error":{"code":"BAD_REQUEST","message":"%s"}}`, msg)
}
func (h *Handler) forbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`))
}
func (h *Handler) notFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"success":false,"error":{"code":"NOT_FOUND","message":"device not found"}}`))
}
func (h *Handler) internalError(w http.ResponseWriter, err error) {
	h.logger.Error("internal error", "error", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
}
func remoteIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}

// ensure auth import is used
var _ = auth.MTLSConfig{}
