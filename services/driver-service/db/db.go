package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/driver-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

const driverCols = `id,full_name_enc,phone_enc,national_id_enc,license_no,license_class,license_expiry,
	status,country,base_location,total_trips,total_km,rating,assigned_vehicle_id,created_at,updated_at`

func (q *Queries) CreateDriver(ctx context.Context, d models.DriverRow) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO drivers(`+driverCols+`) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		d.ID,d.FullNameEnc,d.PhoneEnc,d.NationalIDEnc,d.LicenseNo,d.LicenseClass,d.LicenseExpiry,
		d.Status,d.Country,d.BaseLocation,d.TotalTrips,d.TotalKm,d.Rating,d.AssignedVehicleID,d.CreatedAt,d.UpdatedAt)
	return err
}

func (q *Queries) GetDriver(ctx context.Context, id uuid.UUID) (*models.DriverRow, error) {
	row := q.pool.QueryRow(ctx, `SELECT `+driverCols+` FROM drivers WHERE id=$1`, id)
	var d models.DriverRow
	err := row.Scan(&d.ID,&d.FullNameEnc,&d.PhoneEnc,&d.NationalIDEnc,&d.LicenseNo,&d.LicenseClass,&d.LicenseExpiry,
		&d.Status,&d.Country,&d.BaseLocation,&d.TotalTrips,&d.TotalKm,&d.Rating,&d.AssignedVehicleID,&d.CreatedAt,&d.UpdatedAt)
	return &d, err
}

type ListDriversParams struct{ Country *string; Status *models.DriverStatus; Page, Limit int }

func (q *Queries) ListDrivers(ctx context.Context, p ListDriversParams) ([]models.DriverRow, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.Country != nil { where += fmt.Sprintf(" AND country=$%d",n); args=append(args,*p.Country); n++ }
	if p.Status  != nil { where += fmt.Sprintf(" AND status=$%d",n);  args=append(args,*p.Status);  n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT `+driverCols+` FROM drivers %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.DriverRow
	for rows.Next() {
		var d models.DriverRow
		if err := rows.Scan(&d.ID,&d.FullNameEnc,&d.PhoneEnc,&d.NationalIDEnc,&d.LicenseNo,&d.LicenseClass,&d.LicenseExpiry,
			&d.Status,&d.Country,&d.BaseLocation,&d.TotalTrips,&d.TotalKm,&d.Rating,&d.AssignedVehicleID,&d.CreatedAt,&d.UpdatedAt); err != nil { return nil, err }
		result = append(result, d)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateDriver(ctx context.Context, id uuid.UUID, req models.UpdateDriverRequest, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE drivers SET status=COALESCE($1,status),base_location=COALESCE($2,base_location),updated_at=$3 WHERE id=$4`,
		req.Status, req.BaseLocation, now, id)
	return err
}

func (q *Queries) LogTrip(ctx context.Context, t models.DriverTrip) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO driver_trips(id,driver_id,vehicle_id,route_id,distance_km,start_time,status,notes,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		t.ID,t.DriverID,t.VehicleID,t.RouteID,t.DistanceKm,t.StartTime,t.Status,t.Notes,t.CreatedAt)
	return err
}

func (q *Queries) ListTrips(ctx context.Context, driverID uuid.UUID) ([]models.DriverTrip, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,driver_id,vehicle_id,route_id,distance_km,start_time,end_time,status,notes,created_at FROM driver_trips WHERE driver_id=$1 ORDER BY start_time DESC LIMIT 100`, driverID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.DriverTrip
	for rows.Next() {
		var t models.DriverTrip
		if err := rows.Scan(&t.ID,&t.DriverID,&t.VehicleID,&t.RouteID,&t.DistanceKm,&t.StartTime,&t.EndTime,&t.Status,&t.Notes,&t.CreatedAt); err != nil { return nil, err }
		result = append(result, t)
	}
	return result, rows.Err()
}

func (q *Queries) IncrementTripStats(ctx context.Context, id uuid.UUID, km float64, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE drivers SET total_trips=total_trips+1, total_km=total_km+$1, updated_at=$2 WHERE id=$3`, km, now, id)
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.DriverAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO driver_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}
