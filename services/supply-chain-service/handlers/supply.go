package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/supply-chain-service/db"
	"github.com/klinova/kinara-os/supply-chain-service/models"
)

type Store interface {
	CreateShipment(ctx context.Context, s models.Shipment) error
	GetShipment(ctx context.Context, id uuid.UUID) (*models.Shipment, error)
	ListShipments(ctx context.Context, p db.ListShipmentsParams) ([]models.Shipment, error)
	UpdateShipmentStatus(ctx context.Context, id uuid.UUID, status models.ShipmentStatus, actualCost *float64, now time.Time) error
	AddTrackingEvent(ctx context.Context, e models.TrackingEvent) error
	ListTrackingEvents(ctx context.Context, shipmentID uuid.UUID) ([]models.TrackingEvent, error)
	InsertAuditLog(ctx context.Context, l models.SupplyAuditLog) error
}

type Handler struct {
	store  Store
	logger *slog.Logger
}

func NewHandler(q *db.Queries, logger *slog.Logger) *Handler {
	return &Handler{store: q, logger: logger}
}

func NewHandlerWithStore(s Store) *Handler {
	return &Handler{store: s, logger: slog.Default()}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.health).Methods(http.MethodGet)
	api := r.PathPrefix("/api/v1/supply-chain").Subrouter()
	api.HandleFunc("/shipments", h.createShipment).Methods(http.MethodPost)
	api.HandleFunc("/shipments", h.listShipments).Methods(http.MethodGet)
	api.HandleFunc("/shipments/{id}", h.getShipment).Methods(http.MethodGet)
	api.HandleFunc("/shipments/{id}/status", h.updateStatus).Methods(http.MethodPut)
	api.HandleFunc("/shipments/{id}/tracking", h.getTracking).Methods(http.MethodGet)
	api.HandleFunc("/cost", h.estimateCost).Methods(http.MethodGet)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "supply-chain-service"})
}

func (h *Handler) createShipment(w http.ResponseWriter, r *http.Request) {
	var req models.CreateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.FarmerID == "" || req.CommodityName == "" || req.QuantityKg <= 0 || req.OriginLocation == "" || req.DestLocation == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "farmer_id, commodity_name, quantity_kg, origin_location, and destination_location are required")
		return
	}
	farmerID, err := uuid.Parse(req.FarmerID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid farmer_id")
		return
	}
	var cooperativeID *uuid.UUID
	if req.CooperativeID != "" {
		if cid, err := uuid.Parse(req.CooperativeID); err == nil {
			cooperativeID = &cid
		}
	}
	var buyerID *uuid.UUID
	if req.BuyerID != "" {
		if bid, err := uuid.Parse(req.BuyerID); err == nil {
			buyerID = &bid
		}
	}
	// Cost estimate: 0.05 USD per km per ton (simplified)
	estimatedCost := req.QuantityKg / 1000.0 * 50.0 // ~50 USD per ton for local transport
	if estimatedCost < 5.0 {
		estimatedCost = 5.0
	}

	now := time.Now().UTC()
	id := uuid.New()
	s := models.Shipment{
		ID:               id,
		ShipmentRef:      "SC-" + strings.ToUpper(id.String()[:8]),
		FarmerID:         farmerID,
		CooperativeID:    cooperativeID,
		CommodityName:    req.CommodityName,
		QuantityKg:       req.QuantityKg,
		OriginLocation:   req.OriginLocation,
		DestLocation:     req.DestLocation,
		BuyerID:          buyerID,
		Status:           models.ShipmentPending,
		PillarHandoff:    models.HandoffAgriToLogistics,
		EstimatedCostUSD: estimatedCost,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := h.store.CreateShipment(r.Context(), s); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to create shipment")
		return
	}
	h.audit(r.Context(), id, uuid.Nil, "create_shipment", "ref:"+s.ShipmentRef)
	respond(w, http.StatusCreated, s)
}

func (h *Handler) getShipment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid shipment id")
		return
	}
	s, err := h.store.GetShipment(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "shipment not found")
		return
	}
	respond(w, http.StatusOK, s)
}

func (h *Handler) listShipments(w http.ResponseWriter, r *http.Request) {
	p := db.ListShipmentsParams{Page: 1, Limit: 50}
	if fid := r.URL.Query().Get("farmer_id"); fid != "" {
		if id, err := uuid.Parse(fid); err == nil {
			p.FarmerID = &id
		}
	}
	if s := r.URL.Query().Get("status"); s != "" {
		p.Status = &s
	}
	shipments, err := h.store.ListShipments(r.Context(), p)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list shipments")
		return
	}
	if shipments == nil {
		shipments = []models.Shipment{}
	}
	respond(w, http.StatusOK, shipments)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid shipment id")
		return
	}
	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Status == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "status is required")
		return
	}
	now := time.Now().UTC()
	status := models.ShipmentStatus(req.Status)
	if err := h.store.UpdateShipmentStatus(r.Context(), id, status, req.ActualCostUSD, now); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to update status")
		return
	}
	// Record tracking event
	_ = h.store.AddTrackingEvent(r.Context(), models.TrackingEvent{
		ID:         uuid.New(),
		ShipmentID: id,
		Status:     status,
		Location:   req.Location,
		Note:       req.Note,
		RecordedAt: now,
	})
	h.audit(r.Context(), id, uuid.Nil, "update_status", "status:"+req.Status)
	respond(w, http.StatusOK, map[string]interface{}{"shipment_id": id, "status": req.Status, "updated_at": now})
}

func (h *Handler) getTracking(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid shipment id")
		return
	}
	events, err := h.store.ListTrackingEvents(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to get tracking")
		return
	}
	if events == nil {
		events = []models.TrackingEvent{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"shipment_id": id, "events": events})
}

func (h *Handler) estimateCost(w http.ResponseWriter, r *http.Request) {
	origin := r.URL.Query().Get("origin")
	dest := r.URL.Query().Get("destination")
	qkg := 1000.0
	if q := r.URL.Query().Get("quantity_kg"); q != "" {
		var parsed float64
		if n, _ := fmt.Sscanf(q, "%f", &parsed); n == 1 && parsed > 0 {
			qkg = parsed
		}
	}
	// Simplified: estimate 100km average distance between African cities; 0.05 USD/ton-km
	estimatedKm := 100.0
	costUSD := (qkg / 1000.0) * estimatedKm * 0.05
	costUSD = math.Max(5.0, costUSD)

	respond(w, http.StatusOK, models.CostEstimate{
		OriginLocation:   origin,
		DestLocation:     dest,
		QuantityKg:       qkg,
		EstimatedCostUSD: costUSD,
		CostPerKgUSD:     costUSD / qkg,
		DistanceKm:       estimatedKm,
	})
}

func (h *Handler) audit(ctx context.Context, shipmentID, actorID uuid.UUID, action, detail string) {
	_ = h.store.InsertAuditLog(ctx, models.SupplyAuditLog{
		ID:         uuid.New(),
		ShipmentID: shipmentID,
		ActorID:    actorID,
		Action:     action,
		Detail:     detail,
		CreatedAt:  time.Now().UTC(),
	})
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
