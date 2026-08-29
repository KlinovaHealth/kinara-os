package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/cargo-maritime-service/db"
	"github.com/klinova/kinara-os/cargo-maritime-service/middleware"
	"github.com/klinova/kinara-os/cargo-maritime-service/models"
)

type Store interface {
	RegisterContainer(ctx context.Context, c models.Container) error
	GetContainer(ctx context.Context, id uuid.UUID) (*models.Container, error)
	ListContainers(ctx context.Context, status *models.ContainerStatus, vesselID *uuid.UUID) ([]models.Container, error)
	UpdateContainerStatus(ctx context.Context, id uuid.UUID, status models.ContainerStatus, sealNo string, portID, vesselID *uuid.UUID, now time.Time) error
	CreateManifest(ctx context.Context, m models.CargoManifest) error
	GetManifest(ctx context.Context, id uuid.UUID) (*models.CargoManifest, error)
	AddContainerToManifest(ctx context.Context, mc models.ManifestContainer, weightKg float64) error
	FinalizeManifest(ctx context.Context, id uuid.UUID, now time.Time) error
	ReportDamage(ctx context.Context, d models.DamageReport) error
	ListDamageReports(ctx context.Context, containerID uuid.UUID) ([]models.DamageReport, error)
	InsertAuditLog(ctx context.Context, l models.CargoMaritimeAuditLog) error
}

type CargoMaritimeHandler struct{ store Store }

func NewHandler(q *db.Queries) *CargoMaritimeHandler        { return &CargoMaritimeHandler{store: q} }
func NewHandlerWithStore(s Store) *CargoMaritimeHandler      { return &CargoMaritimeHandler{store: s} }

func (h *CargoMaritimeHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/containers", h.RegisterContainer).Methods(http.MethodPost)
	r.HandleFunc("/containers", h.ListContainers).Methods(http.MethodGet)
	r.HandleFunc("/containers/{id}", h.GetContainer).Methods(http.MethodGet)
	r.HandleFunc("/containers/{id}/status", h.UpdateContainerStatus).Methods(http.MethodPut)
	r.HandleFunc("/containers/{id}/damage", h.ReportDamage).Methods(http.MethodPost)
	r.HandleFunc("/containers/{id}/damage", h.ListDamageReports).Methods(http.MethodGet)
	r.HandleFunc("/manifests", h.CreateManifest).Methods(http.MethodPost)
	r.HandleFunc("/manifests/{id}", h.GetManifest).Methods(http.MethodGet)
	r.HandleFunc("/manifests/{id}/containers", h.AddContainerToManifest).Methods(http.MethodPost)
	r.HandleFunc("/manifests/{id}/finalize", h.FinalizeManifest).Methods(http.MethodPut)
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

func (h *CargoMaritimeHandler) RegisterContainer(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req models.RegisterContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.ContainerNo) == "" {
		respondErr(w, http.StatusBadRequest, "container_no required")
		return
	}
	ownerID := claims.UserID
	now := time.Now().UTC()
	c := models.Container{
		ID: uuid.New(), ContainerNo: req.ContainerNo,
		ContainerType: models.ContainerType(req.ContainerType), OwnerID: ownerID,
		Status: models.StatusEmpty, TareWeightKg: req.TareWeightKg, WeightKg: req.TareWeightKg,
		IsHazmat: req.IsHazmat, HazmatClass: req.HazmatClass, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.RegisterContainer(r.Context(), c); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to register container")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CargoMaritimeAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "register_container", EntityType: "container", EntityID: c.ID, CreatedAt: now})
	respond(w, http.StatusCreated, c)
}

func (h *CargoMaritimeHandler) GetContainer(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	c, err := h.store.GetContainer(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "container not found"); return }
	respond(w, http.StatusOK, c)
}

func (h *CargoMaritimeHandler) ListContainers(w http.ResponseWriter, r *http.Request) {
	var status *models.ContainerStatus
	var vesselID *uuid.UUID
	if s := r.URL.Query().Get("status"); s != "" { st := models.ContainerStatus(s); status = &st }
	if v := r.URL.Query().Get("vessel_id"); v != "" {
		id, err := uuid.Parse(v); if err == nil { vesselID = &id }
	}
	containers, err := h.store.ListContainers(r.Context(), status, vesselID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list containers"); return }
	if containers == nil { containers = []models.Container{} }
	respond(w, http.StatusOK, containers)
}

func (h *CargoMaritimeHandler) UpdateContainerStatus(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req models.UpdateContainerStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	var portID, vesselID *uuid.UUID
	if req.PortID != "" { pid, err := uuid.Parse(req.PortID); if err == nil { portID = &pid } }
	if req.VesselID != "" { vid, err := uuid.Parse(req.VesselID); if err == nil { vesselID = &vid } }
	now := time.Now().UTC()
	if err := h.store.UpdateContainerStatus(r.Context(), id, models.ContainerStatus(req.Status), req.SealNo, portID, vesselID, now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CargoMaritimeAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "update_container_status:" + req.Status, EntityType: "container", EntityID: id, CreatedAt: now})
	c, _ := h.store.GetContainer(r.Context(), id)
	respond(w, http.StatusOK, c)
}

