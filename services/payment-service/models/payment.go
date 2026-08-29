package models

import (
	"time"
	"github.com/google/uuid"
)

type WalletStatus string
type TxnType string
type TxnStatus string
type Currency string

const (
	WalletActive   WalletStatus = "active"
	WalletFrozen   WalletStatus = "frozen"
	WalletClosed   WalletStatus = "closed"

	TxnCredit       TxnType = "credit"
	TxnDebit        TxnType = "debit"
	TxnConversion   TxnType = "conversion"
	TxnSettlement   TxnType = "settlement"
	TxnRefund       TxnType = "refund"

	TxnPending    TxnStatus = "pending"
	TxnCompleted  TxnStatus = "completed"
	TxnFailed     TxnStatus = "failed"
	TxnReversed   TxnStatus = "reversed"

	CurrencyUSD Currency = "USD"
	CurrencyXOF Currency = "XOF"
	CurrencyGHS Currency = "GHS"
	CurrencyKES Currency = "KES"
	CurrencyNGN Currency = "NGN"
	CurrencyETB Currency = "ETB"
	CurrencyTZS Currency = "TZS"
	CurrencyRWF Currency = "RWF"
)

// ExchangeRates relative to USD
var ExchangeRates = map[string]float64{
	"USD": 1.0,
	"XOF": 600.0,
	"GHS": 14.0,
	"KES": 130.0,
	"NGN": 1500.0,
	"ETB": 56.0,
	"TZS": 2600.0,
	"RWF": 1300.0,
}

type Wallet struct {
	ID          uuid.UUID    `json:"id"`
	OwnerID     uuid.UUID    `json:"owner_id"`
	OwnerType   string       `json:"owner_type"`
	Currency    string       `json:"currency"`
	Balance     float64      `json:"balance"`
	ReservedAmt float64      `json:"reserved_amount"`
	Status      WalletStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Transaction struct {
	ID              uuid.UUID `json:"id"`
	TxnRef          string    `json:"txn_ref"`
	WalletID        uuid.UUID `json:"wallet_id"`
	TxnType         TxnType   `json:"txn_type"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	BalanceBefore   float64   `json:"balance_before"`
	BalanceAfter    float64   `json:"balance_after"`
	ReferenceType   string    `json:"reference_type,omitempty"`
	ReferenceID     string    `json:"reference_id,omitempty"`
	Description     string    `json:"description"`
	Status          TxnStatus `json:"status"`
	FailureReason   string    `json:"failure_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type CurrencyConversion struct {
	ID             uuid.UUID `json:"id"`
	FromWalletID   uuid.UUID `json:"from_wallet_id"`
	ToWalletID     uuid.UUID `json:"to_wallet_id"`
	FromCurrency   string    `json:"from_currency"`
	ToCurrency     string    `json:"to_currency"`
	FromAmount     float64   `json:"from_amount"`
	ToAmount       float64   `json:"to_amount"`
	ExchangeRate   float64   `json:"exchange_rate"`
	FeeAmount      float64   `json:"fee_amount"`
	Status         TxnStatus `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type Settlement struct {
	ID              uuid.UUID  `json:"id"`
	SettlementRef   string     `json:"settlement_ref"`
	WalletID        uuid.UUID  `json:"wallet_id"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	BankAccountNo   string     `json:"bank_account_no,omitempty"`
	BankCode        string     `json:"bank_code,omitempty"`
	MobileMoneyNo   string     `json:"mobile_money_no,omitempty"`
	Provider        string     `json:"provider"`
	Status          TxnStatus  `json:"status"`
	SettledAt       *time.Time `json:"settled_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type PaymentAuditLog struct {
	ID         uuid.UUID `json:"id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateWalletRequest struct {
	OwnerID   string `json:"owner_id"`
	OwnerType string `json:"owner_type"`
	Currency  string `json:"currency"`
}

type CreditWalletRequest struct {
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
	ReferenceType string  `json:"reference_type,omitempty"`
	ReferenceID   string  `json:"reference_id,omitempty"`
}

type DebitWalletRequest struct {
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
	ReferenceType string  `json:"reference_type,omitempty"`
	ReferenceID   string  `json:"reference_id,omitempty"`
}

type ConvertCurrencyRequest struct {
	FromWalletID string  `json:"from_wallet_id"`
	ToWalletID   string  `json:"to_wallet_id"`
	FromAmount   float64 `json:"from_amount"`
}

type SettleRequest struct {
	Amount        float64 `json:"amount"`
	Provider      string  `json:"provider"`
	BankAccountNo string  `json:"bank_account_no,omitempty"`
	BankCode      string  `json:"bank_code,omitempty"`
	MobileMoneyNo string  `json:"mobile_money_no,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
