package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/extension-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

// ListResources returns extension resources, optionally filtered by crop_type and/or language.
func (q *Queries) ListResources(ctx context.Context, cropType, language string) ([]models.ExtensionResource, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, title, content_summary, COALESCE(crop_type,''), language, resource_type, viewed_count, created_at
		 FROM extension_resources
		 WHERE ($1 = '' OR crop_type = $1)
		   AND ($2 = '' OR language = $2)
		 ORDER BY created_at DESC`,
		cropType, language)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ExtensionResource
	for rows.Next() {
		var r models.ExtensionResource
		if err := rows.Scan(&r.ID, &r.Title, &r.ContentSummary, &r.CropType, &r.Language, &r.ResourceType, &r.ViewedCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRecommendedResources returns top resources for a crop type ordered by view count.
func (q *Queries) GetRecommendedResources(ctx context.Context, cropType string, limit int) ([]models.ExtensionResource, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, title, content_summary, COALESCE(crop_type,''), language, resource_type, viewed_count, created_at
		 FROM extension_resources
		 WHERE ($1 = '' OR crop_type = $1)
		 ORDER BY viewed_count DESC
		 LIMIT $2`,
		cropType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ExtensionResource
	for rows.Next() {
		var r models.ExtensionResource
		if err := rows.Scan(&r.ID, &r.Title, &r.ContentSummary, &r.CropType, &r.Language, &r.ResourceType, &r.ViewedCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// BookConsultation inserts a new consultation record.
func (q *Queries) BookConsultation(ctx context.Context, c models.Consultation) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO consultations
		 (id, consult_ref, farmer_id, officer_id, topic, crop_type, preferred_date, status, notes, tenant_id, booked_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		c.ID, c.ConsultRef, c.FarmerID, c.OfficerID, c.Topic, c.CropType,
		c.PreferredDate, c.Status, c.Notes, c.TenantID, c.BookedAt)
	return err
}

// GetConsultation fetches a single consultation by primary key.
func (q *Queries) GetConsultation(ctx context.Context, id uuid.UUID) (*models.Consultation, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id, consult_ref, farmer_id, officer_id, topic, COALESCE(crop_type,''),
		        preferred_date, status, COALESCE(notes,''), tenant_id, booked_at
		 FROM consultations WHERE id=$1`, id)
	var c models.Consultation
	err := row.Scan(
		&c.ID, &c.ConsultRef, &c.FarmerID, &c.OfficerID, &c.Topic, &c.CropType,
		&c.PreferredDate, &c.Status, &c.Notes, &c.TenantID, &c.BookedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// InsertFeedback inserts extension feedback for a consultation.
func (q *Queries) InsertFeedback(ctx context.Context, f models.ExtensionFeedback) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO extension_feedback (id, consultation_id, farmer_id, rating, notes, result, submitted_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		f.ID, f.ConsultationID, f.FarmerID, f.Rating, f.Notes, f.Result, f.SubmittedAt)
	return err
}

// GetBestPractices returns best practices for a given crop type.
func (q *Queries) GetBestPractices(ctx context.Context, cropType string) ([]models.BestPractice, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, crop_type, technique, COALESCE(description,''),
		        COALESCE(expected_yield_improvement_pct,0), COALESCE(climate,'')
		 FROM best_practices
		 WHERE crop_type=$1
		 ORDER BY expected_yield_improvement_pct DESC`,
		cropType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.BestPractice
	for rows.Next() {
		var bp models.BestPractice
		if err := rows.Scan(&bp.ID, &bp.CropType, &bp.Technique, &bp.Description, &bp.ExpectedYieldImprovement, &bp.Climate); err != nil {
			return nil, err
		}
		out = append(out, bp)
	}
	return out, rows.Err()
}

// InsertAudit appends an immutable audit log entry for a consultation action.
func (q *Queries) InsertAudit(ctx context.Context, consultID, actorID, action string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO extension_audit_log (consult_id, action, actor_id, occurred_at)
		 VALUES ($1,$2,$3,NOW())`,
		consultID, action, actorID)
	return err
}
