package db

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/livestock-service/models"
)

// Store is the interface consumed by handlers; implemented by *Queries.
type Store interface {
	RegisterAnimal(ctx context.Context, a models.Animal) error
	GetAnimal(ctx context.Context, id uuid.UUID) (*models.Animal, error)
	LogHealthEvent(ctx context.Context, e models.HealthEvent) error
	GetHealthHistory(ctx context.Context, animalID uuid.UUID) ([]models.HealthEvent, error)
	LogProduction(ctx context.Context, p models.ProductionRecord) error
	ListHerd(ctx context.Context, farmerID uuid.UUID) ([]models.Animal, error)
	GetHerdAnalytics(ctx context.Context, farmerID uuid.UUID) (models.HerdAnalytics, error)
	InsertVetAlert(ctx context.Context, a models.VeterinaryAlert) error
	InsertAudit(ctx context.Context, animalID, actorID, action string) error
}

// Queries is the concrete Postgres implementation of Store.
type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RegisterAnimal(ctx context.Context, a models.Animal) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO animals (id, animal_ref, farmer_id, animal_type, breed, age_months, sex, ear_tag, tenant_id, registered_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		a.ID, a.AnimalRef, a.FarmerID, a.AnimalType, a.Breed,
		a.AgeMonths, a.Sex, a.EarTag, a.TenantID, a.RegisteredAt)
	return err
}

func (q *Queries) GetAnimal(ctx context.Context, id uuid.UUID) (*models.Animal, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id, animal_ref, farmer_id, animal_type, breed, age_months, sex, ear_tag, tenant_id, registered_at
		 FROM animals WHERE id=$1`, id)
	var a models.Animal
	err := row.Scan(&a.ID, &a.AnimalRef, &a.FarmerID, &a.AnimalType, &a.Breed,
		&a.AgeMonths, &a.Sex, &a.EarTag, &a.TenantID, &a.RegisteredAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (q *Queries) LogHealthEvent(ctx context.Context, e models.HealthEvent) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO health_events (id, animal_id, event_type, description, treatment, veterinarian_id, event_date, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.ID, e.AnimalID, e.EventType, e.Description, e.Treatment,
		e.VeterinarianID, e.EventDate, e.CreatedBy)
	return err
}

func (q *Queries) GetHealthHistory(ctx context.Context, animalID uuid.UUID) ([]models.HealthEvent, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, animal_id, event_type, description, treatment, veterinarian_id, event_date, created_by
		 FROM health_events WHERE animal_id=$1 ORDER BY event_date DESC`, animalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.HealthEvent
	for rows.Next() {
		var e models.HealthEvent
		if err := rows.Scan(&e.ID, &e.AnimalID, &e.EventType, &e.Description, &e.Treatment,
			&e.VeterinarianID, &e.EventDate, &e.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (q *Queries) LogProduction(ctx context.Context, p models.ProductionRecord) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO production_records (id, animal_id, production_type, quantity, unit, recorded_date, recorded_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.AnimalID, p.ProductionType, p.Quantity, p.Unit, p.RecordedDate, p.RecordedBy)
	return err
}

func (q *Queries) ListHerd(ctx context.Context, farmerID uuid.UUID) ([]models.Animal, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, animal_ref, farmer_id, animal_type, breed, age_months, sex, ear_tag, tenant_id, registered_at
		 FROM animals WHERE farmer_id=$1 ORDER BY registered_at DESC`, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Animal
	for rows.Next() {
		var a models.Animal
		if err := rows.Scan(&a.ID, &a.AnimalRef, &a.FarmerID, &a.AnimalType, &a.Breed,
			&a.AgeMonths, &a.Sex, &a.EarTag, &a.TenantID, &a.RegisteredAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (q *Queries) GetHerdAnalytics(ctx context.Context, farmerID uuid.UUID) (models.HerdAnalytics, error) {
	var analytics models.HerdAnalytics
	err := q.pool.QueryRow(ctx,
		`SELECT
		   COUNT(*) AS total_animals,
		   COUNT(*) FILTER (WHERE id NOT IN (
		     SELECT DISTINCT animal_id FROM health_events
		     WHERE event_type='illness' AND event_date > NOW() - INTERVAL '30 days'
		   )) AS healthy_count
		 FROM animals WHERE farmer_id=$1`, farmerID).
		Scan(&analytics.TotalAnimals, &analytics.HealthyCount)
	if err != nil {
		return analytics, err
	}
	if analytics.TotalAnimals > 0 {
		analytics.HealthRatePct = float64(analytics.HealthyCount) / float64(analytics.TotalAnimals) * 100
	}

	// Monthly production total.
	_ = q.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(pr.quantity), 0)
		 FROM production_records pr
		 JOIN animals a ON a.id = pr.animal_id
		 WHERE a.farmer_id=$1
		   AND pr.recorded_date >= date_trunc('month', NOW())`, farmerID).
		Scan(&analytics.TotalProductionMonth)

	// Top producing type.
	var topType string
	_ = q.pool.QueryRow(ctx,
		`SELECT pr.production_type
		 FROM production_records pr
		 JOIN animals a ON a.id = pr.animal_id
		 WHERE a.farmer_id=$1
		   AND pr.recorded_date >= date_trunc('month', NOW())
		 GROUP BY pr.production_type
		 ORDER BY SUM(pr.quantity) DESC
		 LIMIT 1`, farmerID).Scan(&topType)
	analytics.TopProducingType = topType

	return analytics, nil
}

func (q *Queries) InsertVetAlert(ctx context.Context, a models.VeterinaryAlert) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO veterinary_alerts (id, animal_id, alert_type, priority, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		a.ID, a.AnimalID, a.AlertType, a.Priority, a.CreatedAt)
	return err
}

func (q *Queries) InsertAudit(ctx context.Context, animalID, actorID, action string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO livestock_audit_log (animal_id, action, actor_id) VALUES ($1, $2, $3)`,
		animalID, action, actorID)
	if err != nil {
		slog.Error("livestock audit insert failed", "error", err)
	}
	return err
}
