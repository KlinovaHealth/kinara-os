package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/outbreak-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateResponse(ctx context.Context, r models.OutbreakResponse) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO outbreak_responses (id,response_ref,alert_ref,disease_name,country,region,status,lead_coordinator,team_size,cases_targeted,population,tenant_id,started_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		r.ID, r.ResponseRef, r.AlertRef, r.DiseaseName, r.Country, r.Region,
		r.Status, r.LeadCoordinator, r.TeamSize, r.CasesTargeted, r.Population,
		r.TenantID, r.StartedAt, r.UpdatedAt)
	return err
}

func (q *Queries) GetResponse(ctx context.Context, id uuid.UUID) (*models.OutbreakResponse, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id,response_ref,alert_ref,disease_name,country,region,status,lead_coordinator,team_size,cases_targeted,population,tenant_id,started_at,contained_at,resolved_at,updated_at
		 FROM outbreak_responses WHERE id=$1`, id)
	var r models.OutbreakResponse
	err := row.Scan(&r.ID, &r.ResponseRef, &r.AlertRef, &r.DiseaseName, &r.Country, &r.Region,
		&r.Status, &r.LeadCoordinator, &r.TeamSize, &r.CasesTargeted, &r.Population,
		&r.TenantID, &r.StartedAt, &r.ContainedAt, &r.ResolvedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (q *Queries) ListResponses(ctx context.Context, country string, limit int) ([]models.OutbreakResponse, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,response_ref,alert_ref,disease_name,country,region,status,lead_coordinator,team_size,cases_targeted,population,tenant_id,started_at,contained_at,resolved_at,updated_at
		 FROM outbreak_responses WHERE ($1='' OR country=$1) ORDER BY started_at DESC LIMIT $2`,
		country, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.OutbreakResponse
	for rows.Next() {
		var r models.OutbreakResponse
		if err := rows.Scan(&r.ID, &r.ResponseRef, &r.AlertRef, &r.DiseaseName, &r.Country, &r.Region,
			&r.Status, &r.LeadCoordinator, &r.TeamSize, &r.CasesTargeted, &r.Population,
			&r.TenantID, &r.StartedAt, &r.ContainedAt, &r.ResolvedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (q *Queries) UpdateStatus(ctx context.Context, id uuid.UUID, status models.ResponseStatus, resolvedAt *time.Time, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE outbreak_responses SET status=$1,
		  contained_at = CASE WHEN $1='contained' THEN COALESCE(contained_at,$3) ELSE contained_at END,
		  resolved_at  = CASE WHEN $1='resolved'  THEN COALESCE(resolved_at,$2)  ELSE resolved_at  END,
		  updated_at=$3 WHERE id=$4`,
		status, resolvedAt, now, id)
	return err
}

func (q *Queries) AddAction(ctx context.Context, a models.ResponseAction) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO response_actions (id,response_id,action_type,description,assigned_to,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		a.ID, a.ResponseID, a.ActionType, a.Description, a.AssignedTo, a.CreatedAt)
	return err
}
