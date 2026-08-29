package handlers

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/audit-service/db"
	"github.com/klinova/kinara-os/audit-service/models"
)

type Store interface {
	InsertEvent(ctx context.Context, e models.AuditEvent) error
	ListEvents(ctx context.Context, p db.ListEventsParams) ([]models.AuditEvent, error)
	CountByPillar(ctx context.Context, since, until time.Time) (map[string]int, error)
	CountByService(ctx context.Context, since, until time.Time) (map[string]int, error)
}

type Handler struct {
	store      Store
	signingKey ed25519.PrivateKey
	verifyKey  ed25519.PublicKey
	logger     *slog.Logger
}

func NewHandler(q *db.Queries, logger *slog.Logger) *Handler {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic("audit: failed to generate ed25519 key: " + err.Error())
	}
	return &Handler{store: q, signingKey: priv, verifyKey: pub, logger: logger}
}

func NewHandlerWithStore(s Store) *Handler {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	return &Handler{store: s, signingKey: priv, verifyKey: pub, logger: slog.Default()}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.health).Methods(http.MethodGet)
	api := r.PathPrefix("/api/v1/audit").Subrouter()
	api.HandleFunc("/events", h.logEvent).Methods(http.MethodPost)
	api.HandleFunc("/events", h.listEvents).Methods(http.MethodGet)
	api.HandleFunc("/report", h.generateReport).Methods(http.MethodGet)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "audit-service"})
}

func (h *Handler) logEvent(w http.ResponseWriter, r *http.Request) {
	var req models.LogEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Service == "" || req.EventType == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "service and event_type are required")
		return
	}
	pillar := req.Pillar
	if pillar == "" {
		pillar = inferPillar(req.Service)
	}
	actorID, err := uuid.Parse(req.ActorID)
	if err != nil {
		actorID = uuid.Nil
	}
	now := time.Now().UTC()
	id := uuid.New()
	eventRef := "AE-" + strings.ToUpper(id.String()[:8])

	payload := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		eventRef, req.Service, pillar, req.EventType, actorID, now.Format(time.RFC3339))
	sig := ed25519.Sign(h.signingKey, []byte(payload))
	signature := base64.StdEncoding.EncodeToString(sig)

	event := models.AuditEvent{
		ID:           id,
		EventRef:     eventRef,
		Service:      req.Service,
		Pillar:       pillar,
		EventType:    req.EventType,
		ActorID:      actorID,
		ActorRole:    req.ActorRole,
		ResourceID:   req.ResourceID,
		ResourceType: req.ResourceType,
		Detail:       req.Detail,
		IPAddress:    req.IPAddress,
		TenantID:     req.TenantID,
		Signature:    signature,
		CreatedAt:    now,
	}
	if err := h.store.InsertEvent(r.Context(), event); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to log event")
		return
	}
	respond(w, http.StatusCreated, event)
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	p := db.ListEventsParams{Page: 1, Limit: 100}
	if s := r.URL.Query().Get("service"); s != "" {
		p.Service = &s
	}
	if pl := r.URL.Query().Get("pillar"); pl != "" {
		p.Pillar = &pl
	}
	if t := r.URL.Query().Get("tenant_id"); t != "" {
		p.TenantID = &t
	}
	if a := r.URL.Query().Get("actor_id"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			p.ActorID = &id
		}
	}
	if since := r.URL.Query().Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			p.Since = &t
		}
	}
	events, err := h.store.ListEvents(r.Context(), p)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list events")
		return
	}
	if events == nil {
		events = []models.AuditEvent{}
	}
	respond(w, http.StatusOK, events)
}

func (h *Handler) generateReport(w http.ResponseWriter, r *http.Request) {
	periodStart := time.Now().UTC().AddDate(0, -1, 0)
	periodEnd := time.Now().UTC()
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			periodStart = t
		}
	}
	if u := r.URL.Query().Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			periodEnd = t
		}
	}
	byPillar, _ := h.store.CountByPillar(r.Context(), periodStart, periodEnd)
	byService, _ := h.store.CountByService(r.Context(), periodStart, periodEnd)
	var totalEvents int
	for _, count := range byPillar {
		totalEvents += count
	}
	id := uuid.New()
	report := models.AuditReport{
		ID:          id,
		ReportRef:   "AR-" + strings.ToUpper(id.String()[:8]),
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		TotalEvents: totalEvents,
		ByPillar:    byPillar,
		ByService:   byService,
		GeneratedAt: time.Now().UTC(),
	}
	respond(w, http.StatusOK, report)
}

// inferPillar maps service names to their pillar.
func inferPillar(service string) string {
	switch {
	case strings.Contains(service, "patient") || strings.Contains(service, "clinical") ||
		strings.Contains(service, "pharmacy") || strings.Contains(service, "referral") ||
		strings.Contains(service, "telemedicine") || strings.Contains(service, "health"):
		return "health"
	case strings.Contains(service, "farmer") || strings.Contains(service, "market") ||
		strings.Contains(service, "cooperative") || strings.Contains(service, "weather") ||
		strings.Contains(service, "supply"):
		return "agriculture"
	case strings.Contains(service, "transport") || strings.Contains(service, "warehouse") ||
		strings.Contains(service, "fleet") || strings.Contains(service, "shipment") ||
		strings.Contains(service, "last-mile") || strings.Contains(service, "driver"):
		return "logistics"
	case strings.Contains(service, "port") || strings.Contains(service, "vessel") ||
		strings.Contains(service, "cargo") || strings.Contains(service, "customs") ||
		strings.Contains(service, "dock") || strings.Contains(service, "shipping") ||
		strings.Contains(service, "trade-finance") || strings.Contains(service, "documentation"):
		return "maritime"
	default:
		return "cross-pillar"
	}
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
