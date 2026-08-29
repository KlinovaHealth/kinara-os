package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/governance-service/models"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// ─── Compliance Reports ───────────────────────────────────────────────────────

type CreateComplianceReportParams struct {
	FacilityID      *uuid.UUID
	MinistryID      *uuid.UUID
	ReportType      models.ReportType
	Frequency       models.ReportFrequency
	PeriodStart     time.Time
	PeriodEnd       time.Time
	Country         string
	Region          *string
	Summary         string
	DataPayloadEnc  string
	SubmittedBy     uuid.UUID
}

func (q *Queries) CreateComplianceReport(ctx context.Context, p CreateComplianceReportParams) (*models.ComplianceReportRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO compliance_reports
			(facility_id, ministry_id, report_type, frequency, period_start, period_end,
			 country, region, summary, data_payload_enc, submitted_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, facility_id, ministry_id, report_type, frequency, period_start, period_end,
		          status, country, region, summary, data_payload_enc, submitted_by,
		          submitted_at, reviewed_by, reviewed_at, violation_notes_enc, created_at, updated_at`,
		p.FacilityID, p.MinistryID, p.ReportType, p.Frequency, p.PeriodStart, p.PeriodEnd,
		p.Country, p.Region, p.Summary, p.DataPayloadEnc, p.SubmittedBy,
	)
	return scanComplianceReportRow(row)
}

func (q *Queries) GetComplianceReportByID(ctx context.Context, id uuid.UUID) (*models.ComplianceReportRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, facility_id, ministry_id, report_type, frequency, period_start, period_end,
		       status, country, region, summary, data_payload_enc, submitted_by,
		       submitted_at, reviewed_by, reviewed_at, violation_notes_enc, created_at, updated_at
		FROM compliance_reports WHERE id = $1`, id)
	return scanComplianceReportRow(row)
}

type ListComplianceReportsParams struct {
	Country    string
	Status     *models.ComplianceStatus
	ReportType *models.ReportType
	Page       int
	Limit      int
}

func (q *Queries) ListComplianceReports(ctx context.Context, p ListComplianceReportsParams) ([]*models.ComplianceReportRow, error) {
	offset := (p.Page - 1) * p.Limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, facility_id, ministry_id, report_type, frequency, period_start, period_end,
		       status, country, region, summary, data_payload_enc, submitted_by,
		       submitted_at, reviewed_by, reviewed_at, violation_notes_enc, created_at, updated_at
		FROM compliance_reports
		WHERE ($1 = '' OR country = $1)
		  AND ($2::TEXT IS NULL OR status = $2)
		  AND ($3::TEXT IS NULL OR report_type = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`,
		p.Country, p.Status, p.ReportType, p.Limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.ComplianceReportRow
	for rows.Next() {
		r, err := scanComplianceReportRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *Queries) CountComplianceReports(ctx context.Context, p ListComplianceReportsParams) (int64, error) {
	var count int64
	err := q.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM compliance_reports
		WHERE ($1 = '' OR country = $1)
		  AND ($2::TEXT IS NULL OR status = $2)
		  AND ($3::TEXT IS NULL OR report_type = $3)`,
		p.Country, p.Status, p.ReportType,
	).Scan(&count)
	return count, err
}

type ReviewComplianceReportParams struct {
	ID                uuid.UUID
	Status            models.ComplianceStatus
	ReviewedBy        uuid.UUID
	ViolationNotesEnc *string
}

func (q *Queries) ReviewComplianceReport(ctx context.Context, p ReviewComplianceReportParams) (*models.ComplianceReportRow, error) {
	now := time.Now()
	row := q.pool.QueryRow(ctx, `
		UPDATE compliance_reports SET
			status              = $2,
			reviewed_by         = $3,
			reviewed_at         = $4,
			violation_notes_enc = COALESCE($5, violation_notes_enc),
			updated_at          = NOW()
		WHERE id = $1
		RETURNING id, facility_id, ministry_id, report_type, frequency, period_start, period_end,
		          status, country, region, summary, data_payload_enc, submitted_by,
		          submitted_at, reviewed_by, reviewed_at, violation_notes_enc, created_at, updated_at`,
		p.ID, p.Status, p.ReviewedBy, now, p.ViolationNotesEnc,
	)
	return scanComplianceReportRow(row)
}

func scanComplianceReportRow(scanner interface{ Scan(...any) error }) (*models.ComplianceReportRow, error) {
	var r models.ComplianceReportRow
	err := scanner.Scan(
		&r.ID, &r.FacilityID, &r.MinistryID, &r.ReportType, &r.Frequency,
		&r.PeriodStart, &r.PeriodEnd, &r.Status, &r.Country, &r.Region,
		&r.Summary, &r.DataPayloadEnc, &r.SubmittedBy, &r.SubmittedAt,
		&r.ReviewedBy, &r.ReviewedAt, &r.ViolationNotesEnc, &r.CreatedAt, &r.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("compliance report not found")
	}
	return &r, err
}

