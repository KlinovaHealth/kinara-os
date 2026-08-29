package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/transport-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateTrip(ctx context.Context, t models.TransportTrip) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO transport_trips(id,trip_code,route_id,vehicle_id,driver_id,cargo_id,status,country,origin_address,origin_lat,origin_lng,dest_address,dest_lat,dest_lng,scheduled_pickup,scheduled_delivery,distance_km,cost_per_km,total_cost,currency,fuel_cost,notes,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		t.ID,t.TripCode,t.RouteID,t.VehicleID,t.DriverID,t.CargoID,t.Status,t.Country,t.OriginAddress,t.OriginLat,t.OriginLng,t.DestAddress,t.DestLat,t.DestLng,t.ScheduledPickup,t.ScheduledDelivery,t.DistanceKm,t.CostPerKm,t.TotalCost,t.Currency,t.FuelCost,t.Notes,t.CreatedAt,t.UpdatedAt)
	return err
}

func (q *Queries) GetTrip(ctx context.Context, id uuid.UUID) (*models.TransportTrip, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,trip_code,route_id,vehicle_id,driver_id,cargo_id,status,country,origin_address,origin_lat,origin_lng,dest_address,dest_lat,dest_lng,scheduled_pickup,scheduled_delivery,actual_pickup,actual_delivery,distance_km,cost_per_km,total_cost,currency,fuel_cost,current_lat,current_lng,last_gps_update,delay_reason_code,notes,created_at,updated_at FROM transport_trips WHERE id=$1`, id)
	return scanTrip(row)
}

type ListTripsParams struct{ Country *string; Status *models.TripStatus; DriverID *uuid.UUID; Page, Limit int }

func (q *Queries) ListTrips(ctx context.Context, p ListTripsParams) ([]models.TransportTrip, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.Country != nil { where += fmt.Sprintf(" AND country=$%d",n); args=append(args,*p.Country); n++ }
	if p.Status != nil { where += fmt.Sprintf(" AND status=$%d",n); args=append(args,*p.Status); n++ }
	if p.DriverID != nil { where += fmt.Sprintf(" AND driver_id=$%d",n); args=append(args,*p.DriverID); n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT id,trip_code,route_id,vehicle_id,driver_id,cargo_id,status,country,origin_address,origin_lat,origin_lng,dest_address,dest_lat,dest_lng,scheduled_pickup,scheduled_delivery,actual_pickup,actual_delivery,distance_km,cost_per_km,total_cost,currency,fuel_cost,current_lat,current_lng,last_gps_update,delay_reason_code,notes,created_at,updated_at FROM transport_trips %s ORDER BY scheduled_pickup DESC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.TransportTrip
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil { return nil, err }
		result = append(result, *t)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateTripStatus(ctx context.Context, id uuid.UUID, status models.TripStatus, delay string, now time.Time) error {
	var actualPickup, actualDelivery interface{}
	if status == models.TripEnRoute { actualPickup = now }
	if status == models.TripDelivered { actualDelivery = now }
	_, err := q.pool.Exec(ctx, `UPDATE transport_trips SET status=$1,delay_reason_code=$2,actual_pickup=COALESCE($3,actual_pickup),actual_delivery=COALESCE($4,actual_delivery),updated_at=$5 WHERE id=$6`,
		status,delay,actualPickup,actualDelivery,now,id)
	return err
}

func (q *Queries) UpdateGPS(ctx context.Context, id uuid.UUID, lat, lng float64, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE transport_trips SET current_lat=$1,current_lng=$2,last_gps_update=$3,updated_at=$3 WHERE id=$4`, lat,lng,now,id)
	return err
}

func (q *Queries) AddGPSUpdate(ctx context.Context, g models.GPSUpdate) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO gps_updates(id,trip_id,latitude,longitude,speed_kph,heading,recorded_at,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		g.ID,g.TripID,g.Latitude,g.Longitude,g.SpeedKph,g.Heading,g.RecordedAt,g.CreatedAt)
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.TransportAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO transport_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}

type scannable interface{ Scan(dest ...any) error }

func scanTrip(row scannable) (*models.TransportTrip, error) {
	var t models.TransportTrip
	err := row.Scan(&t.ID,&t.TripCode,&t.RouteID,&t.VehicleID,&t.DriverID,&t.CargoID,&t.Status,&t.Country,&t.OriginAddress,&t.OriginLat,&t.OriginLng,&t.DestAddress,&t.DestLat,&t.DestLng,&t.ScheduledPickup,&t.ScheduledDelivery,&t.ActualPickup,&t.ActualDelivery,&t.DistanceKm,&t.CostPerKm,&t.TotalCost,&t.Currency,&t.FuelCost,&t.CurrentLat,&t.CurrentLng,&t.LastGPSUpdate,&t.DelayReasonCode,&t.Notes,&t.CreatedAt,&t.UpdatedAt)
	return &t, err
}
