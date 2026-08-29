package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/shipment-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateShipment(ctx context.Context, s models.Shipment) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO shipments(id,tracking_code,sender_id,recipient_name,recipient_phone,origin_address,origin_country,dest_address,dest_country,weight_kg,length_cm,width_cm,height_cm,declared_value,currency,service_level,status,freight_charge,insurance_charge,total_charge,est_delivery,notes,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		s.ID,s.TrackingCode,s.SenderID,s.RecipientName,s.RecipientPhone,s.OriginAddress,s.OriginCountry,s.DestAddress,s.DestCountry,s.WeightKg,s.LengthCm,s.WidthCm,s.HeightCm,s.DeclaredValue,s.Currency,s.ServiceLevel,s.Status,s.FreightCharge,s.InsuranceCharge,s.TotalCharge,s.EstDelivery,s.Notes,s.CreatedAt,s.UpdatedAt)
	return err
}

func (q *Queries) GetShipment(ctx context.Context, id uuid.UUID) (*models.Shipment, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,tracking_code,sender_id,recipient_name,recipient_phone,origin_address,origin_country,dest_address,dest_country,weight_kg,length_cm,width_cm,height_cm,declared_value,currency,service_level,status,freight_charge,insurance_charge,total_charge,picked_at,delivered_at,est_delivery,notes,created_at,updated_at FROM shipments WHERE id=$1`, id)
	return scanShipment(row)
}

func (q *Queries) GetShipmentByTrackingCode(ctx context.Context, code string) (*models.Shipment, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,tracking_code,sender_id,recipient_name,recipient_phone,origin_address,origin_country,dest_address,dest_country,weight_kg,length_cm,width_cm,height_cm,declared_value,currency,service_level,status,freight_charge,insurance_charge,total_charge,picked_at,delivered_at,est_delivery,notes,created_at,updated_at FROM shipments WHERE tracking_code=$1`, code)
	return scanShipment(row)
}

type ListShipmentsParams struct{ SenderID *uuid.UUID; Status *models.ShipmentStatus; Page, Limit int }

func (q *Queries) ListShipments(ctx context.Context, p ListShipmentsParams) ([]models.Shipment, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.SenderID != nil { where += fmt.Sprintf(" AND sender_id=$%d",n); args=append(args,*p.SenderID); n++ }
	if p.Status != nil { where += fmt.Sprintf(" AND status=$%d",n); args=append(args,*p.Status); n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT id,tracking_code,sender_id,recipient_name,recipient_phone,origin_address,origin_country,dest_address,dest_country,weight_kg,length_cm,width_cm,height_cm,declared_value,currency,service_level,status,freight_charge,insurance_charge,total_charge,picked_at,delivered_at,est_delivery,notes,created_at,updated_at FROM shipments %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.Shipment
	for rows.Next() {
		s, err := scanShipment(rows)
		if err != nil { return nil, err }
		result = append(result, *s)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateShipmentStatus(ctx context.Context, id uuid.UUID, status models.ShipmentStatus, now time.Time) error {
	var pickedAt, deliveredAt interface{}
	if status == models.ShipmentPicked { pickedAt = now }
	if status == models.ShipmentDelivered { deliveredAt = now }
	_, err := q.pool.Exec(ctx, `UPDATE shipments SET status=$1,picked_at=COALESCE($2,picked_at),delivered_at=COALESCE($3,delivered_at),updated_at=$4 WHERE id=$5`,
		status,pickedAt,deliveredAt,now,id)
	return err
}

func (q *Queries) AddEvent(ctx context.Context, e models.ShipmentEvent) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO shipment_events(id,shipment_id,status,location,notes,event_time,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		e.ID,e.ShipmentID,e.Status,e.Location,e.Notes,e.EventTime,e.CreatedAt)
	return err
}

func (q *Queries) ListEvents(ctx context.Context, shipmentID uuid.UUID) ([]models.ShipmentEvent, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,shipment_id,status,location,notes,event_time,created_at FROM shipment_events WHERE shipment_id=$1 ORDER BY event_time DESC`, shipmentID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.ShipmentEvent
	for rows.Next() {
		var e models.ShipmentEvent
		if err := rows.Scan(&e.ID,&e.ShipmentID,&e.Status,&e.Location,&e.Notes,&e.EventTime,&e.CreatedAt); err != nil { return nil, err }
		result = append(result, e)
	}
	return result, rows.Err()
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.ShipmentAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO shipment_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}

type scannable interface{ Scan(dest ...any) error }

func scanShipment(row scannable) (*models.Shipment, error) {
	var s models.Shipment
	err := row.Scan(&s.ID,&s.TrackingCode,&s.SenderID,&s.RecipientName,&s.RecipientPhone,&s.OriginAddress,&s.OriginCountry,&s.DestAddress,&s.DestCountry,&s.WeightKg,&s.LengthCm,&s.WidthCm,&s.HeightCm,&s.DeclaredValue,&s.Currency,&s.ServiceLevel,&s.Status,&s.FreightCharge,&s.InsuranceCharge,&s.TotalCharge,&s.PickedAt,&s.DeliveredAt,&s.EstDelivery,&s.Notes,&s.CreatedAt,&s.UpdatedAt)
	return &s, err
}
