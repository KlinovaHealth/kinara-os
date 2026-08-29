package models

import (
	"time"

	"github.com/google/uuid"
)

type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyXOF Currency = "XOF"
	CurrencyGHS Currency = "GHS"
	CurrencyKES Currency = "KES"
	CurrencyNGN Currency = "NGN"
	CurrencyETB Currency = "ETB"
	CurrencyTZS Currency = "TZS"
	CurrencyRWF Currency = "RWF"
)

// ExchangeRates maps currency to USD rate (units of currency per 1 USD)
var ExchangeRates = map[Currency]float64{
	CurrencyUSD: 1.0,
	CurrencyXOF: 600.0,
	CurrencyGHS: 14.0,
	CurrencyKES: 130.0,
	CurrencyNGN: 1500.0,
	CurrencyETB: 56.0,
	CurrencyTZS: 2600.0,
	CurrencyRWF: 1300.0,
}

type WalletBalance struct {
	UserID    uuid.UUID `json:"user_id"`
	Currency  Currency  `json:"currency"`
	Balance   float64   `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ReconciliationResult struct {
	UserID         uuid.UUID        `json:"user_id"`
	Balances       []WalletBalance  `json:"balances"`
	TotalBalanceUSD float64         `json:"total_balance_usd"`
	IsBalanced     bool             `json:"is_balanced"`
	CheckedAt      time.Time        `json:"checked_at"`
	DiscrepancyUSD float64          `json:"discrepancy_usd"`
}

type ReconciliationLog struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	IsBalanced  bool      `json:"is_balanced"`
	DiscrepancyUSD float64 `json:"discrepancy_usd"`
	Detail      string    `json:"detail"`
	CreatedAt   time.Time `json:"created_at"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
