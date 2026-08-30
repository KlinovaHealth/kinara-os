package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/vehicle-tracking-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

// InsertPing records a GPS location ping for a vehicle.
func (q *Queries) InsertPing(ctx context.Context, loc models.GPSLocation) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO gps_locations (id, vehicle_id, latitude, longitude, speed_kmh, heading_deg, pinged_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		loc.ID, loc.VehicleID, loc.Latitude, loc.Longitude, loc.SpeedKmh, loc.HeadingDeg, loc.PingedAt)
	return err
}

// GetLatestLocation returns the most recent GPS ping for a vehicle.
func (q *Queries) GetLatestLocation(ctx context.Context, vehicleID uuid.UUID) (*models.GPSLocation, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id, vehicle_id, latitude, longitude, speed_kmh, heading_deg, pinged_at
		 FROM gps_locations
		 WHERE vehicle_id=$1
		 ORDER BY pinged_at DESC
		 LIMIT 1`,
		vehicleID)
	var loc models.GPSLocation
	err := row.Scan(&loc.ID, &loc.VehicleID, &loc.Latitude, &loc.Longitude, &loc.SpeedKmh, &loc.HeadingDeg, &loc.PingedAt)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

// GetActiveRoute returns the current active route for a vehicle.
func (q *Queries) GetActiveRoute(ctx context.Context, vehicleID uuid.UUID) (*models.VehicleRoute, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id, vehicle_id, origin_lat, origin_lng, dest_lat, dest_lng,
		        COALESCE(description,''), active, assigned_at, eta
		 FROM vehicle_routes
		 WHERE vehicle_id=$1 AND active=true
		 ORDER BY assigned_at DESC
		 LIMIT 1`,
		vehicleID)
	var vr models.VehicleRoute
	err := row.Scan(
		&vr.ID, &vr.VehicleID, &vr.OriginLat, &vr.OriginLng, &vr.DestLat, &vr.DestLng,
		&vr.Description, &vr.Active, &vr.AssignedAt, &vr.ETA)
	if err != nil {
		return nil, err
	}
	return &vr, nil
}

// GetFleetStatus returns current status of all vehicles in a tenant.
func (q *Queries) GetFleetStatus(ctx context.Context, tenantID string) ([]models.FleetVehicleStatus, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT v.id, v.vehicle_ref, v.vehicle_type, v.status, COALESCE(v.driver_name,''),
		        g.pinged_at, g.latitude, g.longitude
		 FROM vehicles v
		 LEFT JOIN LATERAL (
		     SELECT pinged_at, latitude, longitude
		     FROM gps_locations
		     WHERE vehicle_id = v.id
		     ORDER BY pinged_at DESC
		     LIMIT 1
		 ) g ON true
		 WHERE v.tenant_id=$1
		 ORDER BY v.vehicle_ref`,
		tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.FleetVehicleStatus
	for rows.Next() {
		var fs models.FleetVehicleStatus
		if err := rows.Scan(
			&fs.VehicleID, &fs.VehicleRef, &fs.VehicleType, &fs.Status, &fs.DriverName,
			&fs.LastPing, &fs.Latitude, &fs.Longitude,
		); err != nil {
			return nil, err
		}
		out = append(out, fs)
	}
	return out, rows.Err()
}

// InsertAlert records a vehicle alert.
func (q *Queries) InsertAlert(ctx context.Context, a models.VehicleAlert) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO vehicle_alerts (id, vehicle_id, alert_type, message, created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		a.ID, a.VehicleID, a.AlertType, a.Message, a.CreatedAt)
	return err
}

// GetVehicle fetches a vehicle record by ID.
func (q *Queries) GetVehicle(ctx context.Context, id uuid.UUID) (*models.Vehicle, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id, vehicle_ref, fleet_id, vehicle_type, COALESCE(capacity_kg,0),
		        COALESCE(driver_name,''), driver_id, status, tenant_id, registered_at
		 FROM vehicles WHERE id=$1`,
		id)
	var v models.Vehicle
	err := row.Scan(
		&v.ID, &v.VehicleRef, &v.FleetID, &v.VehicleType, &v.Capacity,
		&v.DriverName, &v.DriverID, &v.Status, &v.TenantID, &v.RegisteredAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
