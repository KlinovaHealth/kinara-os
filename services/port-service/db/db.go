package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/port-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

type ListBerthsParams struct {
	PortID *uuid.UUID
	Status *models.BerthStatus
}

type ListSchedulesParams struct {
	BerthID  *uuid.UUID
	VesselID *uuid.UUID
	Status   *models.VesselStatus
	Page, Limit int
}

func (q *Queries) CreatePort(ctx context.Context, p models.Port) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO ports (id,name,code,country,city,latitude,longitude,max_draft_m,total_berths,alert_level,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		p.ID, p.Name, p.Code, p.Country, p.City, p.Latitude, p.Longitude,
		p.MaxDraft, p.TotalBerths, p.AlertLevel, p.CreatedAt, p.UpdatedAt)
	return err
}

func (q *Queries) GetPort(ctx context.Context, id uuid.UUID) (*models.Port, error) {
	p := &models.Port{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,name,code,country,city,latitude,longitude,max_draft_m,total_berths,alert_level,created_at,updated_at
         FROM ports WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.Code, &p.Country, &p.City, &p.Latitude, &p.Longitude,
			&p.MaxDraft, &p.TotalBerths, &p.AlertLevel, &p.CreatedAt, &p.UpdatedAt)
	if err != nil { return nil, err }
	return p, nil
}

func (q *Queries) ListPorts(ctx context.Context, country *string) ([]models.Port, error) {
	query := `SELECT id,name,code,country,city,latitude,longitude,max_draft_m,total_berths,alert_level,created_at,updated_at FROM ports`
	args := []interface{}{}
	if country != nil {
		query += " WHERE country=$1"
		args = append(args, *country)
	}
	query += " ORDER BY name ASC"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var ports []models.Port
	for rows.Next() {
		var p models.Port
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.Country, &p.City, &p.Latitude, &p.Longitude,
			&p.MaxDraft, &p.TotalBerths, &p.AlertLevel, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}
	return ports, nil
}

func (q *Queries) CreateBerth(ctx context.Context, b models.Berth) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO berths (id,port_id,berth_number,status,max_length_m,max_draft_m,max_tonnage_t,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		b.ID, b.PortID, b.BerthNumber, b.Status, b.MaxLengthM, b.MaxDraftM, b.MaxTonnage, b.CreatedAt, b.UpdatedAt)
	return err
}

func (q *Queries) GetBerth(ctx context.Context, id uuid.UUID) (*models.Berth, error) {
	b := &models.Berth{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,port_id,berth_number,status,max_length_m,max_draft_m,max_tonnage_t,created_at,updated_at
         FROM berths WHERE id=$1`, id).
		Scan(&b.ID, &b.PortID, &b.BerthNumber, &b.Status, &b.MaxLengthM, &b.MaxDraftM, &b.MaxTonnage, &b.CreatedAt, &b.UpdatedAt)
	if err != nil { return nil, err }
	return b, nil
}

func (q *Queries) ListBerths(ctx context.Context, p ListBerthsParams) ([]models.Berth, error) {
	wheres := []string{}
	args := []interface{}{}
	i := 1
	if p.PortID != nil { wheres = append(wheres, fmt.Sprintf("port_id=$%d", i)); args = append(args, *p.PortID); i++ }
	if p.Status != nil { wheres = append(wheres, fmt.Sprintf("status=$%d", i)); args = append(args, *p.Status); i++ }
	query := "SELECT id,port_id,berth_number,status,max_length_m,max_draft_m,max_tonnage_t,created_at,updated_at FROM berths"
	if len(wheres) > 0 { query += " WHERE " + strings.Join(wheres, " AND ") }
	query += " ORDER BY berth_number ASC"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var berths []models.Berth
	for rows.Next() {
		var b models.Berth
		if err := rows.Scan(&b.ID, &b.PortID, &b.BerthNumber, &b.Status, &b.MaxLengthM, &b.MaxDraftM, &b.MaxTonnage, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		berths = append(berths, b)
	}
	return berths, nil
}

func (q *Queries) UpdateBerthStatus(ctx context.Context, id uuid.UUID, status models.BerthStatus, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE berths SET status=$1,updated_at=$2 WHERE id=$3`, status, now, id)
	return err
}