// ─── Epidemiology Records ─────────────────────────────────────────────────────

type CreateEpidemiologyParams struct {
	ICD10Code      string
	ICD10Desc      string
	Country        string
	Region         *string
	District       *string
	CaseCount      int
	DeathCount     int
	RecoveredCount int
	PeriodStart    time.Time
	PeriodEnd      time.Time
	AgeGroup       *string
	Gender         *string
	FacilityID     *uuid.UUID
	ReportedBy     uuid.UUID
}

func (q *Queries) CreateEpidemiologyRecord(ctx context.Context, p CreateEpidemiologyParams) (*models.EpidemiologyRecord, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO epidemiology_records
			(icd10_code, icd10_description, country, region, district,
			 case_count, death_count, recovered_count,
			 period_start, period_end, age_group, gender, facility_id, reported_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, icd10_code, icd10_description, country, region, district,
		          case_count, death_count, recovered_count,
		          period_start, period_end, age_group, gender, facility_id,
		          reported_by, created_at, updated_at`,
		p.ICD10Code, p.ICD10Desc, p.Country, p.Region, p.District,
		p.CaseCount, p.DeathCount, p.RecoveredCount,
		p.PeriodStart, p.PeriodEnd, p.AgeGroup, p.Gender, p.FacilityID, p.ReportedBy,
	)
	return scanEpidemiologyRow(row)
}

type ListEpidemiologyParams struct {
	Country   string
	ICD10Code string
	Page      int
	Limit     int
}

func (q *Queries) ListEpidemiologyRecords(ctx context.Context, p ListEpidemiologyParams) ([]*models.EpidemiologyRecord, error) {
	offset := (p.Page - 1) * p.Limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, icd10_code, icd10_description, country, region, district,
		       case_count, death_count, recovered_count,
		       period_start, period_end, age_group, gender, facility_id,
		       reported_by, created_at, updated_at
		FROM epidemiology_records
		WHERE ($1 = '' OR country = $1)
		  AND ($2 = '' OR icd10_code = $2)
		ORDER BY period_start DESC, case_count DESC
		LIMIT $3 OFFSET $4`,
		p.Country, p.ICD10Code, p.Limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.EpidemiologyRecord
	for rows.Next() {
		r, err := scanEpidemiologyRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func scanEpidemiologyRow(scanner interface{ Scan(...any) error }) (*models.EpidemiologyRecord, error) {
	var r models.EpidemiologyRecord
	var region, district, ageGroup, gender *string
	err := scanner.Scan(
		&r.ID, &r.ICD10Code, &r.ICD10Desc, &r.Country, &region, &district,
		&r.CaseCount, &r.DeathCount, &r.RecoveredCount,
		&r.PeriodStart, &r.PeriodEnd, &ageGroup, &gender, &r.FacilityID,
		&r.ReportedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("epidemiology record not found")
	}
	if region != nil {
		r.Region = *region
	}
	if district != nil {
		r.District = *district
	}
	if ageGroup != nil {
		r.AgeGroup = *ageGroup
	}
	if gender != nil {
		r.Gender = *gender
	}
	return &r, err
}

// ─── Coordination Rules ───────────────────────────────────────────────────────

type CreateRuleParams struct {
	RuleType       models.RuleType
	Name           string
	Description    string
	Country        string
	Region         *string
	ParametersEnc  string
	EffectiveFrom  time.Time
	EffectiveUntil *time.Time
	CreatedBy      uuid.UUID
}

func (q *Queries) CreateRule(ctx context.Context, p CreateRuleParams) (*models.CoordinationRuleRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO coordination_rules
			(rule_type, name, description, country, region, parameters_enc,
			 effective_from, effective_until, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, rule_type, name, description, country, region, parameters_enc,
		          is_active, effective_from, effective_until, created_by, created_at, updated_at`,
		p.RuleType, p.Name, p.Description, p.Country, p.Region, p.ParametersEnc,
		p.EffectiveFrom, p.EffectiveUntil, p.CreatedBy,
	)
	return scanRuleRow(row)
}

func (q *Queries) ListRules(ctx context.Context, country string, activeOnly bool) ([]*models.CoordinationRuleRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, rule_type, name, description, country, region, parameters_enc,
		       is_active, effective_from, effective_until, created_by, created_at, updated_at
		FROM coordination_rules
		WHERE ($1 = '' OR country = $1)
		  AND (NOT $2 OR is_active = TRUE)
		ORDER BY created_at DESC`,
		country, activeOnly,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.CoordinationRuleRow
	for rows.Next() {
		r, err := scanRuleRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *Queries) DeactivateRule(ctx context.Context, id uuid.UUID) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE coordination_rules SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func scanRuleRow(scanner interface{ Scan(...any) error }) (*models.CoordinationRuleRow, error) {
	var r models.CoordinationRuleRow
	err := scanner.Scan(
		&r.ID, &r.RuleType, &r.Name, &r.Description, &r.Country, &r.Region,
		&r.ParametersEnc, &r.IsActive, &r.EffectiveFrom, &r.EffectiveUntil,
		&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("rule not found")
	}
	return &r, err
}

// ─── Governance Alerts ────────────────────────────────────────────────────────

type CreateAlertParams struct {
	RuleID      *uuid.UUID
	Severity    models.AlertSeverity
	Title       string
	Description string
	Country     string
	Region      *string
	MetadataEnc *string
	RaisedBy    uuid.UUID
}

func (q *Queries) CreateAlert(ctx context.Context, p CreateAlertParams) (*models.GovernanceAlertRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO governance_alerts
			(rule_id, severity, title, description, country, region, metadata_enc, raised_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, rule_id, severity, status, title, description, country, region,
		          metadata_enc, raised_by, acknowledged_by, resolved_by, resolved_at,
		          created_at, updated_at`,
		p.RuleID, p.Severity, p.Title, p.Description, p.Country, p.Region, p.MetadataEnc, p.RaisedBy,
	)
	return scanAlertRow(row)
}

type ListAlertsParams struct {
	Country  string
	Severity *models.AlertSeverity
	Status   *models.AlertStatus
	Page     int
	Limit    int
}

func (q *Queries) ListAlerts(ctx context.Context, p ListAlertsParams) ([]*models.GovernanceAlertRow, error) {
	offset := (p.Page - 1) * p.Limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, rule_id, severity, status, title, description, country, region,
		       metadata_enc, raised_by, acknowledged_by, resolved_by, resolved_at,
		       created_at, updated_at
		FROM governance_alerts
		WHERE ($1 = '' OR country = $1)
		  AND ($2::TEXT IS NULL OR severity = $2)
		  AND ($3::TEXT IS NULL OR status = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`,
		p.Country, p.Severity, p.Status, p.Limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.GovernanceAlertRow
	for rows.Next() {
		r, err := scanAlertRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateAlertStatus(ctx context.Context, id uuid.UUID, status models.AlertStatus, actorID uuid.UUID) (*models.GovernanceAlertRow, error) {
	now := time.Now()
	var resolvedAt *time.Time
	if status == models.AlertResolved {
		resolvedAt = &now
	}

	row := q.pool.QueryRow(ctx, `
		UPDATE governance_alerts SET
			status          = $2,
			acknowledged_by = CASE WHEN $2 = 'acknowledged' THEN $3 ELSE acknowledged_by END,
			resolved_by     = CASE WHEN $2 = 'resolved' THEN $3 ELSE resolved_by END,
			resolved_at     = COALESCE($4, resolved_at),
			updated_at      = NOW()
		WHERE id = $1
		RETURNING id, rule_id, severity, status, title, description, country, region,
		          metadata_enc, raised_by, acknowledged_by, resolved_by, resolved_at,
		          created_at, updated_at`,
		id, status, actorID, resolvedAt,
	)
	return scanAlertRow(row)
}

func scanAlertRow(scanner interface{ Scan(...any) error }) (*models.GovernanceAlertRow, error) {
	var a models.GovernanceAlertRow
	err := scanner.Scan(
		&a.ID, &a.RuleID, &a.Severity, &a.Status, &a.Title, &a.Description,
		&a.Country, &a.Region, &a.MetadataEnc, &a.RaisedBy,
		&a.AcknowledgedBy, &a.ResolvedBy, &a.ResolvedAt,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("alert not found")
	}
	return &a, err
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

type InsertAuditParams struct {
	ResourceType string
	ResourceID   uuid.UUID
	Action       models.AuditAction
	AccessorID   uuid.UUID
	AccessorRole string
	IPAddress    string
	RequestID    string
	Changes      interface{}
}

func (q *Queries) InsertAuditLog(ctx context.Context, p InsertAuditParams) error {
	var changesJSON []byte
	if p.Changes != nil {
		var err error
		changesJSON, err = json.Marshal(p.Changes)
		if err != nil {
			return err
		}
	}
	_, err := q.pool.Exec(ctx, `
		INSERT INTO governance_audit_log
			(resource_type, resource_id, action,
			 accessor_id, accessor_role, ip_address, request_id, changes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.ResourceType, p.ResourceID, p.Action,
		p.AccessorID, p.AccessorRole, p.IPAddress, p.RequestID, changesJSON,
	)
	return err
}
