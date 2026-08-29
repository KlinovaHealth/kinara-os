package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/wallet-service/db"
	"github.com/klinova/kinara-os/wallet-service/models"
)

type Store interface {
	UpsertBalance(ctx context.Context, userID uuid.UUID, currency models.Currency, balance float64) error
	GetBalance(ctx context.Context, userID uuid.UUID, currency models.Currency) (float64, time.Time, error)
	GetAllBalances(ctx context.Context, userID uuid.UUID) ([]models.WalletBalance, error)
	SaveReconciliationLog(ctx context.Context, l models.ReconciliationLog) error
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
	api := r.PathPrefix("/api/v1/wallets").Subrouter()
	api.HandleFunc("/{user_id}/balances", h.getBalances).Methods(http.MethodGet)
	api.HandleFunc("/{user_id}/balance/{currency}", h.getBalance).Methods(http.MethodGet)
	api.HandleFunc("/{user_id}/balance/{currency}", h.upsertBalance).Methods(http.MethodPut)
	api.HandleFunc("/{user_id}/reconcile", h.reconcile).Methods(http.MethodPost)
	api.HandleFunc("/{user_id}/reconciliation-report", h.reconciliationReport).Methods(http.MethodGet)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "wallet-service"})
}

func (h *Handler) getBalances(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(mux.Vars(r)["user_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}
	balances, err := h.store.GetAllBalances(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to get balances")
		return
	}
	if balances == nil {
		balances = []models.WalletBalance{}
	}
	totalUSD := calculateTotalUSD(balances)
	respond(w, http.StatusOK, map[string]interface{}{
		"user_id":         userID,
		"balances":        balances,
		"total_usd":       totalUSD,
	})
}

func (h *Handler) getBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(mux.Vars(r)["user_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}
	currency := models.Currency(strings.ToUpper(mux.Vars(r)["currency"]))
	if _, ok := models.ExchangeRates[currency]; !ok {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "unsupported currency")
		return
	}
	balance, updatedAt, err := h.store.GetBalance(r.Context(), userID, currency)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "wallet balance not found")
		return
	}
	respond(w, http.StatusOK, models.WalletBalance{
		UserID:    userID,
		Currency:  currency,
		Balance:   balance,
		UpdatedAt: updatedAt,
	})
}

func (h *Handler) upsertBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(mux.Vars(r)["user_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}
	currency := models.Currency(strings.ToUpper(mux.Vars(r)["currency"]))
	if _, ok := models.ExchangeRates[currency]; !ok {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "unsupported currency")
		return
	}
	var req struct {
		Balance float64 `json:"balance"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Balance < 0 {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "balance cannot be negative")
		return
	}
	if err := h.store.UpsertBalance(r.Context(), userID, currency, req.Balance); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to upsert balance")
		return
	}
	respond(w, http.StatusOK, models.WalletBalance{
		UserID:    userID,
		Currency:  currency,
		Balance:   req.Balance,
		UpdatedAt: time.Now().UTC(),
	})
}

func (h *Handler) reconcile(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(mux.Vars(r)["user_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}
	balances, err := h.store.GetAllBalances(r.Context(), userID)
	if err != nil || len(balances) == 0 {
		balances = []models.WalletBalance{}
	}
	totalUSD := calculateTotalUSD(balances)
	// Reconciliation: verify sum of all balances converted to USD matches expected
	// In production: compare against payment-service ledger sum
	// Here: balance >= 0 is the invariant (no negative balances allowed)
	isBalanced := true
	var discrepancy float64
	for _, b := range balances {
		if b.Balance < 0 {
			isBalanced = false
			discrepancy += -b.Balance / models.ExchangeRates[b.Currency]
		}
	}
	detail := fmt.Sprintf("Reconciled %d currency balances. Total: $%.2f USD.", len(balances), totalUSD)
	if !isBalanced {
		detail += fmt.Sprintf(" Discrepancy detected: $%.4f USD.", discrepancy)
	}
	log := models.ReconciliationLog{
		ID:             uuid.New(),
		UserID:         userID,
		IsBalanced:     isBalanced,
		DiscrepancyUSD: discrepancy,
		Detail:         detail,
		CreatedAt:      time.Now().UTC(),
	}
	_ = h.store.SaveReconciliationLog(r.Context(), log)
	result := models.ReconciliationResult{
		UserID:          userID,
		Balances:        balances,
		TotalBalanceUSD: totalUSD,
		IsBalanced:      isBalanced,
		CheckedAt:       time.Now().UTC(),
		DiscrepancyUSD:  discrepancy,
	}
	respond(w, http.StatusOK, result)
}

func (h *Handler) reconciliationReport(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(mux.Vars(r)["user_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}
	balances, _ := h.store.GetAllBalances(r.Context(), userID)
	if balances == nil {
		balances = []models.WalletBalance{}
	}
	totalUSD := calculateTotalUSD(balances)
	respond(w, http.StatusOK, map[string]interface{}{
		"user_id":          userID,
		"balances":         balances,
		"total_usd":        totalUSD,
		"currency_count":   len(balances),
		"generated_at":     time.Now().UTC(),
	})
}

func calculateTotalUSD(balances []models.WalletBalance) float64 {
	var total float64
	for _, b := range balances {
		rate, ok := models.ExchangeRates[b.Currency]
		if !ok || rate == 0 {
			continue
		}
		total += b.Balance / rate
	}
	return total
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
