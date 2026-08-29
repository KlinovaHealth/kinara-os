package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/trade-finance-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateLC(ctx context.Context, lc models.LetterOfCredit) error {
	docsJSON, _ := json.Marshal(lc.DocumentsRequired)
	_, err := q.pool.Exec(ctx,
		`INSERT INTO letters_of_credit (id,lc_number,lc_type,applicant_id,applicant_name,beneficiary_name,issuing_bank,advising_bank,amount,currency,expiry_date,shipment_pol,shipment_pod,goods_description,documents_required,status,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		lc.ID, lc.LCNumber, lc.LCType, lc.ApplicantID, lc.ApplicantName, lc.BeneficiaryName,
		lc.IssuingBank, lc.AdvisingBank, lc.Amount, lc.Currency, lc.ExpiryDate,
		lc.ShipmentPOL, lc.ShipmentPOD, lc.GoodsDescription, docsJSON, lc.Status, lc.CreatedAt, lc.UpdatedAt)
	return err
}

func (q *Queries) GetLC(ctx context.Context, id uuid.UUID) (*models.LetterOfCredit, error) {
	lc := &models.LetterOfCredit{}
	var docsRaw []byte
	err := q.pool.QueryRow(ctx,
		`SELECT id,lc_number,lc_type,applicant_id,applicant_name,beneficiary_name,issuing_bank,advising_bank,amount,currency,expiry_date,shipment_pol,shipment_pod,goods_description,documents_required,status,issued_at,realized_at,created_at,updated_at
         FROM letters_of_credit WHERE id=$1`, id).
		Scan(&lc.ID, &lc.LCNumber, &lc.LCType, &lc.ApplicantID, &lc.ApplicantName, &lc.BeneficiaryName,
			&lc.IssuingBank, &lc.AdvisingBank, &lc.Amount, &lc.Currency, &lc.ExpiryDate,
			&lc.ShipmentPOL, &lc.ShipmentPOD, &lc.GoodsDescription, &docsRaw, &lc.Status,
			&lc.IssuedAt, &lc.RealizedAt, &lc.CreatedAt, &lc.UpdatedAt)
	if err != nil { return nil, err }
	json.Unmarshal(docsRaw, &lc.DocumentsRequired)
	return lc, nil
}

func (q *Queries) ListLCs(ctx context.Context, applicantID *uuid.UUID, status *models.LCStatus) ([]models.LetterOfCredit, error) {
	query := `SELECT id,lc_number,lc_type,applicant_id,applicant_name,beneficiary_name,issuing_bank,advising_bank,amount,currency,expiry_date,shipment_pol,shipment_pod,goods_description,documents_required,status,issued_at,realized_at,created_at,updated_at FROM letters_of_credit WHERE 1=1`
	args := []interface{}{}
	i := 1
	if applicantID != nil { query += " AND applicant_id=$1"; args = append(args, *applicantID); i++ }
	if status != nil {
		if i == 1 { query += " AND status=$1" } else { query += " AND status=$2" }
		args = append(args, *status)
	}
	query += " ORDER BY created_at DESC LIMIT 100"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var lcs []models.LetterOfCredit
	for rows.Next() {
		var lc models.LetterOfCredit; var docsRaw []byte
		if err := rows.Scan(&lc.ID, &lc.LCNumber, &lc.LCType, &lc.ApplicantID, &lc.ApplicantName, &lc.BeneficiaryName,
			&lc.IssuingBank, &lc.AdvisingBank, &lc.Amount, &lc.Currency, &lc.ExpiryDate,
			&lc.ShipmentPOL, &lc.ShipmentPOD, &lc.GoodsDescription, &docsRaw, &lc.Status,
			&lc.IssuedAt, &lc.RealizedAt, &lc.CreatedAt, &lc.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(docsRaw, &lc.DocumentsRequired)
		lcs = append(lcs, lc)
	}
	return lcs, nil
}

func (q *Queries) UpdateLCStatus(ctx context.Context, id uuid.UUID, status models.LCStatus, now time.Time) error {
	query := `UPDATE letters_of_credit SET status=$1,updated_at=$2`
	args := []interface{}{status, now}
	if status == models.LCIssued { query += ",issued_at=COALESCE(issued_at,$3)"; args = append(args, now) }
	if status == models.LCRealized { query += ",realized_at=COALESCE(realized_at,$3)"; args = append(args, now) }
	args = append(args, id)
	query += " WHERE id=$" + string(rune('0'+len(args)))
	_, err := q.pool.Exec(ctx, query, args...)
	return err
}

func (q *Queries) CreateFinancing(ctx context.Context, f models.FinancingRequest) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO financing_requests (id,reference_no,applicant_id,booking_id,lc_id,requested_amount,currency,payment_terms,interest_rate_pct,interest_amount,total_repayable,status,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		f.ID, f.RefNo, f.ApplicantID, f.BookingID, f.LCID, f.RequestedAmount, f.Currency,
		f.PaymentTerms, f.InterestRatePct, f.InterestAmount, f.TotalRepayable, f.Status, f.CreatedAt, f.UpdatedAt)
	return err
}

func (q *Queries) GetFinancing(ctx context.Context, id uuid.UUID) (*models.FinancingRequest, error) {
	f := &models.FinancingRequest{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,reference_no,applicant_id,booking_id,lc_id,requested_amount,currency,payment_terms,interest_rate_pct,interest_amount,total_repayable,status,approved_at,disbursed_at,created_at,updated_at
         FROM financing_requests WHERE id=$1`, id).
		Scan(&f.ID, &f.RefNo, &f.ApplicantID, &f.BookingID, &f.LCID, &f.RequestedAmount, &f.Currency,
			&f.PaymentTerms, &f.InterestRatePct, &f.InterestAmount, &f.TotalRepayable, &f.Status,
			&f.ApprovedAt, &f.DisbursedAt, &f.CreatedAt, &f.UpdatedAt)
	if err != nil { return nil, err }
	return f, nil
}

func (q *Queries) ApproveFinancing(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE financing_requests SET status='approved',approved_at=COALESCE(approved_at,$1),updated_at=$2 WHERE id=$3`, now, now, id)
	return err
}

func (q *Queries) DisburseFinancing(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE financing_requests SET status='disbursed',disbursed_at=COALESCE(disbursed_at,$1),updated_at=$2 WHERE id=$3`, now, now, id)
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.TradeFinanceAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO trade_finance_audit_log (id,actor_id,action,entity_type,entity_id,created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
