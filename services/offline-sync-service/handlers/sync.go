package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/offline-sync-service/db"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
	"github.com/klinova/kinara-os/offline-sync-service/middleware"
)

const (
	// appendOnlyTypes never get overwritten — a second push with the same patient
	// generates a new record rather than replacing the existing one.
	// clinical_notes uses last-write-wins (standard upsert).
	payloadConsultation = "consultation"
	payloadPrescription = "prescription"
	payloadReferral     = "referral"
	payloadVitalSigns   = "vital_signs"
)

// appendOnly returns true for payload types that must never overwrite.
func appendOnly(payloadType string) bool {
	return payloadType == payloadPrescription || payloadType == payloadReferral
}

type Handler struct {
	queries           DBQuerier
	patientServiceURL string
	logger            *slog.Logger
}

func New(q *db.Queries, patientServiceURL string, logger *slog.Logger) *Handler {
	return &Handler{queries: q, patientServiceURL: patientServiceURL, logger: logger}
}

// NewWithFakeDB is used by tests to inject a test double.
func NewWithFakeDB(q DBQuerier) *Handler {
	return &Handler{
		queries:           q,
		patientServiceURL: "",
		logger:            slog.Default(),
	}
}

// Pull, Push, and Status are exported for direct handler testing.
func (h *Handler) Pull(w http.ResponseWriter, r *http.Request)   { h.pull(w, r) }
func (h *Handler) Push(w http.ResponseWriter, r *http.Request)   { h.push(w, r) }
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) { h.status(w, r) }

func (h *Handler) Register(r *mux.Router, jwtMW func(http.Handler) http.Handler) {
	sync := r.PathPrefix("/sync").Subrouter()
	sync.Use(jwtMW)
	sync.Use(pkgauth.RequireTenantScope("offline-sync-service", nil))
	sync.Use(middleware.RequireClinicScope)

	sync.HandleFunc("/pull", h.pull).Methods(http.MethodPost)
	sync.HandleFunc("/push", h.push).Methods(http.MethodPost)
	sync.HandleFunc("/status", h.status).Methods(http.MethodGet)
}

