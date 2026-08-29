package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/farmer-finance-service/models"
)

var ErrNotFound = errors.New("db: record not found")

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RecordIncome(ctx context.Context, r models.IncomeRecord) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO income_records (id,farmer_id,source,amount,currency,description,recorded_at,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		r.ID, r.FarmerID, r.Source, r.Amount, r.Currency, r.Description, r.RecordedAt, r.CreatedAt)
	return err
}

func (q *Queries) ListIncome(ctx context.Context, farmerID uuid.UUID, limit int) ([]models.IncomeRecord, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,farmer_id,source,amount,currency,description,recorded_at,created_at
		 FROM income_records WHERE farmer_id=$1 ORDER BY recorded_at DESC LIMIT $2`, farmerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.IncomeRecord
	for rows.Next() {
		var r models.IncomeRecord
		if err := rows.Scan(&r.ID, &r.FarmerID, &r.Source, &r.Amount, &r.Currency, &r.Description, &r.RecordedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func (q *Queries) SumIncome(ctx context.Context, farmerID uuid.UUID, since time.Time) (float64, error) {
	var total float64
	err := q.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM income_records WHERE farmer_id=$1 AND recorded_at>=$2`, farmerID, since).
		Scan(&total)
	return total, err
}

func (q *Queries) CreateLoan(ctx context.Context, l models.Loan) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO loans (id,loan_ref,farmer_id,principal_amount,interest_rate,currency,status,due_date,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		l.ID, l.LoanRef, l.FarmerID, l.PrincipalAmount, l.InterestRate, l.Currency, l.Status, l.DueDate, l.CreatedAt)
	return err
}

func (q *Queries) GetLoan(ctx context.Context, id uuid.UUID) (*models.Loan, error) {
	var l models.Loan
	err := q.pool.QueryRow(ctx,
		`SELECT id,loan_ref,farmer_id,principal_amount,interest_rate,currency,status,due_date,disbursed_at,repaid_at,created_at
		 FROM loans WHERE id=$1`, id).
		Scan(&l.ID, &l.LoanRef, &l.FarmerID, &l.PrincipalAmount, &l.InterestRate, &l.Currency,
			&l.Status, &l.DueDate, &l.DisbursedAt, &l.RepaidAt, &l.CreatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &l, nil
}

func (q *Queries) ListLoans(ctx context.Context, farmerID uuid.UUID) ([]models.Loan, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,loan_ref,farmer_id,principal_amount,interest_rate,currency,status,due_date,disbursed_at,repaid_at,created_at
		 FROM loans WHERE farmer_id=$1 ORDER BY created_at DESC`, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Loan
	for rows.Next() {
		var l models.Loan
		if err := rows.Scan(&l.ID, &l.LoanRef, &l.FarmerID, &l.PrincipalAmount, &l.InterestRate, &l.Currency,
			&l.Status, &l.DueDate, &l.DisbursedAt, &l.RepaidAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, nil
}

func (q *Queries) UpdateLoanStatus(ctx context.Context, id uuid.UUID, status models.LoanStatus, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE loans SET status=$1,
		   disbursed_at=CASE WHEN $1='disbursed' THEN COALESCE(disbursed_at,$2) ELSE disbursed_at END,
		   repaid_at=CASE WHEN $1='repaid' THEN COALESCE(repaid_at,$2) ELSE repaid_at END
		 WHERE id=$3`, status, now, id)
	return err
}

// GetOrCreateSavings returns or creates the savings account for a farmer.
func (q *Queries) GetOrCreateSavings(ctx context.Context, farmerID uuid.UUID, currency models.Currency) (*models.SavingsAccount, error) {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var acc models.SavingsAccount
	err = tx.QueryRow(ctx,
		`SELECT id,farmer_id,balance,currency,total_saved,total_withdrawn,created_at,updated_at
		 FROM savings_accounts WHERE farmer_id=$1`, farmerID).
		Scan(&acc.ID, &acc.FarmerID, &acc.Balance, &acc.Currency, &acc.TotalSaved, &acc.TotalWithdrawn, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		// Create new account
		acc = models.SavingsAccount{
			ID:        uuid.New(),
			FarmerID:  farmerID,
			Balance:   0,
			Currency:  currency,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO savings_accounts (id,farmer_id,balance,currency,total_saved,total_withdrawn,created_at,updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$7)`,
			acc.ID, acc.FarmerID, acc.Balance, acc.Currency, 0, 0, acc.CreatedAt)
		if err != nil {
			return nil, err
		}
	}
	return &acc, tx.Commit(ctx)
}

func (q *Queries) AddSavings(ctx context.Context, farmerID uuid.UUID, amount float64) (*models.SavingsAccount, error) {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var acc models.SavingsAccount
	now := time.Now().UTC()
	err = tx.QueryRow(ctx,
		`UPDATE savings_accounts SET balance=balance+$1, total_saved=total_saved+$1, updated_at=$2
		 WHERE farmer_id=$3
		 RETURNING id,farmer_id,balance,currency,total_saved,total_withdrawn,created_at,updated_at`,
		amount, now, farmerID).
		Scan(&acc.ID, &acc.FarmerID, &acc.Balance, &acc.Currency, &acc.TotalSaved, &acc.TotalWithdrawn, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO savings_transactions (id,account_id,farmer_id,type,amount,balance,created_at)
		 VALUES ($1,$2,$3,'credit',$4,$5,$6)`,
		uuid.New(), acc.ID, farmerID, amount, acc.Balance, now)
	if err != nil {
		return nil, err
	}
	return &acc, tx.Commit(ctx)
}

func (q *Queries) GetSavings(ctx context.Context, farmerID uuid.UUID) (*models.SavingsAccount, error) {
	var acc models.SavingsAccount
	err := q.pool.QueryRow(ctx,
		`SELECT id,farmer_id,balance,currency,total_saved,total_withdrawn,created_at,updated_at
		 FROM savings_accounts WHERE farmer_id=$1`, farmerID).
		Scan(&acc.ID, &acc.FarmerID, &acc.Balance, &acc.Currency, &acc.TotalSaved, &acc.TotalWithdrawn, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &acc, nil
}
