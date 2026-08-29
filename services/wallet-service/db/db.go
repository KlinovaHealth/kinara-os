package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/wallet-service/models"
)

var ErrNotFound = errors.New("db: record not found")

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) UpsertBalance(ctx context.Context, userID uuid.UUID, currency models.Currency, balance float64) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO wallet_balances (id,user_id,currency,balance,updated_at)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (user_id,currency) DO UPDATE SET balance=$4, updated_at=$5`,
		uuid.New(), userID, string(currency), balance, time.Now().UTC())
	return err
}

func (q *Queries) GetBalance(ctx context.Context, userID uuid.UUID, currency models.Currency) (float64, time.Time, error) {
	var balance float64
	var updatedAt time.Time
	err := q.pool.QueryRow(ctx,
		`SELECT balance, updated_at FROM wallet_balances WHERE user_id=$1 AND currency=$2`,
		userID, string(currency)).Scan(&balance, &updatedAt)
	if err != nil {
		return 0, time.Time{}, ErrNotFound
	}
	return balance, updatedAt, nil
}

func (q *Queries) GetAllBalances(ctx context.Context, userID uuid.UUID) ([]models.WalletBalance, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT user_id, currency, balance, updated_at FROM wallet_balances WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.WalletBalance
	for rows.Next() {
		var wb models.WalletBalance
		var currency string
		if err := rows.Scan(&wb.UserID, &currency, &wb.Balance, &wb.UpdatedAt); err != nil {
			return nil, err
		}
		wb.Currency = models.Currency(currency)
		result = append(result, wb)
	}
	return result, nil
}

func (q *Queries) SaveReconciliationLog(ctx context.Context, l models.ReconciliationLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO reconciliation_logs (id,user_id,is_balanced,discrepancy_usd,detail,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.UserID, l.IsBalanced, l.DiscrepancyUSD, l.Detail, l.CreatedAt)
	return err
}
