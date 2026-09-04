package handlers

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/klinova/kinara-os/vehicle-tracking-service/db"
	"github.com/klinova/kinara-os/vehicle-tracking-service/middleware"
	"github.com/klinova/kinara-os/vehicle-tracking-service/models"
)

// Prometheus counters.
var (
	vehiclePingsTotal  = promauto.NewCounter(prometheus.CounterOpts{Name: "vehicle_pings_total"})
	vehicleAlertsTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "vehicle_alerts_total"})
)

// Store is the database interface used by Handler, enabling mock injection in tests.
type Store interface {
	InsertPing(ctx context.Context, loc models.GPSLocation) error
	GetLatestLocation(ctx context.Context, vehicleID uuid.UUID) (*models.GPSLocation, error)
	GetActiveRoute(ctx context.Context, vehicleID uuid.UUID) (*models.VehicleRoute, error)
	GetFleetStatus(ctx context.Context, tenantID string) ([]models.FleetVehicleStatus, error)
	InsertAlert(ctx context.Context, a models.VehicleAlert) error
	GetVehicle(ctx context.Context, id uuid.UUID) (*models.Vehicle, error)
}

// Handler holds the store and exposes HTTP handlers.
type Handler struct{ store Store }

// New wires up a real db.Queries-backed handler.
func New(q *db.Queries) *Handler { return &Handler{store: q} }

// NewWithStore allows injecting a mock Store in tests.
func NewWithStore(s Store) *Handler { return &Handler{store: s} }

// Register mounts all routes. Static sub-paths precede dynamic /{id} routes.
func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/vehicle").Subrouter()

	// Static routes first.
	api.HandleFunc("/ping", h.ping).Methods(http.MethodPost)
	api.HandleFunc("/fleet/status", h.fleetStatus).Methods(http.MethodGet)

	// Dynamic vehicle ID routes.
	api.HandleFunc("/{id}/location", h.getLocation).Methods(http.MethodGet)
	api.HandleFunc("/{id}/route", h.getRoute).Methods(http.MethodGet)
	api.HandleFunc("/{id}/eta", h.calculateETA).Methods(http.MethodPost)
	api.HandleFunc("/{id}/alert", h.sendAlert).Methods(http.MethodPost)
}

// ---------- POST /api/v1/vehicle/ping ----------

func (h *Handler) ping(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("driver", "system", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req models.PingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.VehicleID == uuid.Nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "vehicle_id required"})
		return
	}
	loc := models.GPSLocation{
		ID:         uuid.New(),
		VehicleID:  req.VehicleID,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		SpeedKmh:   req.SpeedKmh,
		HeadingDeg: req.HeadingDeg,
		PingedAt:   time.Now().UTC(),
	}
	if err := h.store.InsertPing(r.Context(), loc); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}
	vehiclePingsTotal.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": loc})
}

// ---------- GET /api/v1/vehicle/{id}/location ----------

func (h *Handler) getLocation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	vehicleID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid vehicle id"})
		return
	}
	loc, err := h.store.GetLatestLocation(r.Context(), vehicleID)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "location not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"vehicle_id":  loc.VehicleID,
			"latitude":    loc.Latitude,
			"longitude":   loc.Longitude,
			"speed_kmh":   loc.SpeedKmh,
			"heading_deg": loc.HeadingDeg,
			"last_ping":   loc.PingedAt,
		},
	})
}

// ---------- GET /api/v1/vehicle/{id}/route ----------

func (h *Handler) getRoute(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	vehicleID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid vehicle id"})
		return
	}
	route, err := h.store.GetActiveRoute(r.Context(), vehicleID)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "no active route"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": route})
}

// ---------- POST /api/v1/vehicle/{id}/eta ----------

func (h *Handler) calculateETA(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	vehicleID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid vehicle id"})
		return
	}
	var req models.ETARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	loc, err := h.store.GetLatestLocation(r.Context(), vehicleID)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "vehicle location not found"})
		return
	}
	distKm := haversineKm(loc.Latitude, loc.Longitude, req.DestinationLat, req.DestinationLng)
	speed := loc.SpeedKmh
	if speed <= 0 {
		speed = 60.0
	}
	estimatedMinutes := int(math.Round(distKm / speed * 60))
	etaUTC := time.Now().UTC().Add(time.Duration(estimatedMinutes) * time.Minute)

	resp := models.ETAResponse{
		VehicleID:        vehicleID,
		CurrentLat:       loc.Latitude,
		CurrentLng:       loc.Longitude,
		DestinationLat:   req.DestinationLat,
		DestinationLng:   req.DestinationLng,
		DistanceKm:       distKm,
		EstimatedMinutes: estimatedMinutes,
		ETAUTC:           etaUTC,
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": resp})
}

// ---------- GET /api/v1/vehicle/fleet/status ----------

func (h *Handler) fleetStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "dispatcher", "logistics_manager") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	items, err := h.store.GetFleetStatus(r.Context(), claims.TenantID.String())
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.FleetVehicleStatus{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

// ---------- POST /api/v1/vehicle/{id}/alert ----------

func (h *Handler) sendAlert(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "dispatcher") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	vehicleID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid vehicle id"})
		return
	}
	var req models.AlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if strings.TrimSpace(req.AlertType) == "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": "alert_type required"})
		return
	}
	alert := models.VehicleAlert{
		ID:        uuid.New(),
		VehicleID: vehicleID,
		AlertType: req.AlertType,
		Message:   req.Message,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.store.InsertAlert(r.Context(), alert); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}
	vehicleAlertsTotal.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": alert})
}

// ---------- helpers ----------

// haversineKm calculates great-circle distance between two GPS coordinates in km.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