// pull returns clinic-scoped patients from the last 72h, capped at 200.
// POST /sync/pull
// Device calls this at startup to hydrate its local cache.
func (h *Handler) pull(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	clinicID, _ := middleware.ClinicIDFromContext(r.Context())

	// Check for revocation or staleness — return wipe directive if needed.
	status, err := h.queries.GetDeviceStatus(r.Context(), *claims.DeviceID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	if status.Revoked {
		h.wipeDirective(w, "device_revoked")
		return
	}
	if status.LastSeenAt != nil && time.Since(*status.LastSeenAt) > 7*24*time.Hour {
		h.wipeDirective(w, "stale_7_days")
		return
	}

	patients, err := h.queries.PullPatients(r.Context(), clinicID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	// Update last_seen so the staleness clock resets on every successful pull.
	h.queries.UpdateLastSeen(r.Context(), *claims.DeviceID, time.Now().UTC())

	h.logger.Info("sync pull",
		"device_id", claims.DeviceID,
		"clinic_id", clinicID,
		"count", len(patients))

	h.json(w, http.StatusOK, map[string]interface{}{
		"wipe":         false,
		"clinic_id":    clinicID,
		"records":      patients,
		"count":        len(patients),
		"cap":          db.MaxCachedRecords,
		"window_hours": 72,
		"synced_at":    time.Now().UTC(),
	})
}

// push accepts queued offline writes from the device.
// POST /sync/push
// Body: array of write operations.
func (h *Handler) push(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	clinicID, _ := middleware.ClinicIDFromContext(r.Context())

	// Check revocation before accepting any writes.
	status, err := h.queries.GetDeviceStatus(r.Context(), *claims.DeviceID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	if status.Revoked {
		h.wipeDirective(w, "device_revoked")
		return
	}

	var req struct {
		Writes []struct {
			IdempotencyKey string          `json:"idempotency_key"`
			PayloadType    string          `json:"payload_type"`
			PatientID      uuid.UUID       `json:"patient_id"`
			Payload        json.RawMessage `json:"payload"`
		} `json:"writes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if len(req.Writes) == 0 {
		h.badRequest(w, "writes array is empty")
		return
	}

	type result struct {
		IdempotencyKey string `json:"idempotency_key"`
		Status         string `json:"status"` // applied|duplicate|rejected
		Reason         string `json:"reason,omitempty"`
	}
	results := make([]result, 0, len(req.Writes))
	now := time.Now().UTC()

	for _, w2 := range req.Writes {
		res := result{IdempotencyKey: w2.IdempotencyKey}

		if w2.IdempotencyKey == "" || w2.PatientID == uuid.Nil || w2.PayloadType == "" {
			res.Status = "rejected"
			res.Reason = "missing required fields"
			results = append(results, res)
			continue
		}

		// Server-side clinic scope check: patient must belong to this clinic.
		inClinic, err := h.queries.PatientInClinic(r.Context(), w2.PatientID, clinicID)
		if err != nil || !inClinic {
			res.Status = "rejected"
			res.Reason = "patient_not_in_clinic_scope"
			h.logger.Warn("scope violation rejected",
				"device_id", claims.DeviceID,
				"patient_id", w2.PatientID,
				"clinic_id", clinicID)
			results = append(results, res)
			continue
		}

		// Idempotency check.
		exists, err := h.queries.PushExists(r.Context(), *claims.DeviceID, w2.IdempotencyKey)
		if err != nil {
			res.Status = "rejected"
			res.Reason = "internal_error"
			results = append(results, res)
			continue
		}
		if exists {
			res.Status = "duplicate"
			results = append(results, res)
			continue
		}

		// Validate payload type.
		switch w2.PayloadType {
		case payloadConsultation, payloadPrescription, payloadReferral, payloadVitalSigns:
			// ok
		default:
			res.Status = "rejected"
			res.Reason = fmt.Sprintf("unknown payload_type: %s", w2.PayloadType)
			results = append(results, res)
			continue
		}

		rec := db.SyncRecord{
			ID:             uuid.New(),
			DeviceID:       *claims.DeviceID,
			IdempotencyKey: w2.IdempotencyKey,
			PayloadType:    w2.PayloadType,
			Payload:        []byte(w2.Payload),
			PatientID:      w2.PatientID,
			ClinicID:       clinicID,
			ReceivedAt:     now,
		}

		if err := h.queries.InsertSyncRecord(r.Context(), rec); err != nil {
			res.Status = "rejected"
			res.Reason = "store_failed"
			results = append(results, res)
			continue
		}

		// For append-only types (prescriptions, referrals), mark applied immediately —
		// the INSERT itself is the authoritative write; we never update these.
		// For other types (consultations, vital_signs), mark applied after store.
		h.queries.MarkApplied(r.Context(), rec.ID, now)
		res.Status = "applied"
		if appendOnly(w2.PayloadType) {
			res.Reason = "append_only"
		}
		results = append(results, res)
	}

	// Reset last_seen after any successful push session.
	h.queries.UpdateLastSeen(r.Context(), *claims.DeviceID, now)

	h.logger.Info("sync push",
		"device_id", claims.DeviceID,
		"clinic_id", clinicID,
		"writes", len(req.Writes),
		"results", len(results))

	h.json(w, http.StatusOK, map[string]interface{}{
		"clinic_id": clinicID,
		"results":   results,
		"synced_at": now,
	})
}

// status returns queue counts and what the device should hold right now.
// GET /sync/status
func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	clinicID, _ := middleware.ClinicIDFromContext(r.Context())

	status, err := h.queries.GetDeviceStatus(r.Context(), *claims.DeviceID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	if status.Revoked {
		h.wipeDirective(w, "device_revoked")
		return
	}

	var isStale bool
	if status.LastSeenAt != nil {
		isStale = time.Since(*status.LastSeenAt) > 7*24*time.Hour
	}
	if isStale {
		h.wipeDirective(w, "stale_7_days")
		return
	}

	pending, applied, rejected, err := h.queries.GetSyncStatus(r.Context(), *claims.DeviceID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, map[string]interface{}{
		"wipe":          false,
		"clinic_id":     clinicID,
		"device_id":     claims.DeviceID,
		"last_seen_at":  status.LastSeenAt,
		"queue": map[string]int{
			"pending":  pending,
			"applied":  applied,
			"rejected": rejected,
		},
		"cache_policy": map[string]interface{}{
			"max_records":       db.MaxCachedRecords,
			"window_hours":      72,
			"stale_wipe_days":   7,
			"session_idle_secs": 300,
			"max_pin_attempts":  10,
		},
	})
}

// ── helpers ────────────────────────────────────────────────────────────────────

func (h *Handler) wipeDirective(w http.ResponseWriter, reason string) {
	w.Header().Set("Content-Type", "application/json")
	// 401 so device re-auth flow is triggered alongside the wipe.
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"wipe":   true,
		"reason": reason,
	})
}

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

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	h.logger.Error("internal error", "error", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
}
