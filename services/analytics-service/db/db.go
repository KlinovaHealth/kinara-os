package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/analytics-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RecordImpact(ctx context.Context, m models.ImpactMetric) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO impact_metrics (id,pillar,country,metric_type,metric_name,metric_value,metric_unit,period_start,period_end,beneficiary_count,notes,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		m.ID, m.Pillar, m.Country, m.MetricType, m.MetricName, m.MetricValue, m.MetricUnit,
		m.PeriodStart, m.PeriodEnd, m.BeneficiaryCount, m.Notes, m.CreatedAt)
	return err
}

type ListImpactParams struct {
	Pillar  *models.Pillar
	Country *string
	Page, Limit int
}

func (q *Queries) ListImpact(ctx context.Context, p ListImpactParams) ([]models.ImpactMetric, error) {
	query := `SELECT id,pillar,country,metric_type,metric_name,metric_value,metric_unit,period_start,period_end,beneficiary_count,notes,created_at FROM impact_metrics WHERE 1=1`
	args := []interface{}{}
	i := 1
	if p.Pillar != nil { query += fmt.Sprintf(" AND pillar=$%d", i); args = append(args, *p.Pillar); i++ }
	if p.Country != nil { query += fmt.Sprintf(" AND country=$%d", i); args = append(args, *p.Country); i++ }
	offset := (p.Page - 1) * p.Limit
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.ImpactMetric
	for rows.Next() {
		var m models.ImpactMetric
		if err := rows.Scan(&m.ID, &m.Pillar, &m.Country, &m.MetricType, &m.MetricName, &m.MetricValue, &m.MetricUnit, &m.PeriodStart, &m.PeriodEnd, &m.BeneficiaryCount, &m.Notes, &m.CreatedAt); err != nil { return nil, err }
		result = append(result, m)
	}
	return result, nil
}

func (q *Queries) CreateSummary(ctx context.Context, s models.CrossPillarSummary) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO cross_pillar_summaries (id,country,period_start,period_end,health_score,agri_score,logistics_score,maritime_score,overall_score,total_beneficiaries,total_services_delivered,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.ID, s.Country, s.PeriodStart, s.PeriodEnd, s.HealthScore, s.AgriScore, s.LogisticsScore,
		s.MaritimeScore, s.OverallScore, s.TotalBeneficiaries, s.TotalServicesDelivered, s.CreatedAt)
	return err
}

func (q *Queries) GetSummary(ctx context.Context, id uuid.UUID) (*models.CrossPillarSummary, error) {
	s := &models.CrossPillarSummary{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,country,period_start,period_end,health_score,agri_score,logistics_score,maritime_score,overall_score,total_beneficiaries,total_services_delivered,created_at FROM cross_pillar_summaries WHERE id=$1`, id).
		Scan(&s.ID, &s.Country, &s.PeriodStart, &s.PeriodEnd, &s.HealthScore, &s.AgriScore, &s.LogisticsScore, &s.MaritimeScore, &s.OverallScore, &s.TotalBeneficiaries, &s.TotalServicesDelivered, &s.CreatedAt)
	if err != nil { return nil, err }
	return s, nil
}

func (q *Queries) ListSummaries(ctx context.Context, country string) ([]models.CrossPillarSummary, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,country,period_start,period_end,health_score,agri_score,logistics_score,maritime_score,overall_score,total_beneficiaries,total_services_delivered,created_at FROM cross_pillar_summaries WHERE country=$1 ORDER BY period_start DESC LIMIT 24`, country)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.CrossPillarSummary
	for rows.Next() {
		var s models.CrossPillarSummary
		if err := rows.Scan(&s.ID, &s.Country, &s.PeriodStart, &s.PeriodEnd, &s.HealthScore, &s.AgriScore, &s.LogisticsScore, &s.MaritimeScore, &s.OverallScore, &s.TotalBeneficiaries, &s.TotalServicesDelivered, &s.CreatedAt); err != nil { return nil, err }
		result = append(result, s)
	}
	return result, nil
}

func (q *Queries) CreateReport(ctx context.Context, r models.GovernmentReport) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO government_reports (id,report_ref,country,report_type,period_start,period_end,generated_at,summary_json,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		r.ID, r.ReportRef, r.Country, r.ReportType, r.PeriodStart, r.PeriodEnd, r.GeneratedAt, r.SummaryJSON, r.CreatedAt)
	return err
}

func (q *Queries) GetReport(ctx context.Context, id uuid.UUID) (*models.GovernmentReport, error) {
	r := &models.GovernmentReport{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,report_ref,country,report_type,period_start,period_end,generated_at,summary_json,created_at FROM government_reports WHERE id=$1`, id).
		Scan(&r.ID, &r.ReportRef, &r.Country, &r.ReportType, &r.PeriodStart, &r.PeriodEnd, &r.GeneratedAt, &r.SummaryJSON, &r.CreatedAt)
	if err != nil { return nil, err }
	return r, nil
}

func (q *Queries) ListReports(ctx context.Context, country string) ([]models.GovernmentReport, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,report_ref,country,report_type,period_start,period_end,generated_at,summary_json,created_at FROM government_reports WHERE country=$1 ORDER BY generated_at DESC LIMIT 50`, country)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.GovernmentReport
	for rows.Next() {
		var r models.GovernmentReport
		if err := rows.Scan(&r.ID, &r.ReportRef, &r.Country, &r.ReportType, &r.PeriodStart, &r.PeriodEnd, &r.GeneratedAt, &r.SummaryJSON, &r.CreatedAt); err != nil { return nil, err }
		result = append(result, r)
	}
	return result, nil
}

func (q *Queries) ReportBottleneck(ctx context.Context, b models.Bottleneck) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO bottlenecks (id,pillar,country,bottleneck_type,description,severity,affected_units,recommended_action,detected_at,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		b.ID, b.Pillar, b.Country, b.BottleneckType, b.Description, b.Severity, b.AffectedUnits, b.RecommendedAction, b.DetectedAt, b.CreatedAt)
	return err
}

func (q *Queries) ListBottlenecks(ctx context.Context, pillar *models.Pillar, country *string) ([]models.Bottleneck, error) {
	query := `SELECT id,pillar,country,bottleneck_type,description,severity,affected_units,recommended_action,detected_at,resolved_at,created_at FROM bottlenecks WHERE resolved_at IS NULL`
	args := []interface{}{}
	i := 1
	if pillar != nil { query += fmt.Sprintf(" AND pillar=$%d", i); args = append(args, *pillar); i++ }
	if country != nil { query += fmt.Sprintf(" AND country=$%d", i); args = append(args, *country) }
	query += " ORDER BY detected_at DESC LIMIT 50"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.Bottleneck
	for rows.Next() {
		var b models.Bottleneck
		if err := rows.Scan(&b.ID, &b.Pillar, &b.Country, &b.BottleneckType, &b.Description, &b.Severity, &b.AffectedUnits, &b.RecommendedAction, &b.DetectedAt, &b.ResolvedAt, &b.CreatedAt); err != nil { return nil, err }
		result = append(result, b)
	}
	return result, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.AnalyticsAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO analytics_audit_log (id,actor_id,action,entity_type,entity_id,created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
