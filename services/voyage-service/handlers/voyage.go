package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/klinova/kinara-os/voyage-service/db"
	"github.com/klinova/kinara-os/voyage-service/middleware"
	"github.com/klinova/kinara-os/voyage-service/models"
)

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "voyage_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/voyages").Subrouter()
	api.HandleFunc("", h.create).Methods(http.MethodPost)
	api.HandleFunc("/{id}", h.get).Methods(http.MethodGet)
	api.HandleFunc("/{id}/status", h.updateStatus).Methods(http.MethodPut)
	api.HandleFunc("/{id}/events", h.logEvent).Methods(http.MethodPost)
	api.HandleFunc("/{id}/events", h.listEvents).Methods(http.MethodGet)
	api.HandleFunc("/vessel/{vessel_id}", h.listByVessel).Methods(http.MethodGet)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "port_authority", "captain", "shipping_agent") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req models.CreateVoyageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	now := time.Now().UTC()
	v := models.Voyage{
		ID:              id,
		VoyageRef:       "VOY-" + strings.ToUpper(id.String()[:8]),
		VesselID:        req.VesselID,
		OriginPort:      req.OriginPort,
		DestinationPort: req.DestinationPort,
		CargoType:       req.CargoType,
		CargoTons:       req.CargoTons,
		Status:          models.VoyagePlanned,
		DepartureAt:     req.DepartureAt,
		EstArrivalAt:    req.EstArrivalAt,
		DistanceNM:      req.DistanceNM,
		FuelTons:        req.FuelTons,
		TenantID:        claims.TenantID.String(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := h.queries.CreateVoyage(r.Context(), v); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/voyages", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": v})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	v, err := h.queries.GetVoyage(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": v})
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	now := time.Now().UTC()
	if err := h.queries.UpdateStatus(r.Context(), id, req.Status, req.ActualArrivalAt, now); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id, "status": req.Status})
}

func (h *Handler) logEvent(w http.ResponseWriter, r *http.Request) {
	voyageID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.LogEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	e := models.VoyageEvent{
		ID:          uuid.New(),
		VoyageID:    voyageID,
		EventType:   req.EventType,
		Description: req.Description,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		OccurredAt:  req.OccurredAt,
	}
	if err := h.queries.LogEvent(r.Context(), e); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "log failed"})
		return
	}
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": e})
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	voyageID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	events, err := h.queries.ListEvents(r.Context(), voyageID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if events == nil {
		events = []models.VoyageEvent{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": events})
}

func (h *Handler) listByVessel(w http.ResponseWriter, r *http.Request) {
	vesselID, err := uuid.Parse(mux.Vars(r)["vessel_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid vessel_id"})
		return
	}
	voyages, err := h.queries.ListByVessel(r.Context(), vesselID, 20)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if voyages == nil {
		voyages = []models.Voyage{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": voyages})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
