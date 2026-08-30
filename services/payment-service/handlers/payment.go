package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/payment-service/middleware"
	"github.com/klinova/kinara-os/payment-service/models"
)

type Store interface {
	CreateWallet(ctx context.Context, w models.Wallet) error
	GetWallet(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
	GetWalletByOwner(ctx context.Context, ownerID uuid.UUID, currency string) (*models.Wallet, error)
	CreditWallet(ctx context.Context, walletID uuid.UUID, amount float64, txn models.Transaction) error
	DebitWallet(ctx context.Context, walletID uuid.UUID, amount float64, txn models.Transaction) error
	ListTransactions(ctx context.Context, walletID uuid.UUID) ([]models.Transaction, error)
	CreateConversion(ctx context.Context, c models.CurrencyConversion) error
	CreateSettlement(ctx context.Context, s models.Settlement) error
	ConfirmSettlement(ctx context.Context, id uuid.UUID, now time.Time) error
	GetSettlement(ctx context.Context, id uuid.UUID) (*models.Settlement, error)
	InsertAuditLog(ctx context.Context, l models.PaymentAuditLog) error
}

type Handler struct{ store Store }

func NewHandler(store Store) *Handler      { return &Handler{store: store} }
func NewHandlerWithStore(s Store) *Handler { return &Handler{store: s} }

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/wallets", h.CreateWallet).Methods("POST")
	r.HandleFunc("/wallets/{id}", h.GetWallet).Methods("GET")
	r.HandleFunc("/wallets/{id}/credit", h.CreditWallet).Methods("POST")
	r.HandleFunc("/wallets/{id}/debit", h.DebitWallet).Methods("POST")
	r.HandleFunc("/wallets/{id}/transactions", h.ListTransactions).Methods("GET")
	r.HandleFunc("/convert", h.ConvertCurrency).Methods("POST")
	r.HandleFunc("/settlements", h.CreateSettlement).Methods("POST")
	r.HandleFunc("/settlements/{id}", h.GetSettlement).Methods("GET")
	r.HandleFunc("/settlements/{id}/confirm", h.ConfirmSettlement).Methods("PUT")
}

func respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func newTxnRef() string {
	id := uuid.New()
	return "TX-" + strings.ToUpper(id.String()[:12])
}

func (h *Handler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid request"})
		return
	}
	if req.OwnerID == "" || req.Currency == "" {
		respond(w, 400, models.APIResponse{Error: "owner_id, currency required"})
		return
	}
	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid owner_id"})
		return
	}
	if _, ok := models.ExchangeRates[req.Currency]; !ok {
		respond(w, 400, models.APIResponse{Error: "unsupported currency"})
		return
	}
	now := time.Now().UTC()
	wallet := models.Wallet{
		ID: uuid.New(), OwnerID: ownerID, OwnerType: req.OwnerType,
		Currency: req.Currency, Status: models.WalletActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateWallet(r.Context(), wallet); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to create wallet"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.PaymentAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "create_wallet", EntityType: "wallet", EntityID: wallet.ID, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: wallet})
}

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	wallet, err := h.store.GetWallet(r.Context(), id)
	if err != nil {
		respond(w, 404, models.APIResponse{Error: "wallet not found"})
		return
	}
	respond(w, 200, models.APIResponse{Success: true, Data: wallet})
}

func (h *Handler) CreditWallet(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	var req models.CreditWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		respond(w, 400, models.APIResponse{Error: "amount required and must be positive"})
		return
	}
	wallet, err := h.store.GetWallet(r.Context(), id)
	if err != nil {
		respond(w, 404, models.APIResponse{Error: "wallet not found"})
		return
	}
	now := time.Now().UTC()
	txnID := uuid.New()
	txn := models.Transaction{
		ID: txnID, TxnRef: newTxnRef(), WalletID: id, TxnType: models.TxnCredit,
		Amount: req.Amount, Currency: wallet.Currency,
		BalanceBefore: wallet.Balance, BalanceAfter: wallet.Balance + req.Amount,
		ReferenceType: req.ReferenceType, ReferenceID: req.ReferenceID,
		Description: req.Description, Status: models.TxnCompleted, CreatedAt: now,
	}
	if err := h.store.CreditWallet(r.Context(), id, req.Amount, txn); err != nil {
		respond(w, 500, models.APIResponse{Error: "credit failed"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.PaymentAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "credit_wallet", EntityType: "transaction", EntityID: txnID, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: txn})
}

