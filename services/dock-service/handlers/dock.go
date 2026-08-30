package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/dock-service/db"
	"github.com/klinova/kinara-os/dock-service/middleware"
	"github.com/klinova/kinara-os/dock-service/models"
)

type Store interface {
	CreateOperation(ctx context.Context, op models.DockOperation) error
	GetOperation(ctx context.Context, id uuid.UUID) (*models.DockOperation, error)
	ListOperations(ctx context.Context, p db.ListOperationsParams) ([]models.DockOperation, error)
	StartOperation(ctx context.Context, id uuid.UUID, startedAt time.Time, now time.Time) error
	CompleteOperation(ctx context.Context, id uuid.UUID, completedAt time.Time, duration float64, safetyIncident bool, details string, now time.Time) error
	CreateEquipment(ctx context.Context, e models.Equipment) error
	GetEquipment(ctx context.Context, id uuid.UUID) (*models.Equipment, error)
	ListEquipment(ctx context.Context, portID uuid.UUID) ([]models.Equipment, error)
	UpdateEquipmentStatus(ctx context.Context, id uuid.UUID, status models.EquipmentStatus, now time.Time) error
	ReportSafetyEvent(ctx context.Context, e models.SafetyEvent) error
	ListSafetyEvents(ctx context.Context, portID uuid.UUID) ([]models.SafetyEvent, error)
	InsertAuditLog(ctx context.Context, l models.DockAuditLog) error
}

type DockHandler struct{ store Store }

func NewHandler(q *db.Queries) *DockHandler        { return &DockHandler{store: q} }
func NewHandlerWithStore(s Store) *DockHandler      { return &DockHandler{store: s} }

func (h *DockHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/operations", h.CreateOperation).Methods(http.MethodPost)
	r.HandleFunc("/operations", h.ListOperations).Methods(http.MethodGet)
	r.HandleFunc("/operations/{id}", h.GetOperation).Methods(http.MethodGet)
	r.HandleFunc("/operations/{id}/start", h.StartOperation).Methods(http.MethodPut)
	r.HandleFunc("/operations/{id}/complete", h.CompleteOperation).Methods(http.MethodPut)
	r.HandleFunc("/operations/{id}/safety", h.ReportSafetyEvent).Methods(http.MethodPost)
	r.HandleFunc("/ports/{port_id}/equipment", h.CreateEquipment).Methods(http.MethodPost)
	r.HandleFunc("/ports/{port_id}/equipment", h.ListEquipment).Methods(http.MethodGet)
	r.HandleFunc("/equipment/{equip_id}/status", h.UpdateEquipmentStatus).Methods(http.MethodPut)
	r.HandleFunc("/ports/{port_id}/safety", h.ListSafetyEvents).Methods(http.MethodGet)
}

func respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.APIResponse{Success: code < 400, Data: data})
}
func respondErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.APIResponse{Error: msg})
}

func (h *DockHandler) CreateOperation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.CreateOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.PortID == "" || req.VesselID == "" || strings.TrimSpace(req.CargoType) == "" {
		respondErr(w, http.StatusBadRequest, "port_id, vessel_id, and cargo_type required")
		return
	}
	portID, err := uuid.Parse(req.PortID)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid port_id"); return }
	vesselID, err := uuid.Parse(req.VesselID)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid vessel_id"); return }
	berthID := uuid.Nil
	if req.BerthID != "" {
		berthID, err = uuid.Parse(req.BerthID)
		if err != nil { respondErr(w, http.StatusBadRequest, "invalid berth_id"); return }
	}
	currency := req.Currency
	if currency == "" { currency = "USD" }
	now := time.Now().UTC()
	op := models.DockOperation{
		ID: uuid.New(), PortID: portID, BerthID: berthID, VesselID: vesselID,
		OperationType: models.OperationType(req.OperationType), CargoType: req.CargoType,
		TonnageT: req.TonnageT, UnitCount: req.UnitCount, StevedoreTeam: req.StevedoreTeam,
		PlannedDuration: req.PlannedDuration, BillingAmount: req.BillingAmount,
		Currency: currency, SafetyIncident: false, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateOperation(r.Context(), op); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create operation")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.DockAuditLog{ID: uuid.New(), PortID: portID, ActorID: claims.UserID.String(), Action: "create_operation", EntityType: "dock_operation", EntityID: op.ID, CreatedAt: now})
	respond(w, http.StatusCreated, op)
}

func (h *DockHandler) GetOperation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	op, err := h.store.GetOperation(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "operation not found"); return }
	respond(w, http.StatusOK, op)
}

func (h *DockHandler) ListOperations(w http.ResponseWriter, r *http.Request) {
	params := db.ListOperationsParams{Page: 1, Limit: 50}
	if v := r.URL.Query().Get("port_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil { params.PortID = &id }
	}
	if v := r.URL.Query().Get("vessel_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil { params.VesselID = &id }
	}
	ops, err := h.store.ListOperations(r.Context(), params)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list operations"); return }
	if ops == nil { ops = []models.DockOperation{} }
	respond(w, http.StatusOK, ops)
}

