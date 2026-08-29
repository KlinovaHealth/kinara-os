package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/cargo-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

const cargoCols = `id,booking_ref,shipper_id,consignee_id,cargo_type,description,weight_kg,volume_m3,status,
	origin_address,origin_lat,origin_lng,destination_address,destination_lat,destination_lng,
	pickup_at,delivered_at,assigned_vehicle_id,assigned_driver_id,estimated_delivery,
	freight_cost,currency,notes,created_at,updated_at`

func (q *Queries) CreateBooking(ctx context.Context, b models.CargoBooking) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO cargo_bookings(`+cargoCols+`) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`,
		b.ID,b.BookingRef,b.ShipperID,b.ConsigneeID,b.CargoType,b.Description,b.WeightKg,b.VolumeM3,b.Status,
		b.OriginAddress,b.OriginLat,b.OriginLng,b.DestinationAddress,b.DestinationLat,b.DestinationLng,
		b.PickupAt,b.DeliveredAt,b.AssignedVehicleID,b.AssignedDriverID,b.EstimatedDelivery,
		b.FreightCost,b.Currency,b.Notes,b.CreatedAt,b.UpdatedAt)
	return err
}

func (q *Queries) GetBooking(ctx context.Context, id uuid.UUID) (*models.CargoBooking, error) {
	row := q.pool.QueryRow(ctx, `SELECT `+cargoCols+` FROM cargo_bookings WHERE id=$1`, id)
	return scanBooking(row)
}

func (q *Queries) GetBookingByRef(ctx context.Context, ref string) (*models.CargoBooking, error) {
	row := q.pool.QueryRow(ctx, `SELECT `+cargoCols+` FROM cargo_bookings WHERE booking_ref=$1`, ref)
	return scanBooking(row)
}

type ListCargoParams struct{ ShipperID *uuid.UUID; Status *models.CargoStatus; Page, Limit int }

func (q *Queries) ListBookings(ctx context.Context, p ListCargoParams) ([]models.CargoBooking, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.ShipperID != nil { where += fmt.Sprintf(" AND shipper_id=$%d",n); args=append(args,*p.ShipperID); n++ }
	if p.Status    != nil { where += fmt.Sprintf(" AND status=$%d",n);     args=append(args,*p.Status);    n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT `+cargoCols+` FROM cargo_bookings %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.CargoBooking
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil { return nil, err }
		result = append(result, *b)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateBookingStatus(ctx context.Context, id uuid.UUID, status models.CargoStatus, now time.Time) error {
	pickupAt := (*time.Time)(nil)
	deliveredAt := (*time.Time)(nil)
	if status == models.CargoPickedUp { pickupAt = &now }
	if status == models.CargoDelivered { deliveredAt = &now }
	_, err := q.pool.Exec(ctx, `UPDATE cargo_bookings SET status=$1,
		pickup_at=CASE WHEN $2::TIMESTAMPTZ IS NOT NULL THEN $2 ELSE pickup_at END,
		delivered_at=CASE WHEN $3::TIMESTAMPTZ IS NOT NULL THEN $3 ELSE delivered_at END,
		updated_at=$4 WHERE id=$5`, status,pickupAt,deliveredAt,now,id)
	return err
}

func (q *Queries) AssignCargo(ctx context.Context, id, vehicleID, driverID uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE cargo_bookings SET assigned_vehicle_id=$1,assigned_driver_id=$2,updated_at=$3 WHERE id=$4`,
		vehicleID,driverID,now,id)
	return err
}

func (q *Queries) AddTrackingEvent(ctx context.Context, e models.TrackingEvent) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO tracking_events(id,cargo_id,status,location,latitude,longitude,notes,event_time,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.ID,e.CargoID,e.Status,e.Location,e.Latitude,e.Longitude,e.Notes,e.EventTime,e.CreatedAt)
	return err
}

func (q *Queries) ListTracking(ctx context.Context, cargoID uuid.UUID) ([]models.TrackingEvent, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,cargo_id,status,location,latitude,longitude,notes,event_time,created_at FROM tracking_events WHERE cargo_id=$1 ORDER BY event_time DESC`, cargoID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.TrackingEvent
	for rows.Next() {
		var e models.TrackingEvent
		if err := rows.Scan(&e.ID,&e.CargoID,&e.Status,&e.Location,&e.Latitude,&e.Longitude,&e.Notes,&e.EventTime,&e.CreatedAt); err != nil { return nil, err }
		result = append(result, e)
	}
	return result, rows.Err()
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.CargoAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO cargo_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}

type scannable interface{ Scan(dest ...any) error }

func scanBooking(row scannable) (*models.CargoBooking, error) {
	var b models.CargoBooking
	err := row.Scan(&b.ID,&b.BookingRef,&b.ShipperID,&b.ConsigneeID,&b.CargoType,&b.Description,&b.WeightKg,&b.VolumeM3,&b.Status,
		&b.OriginAddress,&b.OriginLat,&b.OriginLng,&b.DestinationAddress,&b.DestinationLat,&b.DestinationLng,
		&b.PickupAt,&b.DeliveredAt,&b.AssignedVehicleID,&b.AssignedDriverID,&b.EstimatedDelivery,
		&b.FreightCost,&b.Currency,&b.Notes,&b.CreatedAt,&b.UpdatedAt)
	return &b, err
}
