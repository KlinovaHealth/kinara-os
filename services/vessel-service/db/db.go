package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/vessel-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RegisterVessel(ctx context.Context, v models.Vessel) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO vessels (id,imo_number,name,vessel_type,flag,owner,operator_id,year_built,gross_tonnage_t,deadweight_t,length_m,beam_m,max_draft_m,max_speed_knots,condition,is_active,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		v.ID, v.IMONumber, v.Name, v.VesselType, v.Flag, v.Owner, v.OperatorID, v.YearBuilt,
		v.GrossTonnage, v.DeadweightT, v.LengthM, v.BeamM, v.MaxDraftM, v.MaxSpeed,
		v.Condition, v.IsActive, v.CreatedAt, v.UpdatedAt)
	return err
}

func (q *Queries) GetVessel(ctx context.Context, id uuid.UUID) (*models.Vessel, error) {
	v := &models.Vessel{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,imo_number,name,vessel_type,flag,owner,operator_id,year_built,gross_tonnage_t,deadweight_t,length_m,beam_m,max_draft_m,max_speed_knots,condition,current_port_id,is_active,created_at,updated_at
         FROM vessels WHERE id=$1`, id).
		Scan(&v.ID, &v.IMONumber, &v.Name, &v.VesselType, &v.Flag, &v.Owner, &v.OperatorID, &v.YearBuilt,
			&v.GrossTonnage, &v.DeadweightT, &v.LengthM, &v.BeamM, &v.MaxDraftM, &v.MaxSpeed,
			&v.Condition, &v.CurrentPortID, &v.IsActive, &v.CreatedAt, &v.UpdatedAt)
	if err != nil { return nil, err }
	return v, nil
}

func (q *Queries) ListVessels(ctx context.Context, flag *string, activeOnly bool) ([]models.Vessel, error) {
	query := `SELECT id,imo_number,name,vessel_type,flag,owner,operator_id,year_built,gross_tonnage_t,deadweight_t,length_m,beam_m,max_draft_m,max_speed_knots,condition,current_port_id,is_active,created_at,updated_at FROM vessels WHERE 1=1`
	args := []interface{}{}
	i := 1
	if flag != nil { query += fmt.Sprintf(" AND flag=$%d", i); args = append(args, *flag); i++ }
	if activeOnly { query += fmt.Sprintf(" AND is_active=$%d", i); args = append(args, true); i++ }
	query += " ORDER BY name ASC"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var vessels []models.Vessel
	for rows.Next() {
		var v models.Vessel
		if err := rows.Scan(&v.ID, &v.IMONumber, &v.Name, &v.VesselType, &v.Flag, &v.Owner, &v.OperatorID, &v.YearBuilt,
			&v.GrossTonnage, &v.DeadweightT, &v.LengthM, &v.BeamM, &v.MaxDraftM, &v.MaxSpeed,
			&v.Condition, &v.CurrentPortID, &v.IsActive, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		vessels = append(vessels, v)
	}
	return vessels, nil
}

func (q *Queries) UpdateVesselCondition(ctx context.Context, id uuid.UUID, condition models.VesselCondition, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE vessels SET condition=$1,updated_at=$2 WHERE id=$3`, condition, now, id)
	return err
}

func (q *Queries) LogVoyage(ctx context.Context, v models.VoyageRecord) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO voyage_records (id,vessel_id,voyage_code,departure_port_id,arrival_port_id,departed_at,arrived_at,distance_nm,cargo_tonnage_t,notes,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		v.ID, v.VesselID, v.VoyageCode, v.DeparturePortID, v.ArrivalPortID,
		v.DepartedAt, v.ArrivedAt, v.DistanceNM, v.CargoTonnage, v.Notes, v.CreatedAt)
	return err
}

func (q *Queries) GetVoyage(ctx context.Context, id uuid.UUID) (*models.VoyageRecord, error) {
	v := &models.VoyageRecord{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,vessel_id,voyage_code,departure_port_id,arrival_port_id,departed_at,arrived_at,distance_nm,cargo_tonnage_t,notes,created_at
         FROM voyage_records WHERE id=$1`, id).
		Scan(&v.ID, &v.VesselID, &v.VoyageCode, &v.DeparturePortID, &v.ArrivalPortID,
			&v.DepartedAt, &v.ArrivedAt, &v.DistanceNM, &v.CargoTonnage, &v.Notes, &v.CreatedAt)
	if err != nil { return nil, err }
	return v, nil
}

func (q *Queries) ListVoyages(ctx context.Context, vesselID uuid.UUID) ([]models.VoyageRecord, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,vessel_id,voyage_code,departure_port_id,arrival_port_id,departed_at,arrived_at,distance_nm,cargo_tonnage_t,notes,created_at
         FROM voyage_records WHERE vessel_id=$1 ORDER BY created_at DESC LIMIT 100`, vesselID)
	if err != nil { return nil, err }
	defer rows.Close()
	var voyages []models.VoyageRecord
	for rows.Next() {
		var v models.VoyageRecord
		if err := rows.Scan(&v.ID, &v.VesselID, &v.VoyageCode, &v.DeparturePortID, &v.ArrivalPortID,
			&v.DepartedAt, &v.ArrivedAt, &v.DistanceNM, &v.CargoTonnage, &v.Notes, &v.CreatedAt); err != nil {
			return nil, err
		}
		voyages = append(voyages, v)
	}
	return voyages, nil
}

func (q *Queries) LogMaintenance(ctx context.Context, m models.MaintenanceRecord) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO maintenance_records (id,vessel_id,maintenance_type,description,start_date,cost,currency,vendor,completed,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		m.ID, m.VesselID, m.MaintenanceType, m.Description, m.StartDate,
		m.Cost, m.Currency, m.Vendor, m.Completed, m.CreatedAt)
	return err
}

func (q *Queries) ListMaintenance(ctx context.Context, vesselID uuid.UUID) ([]models.MaintenanceRecord, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,vessel_id,maintenance_type,description,start_date,end_date,cost,currency,vendor,completed,created_at
         FROM maintenance_records WHERE vessel_id=$1 ORDER BY start_date DESC`, vesselID)
	if err != nil { return nil, err }
	defer rows.Close()
	var records []models.MaintenanceRecord
	for rows.Next() {
		var m models.MaintenanceRecord
		if err := rows.Scan(&m.ID, &m.VesselID, &m.MaintenanceType, &m.Description, &m.StartDate,
			&m.EndDate, &m.Cost, &m.Currency, &m.Vendor, &m.Completed, &m.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, m)
	}
	return records, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.VesselAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO vessel_audit_log (id,vessel_id,actor_id,action,created_at) VALUES ($1,$2,$3,$4,$5)`,
		l.ID, l.VesselID, l.ActorID, l.Action, l.CreatedAt)
	return err
}
