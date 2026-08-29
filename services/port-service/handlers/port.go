package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/port-service/db"
	"github.com/klinova/kinara-os/port-service/middleware"
	"github.com/klinova/kinara-os/port-service/models"
)

type Store interface {
	CreatePort(ctx context.Context, p models.Port) error
	GetPort(ctx context.Context, id uuid.UUID) (*models.Port, error)
	ListPorts(ctx context.Context, country *string) ([]models.Port, error)
	CreateBerth(ctx context.Context, b models.Berth) error
	GetBerth(ctx context.Context, id uuid.UUID) (*models.Berth, error)
	ListBerths(ctx context.Context, p db.ListBerthsParams) ([]models.Berth, error)
	UpdateBerthStatus(ctx context.Context, id uuid.UUID, status models.BerthStatus, now time.Time) error
	CreateSchedule(ctx context.Context, s models.BerthSchedule) error
	GetSchedule(ctx context.Context, id uuid.UUID) (*models.BerthSchedule, error)
	ListSchedulesByBerth(ctx context.Context, berthID uuid.UUID) ([]models.BerthSchedule, error)
	UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status models.VesselStatus, now time.Time) error
	CreateCongestionAlert(ctx context.Context, a models.CongestionAlert) error
	ListAlerts(ctx context.Context, portID uuid.UUID) ([]models.CongestionAlert, error)
	InsertAuditLog(ctx context.Context, l models.PortAuditLog) error
}

type PortHandler struct{ store Store }

func NewHandler(q *db.Queries) *PortHandler        { return &PortHandler{store: q} }
func NewHandlerWithStore(s Store) *PortHandler      { return &PortHandler{store: s} }

func (h *PortHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ports", h.CreatePort).Methods(http.MethodPost)
	r.HandleFunc("/ports", h.ListPorts).Methods(http.MethodGet)
	r.HandleFunc("/ports/{id}", h.GetPort).Methods(http.MethodGet)
	r.HandleFunc("/ports/{id}/berths", h.CreateBerth).Methods(http.MethodPost)
	r.HandleFunc("/ports/{id}/berths", h.ListBerths).Methods(http.MethodGet)
	r.HandleFunc("/berths/{berth_id}", h.GetBerth).Methods(http.MethodGet)
	r.HandleFunc("/berths/{berth_id}/status", h.UpdateBerthStatus).Methods(http.MethodPut)
	r.HandleFunc("/berths/{berth_id}/schedules", h.ScheduleBerth).Methods(http.MethodPost)
	r.HandleFunc("/berths/{berth_id}/schedules", h.ListSchedules).Methods(http.MethodGet)
	r.HandleFunc("/schedules/{schedule_id}/status", h.UpdateScheduleStatus).Methods(http.MethodPut)
	r.HandleFunc("/ports/{id}/alerts", h.CreateAlert).Methods(http.MethodPost)
	r.HandleFunc("/ports/{id}/alerts", h.ListAlerts).Methods(http.MethodGet)
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

func (h *PortHandler) CreatePort(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req models.CreatePortRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Country) == "" || req.TotalBerths <= 0 {
		respondErr(w, http.StatusBadRequest, "name, country, and total_berths required")
		return
	}
	now := time.Now().UTC()
	id := uuid.New()
	code := "PT-" + strings.ToUpper(id.String()[:6])
	p := models.Port{
		ID: id, Name: req.Name, Code: code, Country: req.Country, City: req.City,
		Latitude: req.Latitude, Longitude: req.Longitude, MaxDraft: req.MaxDraft,
		TotalBerths: req.TotalBerths, AlertLevel: models.AlertNormal, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreatePort(r.Context(), p); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create port")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.PortAuditLog{ID: uuid.New(), PortID: id, ActorID: claims.UserID.String(), Action: "create_port", EntityType: "port", EntityID: id, CreatedAt: now})
	respond(w, http.StatusCreated, p)
}

func (h *PortHandler) GetPort(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	p, err := h.store.GetPort(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "port not found"); return }
	respond(w, http.StatusOK, p)
}

func (h *PortHandler) ListPorts(w http.ResponseWriter, r *http.Request) {
	var country *string
	if c := r.URL.Query().Get("country"); c != "" { country = &c }
	ports, err := h.store.ListPorts(r.Context(), country)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list ports"); return }
	if ports == nil { ports = []models.Port{} }
	respond(w, http.StatusOK, ports)
}

func (h *PortHandler) CreateBerth(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	portID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid port id"); return }
	var req models.CreateBerthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.BerthNumber) == "" {
		respondErr(w, http.StatusBadRequest, "berth_number required")
		return
	}
	now := time.Now().UTC()
	b := models.Berth{
		ID: uuid.New(), PortID: portID, BerthNumber: req.BerthNumber,
		Status: models.BerthAvailable, MaxLengthM: req.MaxLengthM, MaxDraftM: req.MaxDraftM,
		MaxTonnage: req.MaxTonnage, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateBerth(r.Context(), b); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create berth")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.PortAuditLog{ID: uuid.New(), PortID: portID, ActorID: claims.UserID.String(), Action: "create_berth", EntityType: "berth", EntityID: b.ID, CreatedAt: now})
	respond(w, http.StatusCreated, b)
}