func (h *Handler) DebitWallet(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	var req models.DebitWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		respond(w, 400, models.APIResponse{Error: "amount required and must be positive"})
		return
	}
	wallet, err := h.store.GetWallet(r.Context(), id)
	if err != nil {
		respond(w, 404, models.APIResponse{Error: "wallet not found"})
		return
	}
	if wallet.Balance < req.Amount {
		respond(w, 422, models.APIResponse{Error: fmt.Sprintf("insufficient funds: balance=%.4f requested=%.4f", wallet.Balance, req.Amount)})
		return
	}
	now := time.Now().UTC()
	txnID := uuid.New()
	txn := models.Transaction{
		ID: txnID, TxnRef: newTxnRef(), WalletID: id, TxnType: models.TxnDebit,
		Amount: req.Amount, Currency: wallet.Currency,
		BalanceBefore: wallet.Balance, BalanceAfter: wallet.Balance - req.Amount,
		ReferenceType: req.ReferenceType, ReferenceID: req.ReferenceID,
		Description: req.Description, Status: models.TxnCompleted, CreatedAt: now,
	}
	if err := h.store.DebitWallet(r.Context(), id, req.Amount, txn); err != nil {
		respond(w, 500, models.APIResponse{Error: "debit failed"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.PaymentAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "debit_wallet", EntityType: "transaction", EntityID: txnID, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: txn})
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	txns, err := h.store.ListTransactions(r.Context(), id)
	if err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to list transactions"})
		return
	}
	if txns == nil {
		txns = []models.Transaction{}
	}
	respond(w, 200, models.APIResponse{Success: true, Data: txns})
}

func (h *Handler) ConvertCurrency(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var req models.ConvertCurrencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid request"})
		return
	}
	if req.FromWalletID == "" || req.ToWalletID == "" || req.FromAmount <= 0 {
		respond(w, 400, models.APIResponse{Error: "from_wallet_id, to_wallet_id, from_amount required"})
		return
	}
	fromID, err1 := uuid.Parse(req.FromWalletID)
	toID, err2 := uuid.Parse(req.ToWalletID)
	if err1 != nil || err2 != nil {
		respond(w, 400, models.APIResponse{Error: "invalid wallet ids"})
		return
	}
	fromWallet, err := h.store.GetWallet(r.Context(), fromID)
	if err != nil { respond(w, 404, models.APIResponse{Error: "from wallet not found"}); return }
	toWallet, err := h.store.GetWallet(r.Context(), toID)
	if err != nil { respond(w, 404, models.APIResponse{Error: "to wallet not found"}); return }

	fromRate, ok1 := models.ExchangeRates[fromWallet.Currency]
	toRate, ok2 := models.ExchangeRates[toWallet.Currency]
	if !ok1 || !ok2 {
		respond(w, 400, models.APIResponse{Error: "unsupported currency pair"})
		return
	}
	if fromWallet.Balance < req.FromAmount {
		respond(w, 422, models.APIResponse{Error: "insufficient funds"})
		return
	}
	usdAmount := req.FromAmount / fromRate
	grossToAmount := usdAmount * toRate
	feeAmount := grossToAmount * 0.005
	toAmount := grossToAmount - feeAmount
	exchangeRate := toRate / fromRate

	now := time.Now().UTC()
	convID := uuid.New()

	debitTxn := models.Transaction{
		ID: uuid.New(), TxnRef: newTxnRef(), WalletID: fromID, TxnType: models.TxnConversion,
		Amount: req.FromAmount, Currency: fromWallet.Currency,
		BalanceBefore: fromWallet.Balance, BalanceAfter: fromWallet.Balance - req.FromAmount,
		Description: fmt.Sprintf("FX conversion to %s", toWallet.Currency),
		Status: models.TxnCompleted, CreatedAt: now,
	}
	h.store.DebitWallet(r.Context(), fromID, req.FromAmount, debitTxn)

	creditTxn := models.Transaction{
		ID: uuid.New(), TxnRef: newTxnRef(), WalletID: toID, TxnType: models.TxnConversion,
		Amount: toAmount, Currency: toWallet.Currency,
		BalanceBefore: toWallet.Balance, BalanceAfter: toWallet.Balance + toAmount,
		Description: fmt.Sprintf("FX conversion from %s", fromWallet.Currency),
		Status: models.TxnCompleted, CreatedAt: now,
	}
	h.store.CreditWallet(r.Context(), toID, toAmount, creditTxn)

	conv := models.CurrencyConversion{
		ID: convID, FromWalletID: fromID, ToWalletID: toID,
		FromCurrency: fromWallet.Currency, ToCurrency: toWallet.Currency,
		FromAmount: req.FromAmount, ToAmount: toAmount,
		ExchangeRate: exchangeRate, FeeAmount: feeAmount,
		Status: models.TxnCompleted, CreatedAt: now,
	}
	h.store.CreateConversion(r.Context(), conv)
	h.store.InsertAuditLog(r.Context(), models.PaymentAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "convert_currency", EntityType: "currency_conversion", EntityID: convID, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: conv})
}

