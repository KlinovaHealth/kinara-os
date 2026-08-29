package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/payment-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateWallet(ctx context.Context, w models.Wallet) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO wallets (id,owner_id,owner_type,currency,balance,reserved_amount,status,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		w.ID, w.OwnerID, w.OwnerType, w.Currency, w.Balance, w.ReservedAmt, w.Status, w.CreatedAt, w.UpdatedAt)
	return err
}

func (q *Queries) GetWallet(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	w := &models.Wallet{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,owner_id,owner_type,currency,balance,reserved_amount,status,created_at,updated_at FROM wallets WHERE id=$1`, id).
		Scan(&w.ID, &w.OwnerID, &w.OwnerType, &w.Currency, &w.Balance, &w.ReservedAmt, &w.Status, &w.CreatedAt, &w.UpdatedAt)
	if err != nil { return nil, err }
	return w, nil
}

func (q *Queries) GetWalletByOwner(ctx context.Context, ownerID uuid.UUID, currency string) (*models.Wallet, error) {
	w := &models.Wallet{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,owner_id,owner_type,currency,balance,reserved_amount,status,created_at,updated_at FROM wallets WHERE owner_id=$1 AND currency=$2`, ownerID, currency).
		Scan(&w.ID, &w.OwnerID, &w.OwnerType, &w.Currency, &w.Balance, &w.ReservedAmt, &w.Status, &w.CreatedAt, &w.UpdatedAt)
	if err != nil { return nil, err }
	return w, nil
}

func (q *Queries) CreditWallet(ctx context.Context, walletID uuid.UUID, amount float64, txn models.Transaction) error {
	tx, err := q.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE wallets SET balance=balance+$1,updated_at=$2 WHERE id=$3`, amount, time.Now().UTC(), walletID)
	if err != nil { return err }
	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (id,txn_ref,wallet_id,txn_type,amount,currency,balance_before,balance_after,reference_type,reference_id,description,status,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		txn.ID, txn.TxnRef, txn.WalletID, txn.TxnType, txn.Amount, txn.Currency, txn.BalanceBefore, txn.BalanceAfter,
		txn.ReferenceType, txn.ReferenceID, txn.Description, txn.Status, txn.CreatedAt)
	if err != nil { return err }
	return tx.Commit(ctx)
}

func (q *Queries) DebitWallet(ctx context.Context, walletID uuid.UUID, amount float64, txn models.Transaction) error {
	tx, err := q.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE wallets SET balance=balance-$1,updated_at=$2 WHERE id=$3 AND balance>=$1`, amount, time.Now().UTC(), walletID)
	if err != nil { return err }
	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (id,txn_ref,wallet_id,txn_type,amount,currency,balance_before,balance_after,reference_type,reference_id,description,status,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		txn.ID, txn.TxnRef, txn.WalletID, txn.TxnType, txn.Amount, txn.Currency, txn.BalanceBefore, txn.BalanceAfter,
		txn.ReferenceType, txn.ReferenceID, txn.Description, txn.Status, txn.CreatedAt)
	if err != nil { return err }
	return tx.Commit(ctx)
}

func (q *Queries) ListTransactions(ctx context.Context, walletID uuid.UUID) ([]models.Transaction, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,txn_ref,wallet_id,txn_type,amount,currency,balance_before,balance_after,reference_type,reference_id,description,status,failure_reason,created_at FROM transactions WHERE wallet_id=$1 ORDER BY created_at DESC LIMIT 100`, walletID)
	if err != nil { return nil, err }
	defer rows.Close()
	var txns []models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.TxnRef, &t.WalletID, &t.TxnType, &t.Amount, &t.Currency, &t.BalanceBefore, &t.BalanceAfter, &t.ReferenceType, &t.ReferenceID, &t.Description, &t.Status, &t.FailureReason, &t.CreatedAt); err != nil { return nil, err }
		txns = append(txns, t)
	}
	return txns, nil
}

func (q *Queries) CreateConversion(ctx context.Context, c models.CurrencyConversion) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO currency_conversions (id,from_wallet_id,to_wallet_id,from_currency,to_currency,from_amount,to_amount,exchange_rate,fee_amount,status,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		c.ID, c.FromWalletID, c.ToWalletID, c.FromCurrency, c.ToCurrency, c.FromAmount, c.ToAmount, c.ExchangeRate, c.FeeAmount, c.Status, c.CreatedAt)
	return err
}

func (q *Queries) CreateSettlement(ctx context.Context, s models.Settlement) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO settlements (id,settlement_ref,wallet_id,amount,currency,bank_account_no,bank_code,mobile_money_no,provider,status,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.ID, s.SettlementRef, s.WalletID, s.Amount, s.Currency, s.BankAccountNo, s.BankCode, s.MobileMoneyNo, s.Provider, s.Status, s.CreatedAt, s.UpdatedAt)
	return err
}

func (q *Queries) ConfirmSettlement(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE settlements SET status='completed',settled_at=COALESCE(settled_at,$1),updated_at=$2 WHERE id=$3`,
		now, now, id)
	return err
}

func (q *Queries) GetSettlement(ctx context.Context, id uuid.UUID) (*models.Settlement, error) {
	s := &models.Settlement{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,settlement_ref,wallet_id,amount,currency,bank_account_no,bank_code,mobile_money_no,provider,status,settled_at,created_at,updated_at FROM settlements WHERE id=$1`, id).
		Scan(&s.ID, &s.SettlementRef, &s.WalletID, &s.Amount, &s.Currency, &s.BankAccountNo, &s.BankCode, &s.MobileMoneyNo, &s.Provider, &s.Status, &s.SettledAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil { return nil, err }
	return s, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.PaymentAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO payment_audit_log (id,actor_id,action,entity_type,entity_id,created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
