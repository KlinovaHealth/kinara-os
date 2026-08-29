package models

import (
	"time"

	"github.com/google/uuid"
)

type LoanStatus string

const (
	LoanPending   LoanStatus = "pending"
	LoanApproved  LoanStatus = "approved"
	LoanDisbursed LoanStatus = "disbursed"
	LoanRepaid    LoanStatus = "repaid"
	LoanDefaulted LoanStatus = "defaulted"
)

type Currency string

const (
	CurrencyXOF Currency = "XOF"
	CurrencyUSD Currency = "USD"
	CurrencyGHS Currency = "GHS"
	CurrencyKES Currency = "KES"
)

// IncomeRecord tracks a single income event (crop sale, etc.)
type IncomeRecord struct {
	ID          uuid.UUID `json:"id"`
	FarmerID    uuid.UUID `json:"farmer_id"`
	Source      string    `json:"source"`
	Amount      float64   `json:"amount"`
	Currency    Currency  `json:"currency"`
	Description string    `json:"description"`
	RecordedAt  time.Time `json:"recorded_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// Loan is a microloan issued to a farmer against future income.
type Loan struct {
	ID              uuid.UUID  `json:"id"`
	LoanRef         string     `json:"loan_ref"`
	FarmerID        uuid.UUID  `json:"farmer_id"`
	PrincipalAmount float64    `json:"principal_amount"`
	InterestRate    float64    `json:"interest_rate"`
	Currency        Currency   `json:"currency"`
	Status          LoanStatus `json:"status"`
	DueDate         time.Time  `json:"due_date"`
	DisbursedAt     *time.Time `json:"disbursed_at,omitempty"`
	RepaidAt        *time.Time `json:"repaid_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// SavingsAccount holds a farmer's savings balance.
type SavingsAccount struct {
	ID           uuid.UUID `json:"id"`
	FarmerID     uuid.UUID `json:"farmer_id"`
	Balance      float64   `json:"balance"`
	Currency     Currency  `json:"currency"`
	TotalSaved   float64   `json:"total_saved"`
	TotalWithdrawn float64 `json:"total_withdrawn"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SavingsTransaction records a credit or debit on the savings account.
type SavingsTransaction struct {
	ID        uuid.UUID `json:"id"`
	AccountID uuid.UUID `json:"account_id"`
	FarmerID  uuid.UUID `json:"farmer_id"`
	Type      string    `json:"type"`
	Amount    float64   `json:"amount"`
	Balance   float64   `json:"balance_after"`
	CreatedAt time.Time `json:"created_at"`
}

type LoanEligibility struct {
	FarmerID         uuid.UUID `json:"farmer_id"`
	IsEligible       bool      `json:"is_eligible"`
	MaxLoanAmount    float64   `json:"max_loan_amount"`
	Currency         Currency  `json:"currency"`
	Reason           string    `json:"reason"`
	AvgMonthlyIncome float64   `json:"avg_monthly_income"`
}

type RecordIncomeRequest struct {
	Source      string  `json:"source"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	RecordedAt  string  `json:"recorded_at"`
}

type RequestLoanRequest struct {
	PrincipalAmount float64 `json:"principal_amount"`
	Currency        string  `json:"currency"`
	DueDate         string  `json:"due_date"`
}

type SaveRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
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
