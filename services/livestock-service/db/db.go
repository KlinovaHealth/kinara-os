package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/livestock-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RegisterAnimal(ctx context.Context, a models.Animal) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO animals (id,tag_ref,farmer_id,species,breed,birth_date,weight_kg,health_status,is_active,tenant_id,registered_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		a.ID, a.TagRef, a.FarmerID, a.Species, a.Breed, a.BirthDate,
		a.WeightKg, a.HealthStatus, a.IsActive, a.TenantID, a.RegisteredAt, a.UpdatedAt)
	return err
}

func (q *Queries) GetAnimal(ctx context.Context, id uuid.UUID) (*models.Animal, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id,tag_ref,farmer_id,species,breed,birth_date,weight_kg,health_status,is_active,tenant_id,registered_at,updated_at
		 FROM animals WHERE id=$1`, id)
	var a models.Animal
	err := row.Scan(&a.ID, &a.TagRef, &a.FarmerID, &a.Species, &a.Breed, &a.BirthDate,
		&a.WeightKg, &a.HealthStatus, &a.IsActive, &a.TenantID, &a.RegisteredAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (q *Queries) ListByFarmer(ctx context.Context, farmerID uuid.UUID) ([]models.Animal, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,tag_ref,farmer_id,species,breed,birth_date,weight_kg,health_status,is_active,tenant_id,registered_at,updated_at
		 FROM animals WHERE farmer_id=$1 AND is_active=true ORDER BY registered_at DESC`, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Animal
	for rows.Next() {
		var a models.Animal
		if err := rows.Scan(&a.ID, &a.TagRef, &a.FarmerID, &a.Species, &a.Breed, &a.BirthDate,
			&a.WeightKg, &a.HealthStatus, &a.IsActive, &a.TenantID, &a.RegisteredAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (q *Queries) UpdateHealth(ctx context.Context, id uuid.UUID, status models.HealthStatus, weightKg float64, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE animals SET health_status=$1, weight_kg=$2, updated_at=$3 WHERE id=$4`,
		status, weightKg, now, id)
	return err
}

func (q *Queries) RecordProduction(ctx context.Context, p models.ProductionRecord) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO production_records (id,animal_id,farmer_id,product_type,quantity_kg,recorded_at,notes,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.ID, p.AnimalID, p.FarmerID, p.ProductType, p.QuantityKg, p.RecordedAt, p.Notes, p.RecordedAt)
	return err
}

func (q *Queries) SumProduction(ctx context.Context, farmerID uuid.UUID, since time.Time) (float64, error) {
	var total float64
	err := q.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity_kg),0) FROM production_records WHERE farmer_id=$1 AND recorded_at>=$2`,
		farmerID, since).Scan(&total)
	return total, err
}
