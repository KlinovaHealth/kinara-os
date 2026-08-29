package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/dock-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

type ListOperationsParams struct {
	PortID   *uuid.UUID
	VesselID *uuid.UUID
	Page, Limit int
}

func (q *Queries) CreateOperation(ctx context.Context, op models.DockOperation) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO dock_operations (id,port_id,berth_id,vessel_id,operation_type,cargo_type,tonnage_t,unit_count,stevedore_team,planned_duration_hrs,billing_amount,currency,safety_incident,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		op.ID, op.PortID, op.BerthID, op.VesselID, op.OperationType, op.CargoType,
		op.TonnageT, op.UnitCount, op.StevedoreTeam, op.PlannedDuration, op.BillingAmount,
		op.Currency, op.SafetyIncident, op.CreatedAt, op.UpdatedAt)
	return err
}

func (q *Queries) GetOperation(ctx context.Context, id uuid.UUID) (*models.DockOperation, error) {
	op := &models.DockOperation{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,port_id,berth_id,vessel_id,operation_type,cargo_type,tonnage_t,unit_count,stevedore_team,started_at,completed_at,planned_duration_hrs,actual_duration_hrs,safety_incident,incident_details,billing_amount,currency,created_at,updated_at
         FROM dock_operations WHERE id=$1`, id).
		Scan(&op.ID, &op.PortID, &op.BerthID, &op.VesselID, &op.OperationType, &op.CargoType,
			&op.TonnageT, &op.UnitCount, &op.StevedoreTeam, &op.StartedAt, &op.CompletedAt,
			&op.PlannedDuration, &op.ActualDuration, &op.SafetyIncident, &op.IncidentDetails,
			&op.BillingAmount, &op.Currency, &op.CreatedAt, &op.UpdatedAt)
	if err != nil { return nil, err }
	return op, nil
}

func (q *Queries) ListOperations(ctx context.Context, p ListOperationsParams) ([]models.DockOperation, error) {
	wheres := []string{}
	args := []interface{}{}
	i := 1
	if p.PortID != nil { wheres = append(wheres, fmt.Sprintf("port_id=$%d", i)); args = append(args, *p.PortID); i++ }
	if p.VesselID != nil { wheres = append(wheres, fmt.Sprintf("vessel_id=$%d", i)); args = append(args, *p.VesselID); i++ }
	query := "SELECT id,port_id,berth_id,vessel_id,operation_type,cargo_type,tonnage_t,unit_count,stevedore_team,started_at,completed_at,planned_duration_hrs,actual_duration_hrs,safety_incident,incident_details,billing_amount,currency,created_at,updated_at FROM dock_operations"
	if len(wheres) > 0 { query += " WHERE " + strings.Join(wheres, " AND ") }
	offset := (p.Page - 1) * p.Limit
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var ops []models.DockOperation
	for rows.Next() {
		var op models.DockOperation
		if err := rows.Scan(&op.ID, &op.PortID, &op.BerthID, &op.VesselID, &op.OperationType, &op.CargoType,
			&op.TonnageT, &op.UnitCount, &op.StevedoreTeam, &op.StartedAt, &op.CompletedAt,
			&op.PlannedDuration, &op.ActualDuration, &op.SafetyIncident, &op.IncidentDetails,
			&op.BillingAmount, &op.Currency, &op.CreatedAt, &op.UpdatedAt); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func (q *Queries) StartOperation(ctx context.Context, id uuid.UUID, startedAt time.Time, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE dock_operations SET started_at=COALESCE(started_at,$1),updated_at=$2 WHERE id=$3`, startedAt, now, id)
	return err
}

func (q *Queries) CompleteOperation(ctx context.Context, id uuid.UUID, completedAt time.Time, duration float64, safetyIncident bool, details string, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE dock_operations SET completed_at=COALESCE(completed_at,$1),actual_duration_hrs=$2,safety_incident=$3,incident_details=$4,updated_at=$5 WHERE id=$6`,
		completedAt, duration, safetyIncident, details, now, id)
	return err
}

func (q *Queries) CreateEquipment(ctx context.Context, e models.Equipment) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO dock_equipment (id,port_id,equipment_code,equipment_type,model,status,capacity_t,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.ID, e.PortID, e.EquipmentCode, e.EquipmentType, e.Model, e.Status, e.CapacityT, e.CreatedAt, e.UpdatedAt)
	return err
}

func (q *Queries) GetEquipment(ctx context.Context, id uuid.UUID) (*models.Equipment, error) {
	e := &models.Equipment{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,port_id,equipment_code,equipment_type,model,status,capacity_t,last_service_at,next_service_at,created_at,updated_at
         FROM dock_equipment WHERE id=$1`, id).
		Scan(&e.ID, &e.PortID, &e.EquipmentCode, &e.EquipmentType, &e.Model, &e.Status,
			&e.CapacityT, &e.LastServiceAt, &e.NextServiceAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil { return nil, err }
	return e, nil
}

func (q *Queries) ListEquipment(ctx context.Context, portID uuid.UUID) ([]models.Equipment, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,port_id,equipment_code,equipment_type,model,status,capacity_t,last_service_at,next_service_at,created_at,updated_at
         FROM dock_equipment WHERE port_id=$1 ORDER BY equipment_code ASC`, portID)
	if err != nil { return nil, err }
	defer rows.Close()
	var equip []models.Equipment
	for rows.Next() {
		var e models.Equipment
		if err := rows.Scan(&e.ID, &e.PortID, &e.EquipmentCode, &e.EquipmentType, &e.Model, &e.Status,
			&e.CapacityT, &e.LastServiceAt, &e.NextServiceAt, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		equip = append(equip, e)
	}
	return equip, nil
}

func (q *Queries) UpdateEquipmentStatus(ctx context.Context, id uuid.UUID, status models.EquipmentStatus, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE dock_equipment SET status=$1,updated_at=$2 WHERE id=$3`, status, now, id)
	return err
}

func (q *Queries) ReportSafetyEvent(ctx context.Context, e models.SafetyEvent) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO safety_events (id,operation_id,port_id,event_type,severity,description,injured_count,reported_by,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.ID, e.OperationID, e.PortID, e.EventType, e.Severity, e.Description, e.Injured, e.ReportedBy, e.CreatedAt)
	return err
}

func (q *Queries) ListSafetyEvents(ctx context.Context, portID uuid.UUID) ([]models.SafetyEvent, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,operation_id,port_id,event_type,severity,description,injured_count,reported_by,created_at
         FROM safety_events WHERE port_id=$1 ORDER BY created_at DESC LIMIT 100`, portID)
	if err != nil { return nil, err }
	defer rows.Close()
	var events []models.SafetyEvent
	for rows.Next() {
		var e models.SafetyEvent
		if err := rows.Scan(&e.ID, &e.OperationID, &e.PortID, &e.EventType, &e.Severity, &e.Description, &e.Injured, &e.ReportedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.DockAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO dock_audit_log (id,port_id,actor_id,action,entity_type,entity_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		l.ID, l.PortID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
