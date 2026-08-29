package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/input-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreatePurchase(ctx context.Context, p models.InputPurchase) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO input_purchases (id,purchase_ref,farmer_id,coop_id,input_type,input_name,quantity,unit,cost_xof,supplier,purchased_at,tenant_id,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		p.ID, p.PurchaseRef, p.FarmerID, p.CoopID, p.InputType, p.InputName,
		p.Quantity, p.Unit, p.CostXOF, p.Supplier, p.PurchasedAt, p.TenantID, p.CreatedAt)
	return err
}

func (q *Queries) ListByFarmer(ctx context.Context, farmerID uuid.UUID, limit int) ([]models.InputPurchase, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,purchase_ref,farmer_id,coop_id,input_type,input_name,quantity,unit,cost_xof,supplier,purchased_at,tenant_id,created_at
		 FROM input_purchases WHERE farmer_id=$1 ORDER BY purchased_at DESC LIMIT $2`,
		farmerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.InputPurchase
	for rows.Next() {
		var p models.InputPurchase
		if err := rows.Scan(&p.ID, &p.PurchaseRef, &p.FarmerID, &p.CoopID, &p.InputType, &p.InputName,
			&p.Quantity, &p.Unit, &p.CostXOF, &p.Supplier, &p.PurchasedAt, &p.TenantID, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (q *Queries) RecordUsage(ctx context.Context, u models.InputUsage) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO input_usages (id,purchase_id,farmer_id,field_id,quantity,used_at,notes,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		u.ID, u.PurchaseID, u.FarmerID, u.FieldID, u.Quantity, u.UsedAt, u.Notes, u.UsedAt)
	return err
}
