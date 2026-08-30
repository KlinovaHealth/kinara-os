package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/outbreak-service/models"
)

// Store is the interface over all DB operations, enabling mock injection in tests.
type Store interface {
	InsertCase(ctx context.Context, c models.SuspectedCase) error
	CountRecentCases(ctx context.Context, diseaseCode, clinicID string, window time.Duration) (int, error)
	UpsertOutbreak(ctx context.Context, diseaseCode, clinicID, diseaseName string, caseCount int) error
	ListActiveOutbreaks(ctx context.Context) ([]models.ConfirmedOutbreak, error)
	ConfirmOutbreak(ctx context.Context, id uuid.UUID, actorID string) error
	GetClusters(ctx context.Context) ([]models.DiseaseCluster, error)
	GetTrends(ctx context.Context) ([]models.DiseaseTrend, error)
	InsertNotification(ctx context.Context, n models.OutbreakNotification) error
	InsertAudit(ctx context.Context, outbreakID, actorID, action string) error
}

// Queries is the concrete postgres implementation of Store.
type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

// Ensure Queries satisfies Store at compile time.
var _ Store = (*Queries)(nil)

func (q *Queries) InsertCase(ctx context.Context, c models.SuspectedCase) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO suspected_cases
		 (id, case_ref, patient_id, disease_code, disease_name, clinic_id, location, symptoms, reported_by, tenant_id, reported_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		c.ID, c.CaseRef, c.PatientID, c.DiseaseCode, c.DiseaseName,
		c.ClinicID, c.Location, c.Symptoms, c.ReportedBy, c.TenantID, c.ReportedAt)
	return err
}

func (q *Queries) CountRecentCases(ctx context.Context, diseaseCode, clinicID string, window time.Duration) (int, error) {
	since := time.Now().UTC().Add(-window)
	var count int
	err := q.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM suspected_cases
		 WHERE disease_code=$1 AND clinic_id=$2 AND reported_at >= $3`,
		diseaseCode, clinicID, since).Scan(&count)
	return count, err
}

func (q *Queries) UpsertOutbreak(ctx context.Context, diseaseCode, clinicID, diseaseName string, caseCount int) error {
	id := uuid.New()
	alertRef := "OBK-" + strings.ToUpper(id.String()[:8])
	_, err := q.pool.Exec(ctx,
		`INSERT INTO confirmed_outbreaks
		 (id, alert_ref, disease_code, disease_name, clinic_id, case_count, status, tenant_id, detected_at)
		 VALUES ($1,$2,$3,$4,$5,$6,'active','system',NOW())
		 ON CONFLICT (disease_code, clinic_id) WHERE status != 'contained'
		 DO UPDATE SET case_count = EXCLUDED.case_count`,
		id, alertRef, diseaseCode, diseaseName, clinicID, caseCount)
	return err
}

func (q *Queries) ListActiveOutbreaks(ctx context.Context) ([]models.ConfirmedOutbreak, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, alert_ref, disease_code, disease_name, clinic_id, case_count, status, tenant_id, detected_at, contained_at
		 FROM confirmed_outbreaks
		 WHERE status = 'active'
		 ORDER BY detected_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ConfirmedOutbreak
	for rows.Next() {
		var o models.ConfirmedOutbreak
		if err := rows.Scan(&o.ID, &o.AlertRef, &o.DiseaseCode, &o.DiseaseName,
			&o.ClinicID, &o.CaseCount, &o.Status, &o.TenantID, &o.DetectedAt, &o.ContainedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (q *Queries) ConfirmOutbreak(ctx context.Context, id uuid.UUID, actorID string) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE confirmed_outbreaks SET status='confirmed' WHERE id=$1`, id)
	if err != nil {
		return err
	}
	return q.InsertAudit(ctx, id.String(), actorID, "confirm_outbreak")
}

func (q *Queries) GetClusters(ctx context.Context) ([]models.DiseaseCluster, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT disease_code, disease_name, clinic_id, COUNT(*) AS case_count
		 FROM suspected_cases
		 GROUP BY disease_code, disease_name, clinic_id
		 ORDER BY case_count DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.DiseaseCluster
	for rows.Next() {
		var c models.DiseaseCluster
		if err := rows.Scan(&c.DiseaseCode, &c.DiseaseName, &c.ClinicID, &c.CaseCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (q *Queries) GetTrends(ctx context.Context) ([]models.DiseaseTrend, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT disease_code, DATE_TRUNC('day', reported_at) AS day, COUNT(*) AS case_count
		 FROM suspected_cases
		 GROUP BY disease_code, day
		 ORDER BY day DESC
		 LIMIT 30`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.DiseaseTrend
	for rows.Next() {
		var t models.DiseaseTrend
		if err := rows.Scan(&t.DiseaseCode, &t.Date, &t.CaseCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (q *Queries) InsertNotification(ctx context.Context, n models.OutbreakNotification) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO outbreak_notifications (id, outbreak_id, message, recipients, sent_by, sent_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		n.ID, n.OutbreakID, n.Message, n.Recipients, n.SentBy, n.SentAt)
	return err
}

func (q *Queries) InsertAudit(ctx context.Context, outbreakID, actorID, action string) error {
	// outbreakID may be empty string for case-report audits
	var outbreakIDVal interface{}
	if outbreakID == "" {
		outbreakIDVal = nil
	} else {
		parsed, err := uuid.Parse(outbreakID)
		if err != nil {
			// store as-is using a text fallback; wrap in a format error for clarity
			return fmt.Errorf("invalid outbreak_id %q: %w", outbreakID, err)
		}
		outbreakIDVal = parsed
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO outbreak_audit_log (outbreak_id, action, actor_id, occurred_at)
		 VALUES ($1,$2,$3,NOW())`,
		outbreakIDVal, action, actorID)
	return err
}