func (q *Queries) CreateSchedule(ctx context.Context, s models.BerthSchedule) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO berth_schedules (id,berth_id,vessel_id,vessel_name,status,eta,etd,cargo_type,tonnage_t,notes,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.ID, s.BerthID, s.VesselID, s.VesselName, s.Status, s.ETA, s.ETD,
		s.CargoType, s.TonnageT, s.Notes, s.CreatedAt, s.UpdatedAt)
	return err
}

func (q *Queries) GetSchedule(ctx context.Context, id uuid.UUID) (*models.BerthSchedule, error) {
	s := &models.BerthSchedule{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,berth_id,vessel_id,vessel_name,status,eta,etd,actual_arrival,actual_departure,cargo_type,tonnage_t,notes,created_at,updated_at
         FROM berth_schedules WHERE id=$1`, id).
		Scan(&s.ID, &s.BerthID, &s.VesselID, &s.VesselName, &s.Status, &s.ETA, &s.ETD,
			&s.ActualArrival, &s.ActualDeparture, &s.CargoType, &s.TonnageT, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err != nil { return nil, err }
	return s, nil
}

func (q *Queries) ListSchedulesByBerth(ctx context.Context, berthID uuid.UUID) ([]models.BerthSchedule, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,berth_id,vessel_id,vessel_name,status,eta,etd,actual_arrival,actual_departure,cargo_type,tonnage_t,notes,created_at,updated_at
         FROM berth_schedules WHERE berth_id=$1 ORDER BY eta ASC`, berthID)
	if err != nil { return nil, err }
	defer rows.Close()
	var schedules []models.BerthSchedule
	for rows.Next() {
		var s models.BerthSchedule
		if err := rows.Scan(&s.ID, &s.BerthID, &s.VesselID, &s.VesselName, &s.Status, &s.ETA, &s.ETD,
			&s.ActualArrival, &s.ActualDeparture, &s.CargoType, &s.TonnageT, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (q *Queries) UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status models.VesselStatus, now time.Time) error {
	query := `UPDATE berth_schedules SET status=$1,updated_at=$2`
	args := []interface{}{status, now}
	if status == models.VesselArrived {
		query += ",actual_arrival=COALESCE(actual_arrival,$3)"
		args = append(args, now)
	} else if status == models.VesselDeparted {
		query += ",actual_departure=COALESCE(actual_departure,$3)"
		args = append(args, now)
	}
	args = append(args, id)
	query += fmt.Sprintf(" WHERE id=$%d", len(args))
	_, err := q.pool.Exec(ctx, query, args...)
	return err
}

func (q *Queries) CreateCongestionAlert(ctx context.Context, a models.CongestionAlert) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO congestion_alerts (id,port_id,alert_level,message,occupied_berths,total_berths,occupancy_pct,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		a.ID, a.PortID, a.AlertLevel, a.Message, a.OccupiedBerths, a.TotalBerths, a.OccupancyPct, a.CreatedAt)
	return err
}

func (q *Queries) ListAlerts(ctx context.Context, portID uuid.UUID) ([]models.CongestionAlert, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,port_id,alert_level,message,occupied_berths,total_berths,occupancy_pct,resolved_at,created_at
         FROM congestion_alerts WHERE port_id=$1 ORDER BY created_at DESC LIMIT 50`, portID)
	if err != nil { return nil, err }
	defer rows.Close()
	var alerts []models.CongestionAlert
	for rows.Next() {
		var a models.CongestionAlert
		if err := rows.Scan(&a.ID, &a.PortID, &a.AlertLevel, &a.Message, &a.OccupiedBerths, &a.TotalBerths, &a.OccupancyPct, &a.ResolvedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.PortAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO port_audit_log (id,port_id,actor_id,action,entity_type,entity_id,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		l.ID, l.PortID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