func (h *CargoMaritimeHandler) ReportDamage(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	containerID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid container id"); return }
	var req models.ReportDamageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		respondErr(w, http.StatusBadRequest, "description required")
		return
	}
	c, err := h.store.GetContainer(r.Context(), containerID)
	if err != nil { respondErr(w, http.StatusNotFound, "container not found"); return }
	portID := uuid.Nil
	if req.PortID != "" { portID, _ = uuid.Parse(req.PortID) }
	currency := req.Currency
	if currency == "" { currency = "USD" }
	now := time.Now().UTC()
	d := models.DamageReport{
		ID: uuid.New(), ContainerID: containerID, ContainerNo: c.ContainerNo,
		DamageLevel: models.DamageLevel(req.DamageLevel), Description: req.Description,
		PhotoURL: req.PhotoURL, ReportedBy: claims.UserID.String(),
		EstimatedCost: req.EstimatedCost, Currency: currency, PortID: portID, CreatedAt: now,
	}
	if err := h.store.ReportDamage(r.Context(), d); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to report damage")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CargoMaritimeAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "report_damage", EntityType: "damage_report", EntityID: d.ID, CreatedAt: now})
	respond(w, http.StatusCreated, d)
}

func (h *CargoMaritimeHandler) ListDamageReports(w http.ResponseWriter, r *http.Request) {
	containerID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid container id"); return }
	reports, err := h.store.ListDamageReports(r.Context(), containerID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list damage reports"); return }
	if reports == nil { reports = []models.DamageReport{} }
	respond(w, http.StatusOK, reports)
}

func (h *CargoMaritimeHandler) CreateManifest(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req models.CreateManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.VesselID == "" || req.ShipperName == "" {
		respondErr(w, http.StatusBadRequest, "vessel_id and shipper_name required")
		return
	}
	vesselID, err := uuid.Parse(req.VesselID)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid vessel_id"); return }
	voyageID, _ := uuid.Parse(req.VoyageID)
	polID, _ := uuid.Parse(req.PortOfLoading)
	podID, _ := uuid.Parse(req.PortOfDischarge)
	now := time.Now().UTC()
	id := uuid.New()
	manifestNo := "MF-" + strings.ToUpper(id.String()[:10])
	m := models.CargoManifest{
		ID: id, ManifestNo: manifestNo, VoyageID: voyageID, VesselID: vesselID,
		PortOfLoading: polID, PortOfDischarge: podID,
		ShipperName: req.ShipperName, ConsigneeName: req.ConsigneeName,
		Commodity: req.Commodity, IsFinalized: false, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateManifest(r.Context(), m); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create manifest")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CargoMaritimeAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "create_manifest", EntityType: "cargo_manifest", EntityID: m.ID, CreatedAt: now})
	respond(w, http.StatusCreated, m)
}

func (h *CargoMaritimeHandler) GetManifest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	m, err := h.store.GetManifest(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "manifest not found"); return }
	respond(w, http.StatusOK, m)
}

func (h *CargoMaritimeHandler) AddContainerToManifest(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	manifestID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid manifest id"); return }
	var req models.AddContainerToManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	containerID, err := uuid.Parse(req.ContainerID)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid container_id"); return }
	c, err := h.store.GetContainer(r.Context(), containerID)
	if err != nil { respondErr(w, http.StatusNotFound, "container not found"); return }
	now := time.Now().UTC()
	mc := models.ManifestContainer{ID: uuid.New(), ManifestID: manifestID, ContainerID: containerID, ContainerNo: c.ContainerNo, AddedAt: now}
	if err := h.store.AddContainerToManifest(r.Context(), mc, c.WeightKg); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to add container to manifest")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CargoMaritimeAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "add_container_to_manifest", EntityType: "manifest_container", EntityID: mc.ID, CreatedAt: now})
	respond(w, http.StatusCreated, mc)
}

func (h *CargoMaritimeHandler) FinalizeManifest(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid manifest id"); return }
	now := time.Now().UTC()
	if err := h.store.FinalizeManifest(r.Context(), id, now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to finalize manifest")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CargoMaritimeAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "finalize_manifest", EntityType: "cargo_manifest", EntityID: id, CreatedAt: now})
	m, _ := h.store.GetManifest(r.Context(), id)
	respond(w, http.StatusOK, m)
}
