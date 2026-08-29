package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/voyage-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateVoyage(ctx context.Context, v models.Voyage) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO voyages (id,voyage_ref,vessel_id,origin_port,destination_port,cargo_type,cargo_tons,status,departure_at,est_arrival_at,distance_nm,fuel_tons,tenant_id,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		v.ID, v.VoyageRef, v.VesselID, v.OriginPort, v.DestinationPort, v.CargoType,
		v.CargoTons, v.Status, v.DepartureAt, v.EstArrivalAt, v.DistanceNM,
		v.FuelTons, v.TenantID, v.CreatedAt, v.UpdatedAt)
	return err
}

func (q *Queries) GetVoyage(ctx context.Context, id uuid.UUID) (*models.Voyage, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id,voyage_ref,vessel_id,origin_port,destination_port,cargo_type,cargo_tons,status,departure_at,est_arrival_at,actual_arrival_at,distance_nm,fuel_tons,tenant_id,created_at,updated_at
		 FROM voyages WHERE id=$1`, id)
	var v models.Voyage
	err := row.Scan(&v.ID, &v.VoyageRef, &v.VesselID, &v.OriginPort, &v.DestinationPort,
		&v.CargoType, &v.CargoTons, &v.Status, &v.DepartureAt, &v.EstArrivalAt,
		&v.ActualArrivalAt, &v.DistanceNM, &v.FuelTons, &v.TenantID, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (q *Queries) ListByVessel(ctx context.Context, vesselID uuid.UUID, limit int) ([]models.Voyage, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,voyage_ref,vessel_id,origin_port,destination_port,cargo_type,cargo_tons,status,departure_at,est_arrival_at,actual_arrival_at,distance_nm,fuel_tons,tenant_id,created_at,updated_at
		 FROM voyages WHERE vessel_id=$1 ORDER BY created_at DESC LIMIT $2`, vesselID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Voyage
	for rows.Next() {
		var v models.Voyage
		if err := rows.Scan(&v.ID, &v.VoyageRef, &v.VesselID, &v.OriginPort, &v.DestinationPort,
			&v.CargoType, &v.CargoTons, &v.Status, &v.DepartureAt, &v.EstArrivalAt,
			&v.ActualArrivalAt, &v.DistanceNM, &v.FuelTons, &v.TenantID, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (q *Queries) UpdateStatus(ctx context.Context, id uuid.UUID, status models.VoyageStatus, arrivalAt *time.Time, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE voyages SET status=$1,
		  departure_at     = CASE WHEN $1='departed'  THEN COALESCE(departure_at,$3)        ELSE departure_at     END,
		  actual_arrival_at= CASE WHEN $1='arrived'   THEN COALESCE(actual_arrival_at,$2)   ELSE actual_arrival_at END,
		  updated_at=$3 WHERE id=$4`,
		status, arrivalAt, now, id)
	return err
}

func (q *Queries) LogEvent(ctx context.Context, e models.VoyageEvent) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO voyage_events (id,voyage_id,event_type,description,latitude,longitude,occurred_at,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		e.ID, e.VoyageID, e.EventType, e.Description, e.Latitude, e.Longitude, e.OccurredAt, time.Now().UTC())
	return err
}

func (q *Queries) ListEvents(ctx context.Context, voyageID uuid.UUID) ([]models.VoyageEvent, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,voyage_id,event_type,description,latitude,longitude,occurred_at
		 FROM voyage_events WHERE voyage_id=$1 ORDER BY occurred_at`, voyageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.VoyageEvent
	for rows.Next() {
		var e models.VoyageEvent
		if err := rows.Scan(&e.ID, &e.VoyageID, &e.EventType, &e.Description,
			&e.Latitude, &e.Longitude, &e.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}
