package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/last-mile-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateDelivery(ctx context.Context, d models.Delivery) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO deliveries(id,delivery_code,cargo_id,driver_id,recipient_name,recipient_phone,delivery_address,delivery_lat,delivery_lng,status,window_start,window_end,attempt_count,sms_notified,country,notes,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		d.ID,d.DeliveryCode,d.CargoID,d.DriverID,d.RecipientName,d.RecipientPhone,d.DeliveryAddress,d.DeliveryLat,d.DeliveryLng,d.Status,d.WindowStart,d.WindowEnd,d.AttemptCount,d.SMSNotified,d.Country,d.Notes,d.CreatedAt,d.UpdatedAt)
	return err
}

func (q *Queries) GetDelivery(ctx context.Context, id uuid.UUID) (*models.Delivery, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,delivery_code,cargo_id,driver_id,recipient_name,recipient_phone,delivery_address,delivery_lat,delivery_lng,status,window_start,window_end,attempt_count,delivered_at,proof_photo_url,signature_url,failure_reason,next_attempt_at,sms_notified,country,notes,created_at,updated_at FROM deliveries WHERE id=$1`, id)
	return scanDelivery(row)
}

type ListDeliveryParams struct{ Status *models.DeliveryStatus; DriverID *uuid.UUID; Page, Limit int }

func (q *Queries) ListDeliveries(ctx context.Context, p ListDeliveryParams) ([]models.Delivery, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.Status != nil { where += fmt.Sprintf(" AND status=$%d",n); args=append(args,*p.Status); n++ }
	if p.DriverID != nil { where += fmt.Sprintf(" AND driver_id=$%d",n); args=append(args,*p.DriverID); n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT id,delivery_code,cargo_id,driver_id,recipient_name,recipient_phone,delivery_address,delivery_lat,delivery_lng,status,window_start,window_end,attempt_count,delivered_at,proof_photo_url,signature_url,failure_reason,next_attempt_at,sms_notified,country,notes,created_at,updated_at FROM deliveries %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.Delivery
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil { return nil, err }
		result = append(result, *d)
	}
	return result, rows.Err()
}

func (q *Queries) AssignDriver(ctx context.Context, id, driverID uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE deliveries SET driver_id=$1,status='assigned',updated_at=$2 WHERE id=$3`, driverID,now,id)
	return err
}

func (q *Queries) RecordDelivered(ctx context.Context, id uuid.UUID, photoURL, sigURL, notes string, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE deliveries SET status='delivered',delivered_at=$1,proof_photo_url=$2,signature_url=$3,notes=$4,updated_at=$1 WHERE id=$5`,
		now,photoURL,sigURL,notes,id)
	return err
}

func (q *Queries) RecordFailure(ctx context.Context, id uuid.UUID, reason models.FailureReason, nextAt *time.Time, notes string, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE deliveries SET status='failed',failure_reason=$1,next_attempt_at=$2,attempt_count=attempt_count+1,notes=$3,updated_at=$4 WHERE id=$5`,
		reason,nextAt,notes,now,id)
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.LastMileAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO lastmile_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}

type scannable interface{ Scan(dest ...any) error }

func scanDelivery(row scannable) (*models.Delivery, error) {
	var d models.Delivery
	err := row.Scan(&d.ID,&d.DeliveryCode,&d.CargoID,&d.DriverID,&d.RecipientName,&d.RecipientPhone,&d.DeliveryAddress,&d.DeliveryLat,&d.DeliveryLng,&d.Status,&d.WindowStart,&d.WindowEnd,&d.AttemptCount,&d.DeliveredAt,&d.ProofPhotoURL,&d.SignatureURL,&d.FailureReason,&d.NextAttemptAt,&d.SMSNotified,&d.Country,&d.Notes,&d.CreatedAt,&d.UpdatedAt)
	return &d, err
}
