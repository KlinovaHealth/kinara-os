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
	"github.com/klinova/kinara-os/farmer-finance-service/db"
	"github.com/klinova/kinara-os/farmer-finance-service/models"
)

type Store interface {
	RecordIncome(ctx context.Context, r models.IncomeRecord) error
	ListIncome(ctx context.Context, farmerID uuid.UUID, limit int) ([]models.IncomeRecord, error)
	SumIncome(ctx context.Context, farmerID uuid.UUID, since time.Time) (float64, error)
	CreateLoan(ctx context.Context, l models.Loan) error
	GetLoan(ctx context.Context, id uuid.UUID) (*models.Loan, error)
	ListLoans(ctx context.Context, farmerID uuid.UUID) ([]models.Loan, error)
	UpdateLoanStatus(ctx context.Context, id uuid.UUID, status models.LoanStatus, now time.Time) error
	GetOrCreateSavings(ctx context.Context, farmerID uuid.UUID, currency models.Currency) (*models.SavingsAccount, error)
	AddSavings(ctx context.Context, farmerID uuid.UUID, amount float64) (*models.SavingsAccount, error)
	GetSavings(ctx context.Context, farmerID uuid.UUID) (*models.SavingsAccount, error)
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
	api := r.PathPrefix("/api/v1/finance").Subrouter()
	api.HandleFunc("/{farmer_id}/income", h.recordIncome).Methods(http.MethodPost)
	api.HandleFunc("/{farmer_id}/income", h.getIncome).Methods(http.MethodGet)
	api.HandleFunc("/{farmer_id}/loan", h.requestLoan).Methods(http.MethodPost)
	api.HandleFunc("/{farmer_id}/loans", h.listLoans).Methods(http.MethodGet)
	api.HandleFunc("/{farmer_id}/loan-eligibility", h.loanEligibility).Methods(http.MethodGet)
	api.HandleFunc("/{farmer_id}/savings", h.getSavings).Methods(http.MethodGet)
	api.HandleFunc("/{farmer_id}/save", h.addSavings).Methods(http.MethodPost)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "farmer-finance-service"})
}

func (h *Handler) recordIncome(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid farmer_id")
		return
	}
	var req models.RecordIncomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Amount <= 0 || req.Source == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "amount and source are required")
		return
	}
	currency := models.Currency(strings.ToUpper(req.Currency))
	if currency == "" {
		currency = models.CurrencyXOF
	}
	recordedAt := time.Now().UTC()
	if req.RecordedAt != "" {
		if t, err := time.Parse(time.RFC3339, req.RecordedAt); err == nil {
			recordedAt = t
		}
	}
	rec := models.IncomeRecord{
		ID:          uuid.New(),
		FarmerID:    farmerID,
		Source:      req.Source,
		Amount:      req.Amount,
		Currency:    currency,
		Description: req.Description,
		RecordedAt:  recordedAt,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.store.RecordIncome(r.Context(), rec); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to record income")
		return
	}
	respond(w, http.StatusCreated, rec)
}

func (h *Handler) getIncome(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid farmer_id")
		return
	}
	records, err := h.store.ListIncome(r.Context(), farmerID, 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list income")
		return
	}
	if records == nil {
		records = []models.IncomeRecord{}
	}
	// Compute totals for response
	var total float64
	for _, r := range records {
		total += r.Amount
	}
	respond(w, http.StatusOK, map[string]interface{}{
		"farmer_id": farmerID,
		"records":   records,
		"total":     total,
		"count":     len(records),
	})
}

func (h *Handler) loanEligibility(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid farmer_id")
		return
	}
	// Eligibility: average monthly income over past 3 months > 0; max loan = 3x monthly income
	threeMonthsAgo := time.Now().UTC().AddDate(0, -3, 0)
	total, _ := h.store.SumIncome(r.Context(), farmerID, threeMonthsAgo)
	avgMonthly := total / 3.0

	eligibility := models.LoanEligibility{
		FarmerID:         farmerID,
		AvgMonthlyIncome: avgMonthly,
		Currency:         models.CurrencyXOF,
	}
	if avgMonthly >= 10000 { // 10,000 XOF minimum monthly income
		eligibility.IsEligible = true
		eligibility.MaxLoanAmount = avgMonthly * 3
		eligibility.Reason = "Eligible based on 3-month income history"
	} else {
		eligibility.IsEligible = false
		eligibility.MaxLoanAmount = 0
		eligibility.Reason = fmt.Sprintf("Insufficient income history. Avg monthly: %.0f XOF. Minimum: 10,000 XOF.", avgMonthly)
	}
	respond(w, http.StatusOK, eligibility)
}

func (h *Handler) requestLoan(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid farmer_id")
		return
	}
	var req models.RequestLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.PrincipalAmount <= 0 {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "principal_amount is required and must be > 0")
		return
	}
	currency := models.Currency(strings.ToUpper(req.Currency))
	if currency == "" {
		currency = models.CurrencyXOF
	}
	dueDate := time.Now().UTC().AddDate(0, 3, 0) // Default 3-month repayment
	if req.DueDate != "" {
		if t, err := time.Parse(time.RFC3339, req.DueDate); err == nil {
			dueDate = t
		}
	}
	now := time.Now().UTC()
	id := uuid.New()
	loan := models.Loan{
		ID:              id,
		LoanRef:         "LN-" + strings.ToUpper(id.String()[:8]),
		FarmerID:        farmerID,
		PrincipalAmount: req.PrincipalAmount,
		InterestRate:    5.0,
		Currency:        currency,
		Status:          models.LoanPending,
		DueDate:         dueDate,
		CreatedAt:       now,
	}
	if err := h.store.CreateLoan(r.Context(), loan); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to create loan")
		return
	}
	respond(w, http.StatusCreated, loan)
}

func (h *Handler) listLoans(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid farmer_id")
		return
	}
	loans, err := h.store.ListLoans(r.Context(), farmerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list loans")
		return
	}
	if loans == nil {
		loans = []models.Loan{}
	}
	respond(w, http.StatusOK, loans)
}

func (h *Handler) getSavings(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid farmer_id")
		return
	}
	acc, err := h.store.GetSavings(r.Context(), farmerID)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "no savings account found")
		return
	}
	respond(w, http.StatusOK, acc)
}

func (h *Handler) addSavings(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid farmer_id")
		return
	}
	var req models.SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Amount <= 0 {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "amount must be > 0")
		return
	}
	currency := models.Currency(strings.ToUpper(req.Currency))
	if currency == "" {
		currency = models.CurrencyXOF
	}
	// Ensure account exists
	_, _ = h.store.GetOrCreateSavings(r.Context(), farmerID, currency)
	acc, err := h.store.AddSavings(r.Context(), farmerID, req.Amount)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "failed to add savings")
		return
	}
	respond(w, http.StatusOK, acc)
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
