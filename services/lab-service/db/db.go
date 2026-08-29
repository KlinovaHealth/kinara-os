package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/lab-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateOrder(ctx context.Context, o models.LabOrder) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO lab_orders (id,order_ref,patient_id,ordered_by,clinic_id,test_code,test_name,priority,status,notes,tenant_id,ordered_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		o.ID, o.OrderRef, o.PatientID, o.OrderedBy, o.ClinicID, o.TestCode,
		o.TestName, o.Priority, o.Status, o.Notes, o.TenantID, o.OrderedAt)
	return err
}

func (q *Queries) GetOrder(ctx context.Context, id uuid.UUID) (*models.LabOrder, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id,order_ref,patient_id,ordered_by,clinic_id,test_code,test_name,priority,status,notes,tenant_id,ordered_at,completed_at
		 FROM lab_orders WHERE id=$1`, id)
	var o models.LabOrder
	err := row.Scan(&o.ID, &o.OrderRef, &o.PatientID, &o.OrderedBy, &o.ClinicID, &o.TestCode,
		&o.TestName, &o.Priority, &o.Status, &o.Notes, &o.TenantID, &o.OrderedAt, &o.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (q *Queries) ListByPatient(ctx context.Context, patientID uuid.UUID) ([]models.LabOrder, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,order_ref,patient_id,ordered_by,clinic_id,test_code,test_name,priority,status,notes,tenant_id,ordered_at,completed_at
		 FROM lab_orders WHERE patient_id=$1 ORDER BY ordered_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LabOrder
	for rows.Next() {
		var o models.LabOrder
		if err := rows.Scan(&o.ID, &o.OrderRef, &o.PatientID, &o.OrderedBy, &o.ClinicID, &o.TestCode,
			&o.TestName, &o.Priority, &o.Status, &o.Notes, &o.TenantID, &o.OrderedAt, &o.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func (q *Queries) InsertResult(ctx context.Context, r models.LabResult) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO lab_results (id,order_id,patient_id,test_code,result_value,unit,reference_range,flag,analyzed_by,result_at,tenant_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		r.ID, r.OrderID, r.PatientID, r.TestCode, r.ResultValue, r.Unit,
		r.ReferenceRange, r.Flag, r.AnalyzedBy, r.ResultAt, r.TenantID)
	return err
}

func (q *Queries) MarkCompleted(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE lab_orders SET status='completed', completed_at=COALESCE(completed_at,$1) WHERE id=$2`, now, id)
	return err
}