func (h *PortHandler) GetBerth(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["berth_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	b, err := h.store.GetBerth(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "berth not found"); return }
	respond(w, http.StatusOK, b)
}

func (h *PortHandler) ListBerths(w http.ResponseWriter, r *http.Request) {
	portID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid port id"); return }
	params := db.ListBerthsParams{PortID: &portID}
	if s := r.URL.Query().Get("status"); s != "" { st := models.BerthStatus(s); params.Status = &st }
	berths, err := h.store.ListBerths(r.Context(), params)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list berths"); return }
	if berths == nil { berths = []models.Berth{} }
	respond(w, http.StatusOK, berths)
}

func (h *PortHandler) UpdateBerthStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["berth_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req struct{ Status string `json:"status"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.store.UpdateBerthStatus(r.Context(), id, models.BerthStatus(req.Status), time.Now().UTC()); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	b, _ := h.store.GetBerth(r.Context(), id)
	respond(w, http.StatusOK, b)
}

func (h *PortHandler) ScheduleBerth(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	berthID, err := uuid.Parse(mux.Vars(r)["berth_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid berth id"); return }
	var req models.ScheduleBerthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.VesselID == "" || req.VesselName == "" || req.ETA == "" || req.ETD == "" {
		respondErr(w, http.StatusBadRequest, "vessel_id, vessel_name, eta, etd required")
		return
	}
	vid, err := uuid.Parse(req.VesselID)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid vessel_id"); return }
	eta, err := time.Parse(time.RFC3339, req.ETA)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid eta"); return }
	etd, err := time.Parse(time.RFC3339, req.ETD)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid etd"); return }
	now := time.Now().UTC()
	berth, err := h.store.GetBerth(r.Context(), berthID)
	if err != nil { respondErr(w, http.StatusNotFound, "berth not found"); return }
	s := models.BerthSchedule{
		ID: uuid.New(), BerthID: berthID, VesselID: vid, VesselName: req.VesselName,
		Status: models.VesselExpected, ETA: eta, ETD: etd, CargoType: req.CargoType,
		TonnageT: req.TonnageT, Notes: req.Notes, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateSchedule(r.Context(), s); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to schedule berth")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.PortAuditLog{ID: uuid.New(), PortID: berth.PortID, ActorID: claims.UserID.String(), Action: "schedule_berth", EntityType: "berth_schedule", EntityID: s.ID, CreatedAt: now})
	respond(w, http.StatusCreated, s)
}

func (h *PortHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	berthID, err := uuid.Parse(mux.Vars(r)["berth_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid berth id"); return }
	schedules, err := h.store.ListSchedulesByBerth(r.Context(), berthID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list schedules"); return }
	if schedules == nil { schedules = []models.BerthSchedule{} }
	respond(w, http.StatusOK, schedules)
}

func (h *PortHandler) UpdateScheduleStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["schedule_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req models.UpdateScheduleStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.store.UpdateScheduleStatus(r.Context(), id, models.VesselStatus(req.Status), time.Now().UTC()); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	s, _ := h.store.GetSchedule(r.Context(), id)
	respond(w, http.StatusOK, s)
}

func (h *PortHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	portID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid port id"); return }
	var req models.CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		respondErr(w, http.StatusBadRequest, "message required")
		return
	}
	berths, _ := h.store.ListBerths(r.Context(), db.ListBerthsParams{PortID: &portID})
	occupied := 0
	for _, b := range berths { if b.Status == models.BerthOccupied { occupied++ } }
	total := len(berths)
	pct := 0.0
	if total > 0 { pct = float64(occupied) / float64(total) * 100 }
	now := time.Now().UTC()
	a := models.CongestionAlert{
		ID: uuid.New(), PortID: portID, AlertLevel: models.PortAlertLevel(req.AlertLevel),
		Message: req.Message, OccupiedBerths: occupied, TotalBerths: total,
		OccupancyPct: pct, CreatedAt: now,
	}
	if err := h.store.CreateCongestionAlert(r.Context(), a); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create alert")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.PortAuditLog{ID: uuid.New(), PortID: portID, ActorID: claims.UserID.String(), Action: "create_alert", EntityType: "congestion_alert", EntityID: a.ID, CreatedAt: now})
	respond(w, http.StatusCreated, a)
}

func (h *PortHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	portID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid port id"); return }
	alerts, err := h.store.ListAlerts(r.Context(), portID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list alerts"); return }
	if alerts == nil { alerts = []models.CongestionAlert{} }
	respond(w, http.StatusOK, alerts)
}
