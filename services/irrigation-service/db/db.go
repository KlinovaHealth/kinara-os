package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/irrigation-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateSchedule(ctx context.Context, s models.IrrigationSchedule) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO irrigation_schedules (id,schedule_ref,farmer_id,field_id,crop_type,method,frequency_days,duration_min,water_liters,is_active,tenant_id,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		s.ID, s.ScheduleRef, s.FarmerID, s.FieldID, s.CropType, s.Method,
		s.FrequencyDays, s.DurationMin, s.WaterLiters, s.IsActive, s.TenantID, s.CreatedAt, s.UpdatedAt)
	return err
}

func (q *Queries) ListSchedules(ctx context.Context, farmerID uuid.UUID) ([]models.IrrigationSchedule, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,schedule_ref,farmer_id,field_id,crop_type,method,frequency_days,duration_min,water_liters,is_active,tenant_id,created_at,updated_at
		 FROM irrigation_schedules WHERE farmer_id=$1 AND is_active=true ORDER BY created_at DESC`, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.IrrigationSchedule
	for rows.Next() {
		var s models.IrrigationSchedule
		if err := rows.Scan(&s.ID, &s.ScheduleRef, &s.FarmerID, &s.FieldID, &s.CropType, &s.Method,
			&s.FrequencyDays, &s.DurationMin, &s.WaterLiters, &s.IsActive, &s.TenantID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (q *Queries) LogEvent(ctx context.Context, e models.IrrigationEvent) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO irrigation_events (id,schedule_id,farmer_id,field_id,scheduled_at,water_used_l,status,notes,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.ID, e.ScheduleID, e.FarmerID, e.FieldID, e.ScheduledAt, e.WaterUsedL, e.Status, e.Notes, e.CreatedAt)
	return err
}

func (q *Queries) ListEvents(ctx context.Context, farmerID uuid.UUID, limit int) ([]models.IrrigationEvent, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,schedule_id,farmer_id,field_id,scheduled_at,started_at,completed_at,water_used_l,status,notes,created_at
		 FROM irrigation_events WHERE farmer_id=$1 ORDER BY scheduled_at DESC LIMIT $2`, farmerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.IrrigationEvent
	for rows.Next() {
		var e models.IrrigationEvent
		if err := rows.Scan(&e.ID, &e.ScheduleID, &e.FarmerID, &e.FieldID, &e.ScheduledAt,
			&e.StartedAt, &e.CompletedAt, &e.WaterUsedL, &e.Status, &e.Notes, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (q *Queries) SumWaterUsed(ctx context.Context, farmerID uuid.UUID, since time.Time) (float64, error) {
	var total float64
	err := q.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(water_used_l),0) FROM irrigation_events WHERE farmer_id=$1 AND scheduled_at>=$2 AND status='completed'`,
		farmerID, since).Scan(&total)
	return total, err
}
