package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/trade-finance-service/db"
	"github.com/klinova/kinara-os/trade-finance-service/middleware"
	"github.com/klinova/kinara-os/trade-finance-service/models"
)

type Store interface {
	CreateLC(ctx context.Context, lc models.LetterOfCredit) error
	GetLC(ctx context.Context, id uuid.UUID) (*models.LetterOfCredit, error)
	ListLCs(ctx context.Context, applicantID *uuid.UUID, status *models.LCStatus) ([]models.LetterOfCredit, error)
	UpdateLCStatus(ctx context.Context, id uuid.UUID, status models.LCStatus, now time.Time) error
	CreateFinancing(ctx context.Context, f models.FinancingRequest) error
	GetFinancing(ctx context.Context, id uuid.UUID) (*models.FinancingRequest, error)
	ApproveFinancing(ctx context.Context, id uuid.UUID, now time.Time) error
	DisburseFinancing(ctx context.Context, id uuid.UUID, now time.Time) error
	InsertAuditLog(ctx context.Context, l models.TradeFinanceAuditLog) error
}

type TFHandler struct{ store Store }

func NewHandler(q *db.Queries) *TFHandler        { return &TFHandler{store: q} }
func NewHandlerWithStore(s Store) *TFHandler      { return &TFHandler{store: s} }

func (h *TFHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/lc", h.CreateLC).Methods(http.MethodPost)
	r.HandleFunc("/lc", h.ListLCs).Methods(http.MethodGet)
	r.HandleFunc("/lc/{id}", h.GetLC).Methods(http.MethodGet)
	r.HandleFunc("/lc/{id}/status", h.UpdateLCStatus).Methods(http.MethodPut)
	r.HandleFunc("/financing", h.CreateFinancing).Methods(http.MethodPost)
	r.HandleFunc("/financing/{id}", h.GetFinancing).Methods(http.MethodGet)
	r.HandleFunc("/financing/{id}/approve", h.ApproveFinancing).Methods(http.MethodPut)
	r.HandleFunc("/financing/{id}/disburse", h.DisburseFinancing).Methods(http.MethodPut)
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

func (h *TFHandler) CreateLC(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.CreateLCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.BeneficiaryName) == "" || strings.TrimSpace(req.IssuingBank) == "" || req.Amount <= 0 {
		respondErr(w, http.StatusBadRequest, "beneficiary_name, issuing_bank, and amount required")
		return
	}
	expiry, err := time.Parse(time.RFC3339, req.ExpiryDate)
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid expiry_date"); return }
	applicantID := claims.UserID
	now := time.Now().UTC()
	id := uuid.New()
	lcNo := "LC-" + strings.ToUpper(id.String()[:10])
	lcType := models.LCStandard
	if req.LCType == "standby" { lcType = models.LCStandby }
	currency := req.Currency
	if currency == "" { currency = "USD" }
	docs := req.DocumentsRequired
	if docs == nil { docs = []string{"bill_of_lading", "commercial_invoice", "packing_list"} }
	lc := models.LetterOfCredit{
		ID: id, LCNumber: lcNo, LCType: lcType, ApplicantID: applicantID,
		ApplicantName: req.ApplicantName, BeneficiaryName: req.BeneficiaryName,
		IssuingBank: req.IssuingBank, AdvisingBank: req.AdvisingBank,
		Amount: req.Amount, Currency: currency, ExpiryDate: expiry,
		ShipmentPOL: req.ShipmentPOL, ShipmentPOD: req.ShipmentPOD,
		GoodsDescription: req.GoodsDescription, DocumentsRequired: docs,
		Status: models.LCDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateLC(r.Context(), lc); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create LC")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.TradeFinanceAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "create_lc", EntityType: "letter_of_credit", EntityID: lc.ID, CreatedAt: now})
	respond(w, http.StatusCreated, lc)
}

func (h *TFHandler) GetLC(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	lc, err := h.store.GetLC(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "LC not found"); return }
	respond(w, http.StatusOK, lc)
}

