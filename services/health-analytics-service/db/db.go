package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/health-analytics-service/models"
)

var ErrNotFound = errors.New("db: record not found")

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) ReportDisease(ctx context.Context, r models.DiseaseReport) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO disease_reports (id,clinic_id,country,region,icd10_code,disease_name,case_count,period,period_start,period_end,severity,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		r.ID, r.ClinicID, r.Country, r.Region, r.ICD10Code, r.DiseaseName,
		r.CaseCount, r.Period, r.PeriodStart, r.PeriodEnd, r.Severity, r.CreatedAt)
	return err
}

type ListDiseaseParams struct {
	Country  *string
	ICD10    *string
	Page     int
	Limit    int
}

func (q *Queries) ListDiseases(ctx context.Context, p ListDiseaseParams) ([]models.DiseaseReport, error) {
	query := `SELECT id,clinic_id,country,region,icd10_code,disease_name,case_count,period,period_start,period_end,severity,created_at
	          FROM disease_reports WHERE 1=1`
	args := []interface{}{}
	i := 1
	if p.Country != nil {
		query += fmt.Sprintf(" AND country=$%d", i)
		args = append(args, *p.Country)
		i++
	}
	if p.ICD10 != nil {
		query += fmt.Sprintf(" AND icd10_code=$%d", i)
		args = append(args, *p.ICD10)
		i++
	}
	if p.Limit == 0 {
		p.Limit = 50
	}
	if p.Page < 1 {
		p.Page = 1
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, p.Limit, (p.Page-1)*p.Limit)

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.DiseaseReport
	for rows.Next() {
		var r models.DiseaseReport
		if err := rows.Scan(&r.ID, &r.ClinicID, &r.Country, &r.Region, &r.ICD10Code, &r.DiseaseName,
			&r.CaseCount, &r.Period, &r.PeriodStart, &r.PeriodEnd, &r.Severity, &r.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func (q *Queries) CreateOutbreakAlert(ctx context.Context, a models.OutbreakAlert) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO outbreak_alerts (id,alert_ref,clinic_id,country,region,icd10_code,disease_name,case_count,threshold,status,detected_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		a.ID, a.AlertRef, a.ClinicID, a.Country, a.Region, a.ICD10Code, a.DiseaseName,
		a.CaseCount, a.Threshold, a.Status, a.DetectedAt)
	return err
}

func (q *Queries) ListActiveAlerts(ctx context.Context, country *string) ([]models.OutbreakAlert, error) {
	query := `SELECT id,alert_ref,clinic_id,country,region,icd10_code,disease_name,case_count,threshold,status,detected_at,resolved_at
	          FROM outbreak_alerts WHERE status != 'resolved'`
	args := []interface{}{}
	if country != nil {
		query += " AND country=$1"
		args = append(args, *country)
	}
	query += " ORDER BY detected_at DESC"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.OutbreakAlert
	for rows.Next() {
		var a models.OutbreakAlert
		if err := rows.Scan(&a.ID, &a.AlertRef, &a.ClinicID, &a.Country, &a.Region, &a.ICD10Code, &a.DiseaseName,
			&a.CaseCount, &a.Threshold, &a.Status, &a.DetectedAt, &a.ResolvedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, nil
}

func (q *Queries) ResolveAlert(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE outbreak_alerts SET status='resolved', resolved_at=$1 WHERE id=$2`, now, id)
	return err
}

func (q *Queries) RecordClinicMetric(ctx context.Context, m models.ClinicMetric) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO clinic_metrics
		 (id,clinic_id,country,period,period_start,period_end,total_patients,avg_visit_minutes,
		  referral_count,referral_success_rate,patient_outcome_improved,patient_outcome_stable,
		  patient_outcome_worsened,cost_per_visit_usd,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		m.ID, m.ClinicID, m.Country, m.Period, m.PeriodStart, m.PeriodEnd,
		m.TotalPatients, m.AvgVisitMinutes, m.ReferralCount, m.ReferralSuccessRate,
		m.PatientOutcomeImproved, m.PatientOutcomeStable, m.PatientOutcomeWorsened,
		m.CostPerVisitUSD, m.CreatedAt)
	return err
}

func (q *Queries) GetClinicMetrics(ctx context.Context, clinicID uuid.UUID, limit int) ([]models.ClinicMetric, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,clinic_id,country,period,period_start,period_end,total_patients,avg_visit_minutes,
		        referral_count,referral_success_rate,patient_outcome_improved,patient_outcome_stable,
		        patient_outcome_worsened,cost_per_visit_usd,created_at
		 FROM clinic_metrics WHERE clinic_id=$1 ORDER BY created_at DESC LIMIT $2`, clinicID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ClinicMetric
	for rows.Next() {
		var m models.ClinicMetric
		if err := rows.Scan(&m.ID, &m.ClinicID, &m.Country, &m.Period, &m.PeriodStart, &m.PeriodEnd,
			&m.TotalPatients, &m.AvgVisitMinutes, &m.ReferralCount, &m.ReferralSuccessRate,
			&m.PatientOutcomeImproved, &m.PatientOutcomeStable, &m.PatientOutcomeWorsened,
			&m.CostPerVisitUSD, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

func (q *Queries) GetImpactSummary(ctx context.Context, country string) (*models.ImpactSummary, error) {
	var s models.ImpactSummary
	s.Country = country
	s.GeneratedAt = time.Now().UTC()

	err := q.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_patients),0),
		        COALESCE(AVG(cost_per_visit_usd),0),
		        COUNT(DISTINCT clinic_id)
		 FROM clinic_metrics WHERE country=$1`, country).
		Scan(&s.TotalPatients, &s.AvgCostPerVisitUSD, &s.TotalClinics)
	if err != nil {
		return nil, err
	}

	// Outbreak count
	_ = q.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbreak_alerts WHERE country=$1 AND status='active'`, country).
		Scan(&s.ActiveOutbreaks)

	// Outcome improvement rate
	var improved, total int
	_ = q.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(patient_outcome_improved),0),
		        COALESCE(SUM(patient_outcome_improved)+SUM(patient_outcome_stable)+SUM(patient_outcome_worsened),1)
		 FROM clinic_metrics WHERE country=$1`, country).
		Scan(&improved, &total)
	if total > 0 {
		s.OutcomeImprovementRate = float64(improved) / float64(total) * 100
	}

	return &s, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.HealthAnalyticsAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO health_analytics_audit_log (id,actor_id,action,resource,created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		l.ID, l.ActorID, l.Action, l.Resource, l.CreatedAt)
	return err
}
