package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/shipping-service/db"
	"github.com/klinova/kinara-os/shipping-service/middleware"
	"github.com/klinova/kinara-os/shipping-service/models"
)

type Store interface {
	CreateBooking(ctx context.Context, b models.FreightBooking) error
	GetBooking(ctx context.Context, id uuid.UUID) (*models.FreightBooking, error)
	ListBookings(ctx context.Context, shipperID *uuid.UUID, status *models.FreightStatus) ([]models.FreightBooking, error)
	UpdateBookingStatus(ctx context.Context, id uuid.UUID, status models.FreightStatus, now time.Time) error
	IssueBOL(ctx context.Context, bol models.BillOfLading) error
	GetBOL(ctx context.Context, id uuid.UUID) (*models.BillOfLading, error)
	SurrenderBOL(ctx context.Context, id uuid.UUID, now time.Time) error
	RecordDemurrage(ctx context.Context, d models.DemurrageRecord) error
	ListDemurrage(ctx context.Context, bookingID uuid.UUID) ([]models.DemurrageRecord, error)
	InsertAuditLog(ctx context.Context, l models.ShippingAuditLog) error
}

type ShippingHandler struct{ store Store }

func NewHandler(q *db.Queries) *ShippingHandler        { return &ShippingHandler{store: q} }
func NewHandlerWithStore(s Store) *ShippingHandler      { return &ShippingHandler{store: s} }

func (h *ShippingHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/bookings", h.CreateBooking).Methods(http.MethodPost)
	r.HandleFunc("/bookings", h.ListBookings).Methods(http.MethodGet)
	r.HandleFunc("/bookings/{id}", h.GetBooking).Methods(http.MethodGet)
	r.HandleFunc("/bookings/{id}/status", h.UpdateBookingStatus).Methods(http.MethodPut)
	r.HandleFunc("/bookings/{id}/bol", h.IssueBOL).Methods(http.MethodPost)
	r.HandleFunc("/bookings/{id}/demurrage", h.RecordDemurrage).Methods(http.MethodPost)
	r.HandleFunc("/bookings/{id}/demurrage", h.ListDemurrage).Methods(http.MethodGet)
	r.HandleFunc("/bol/{bol_id}", h.GetBOL).Methods(http.MethodGet)
	r.HandleFunc("/bol/{bol_id}/surrender", h.SurrenderBOL).Methods(http.MethodPut)
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

func (h *ShippingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req models.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.ShipperName) == "" || strings.TrimSpace(req.CommodityDesc) == "" {
		respondErr(w, http.StatusBadRequest, "shipper_name and commodity_description required")
		return
	}
	polID, _ := uuid.Parse(req.PortOfLoading)
	podID, _ := uuid.Parse(req.PortOfDischarge)
	shipperID, _ := uuid.Parse(claims.UserID)
	currency := req.Currency
	if currency == "" { currency = "USD" }
	insurancePct := req.InsurancePct
	if insurancePct == 0 { insurancePct = 0.5 } // default 0.5%
	insuranceAmount := req.DeclaredValue * insurancePct / 100
	totalFreight := req.FreightRate*float64(req.ContainerCount) + insuranceAmount
	if req.ContainerCount == 0 { req.ContainerCount = 1 }
	now := time.Now().UTC()
	id := uuid.New()
	ref := "SB-" + strings.ToUpper(id.String()[:10])
	shipType := models.ShipFCL
	if strings.ToLower(req.ShipmentType) == "lcl" { shipType = models.ShipLCL }
	b := models.FreightBooking{
		ID: id, BookingRef: ref, ShipperID: shipperID, ShipperName: req.ShipperName,
		ConsigneeName: req.ConsigneeName, ShipmentType: shipType,
		PortOfLoading: polID, PortOfDischarge: podID, CommodityDesc: req.CommodityDesc,
		ContainerCount: req.ContainerCount, WeightKg: req.WeightKg,
		FreightRate: req.FreightRate, InsurancePct: insurancePct,
		InsuranceAmount: insuranceAmount, DeclaredValue: req.DeclaredValue,
		TotalFreight: totalFreight, Currency: currency,
		Status: models.FreightPending, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateBooking(r.Context(), b); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create booking")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.ShippingAuditLog{ID: uuid.New(), ActorID: claims.UserID, Action: "create_booking", EntityType: "freight_booking", EntityID: b.ID, CreatedAt: now})
	respond(w, http.StatusCreated, b)
}

func (h *ShippingHandler) GetBooking(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	b, err := h.store.GetBooking(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "booking not found"); return }
	respond(w, http.StatusOK, b)
}