func (h *TFHandler) ListLCs(w http.ResponseWriter, r *http.Request) {
	var applicantID *uuid.UUID
	var status *models.LCStatus
	if v := r.URL.Query().Get("applicant_id"); v != "" {
		id, err := uuid.Parse(v); if err == nil { applicantID = &id }
	}
	if s := r.URL.Query().Get("status"); s != "" { st := models.LCStatus(s); status = &st }
	lcs, err := h.store.ListLCs(r.Context(), applicantID, status)
	if err != nil { respondErr(w, http.StatusInternalServerError, "failed to list LCs"); return }
	if lcs == nil { lcs = []models.LetterOfCredit{} }
	respond(w, http.StatusOK, lcs)
}

func (h *TFHandler) UpdateLCStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	var req models.UpdateLCStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	now := time.Now().UTC()
	if err := h.store.UpdateLCStatus(r.Context(), id, models.LCStatus(req.Status), now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to update LC status")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.TradeFinanceAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "update_lc_status:" + req.Status, EntityType: "letter_of_credit", EntityID: id, CreatedAt: now})
	lc, _ := h.store.GetLC(r.Context(), id)
	respond(w, http.StatusOK, lc)
}

func (h *TFHandler) CreateFinancing(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.CreateFinancingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.RequestedAmount <= 0 {
		respondErr(w, http.StatusBadRequest, "requested_amount required")
		return
	}
	applicantID := claims.UserID
	currency := req.Currency
	if currency == "" { currency = "USD" }
	interestRate := req.InterestRatePct
	if interestRate == 0 { interestRate = 3.5 }
	interestAmount := req.RequestedAmount * interestRate / 100
	totalRepayable := req.RequestedAmount + interestAmount
	var bookingID, lcID *uuid.UUID
	if req.BookingID != "" { bid, err := uuid.Parse(req.BookingID); if err == nil { bookingID = &bid } }
	if req.LCID != "" { lid, err := uuid.Parse(req.LCID); if err == nil { lcID = &lid } }
	now := time.Now().UTC()
	id := uuid.New()
	ref := "TF-" + strings.ToUpper(id.String()[:10])
	f := models.FinancingRequest{
		ID: id, RefNo: ref, ApplicantID: applicantID, BookingID: bookingID, LCID: lcID,
		RequestedAmount: req.RequestedAmount, Currency: currency,
		PaymentTerms: models.PaymentTerms(req.PaymentTerms),
		InterestRatePct: interestRate, InterestAmount: interestAmount,
		TotalRepayable: totalRepayable, Status: "pending", CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateFinancing(r.Context(), f); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to create financing request")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.TradeFinanceAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "create_financing", EntityType: "financing_request", EntityID: f.ID, CreatedAt: now})
	respond(w, http.StatusCreated, f)
}

func (h *TFHandler) GetFinancing(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	f, err := h.store.GetFinancing(r.Context(), id)
	if err != nil { respondErr(w, http.StatusNotFound, "financing request not found"); return }
	respond(w, http.StatusOK, f)
}

func (h *TFHandler) ApproveFinancing(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	now := time.Now().UTC()
	if err := h.store.ApproveFinancing(r.Context(), id, now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to approve")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.TradeFinanceAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "approve_financing", EntityType: "financing_request", EntityID: id, CreatedAt: now})
	f, _ := h.store.GetFinancing(r.Context(), id)
	respond(w, http.StatusOK, f)
}

func (h *TFHandler) DisburseFinancing(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { respondErr(w, http.StatusBadRequest, "invalid id"); return }
	now := time.Now().UTC()
	if err := h.store.DisburseFinancing(r.Context(), id, now); err != nil {
		respondErr(w, http.StatusInternalServerError, "failed to disburse")
		return
	}
	h.store.InsertAuditLog(r.Context(), models.TradeFinanceAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "disburse_financing", EntityType: "financing_request", EntityID: id, CreatedAt: now})
	f, _ := h.store.GetFinancing(r.Context(), id)
	respond(w, http.StatusOK, f)
}
