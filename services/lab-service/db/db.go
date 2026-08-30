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
		`INSERT INTO lab_orders
		 (id,order_ref,patient_id,ordered_by,clinic_id,test_code,test_name,priority,status,notes,tenant_id,ordered_at)
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
	err := row.Scan(
		&o.ID, &o.OrderRef, &o.PatientID, &o.OrderedBy, &o.ClinicID, &o.TestCode,
		&o.TestName, &o.Priority, &o.Status, &o.Notes, &o.TenantID, &o.OrderedAt, &o.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (q *Queries) UploadResult(ctx context.Context, r models.LabResult) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO lab_results
		 (id,order_id,result_value,unit,normal_low,normal_high,flag,notes,recorded_by,recorded_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		r.ID, r.OrderID, r.ResultValue, r.Unit, r.NormalLow, r.NormalHigh,
		r.Flag, r.Notes, r.RecordedBy, r.RecordedAt)
	return err
}

func (q *Queries) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string, completedAt time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE lab_orders SET status=$1, completed_at=COALESCE(completed_at,$2) WHERE id=$3`,
		status, completedAt, id)
	return err
}

func (q *Queries) ListPatientResults(ctx context.Context, patientID uuid.UUID) ([]models.LabResultWithOrder, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT
		    o.id,o.order_ref,o.patient_id,o.ordered_by,o.clinic_id,o.test_code,o.test_name,
		    o.priority,o.status,o.notes,o.tenant_id,o.ordered_at,o.completed_at,
		    r.id,r.order_id,r.result_value,r.unit,r.normal_low,r.normal_high,r.flag,r.notes,r.recorded_by,r.recorded_at
		 FROM lab_orders o
		 LEFT JOIN lab_results r ON r.order_id=o.id
		 WHERE o.patient_id=$1
		 ORDER BY o.ordered_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LabResultWithOrder
	for rows.Next() {
		var o models.LabOrder
		var rID, rOrderID, rRecordedBy *uuid.UUID
		var rValue *float64
		var rUnit, rFlag, rNotes *string
		var rNormalLow, rNormalHigh *float64
		var rRecordedAt *time.Time

		if err := rows.Scan(
			&o.ID, &o.OrderRef, &o.PatientID, &o.OrderedBy, &o.ClinicID, &o.TestCode, &o.TestName,
			&o.Priority, &o.Status, &o.Notes, &o.TenantID, &o.OrderedAt, &o.CompletedAt,
			&rID, &rOrderID, &rValue, &rUnit, &rNormalLow, &rNormalHigh, &rFlag, &rNotes, &rRecordedBy, &rRecordedAt,
		); err != nil {
			return nil, err
		}
		item := models.LabResultWithOrder{Order: o}
		if rID != nil {
			item.Result = &models.LabResult{
				ID:          *rID,
				OrderID:     *rOrderID,
				ResultValue: *rValue,
				Unit:        *rUnit,
				NormalLow:   *rNormalLow,
				NormalHigh:  *rNormalHigh,
				Flag:        *rFlag,
				RecordedBy:  *rRecordedBy,
				RecordedAt:  *rRecordedAt,
			}
			if rNotes != nil {
				item.Result.Notes = *rNotes
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (q *Queries) GetOrderStatus(ctx context.Context, id uuid.UUID) (string, error) {
	var status string
	err := q.pool.QueryRow(ctx,
		`SELECT status FROM lab_orders WHERE id=$1`, id).Scan(&status)
	return status, err
}

func (q *Queries) GetTestCatalog(ctx context.Context, testCode string) (*models.TestCatalogEntry, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT test_code,test_name,normal_low,normal_high,unit
		 FROM test_catalog WHERE test_code=$1`, testCode)
	var e models.TestCatalogEntry
	err := row.Scan(&e.TestCode, &e.TestName, &e.NormalLow, &e.NormalHigh, &e.Unit)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (q *Queries) InsertAudit(ctx context.Context, orderID, actorID, action string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO lab_audit_log (order_id,action,actor_id,occurred_at) VALUES ($1,$2,$3,NOW())`,
		orderID, action, actorID)
	return err
}

// ---------------------------------------------------------------------------
// Backward-compat aliases used by the old stub (main.go / other callers)
// ---------------------------------------------------------------------------

func (q *Queries) InsertResult(ctx context.Context, r models.LabResult) error {
	return q.UploadResult(ctx, r)
}

func (q *Queries) MarkCompleted(ctx context.Context, id uuid.UUID, now time.Time) error {
	return q.UpdateOrderStatus(ctx, id, "completed", now)
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
		if err := rows.Scan(
			&o.ID, &o.OrderRef, &o.PatientID, &o.OrderedBy, &o.ClinicID, &o.TestCode,
			&o.TestName, &o.Priority, &o.Status, &o.Notes, &o.TenantID, &o.OrderedAt, &o.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}