func (h *DockHandler) StartOperation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req models.StartOperationRequest
	json.NewDecoder(r.Body).Decode(&req)
	now := time.Now().UTC()
	startedAt := now
	if req.StartedAt != "" {
		t, err := time.Parse(time.RFC3339, req.StartedAt)
		if err == nil { startedAt = t }
	}
	if err := h.store.StartOperation(r.Context(), id, startedAt, now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to start operation")
		return
	}
	op, _ := h.store.GetOperation(r.Context(), id)
	if op != nil {
		h.store.InsertAuditLog(r.Context(), models.DockAuditLog{ID: uuid.New(), PortID: op.PortID, ActorID: claims.UserID.String(), Action: "start_operation", EntityType: "dock_operation", EntityID: id, CreatedAt: now})
	}
	respond(w, http.StatusOK, op)
}

func (h *DockHandler) CompleteOperation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req models.CompleteOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	now := time.Now().UTC()
	completedAt := now
	if req.CompletedAt != "" {
		t, err := time.Parse(time.RFC3339, req.CompletedAt)
		if err == nil { completedAt = t }
	}
	if err := h.store.CompleteOperation(r.Context(), id, completedAt, req.ActualDuration, req.SafetyIncident, req.IncidentDetails, now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to complete operation")
		return
	}
	op, _ := h.store.GetOperation(r.Context(), id)
	if op != nil {
		h.store.InsertAuditLog(r.Context(), models.DockAuditLog{ID: uuid.New(), PortID: op.PortID, ActorID: claims.UserID.String(), Action: "complete_operation", EntityType: "dock_operation", EntityID: id, CreatedAt: now})
	}
	respond(w, http.StatusOK, op)
}

func (h *DockHandler) ReportSafetyEvent(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	opID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid operation id"); return }
	var req models.ReportSafetyEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		respondErr(w, http.StatusBadRequest, "description required")
		return
	}
	op, err := h.store.GetOperation(r.Context(), opID)
	if err != nil { respondErr(w, http.StatusNotFound, "operation not found"); return }
	now := time.Now().UTC()
	e := models.SafetyEvent{
		ID: uuid.New(), OperationID: opID, PortID: op.PortID,
		EventType: req.EventType, Severity: req.Severity,
		Description: req.Description, Injured: req.Injured,
		ReportedBy: claims.UserID.String(), CreatedAt: now,
	}
	if err := h.store.ReportSafetyEvent(r.Context(), e); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to report safety event")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.DockAuditLog{ID: uuid.New(), PortID: op.PortID, ActorID: claims.UserID.String(), Action: "report_safety_event", EntityType: "safety_event", EntityID: e.ID, CreatedAt: now})
	respond(w, http.StatusCreated, e)
}

func (h *DockHandler) CreateEquipment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	portID, err := uuid.Parse(mux.Vars(r)["port_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid port_id"); return }
	var req models.CreateEquipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		respondErr(w, http.StatusBadRequest, "model required")
		return
	}
	now := time.Now().UTC()
	id := uuid.New()
	code := "EQ-" + strings.ToUpper(id.String()[:8])
	e := models.Equipment{
		ID: id, PortID: portID, EquipmentCode: code,
		EquipmentType: models.EquipmentType(req.EquipmentType), Model: req.Model,
		Status: models.EquipAvailable, CapacityT: req.CapacityT, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateEquipment(r.Context(), e); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create equipment")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.DockAuditLog{ID: uuid.New(), PortID: portID, ActorID: claims.UserID.String(), Action: "create_equipment", EntityType: "dock_equipment", EntityID: e.ID, CreatedAt: now})
	respond(w, http.StatusCreated, e)
}

func (h *DockHandler) ListEquipment(w http.ResponseWriter, r *http.Request) {
	portID, err := uuid.Parse(mux.Vars(r)["port_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid port_id"); return }
	equip, err := h.store.ListEquipment(r.Context(), portID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list equipment"); return }
	if equip == nil { equip = []models.Equipment{} }
	respond(w, http.StatusOK, equip)
}

func (h *DockHandler) UpdateEquipmentStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["equip_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid equipment id"); return }
	var req struct{ Status string `json:"status"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.store.UpdateEquipmentStatus(r.Context(), id, models.EquipmentStatus(req.Status), time.Now().UTC()); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	e, _ := h.store.GetEquipment(r.Context(), id)
	respond(w, http.StatusOK, e)
}

func (h *DockHandler) ListSafetyEvents(w http.ResponseWriter, r *http.Request) {
	portID, err := uuid.Parse(mux.Vars(r)["port_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid port_id"); return }
	events, err := h.store.ListSafetyEvents(r.Context(), portID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list safety events"); return }
	if events == nil { events = []models.SafetyEvent{} }
	respond(w, http.StatusOK, events)
}
