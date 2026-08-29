package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/route-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateRoute(ctx context.Context, r models.Route) error {
	wp, _ := json.Marshal(r.Waypoints)
	_, err := q.pool.Exec(ctx, `INSERT INTO routes(id,name,route_code,route_type,status,country,origin_name,origin_lat,origin_lng,dest_name,dest_lat,dest_lng,distance_km,est_hours,waypoints,freight_class,notes,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		r.ID,r.Name,r.RouteCode,r.RouteType,r.Status,r.Country,r.OriginName,r.OriginLat,r.OriginLng,
		r.DestName,r.DestLat,r.DestLng,r.DistanceKm,r.EstHours,wp,r.FreightClass,r.Notes,r.CreatedAt,r.UpdatedAt)
	return err
}

func (q *Queries) GetRoute(ctx context.Context, id uuid.UUID) (*models.Route, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,name,route_code,route_type,status,country,origin_name,origin_lat,origin_lng,dest_name,dest_lat,dest_lng,distance_km,est_hours,waypoints,freight_class,notes,created_at,updated_at FROM routes WHERE id=$1`, id)
	return scanRoute(row)
}

type ListRoutesParams struct{ Country *string; Page, Limit int }

func (q *Queries) ListRoutes(ctx context.Context, p ListRoutesParams) ([]models.Route, error) {
	where := "WHERE status='active'"; var args []interface{}; n := 1
	if p.Country != nil { where += fmt.Sprintf(" AND country=$%d",n); args=append(args,*p.Country); n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT id,name,route_code,route_type,status,country,origin_name,origin_lat,origin_lng,dest_name,dest_lat,dest_lng,distance_km,est_hours,waypoints,freight_class,notes,created_at,updated_at FROM routes %s ORDER BY name ASC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.Route
	for rows.Next() {
		r, err := scanRoute(rows)
		if err != nil { return nil, err }
		result = append(result, *r)
	}
	return result, rows.Err()
}

func (q *Queries) ScheduleRoute(ctx context.Context, s models.RouteSchedule) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO route_schedules(id,route_id,vehicle_id,driver_id,departure_time,status,notes,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		s.ID,s.RouteID,s.VehicleID,s.DriverID,s.DepartureTime,s.Status,s.Notes,s.CreatedAt,s.UpdatedAt)
	return err
}

func (q *Queries) ListSchedules(ctx context.Context, routeID uuid.UUID) ([]models.RouteSchedule, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,route_id,vehicle_id,driver_id,departure_time,arrival_time,status,actual_dept_at,actual_arr_at,notes,created_at,updated_at FROM route_schedules WHERE route_id=$1 ORDER BY departure_time DESC LIMIT 50`, routeID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.RouteSchedule
	for rows.Next() {
		var s models.RouteSchedule
		if err := rows.Scan(&s.ID,&s.RouteID,&s.VehicleID,&s.DriverID,&s.DepartureTime,&s.ArrivalTime,&s.Status,&s.ActualDeptAt,&s.ActualArrAt,&s.Notes,&s.CreatedAt,&s.UpdatedAt); err != nil { return nil, err }
		result = append(result, s)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status string, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE route_schedules SET status=$1,updated_at=$2 WHERE id=$3`, status,now,id)
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.RouteAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO route_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}

type scannable interface{ Scan(dest ...any) error }

func scanRoute(row scannable) (*models.Route, error) {
	var r models.Route
	var wpJSON []byte
	err := row.Scan(&r.ID,&r.Name,&r.RouteCode,&r.RouteType,&r.Status,&r.Country,&r.OriginName,&r.OriginLat,&r.OriginLng,&r.DestName,&r.DestLat,&r.DestLng,&r.DistanceKm,&r.EstHours,&wpJSON,&r.FreightClass,&r.Notes,&r.CreatedAt,&r.UpdatedAt)
	if err == nil { json.Unmarshal(wpJSON, &r.Waypoints) }
	return &r, err
}
