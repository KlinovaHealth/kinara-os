package db

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/irrigation-service/models"
)

// Store is the interface consumed by handlers; implemented by *Queries.
type Store interface {
	RegisterSystem(ctx context.Context, s models.IrrigationSystem) error
	GetSystem(ctx context.Context, farmID string) (*models.IrrigationSystem, error)
	GetLatestMoisture(ctx context.Context, farmID string) (*models.SoilMoistureReading, error)
	CreateSchedule(ctx context.Context, s models.WateringSchedule) error
	InsertMoisture(ctx context.Context, r models.SoilMoistureReading) error
	InsertAlert(ctx context.Context, a models.IrrigationAlert) error
	GetHistory(ctx context.Context, farmID string, limit int) ([]models.WateringHistory, error)
	InsertAudit(ctx context.Context, farmID, actorID, action string) error
}

// Queries is the concrete Postgres implementation of Store.
type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RegisterSystem(ctx context.Context, s models.IrrigationSystem) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO irrigation_systems (id, farm_id, system_type, capacity_liters, sensor_id, tenant_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (farm_id) DO UPDATE
		   SET system_type=$3, capacity_liters=$4, sensor_id=$5`,
		s.ID, s.FarmID, s.SystemType, s.CapacityLiters, s.SensorID, s.TenantID, s.CreatedAt)
	return err
}

func (q *Queries) GetSystem(ctx context.Context, farmID string) (*models.IrrigationSystem, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id, farm_id, system_type, capacity_liters, sensor_id, tenant_id, created_at
		 FROM irrigation_systems WHERE farm_id=$1`, farmID)
	var s models.IrrigationSystem
	err := row.Scan(&s.ID, &s.FarmID, &s.SystemType, &s.CapacityLiters, &s.SensorID, &s.TenantID, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (q *Queries) GetLatestMoisture(ctx context.Context, farmID string) (*models.SoilMoistureReading, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id, farm_id, moisture_pct, sensor_id, recorded_at
		 FROM soil_moisture_readings WHERE farm_id=$1
		 ORDER BY recorded_at DESC LIMIT 1`, farmID)
	var r models.SoilMoistureReading
	err := row.Scan(&r.ID, &r.FarmID, &r.MoisturePct, &r.SensorID, &r.RecordedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (q *Queries) CreateSchedule(ctx context.Context, s models.WateringSchedule) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO watering_schedules (id, farm_id, cron_expression, duration_min, crop_type, tenant_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.FarmID, s.CronExpression, s.DurationMin, s.CropType, s.TenantID, s.CreatedAt)
	return err
}

func (q *Queries) InsertMoisture(ctx context.Context, r models.SoilMoistureReading) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO soil_moisture_readings (id, farm_id, moisture_pct, sensor_id, recorded_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		r.ID, r.FarmID, r.MoisturePct, r.SensorID, r.RecordedAt)
	return err
}

func (q *Queries) InsertAlert(ctx context.Context, a models.IrrigationAlert) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO irrigation_alerts (id, farm_id, message, alert_type, sent_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		a.ID, a.FarmID, a.Message, a.AlertType, a.SentAt)
	return err
}

func (q *Queries) GetHistory(ctx context.Context, farmID string, limit int) ([]models.WateringHistory, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, farm_id, duration_min, COALESCE(amount_liters,0), trigger_type, irrigated_at
		 FROM watering_history WHERE farm_id=$1 ORDER BY irrigated_at DESC LIMIT $2`,
		farmID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.WateringHistory
	for rows.Next() {
		var h models.WateringHistory
		if err := rows.Scan(&h.ID, &h.FarmID, &h.DurationMin, &h.AmountL, &h.TriggerType, &h.IrrigatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

func (q *Queries) InsertAudit(ctx context.Context, farmID, actorID, action string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO irrigation_audit_log (farm_id, action, actor_id) VALUES ($1, $2, $3)`,
		farmID, action, actorID)
	if err != nil {
		slog.Error("irrigation audit insert failed", "error", err)
	}
	return err
}
