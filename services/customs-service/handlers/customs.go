package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/customs-service/db"
	"github.com/klinova/kinara-os/customs-service/middleware"
	"github.com/klinova/kinara-os/customs-service/models"
)

type Store interface {
	CreateTariff(ctx context.Context, t models.TariffCode) error
	LookupTariff(ctx context.Context, hsCode, country string) (*models.TariffCode, error)
	ListTariffs(ctx context.Context, country *string) ([]models.TariffCode, error)
	CreateClearance(ctx context.Context, c models.ClearanceRequest) error
	GetClearance(ctx context.Context, id uuid.UUID) (*models.ClearanceRequest, error)
	ListClearances(ctx context.Context, portID *uuid.UUID, status *models.ClearanceStatus) ([]models.ClearanceRequest, error)
	UpdateClearanceStatus(ctx context.Context, id uuid.UUID, status models.ClearanceStatus, reviewerID, rejectionReason string, now time.Time) error
	InsertAuditLog(ctx context.Context, l models.CustomsAuditLog) error
}

type CustomsHandler struct{ store Store }

func NewHandler(q *db.Queries) *CustomsHandler        { return &CustomsHandler{store: q} }
func NewHandlerWithStore(s Store) *CustomsHandler      { return &CustomsHandler{store: s} }

func (h *CustomsHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/tariffs", h.CreateTariff).Methods(http.MethodPost)
	r.HandleFunc("/tariffs", h.ListTariffs).Methods(http.MethodGet)
	r.HandleFunc("/tariffs/lookup", h.LookupTariff).Methods(http.MethodGet)
	r.HandleFunc("/clearances", h.CreateClearance).Methods(http.MethodPost)
	r.HandleFunc("/clearances", h.ListClearances).Methods(http.MethodGet)
	r.HandleFunc("/clearances/{id}", h.GetClearance).Methods(http.MethodGet)
	r.HandleFunc("/clearances/{id}/status", h.UpdateClearanceStatus).Methods(http.MethodPut)
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

func (h *CustomsHandler) CreateTariff(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req models.CreateTariffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.HSCode) == "" || strings.TrimSpace(req.Country) == "" {
		respondErr(w, http.StatusBadRequest, "hs_code and country required")
		return
	}
	now := time.Now().UTC()
	portID := uuid.Nil
	t := models.TariffCode{
		ID: uuid.New(), HSCode: req.HSCode, Description: req.Description,
		Category: models.TariffCategory(req.Category), DutyRate: req.DutyRate, VATRate: req.VATRate,
		Country: req.Country, IsRestricted: req.IsRestricted, Notes: req.Notes, CreatedAt: now,
	}
	if err := h.store.CreateTariff(r.Context(), t); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create tariff")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CustomsAuditLog{ID: uuid.New(), PortID: portID, ActorID: claims.UserID.String(), Action: "create_tariff", EntityType: "tariff_code", EntityID: t.ID, CreatedAt: now})
	respond(w, http.StatusCreated, t)
}

func (h *CustomsHandler) LookupTariff(w http.ResponseWriter, r *http.Request) {
	hsCode := r.URL.Query().Get("hs_code")
	country := r.URL.Query().Get("country")
	if hsCode == "" || country == "" {
		respondErr(w, http.StatusBadRequest, "hs_code and country query params required")
		return
	}
	t, err := h.store.LookupTariff(r.Context(), hsCode, country)
	if err != nil { respondErr(w, http.StatusNotFound, "tariff not found"); return }
	// Calculate example duty on 1 unit
	respond(w, http.StatusOK, t)
}

func (h *CustomsHandler) ListTariffs(w http.ResponseWriter, r *http.Request) {
	var country *string
	if c := r.URL.Query().Get("country"); c != "" { country = &c }
	tariffs, err := h.store.ListTariffs(r.Context(), country)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list tariffs"); return }
	if tariffs == nil { tariffs = []models.TariffCode{} }
	respond(w, http.StatusOK, tariffs)
}

func (h *CustomsHandler) CreateClearance(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req models.CreateClearanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.ImporterName) == "" || strings.TrimSpace(req.HSCode) == "" || req.DeclaredValue <= 0 {
		respondErr(w, http.StatusBadRequest, "importer_name, hs_code, and declared_value required")
		return
	}
	manifestID, _ := uuid.Parse(req.ManifestID)
	vesselID, _ := uuid.Parse(req.VesselID)
	portID, _ := uuid.Parse(req.PortID)
	currency := req.Currency
	if currency == "" { currency = "USD" }

	// Auto-calculate duty and VAT using tariff lookup
	dutyRate := 0.0
	vatRate := 0.0
	t, err := h.store.LookupTariff(r.Context(), req.HSCode, "")
	if err == nil { dutyRate = t.DutyRate; vatRate = t.VATRate }
	dutyAmount := req.DeclaredValue * dutyRate / 100
	vatAmount := req.DeclaredValue * vatRate / 100
	totalDue := dutyAmount + vatAmount

	now := time.Now().UTC()
	id := uuid.New()
	refNo := "CR-" + strings.ToUpper(id.String()[:10])
	c := models.ClearanceRequest{
		ID: id, ReferenceNo: refNo, ImporterName: req.ImporterName, ImporterID: req.ImporterID,
		ManifestID: manifestID, VesselID: vesselID, PortID: portID,
		HSCode: req.HSCode, GoodsDescription: req.GoodsDescription,
		DeclaredValue: req.DeclaredValue, Currency: currency, WeightKg: req.WeightKg,
		DutyAmount: dutyAmount, VATAmount: vatAmount, TotalDue: totalDue,
		Status: models.ClearancePending, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateClearance(r.Context(), c); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create clearance")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CustomsAuditLog{ID: uuid.New(), PortID: portID, ActorID: claims.UserID.String(), Action: "create_clearance", EntityType: "clearance_request", EntityID: c.ID, CreatedAt: now})
	respond(w, http.StatusCreated, c)
}

func (h *CustomsHandler) GetClearance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	c, err := h.store.GetClearance(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "clearance not found"); return }
	respond(w, http.StatusOK, c)
}

func (h *CustomsHandler) ListClearances(w http.ResponseWriter, r *http.Request) {
	var portID *uuid.UUID
	var status *models.ClearanceStatus
	if v := r.URL.Query().Get("port_id"); v != "" {
		id, err := uuid.Parse(v); if err == nil { portID = &id }
	}
	if s := r.URL.Query().Get("status"); s != "" { st := models.ClearanceStatus(s); status = &st }
	clearances, err := h.store.ListClearances(r.Context(), portID, status)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list clearances"); return }
	if clearances == nil { clearances = []models.ClearanceRequest{} }
	respond(w, http.StatusOK, clearances)
}

func (h *CustomsHandler) UpdateClearanceStatus(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req models.UpdateClearanceStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	now := time.Now().UTC()
	c, err := h.store.GetClearance(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "clearance not found"); return }
	if err := h.store.UpdateClearanceStatus(r.Context(), id, models.ClearanceStatus(req.Status), claims.UserID.String(), req.RejectionReason, now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.CustomsAuditLog{ID: uuid.New(), PortID: c.PortID, ActorID: claims.UserID.String(), Action: "update_clearance_status:" + req.Status, EntityType: "clearance_request", EntityID: id, CreatedAt: now})
	updated, _ := h.store.GetClearance(r.Context(), id)
	respond(w, http.StatusOK, updated)
}