type createSettlementBody struct {
	WalletID      string  `json:"wallet_id"`
	Amount        float64 `json:"amount"`
	Provider      string  `json:"provider"`
	BankAccountNo string  `json:"bank_account_no,omitempty"`
	BankCode      string  `json:"bank_code,omitempty"`
	MobileMoneyNo string  `json:"mobile_money_no,omitempty"`
}

func (h *Handler) CreateSettlement(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	bodyBytes, _ := io.ReadAll(r.Body)
	var req createSettlementBody
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&req); err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid request"})
		return
	}
	if req.WalletID == "" || req.Amount <= 0 || req.Provider == "" {
		respond(w, 400, models.APIResponse{Error: "wallet_id, amount, provider required"})
		return
	}
	walletID, err := uuid.Parse(req.WalletID)
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid wallet_id"})
		return
	}
	wallet, err := h.store.GetWallet(r.Context(), walletID)
	if err != nil {
		respond(w, 404, models.APIResponse{Error: "wallet not found"})
		return
	}
	if wallet.Balance < req.Amount {
		respond(w, 422, models.APIResponse{Error: "insufficient funds"})
		return
	}
	now := time.Now().UTC()
	id := uuid.New()
	settlement := models.Settlement{
		ID: id, SettlementRef: "ST-" + strings.ToUpper(id.String()[:12]),
		WalletID: walletID, Amount: req.Amount, Currency: wallet.Currency,
		BankAccountNo: req.BankAccountNo, BankCode: req.BankCode,
		MobileMoneyNo: req.MobileMoneyNo, Provider: req.Provider,
		Status: models.TxnPending, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.store.CreateSettlement(r.Context(), settlement); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to create settlement"})
		return
	}
	h.store.InsertAuditLog(r.Context(), models.PaymentAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "create_settlement", EntityType: "settlement", EntityID: id, CreatedAt: now})
	respond(w, 201, models.APIResponse{Success: true, Data: settlement})
}

func (h *Handler) GetSettlement(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	s, err := h.store.GetSettlement(r.Context(), id)
	if err != nil {
		respond(w, 404, models.APIResponse{Error: "settlement not found"})
		return
	}
	respond(w, 200, models.APIResponse{Success: true, Data: s})
}

func (h *Handler) ConfirmSettlement(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, 400, models.APIResponse{Error: "invalid id"})
		return
	}
	now := time.Now().UTC()
	if err := h.store.ConfirmSettlement(r.Context(), id, now); err != nil {
		respond(w, 500, models.APIResponse{Error: "failed to confirm settlement"})
		return
	}
	s, _ := h.store.GetSettlement(r.Context(), id)
	h.store.InsertAuditLog(r.Context(), models.PaymentAuditLog{ID: uuid.New(), ActorID: claims.UserID.String(), Action: "confirm_settlement", EntityType: "settlement", EntityID: id, CreatedAt: now})
	respond(w, 200, models.APIResponse{Success: true, Data: s})
}
