package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/fleet-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateVehicle(ctx context.Context, v models.Vehicle) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO vehicles
		(id,registration_no,vehicle_type,make,model,year,fuel_type,payload_capacity_kg,volume_capacity_m3,
		 status,country,base_location,current_odometer_km,last_service_km,next_service_km,
		 insurance_expiry,inspection_expiry,assigned_driver_id,notes,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		v.ID, v.RegistrationNo, v.VehicleType, v.Make, v.Model, v.Year, v.FuelType,
		v.PayloadCapacityKg, v.VolumeCapacityM3, v.Status, v.Country, v.BaseLocation,
		v.CurrentOdometerKm, v.LastServiceKm, v.NextServiceKm,
		v.InsuranceExpiry, v.InspectionExpiry, v.AssignedDriverID, v.Notes, v.CreatedAt, v.UpdatedAt)
	return err
}

func (q *Queries) GetVehicle(ctx context.Context, id uuid.UUID) (*models.Vehicle, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,registration_no,vehicle_type,make,model,year,fuel_type,
		payload_capacity_kg,volume_capacity_m3,status,country,base_location,
		current_odometer_km,last_service_km,next_service_km,
		insurance_expiry,inspection_expiry,assigned_driver_id,notes,created_at,updated_at
		FROM vehicles WHERE id=$1`, id)
	var v models.Vehicle
	err := row.Scan(&v.ID,&v.RegistrationNo,&v.VehicleType,&v.Make,&v.Model,&v.Year,&v.FuelType,
		&v.PayloadCapacityKg,&v.VolumeCapacityM3,&v.Status,&v.Country,&v.BaseLocation,
		&v.CurrentOdometerKm,&v.LastServiceKm,&v.NextServiceKm,
		&v.InsuranceExpiry,&v.InspectionExpiry,&v.AssignedDriverID,&v.Notes,&v.CreatedAt,&v.UpdatedAt)
	return &v, err
}

type ListVehiclesParams struct{ Country *string; Status *models.VehicleStatus; Page, Limit int }

func (q *Queries) ListVehicles(ctx context.Context, p ListVehiclesParams) ([]models.Vehicle, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.Country != nil { where += fmt.Sprintf(" AND country=$%d",n); args=append(args,*p.Country); n++ }
	if p.Status  != nil { where += fmt.Sprintf(" AND status=$%d",n);  args=append(args,*p.Status);  n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT id,registration_no,vehicle_type,make,model,year,fuel_type,
		payload_capacity_kg,volume_capacity_m3,status,country,base_location,
		current_odometer_km,last_service_km,next_service_km,
		insurance_expiry,inspection_expiry,assigned_driver_id,notes,created_at,updated_at
		FROM vehicles %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.Vehicle
	for rows.Next() {
		var v models.Vehicle
		if err := rows.Scan(&v.ID,&v.RegistrationNo,&v.VehicleType,&v.Make,&v.Model,&v.Year,&v.FuelType,
			&v.PayloadCapacityKg,&v.VolumeCapacityM3,&v.Status,&v.Country,&v.BaseLocation,
			&v.CurrentOdometerKm,&v.LastServiceKm,&v.NextServiceKm,
			&v.InsuranceExpiry,&v.InspectionExpiry,&v.AssignedDriverID,&v.Notes,&v.CreatedAt,&v.UpdatedAt); err != nil { return nil, err }
		result = append(result, v)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateVehicle(ctx context.Context, id uuid.UUID, req models.UpdateVehicleRequest, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE vehicles SET
		status=COALESCE($1,status), notes=COALESCE($2,notes), updated_at=$3 WHERE id=$4`,
		req.Status, req.Notes, now, id)
	return err
}

func (q *Queries) LogMaintenance(ctx context.Context, m models.MaintenanceRecord) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO maintenance_records
		(id,vehicle_id,service_type,description,odometer_km,cost,currency,serviced_by,serviced_at,next_service_km,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		m.ID,m.VehicleID,m.ServiceType,m.Description,m.OdometerKm,m.Cost,m.Currency,m.ServicedBy,m.ServicedAt,m.NextServiceKm,m.CreatedAt)
	return err
}

func (q *Queries) ListMaintenance(ctx context.Context, vehicleID uuid.UUID) ([]models.MaintenanceRecord, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,vehicle_id,service_type,description,odometer_km,cost,currency,serviced_by,serviced_at,next_service_km,created_at FROM maintenance_records WHERE vehicle_id=$1 ORDER BY serviced_at DESC`, vehicleID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.MaintenanceRecord
	for rows.Next() {
		var m models.MaintenanceRecord
		if err := rows.Scan(&m.ID,&m.VehicleID,&m.ServiceType,&m.Description,&m.OdometerKm,&m.Cost,&m.Currency,&m.ServicedBy,&m.ServicedAt,&m.NextServiceKm,&m.CreatedAt); err != nil { return nil, err }
		result = append(result, m)
	}
	return result, rows.Err()
}

func (q *Queries) LogFuel(ctx context.Context, f models.FuelLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO fuel_logs
		(id,vehicle_id,driver_id,litres_filled,cost_per_litre,total_cost,currency,odometer_km,station,filled_at,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		f.ID,f.VehicleID,f.DriverID,f.LitresFilled,f.CostPerLitre,f.TotalCost,f.Currency,f.OdometerKm,f.Station,f.FilledAt,f.CreatedAt)
	return err
}

func (q *Queries) ListFuelLogs(ctx context.Context, vehicleID uuid.UUID) ([]models.FuelLog, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,vehicle_id,driver_id,litres_filled,cost_per_litre,total_cost,currency,odometer_km,station,filled_at,created_at FROM fuel_logs WHERE vehicle_id=$1 ORDER BY filled_at DESC LIMIT 100`, vehicleID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.FuelLog
	for rows.Next() {
		var f models.FuelLog
		if err := rows.Scan(&f.ID,&f.VehicleID,&f.DriverID,&f.LitresFilled,&f.CostPerLitre,&f.TotalCost,&f.Currency,&f.OdometerKm,&f.Station,&f.FilledAt,&f.CreatedAt); err != nil { return nil, err }
		result = append(result, f)
	}
	return result, rows.Err()
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.FleetAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO fleet_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}
