package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/customs-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateTariff(ctx context.Context, t models.TariffCode) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO tariff_codes (id,hs_code,description,category,duty_rate_pct,vat_rate_pct,country,is_restricted,notes,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		t.ID, t.HSCode, t.Description, t.Category, t.DutyRate, t.VATRate,
		t.Country, t.IsRestricted, t.Notes, t.CreatedAt)
	return err
}

func (q *Queries) LookupTariff(ctx context.Context, hsCode, country string) (*models.TariffCode, error) {
	t := &models.TariffCode{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,hs_code,description,category,duty_rate_pct,vat_rate_pct,country,is_restricted,notes,created_at
         FROM tariff_codes WHERE hs_code=$1 AND country=$2`, hsCode, country).
		Scan(&t.ID, &t.HSCode, &t.Description, &t.Category, &t.DutyRate, &t.VATRate,
			&t.Country, &t.IsRestricted, &t.Notes, &t.CreatedAt)
	if err != nil { return nil, err }
	return t, nil
}

func (q *Queries) ListTariffs(ctx context.Context, country *string) ([]models.TariffCode, error) {
	query := `SELECT id,hs_code,description,category,duty_rate_pct,vat_rate_pct,country,is_restricted,notes,created_at FROM tariff_codes`
	args := []interface{}{}
	if country != nil { query += " WHERE country=$1"; args = append(args, *country) }
	query += " ORDER BY hs_code ASC LIMIT 500"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var tariffs []models.TariffCode
	for rows.Next() {
		var t models.TariffCode
		if err := rows.Scan(&t.ID, &t.HSCode, &t.Description, &t.Category, &t.DutyRate, &t.VATRate,
			&t.Country, &t.IsRestricted, &t.Notes, &t.CreatedAt); err != nil {
			return nil, err
		}
		tariffs = append(tariffs, t)
	}
	return tariffs, nil
}

func (q *Queries) CreateClearance(ctx context.Context, c models.ClearanceRequest) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO clearance_requests (id,reference_no,importer_name,importer_id,manifest_id,vessel_id,port_id,hs_code,goods_description,declared_value,currency,weight_kg,duty_amount,vat_amount,total_due,status,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		c.ID, c.ReferenceNo, c.ImporterName, c.ImporterID, c.ManifestID, c.VesselID, c.PortID,
		c.HSCode, c.GoodsDescription, c.DeclaredValue, c.Currency, c.WeightKg,
		c.DutyAmount, c.VATAmount, c.TotalDue, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

func (q *Queries) GetClearance(ctx context.Context, id uuid.UUID) (*models.ClearanceRequest, error) {
	c := &models.ClearanceRequest{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,reference_no,importer_name,importer_id,manifest_id,vessel_id,port_id,hs_code,goods_description,declared_value,currency,weight_kg,duty_amount,vat_amount,total_due,status,reviewed_by,reviewed_at,rejection_reason,created_at,updated_at
         FROM clearance_requests WHERE id=$1`, id).
		Scan(&c.ID, &c.ReferenceNo, &c.ImporterName, &c.ImporterID, &c.ManifestID, &c.VesselID, &c.PortID,
			&c.HSCode, &c.GoodsDescription, &c.DeclaredValue, &c.Currency, &c.WeightKg,
			&c.DutyAmount, &c.VATAmount, &c.TotalDue, &c.Status, &c.ReviewedBy, &c.ReviewedAt,
			&c.RejectionReason, &c.CreatedAt, &c.UpdatedAt)
	if err != nil { return nil, err }
	return c, nil
}

func (q *Queries) ListClearances(ctx context.Context, portID *uuid.UUID, status *models.ClearanceStatus) ([]models.ClearanceRequest, error) {
	query := `SELECT id,reference_no,importer_name,importer_id,manifest_id,vessel_id,port_id,hs_code,goods_description,declared_value,currency,weight_kg,duty_amount,vat_amount,total_due,status,reviewed_by,reviewed_at,rejection_reason,created_at,updated_at FROM clearance_requests WHERE 1=1`
	args := []interface{}{}
	i := 1
	if portID != nil { query += " AND port_id=$1"; args = append(args, *portID); i++ }
	if status != nil {
		if i == 1 { query += " AND status=$1" } else { query += " AND status=$2" }
		args = append(args, *status)
	}
	query += " ORDER BY created_at DESC LIMIT 100"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var clearances []models.ClearanceRequest
	for rows.Next() {
		var c models.ClearanceRequest
		if err := rows.Scan(&c.ID, &c.ReferenceNo, &c.ImporterName, &c.ImporterID, &c.ManifestID, &c.VesselID, &c.PortID,
			&c.HSCode, &c.GoodsDescription, &c.DeclaredValue, &c.Currency, &c.WeightKg,
			&c.DutyAmount, &c.VATAmount, &c.TotalDue, &c.Status, &c.ReviewedBy, &c.ReviewedAt,
			&c.RejectionReason, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		clearances = append(clearances, c)
	}
	return clearances, nil
}

func (q *Queries) UpdateClearanceStatus(ctx context.Context, id uuid.UUID, status models.ClearanceStatus, reviewerID, rejectionReason string, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE clearance_requests SET status=$1,reviewed_by=$2,reviewed_at=COALESCE(reviewed_at,$3),rejection_reason=COALESCE(NULLIF($4,''),rejection_reason),updated_at=$5 WHERE id=$6`,
		status, reviewerID, now, rejectionReason, now, id)
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.CustomsAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO customs_audit_log (id,port_id,actor_id,action,entity_type,entity_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		l.ID, l.PortID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
