package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/vessel-service/db"
	"github.com/klinova/kinara-os/vessel-service/middleware"
	"github.com/klinova/kinara-os/vessel-service/models"
)

type Store interface {
	RegisterVessel(ctx context.Context, v models.Vessel) error
	GetVessel(ctx context.Context, id uuid.UUID) (*models.Vessel, error)
	ListVessels(ctx context.Context, flag *string, activeOnly bool) ([]models.Vessel, error)
	UpdateVesselCondition(ctx context.Context, id uuid.UUID, condition models.VesselCondition, now time.Time) error
	LogVoyage(ctx context.Context, v models.VoyageRecord) error
	GetVoyage(ctx context.Context, id uuid.UUID) (*models.VoyageRecord, error)
	ListVoyages(ctx context.Context, vesselID uuid.UUID) ([]models.VoyageRecord, error)
	LogMaintenance(ctx context.Context, m models.MaintenanceRecord) error
	ListMaintenance(ctx context.Context, vesselID uuid.UUID) ([]models.MaintenanceRecord, error)
	InsertAuditLog(ctx context.Context, l models.VesselAuditLog) error
}

type VesselHandler struct{ store Store }

func NewHandler(q *db.Queries) *VesselHandler       { return &VesselHandler{store: q} }
func NewHandlerWithStore(s Store) *VesselHandler     { return &VesselHandler{store: s} }

func (h *VesselHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/vessels", h.RegisterVessel).Methods(http.MethodPost)
	r.HandleFunc("/vessels", h.ListVessels).Methods(http.MethodGet)
	r.HandleFunc("/vessels/{id}", h.GetVessel).Methods(http.MethodGet)
	r.HandleFunc("/vessels/{id}/condition", h.UpdateCondition).Methods(http.MethodPut)
	r.HandleFunc("/vessels/{id}/voyages", h.LogVoyage).Methods(http.MethodPost)
	r.HandleFunc("/vessels/{id}/voyages", h.ListVoyages).Methods(http.MethodGet)
	r.HandleFunc("/vessels/{id}/maintenance", h.LogMaintenance).Methods(http.MethodPost)
	r.HandleFunc("/vessels/{id}/maintenance", h.ListMaintenance).Methods(http.MethodGet)
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

func (h *VesselHandler) RegisterVessel(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.RegisterVesselRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.IMONumber) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Flag) == "" {
		respondErr(w, http.StatusBadRequest, "imo_number, name, and flag required")
		return
	}
	now := time.Now().UTC()
	operatorID := claims.UserID
	v := models.Vessel{
		ID: uuid.New(), IMONumber: req.IMONumber, Name: req.Name,
		VesselType: models.VesselType(req.VesselType), Flag: req.Flag, Owner: req.Owner,
		OperatorID: operatorID, YearBuilt: req.YearBuilt, GrossTonnage: req.GrossTonnage,
		DeadweightT: req.DeadweightT, LengthM: req.LengthM, BeamM: req.BeamM,
		MaxDraftM: req.MaxDraftM, MaxSpeed: req.MaxSpeed,
		Condition: models.ConditionGood, IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.RegisterVessel(r.Context(), v); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to register vessel")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.VesselAuditLog{ID: uuid.New(), VesselID: v.ID, ActorID: claims.UserID.String(), Action: "register_vessel", CreatedAt: now})
	respond(w, http.StatusCreated, v)
}

func (h *VesselHandler) GetVessel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	v, err := h.store.GetVessel(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "vessel not found"); return }
	respond(w, http.StatusOK, v)
}

func (h *VesselHandler) ListVessels(w http.ResponseWriter, r *http.Request) {
	var flag *string
	if f := r.URL.Query().Get("flag"); f != "" { flag = &f }
	activeOnly := r.URL.Query().Get("active") != "false"
	vessels, err := h.store.ListVessels(r.Context(), flag, activeOnly)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list vessels"); return }
	if vessels == nil { vessels = []models.Vessel{} }
	respond(w, http.StatusOK, vessels)
}