func (h *ShippingHandler) ListBookings(w http.ResponseWriter, r *http.Request) {
	var shipperID *uuid.UUID
	var status *models.FreightStatus
	if v := r.URL.Query().Get("shipper_id"); v != "" {
		id, err := uuid.Parse(v); if err == nil { shipperID = &id }
	}
	if s := r.URL.Query().Get("status"); s != "" { st := models.FreightStatus(s); status = &st }
	bookings, err := h.store.ListBookings(r.Context(), shipperID, status)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list bookings"); return }
	if bookings == nil { bookings = []models.FreightBooking{} }
	respond(w, http.StatusOK, bookings)
}

func (h *ShippingHandler) UpdateBookingStatus(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req struct{ Status string `json:"status"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	now := time.Now().UTC()
	if err := h.store.UpdateBookingStatus(r.Context(), id, models.FreightStatus(req.Status), now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.ShippingAuditLog{ID: uuid.New(), ActorID: claims.UserID, Action: "update_booking_status:" + req.Status, EntityType: "freight_booking", EntityID: id, CreatedAt: now})
	b, _ := h.store.GetBooking(r.Context(), id)
	respond(w, http.StatusOK, b)
}

func (h *ShippingHandler) IssueBOL(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	bookingID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid booking id"); return }
	var req models.IssueBOLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.VesselName) == "" {
		respondErr(w, http.StatusBadRequest, "vessel_name required")
		return
	}
	b, err := h.store.GetBooking(r.Context(), bookingID)
	if err != nil { respondErr(w, http.StatusNotFound, "booking not found"); return }
	now := time.Now().UTC()
	id := uuid.New()
	bolNo := "BL-" + strings.ToUpper(id.String()[:10])
	bol := models.BillOfLading{
		ID: id, BOLNumber: bolNo, BookingID: bookingID,
		VesselName: req.VesselName, VoyageNo: req.VoyageNo,
		ShipperName: b.ShipperName, ConsigneeName: b.ConsigneeName,
		NotifyParty: req.NotifyParty, POL: req.POL, POD: req.POD,
		CommodityDesc: b.CommodityDesc, ContainerCount: b.ContainerCount,
		GrossWeightKg: b.WeightKg, FreightPrepaid: req.FreightPrepaid,
		Status: models.BOLIssued, IssuedAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.IssueBOL(r.Context(), bol); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to issue BOL")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.ShippingAuditLog{ID: uuid.New(), ActorID: claims.UserID, Action: "issue_bol", EntityType: "bill_of_lading", EntityID: bol.ID, CreatedAt: now})
	respond(w, http.StatusCreated, bol)
}

func (h *ShippingHandler) GetBOL(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["bol_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid bol id"); return }
	bol, err := h.store.GetBOL(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "BOL not found"); return }
	respond(w, http.StatusOK, bol)
}

func (h *ShippingHandler) SurrenderBOL(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["bol_id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid bol id"); return }
	now := time.Now().UTC()
	if err := h.store.SurrenderBOL(r.Context(), id, now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to surrender BOL")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.ShippingAuditLog{ID: uuid.New(), ActorID: claims.UserID, Action: "surrender_bol", EntityType: "bill_of_lading", EntityID: id, CreatedAt: now})
	bol, _ := h.store.GetBOL(r.Context(), id)
	respond(w, http.StatusOK, bol)
}

func (h *ShippingHandler) RecordDemurrage(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	bookingID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid booking id"); return }
	var req models.RecordDemurrageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.ContainerNo) == "" || req.UsedDays <= req.FreeDays {
		respondErr(w, http.StatusBadRequest, "container_no required and used_days must exceed free_days")
		return
	}
	portID, _ := uuid.Parse(req.PortID)
	currency := req.Currency
	if currency == "" { currency = "USD" }
	chargeableDays := req.UsedDays - req.FreeDays
	totalCharge := float64(chargeableDays) * req.DailyRate
	now := time.Now().UTC()
	d := models.DemurrageRecord{
		ID: uuid.New(), BookingID: bookingID, ContainerNo: req.ContainerNo,
		FreeDays: req.FreeDays, UsedDays: req.UsedDays, DailyRate: req.DailyRate,
		TotalCharge: totalCharge, Currency: currency, PortID: portID, CreatedAt: now,
	}
	if err := h.store.RecordDemurrage(r.Context(), d); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to record demurrage")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.ShippingAuditLog{ID: uuid.New(), ActorID: claims.UserID, Action: "record_demurrage", EntityType: "demurrage_record", EntityID: d.ID, CreatedAt: now})
	respond(w, http.StatusCreated, d)
}

func (h *ShippingHandler) ListDemurrage(w http.ResponseWriter, r *http.Request) {
	bookingID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid booking id"); return }
	records, err := h.store.ListDemurrage(r.Context(), bookingID)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list demurrage"); return }
	if records == nil { records = []models.DemurrageRecord{} }
	respond(w, http.StatusOK, records)
}
