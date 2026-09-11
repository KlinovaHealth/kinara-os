package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Querier interface {
	Create(ctx context.Context, r Record) error
	Get(ctx context.Context, id uuid.UUID) (*Record, error)
	List(ctx context.Context, limit, offset int, tenantID uuid.UUID) ([]Record, error)
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
}

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

type Record struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	DataEnc   string    `db:"data_enc"   json:"data_enc"`
	CreatedBy uuid.UUID `db:"created_by" json:"created_by"`
	TenantID  uuid.UUID `db:"tenant_id"  json:"tenant_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (q *Queries) Create(ctx context.Context, r Record) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO records (id, data_enc, created_by, tenant_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		r.ID, r.DataEnc, r.CreatedBy, r.TenantID, r.CreatedAt, r.UpdatedAt)
	return err
}

func (q *Queries) Get(ctx context.Context, id uuid.UUID) (*Record, error) {
	var r Record
	err := q.pool.QueryRow(ctx,
		`SELECT id, data_enc, created_by, tenant_id, created_at, updated_at FROM records WHERE id = $1`, id).
		Scan(&r.ID, &r.DataEnc, &r.CreatedBy, &r.TenantID, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (q *Queries) List(ctx context.Context, limit, offset int, tenantID uuid.UUID) ([]Record, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, data_enc, created_by, tenant_id, created_at, updated_at FROM records
		 WHERE tenant_id = $3 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.ID, &r.DataEnc, &r.CreatedBy, &r.TenantID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *Queries) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := q.pool.Exec(ctx, `DELETE FROM records WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}