func (h *VesselHandler) UpdateCondition(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req struct{ Condition string `json:"condition"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	now := time.Now().UTC()
	if err := h.store.UpdateVesselCondition(r.Context(), id, models.VesselCondition(req.Condition), now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to update condition")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.VesselAuditLog{ID: uuid.New(), VesselID: id, ActorID: claims.UserID.String(), Action: "update_condition:" + req.Condition, CreatedAt: now})
	v, _ := h.store.GetVessel(r.Context(), id)
	respond(w, http.StatusOK, v)
}

func (h *VesselHandler) LogVoyage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	vesselID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid vessel id"); return }
	var req models.LogVoyageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.DeparturePortID == "" || req.ArrivalPortID == "" {
		respondErr(w, http.StatusBadRequest, "departure_port_id and arrival_port_id required")
		return
	}
	depID, err := uuid.Parse(req.DeparturePortID)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid departure_port_id"); return }
	arrID, err := uuid.Parse(req.ArrivalPortID)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid arrival_port_id"); return }
	now := time.Now().UTC()
	id := uuid.New()
	code := "VO-" + strings.ToUpper(id.String()[:8])
	voy := models.VoyageRecord{
		ID: id, VesselID: vesselID, VoyageCode: code,
		DeparturePortID: depID, ArrivalPortID: arrID,
		DistanceNM: req.DistanceNM, CargoTonnage: req.CargoTonnage, Notes: req.Notes, CreatedAt: now,
	}
	if req.DepartedAt != "" {
		t, err := time.Parse(time.RFC3339, req.DepartedAt)
		if err == nil { voy.DepartedAt = &t }
	}
	if req.ArrivedAt != "" {
		t, err := time.Parse(time.RFC3339, req.ArrivedAt)
		if err == nil { voy.ArrivedAt = &t }
	}
	if err := h.store.LogVoyage(r.Context(), voy); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to log voyage")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.VesselAuditLog{ID: uuid.New(), VesselID: vesselID, ActorID: claims.UserID.String(), Action: "log_voyage", CreatedAt: now})
	respond(w, http.StatusCreated, voy)
}

func (h *VesselHandler) ListVoyages(w http.ResponseWriter, r *http.Request) {
	vesselID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid vessel id"); return }
	voyages, err := h.store.ListVoyages(r.Context(), vesselID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list voyages"); return }
	if voyages == nil { voyages = []models.VoyageRecord{} }
	respond(w, http.StatusOK, voyages)
}

func (h *VesselHandler) LogMaintenance(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	vesselID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid vessel id"); return }
	var req models.LogMaintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Description) == "" || req.StartDate == "" {
		respondErr(w, http.StatusBadRequest, "description and start_date required")
		return
	}
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid start_date"); return }
	now := time.Now().UTC()
	currency := req.Currency
	if currency == "" { currency = "USD" }
	m := models.MaintenanceRecord{
		ID: uuid.New(), VesselID: vesselID, MaintenanceType: models.MaintenanceType(req.MaintenanceType),
		Description: req.Description, StartDate: startDate, Cost: req.Cost,
		Currency: currency, Vendor: req.Vendor, Completed: false, CreatedAt: now,
	}
	if err := h.store.LogMaintenance(r.Context(), m); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to log maintenance")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.VesselAuditLog{ID: uuid.New(), VesselID: vesselID, ActorID: claims.UserID.String(), Action: "log_maintenance:" + req.MaintenanceType, CreatedAt: now})
	respond(w, http.StatusCreated, m)
}

func (h *VesselHandler) ListMaintenance(w http.ResponseWriter, r *http.Request) {
	vesselID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid vessel id"); return }
	records, err := h.store.ListMaintenance(r.Context(), vesselID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list maintenance"); return }
	if records == nil { records = []models.MaintenanceRecord{} }
	respond(w, http.StatusOK, records)
}
