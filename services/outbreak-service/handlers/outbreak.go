package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/klinova/kinara-os/outbreak-service/db"
	"github.com/klinova/kinara-os/outbreak-service/middleware"
	"github.com/klinova/kinara-os/outbreak-service/models"
)

var (
	casesReported     = promauto.NewCounter(prometheus.CounterOpts{Name: "outbreak_cases_reported_total", Help: "Total suspected cases reported"})
	outbreaksDetected = promauto.NewCounter(prometheus.CounterOpts{Name: "outbreak_outbreaks_detected_total", Help: "Total outbreak alerts auto-created"})
	notificationsSent = promauto.NewCounter(prometheus.CounterOpts{Name: "outbreak_notifications_sent_total", Help: "Total notifications sent to health ministry"})
)

// Store is the handler's view of the database layer.
type Store = db.Store

// Handler holds the store interface.
type Handler struct{ store Store }

// New creates a Handler backed by a real *db.Queries.
func New(q *db.Queries) *Handler { return &Handler{store: q} }

// NewWithStore creates a Handler with any Store (for tests).
func NewWithStore(s Store) *Handler { return &Handler{store: s} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/outbreak").Subrouter()
	api.HandleFunc("/cases", h.reportCase).Methods(http.MethodPost)
	api.HandleFunc("/alerts", h.listAlerts).Methods(http.MethodGet)
	api.HandleFunc("/alerts/{id}/confirm", h.confirmOutbreak).Methods(http.MethodPost)
	api.HandleFunc("/clusters", h.getClusters).Methods(http.MethodGet)
	api.HandleFunc("/trends", h.getTrends).Methods(http.MethodGet)
	api.HandleFunc("/notify", h.notify).Methods(http.MethodPost)
}

// POST /api/v1/outbreak/cases
func (h *Handler) reportCase(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("doctor", "nurse", "admin", "epidemiologist") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req models.ReportCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.DiseaseCode == "" || req.ClinicID == "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": "disease_code and clinic_id are required"})
		return
	}

	id := uuid.New()
	now := time.Now().UTC()
	c := models.SuspectedCase{
		ID:          id,
		CaseRef:     "CASE-" + strings.ToUpper(id.String()[:8]),
		PatientID:   req.PatientID,
		DiseaseCode: req.DiseaseCode,
		DiseaseName: req.DiseaseName,
		ClinicID:    req.ClinicID,
		Location:    req.Location,
		Symptoms:    req.Symptoms,
		ReportedBy:  claims.UserID,
		TenantID:    claims.TenantID.String(),
		ReportedAt:  now,
	}

	if err := h.store.InsertCase(r.Context(), c); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}
	casesReported.Inc()

	// Fire-and-forget outbreak threshold check.
	go func() {
		ctx := context.Background()
		if triggered := h.checkOutbreakThreshold(ctx, req.DiseaseCode, req.ClinicID, req.DiseaseName); triggered {
			outbreaksDetected.Inc()
		}
	}()

	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": c})
}

// checkOutbreakThreshold counts cases in the window and upserts an outbreak if threshold reached.
// Returns true if the threshold was met and upsert was attempted.
func (h *Handler) checkOutbreakThreshold(ctx context.Context, diseaseCode, clinicID, diseaseName string) bool {
	count, err := h.store.CountRecentCases(ctx, diseaseCode, clinicID, 7*24*time.Hour)
	if err != nil {
		slog.Error("count recent cases", "error", err)
		return false
	}
	if count >= 5 {
		if err := h.store.UpsertOutbreak(ctx, diseaseCode, clinicID, diseaseName, count); err != nil {
			slog.Error("upsert outbreak", "error", err)
		}
		return true
	}
	return false
}

// GET /api/v1/outbreak/alerts
func (h *Handler) listAlerts(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	items, err := h.store.ListActiveOutbreaks(r.Context())
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.ConfirmedOutbreak{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

// POST /api/v1/outbreak/alerts/{id}/confirm
func (h *Handler) confirmOutbreak(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("epidemiologist", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.store.ConfirmOutbreak(r.Context(), id, claims.UserID.String()); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "confirm failed"})
		return
	}

	// Audit is handled inside ConfirmOutbreak; no extra insert needed.
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

// GET /api/v1/outbreak/clusters
func (h *Handler) getClusters(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	clusters, err := h.store.GetClusters(r.Context())
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if clusters == nil {
		clusters = []models.DiseaseCluster{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": clusters})
}

// GET /api/v1/outbreak/trends
func (h *Handler) getTrends(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	trends, err := h.store.GetTrends(r.Context())
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if trends == nil {
		trends = []models.DiseaseTrend{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": trends})
}

// POST /api/v1/outbreak/notify
func (h *Handler) notify(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("epidemiologist", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req models.NotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Message == "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	n := models.OutbreakNotification{
		ID:         uuid.New(),
		OutbreakID: req.OutbreakID,
		Message:    req.Message,
		Recipients: req.Recipients,
		SentBy:     claims.UserID,
		SentAt:     time.Now().UTC(),
	}

	if err := h.store.InsertNotification(r.Context(), n); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "notify failed"})
		return
	}
	notificationsSent.Inc()

	// Fire-and-forget audit.
	go func() {
		ctx := context.Background()
		if err := h.store.InsertAudit(ctx, req.OutbreakID.String(), claims.UserID.String(), "send_notification"); err != nil {
			slog.Error("audit insert", "error", err)
		}
	}()

	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": n})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
