package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/health-compliance-service/models"
)

var ErrNotFound = errors.New("db: record not found")

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) InsertAuditEntry(ctx context.Context, e models.AuditEntry) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO audit_entries (id,entry_ref,service,resource_type,resource_id,actor_id,actor_role,action,detail,ip_address,signature,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		e.ID, e.EntryRef, e.Service, e.ResourceType, e.ResourceID,
		e.ActorID, e.ActorRole, e.Action, e.Detail, e.IPAddress, e.Signature, e.CreatedAt)
	return err
}

type ListAuditParams struct {
	Service    *string
	ActorID    *uuid.UUID
	ResourceID *uuid.UUID
	Since      *time.Time
	Page       int
	Limit      int
}

func (q *Queries) ListAuditEntries(ctx context.Context, p ListAuditParams) ([]models.AuditEntry, error) {
	query := `SELECT id,entry_ref,service,resource_type,resource_id,actor_id,actor_role,action,detail,ip_address,signature,created_at
	          FROM audit_entries WHERE 1=1`
	args := []interface{}{}
	i := 1
	if p.Service != nil {
		query += fmt.Sprintf(" AND service=$%d", i)
		args = append(args, *p.Service)
		i++
	}
	if p.ActorID != nil {
		query += fmt.Sprintf(" AND actor_id=$%d", i)
		args = append(args, *p.ActorID)
		i++
	}
	if p.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id=$%d", i)
		args = append(args, *p.ResourceID)
		i++
	}
	if p.Since != nil {
		query += fmt.Sprintf(" AND created_at>=$%d", i)
		args = append(args, *p.Since)
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
	var result []models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		if err := rows.Scan(&e.ID, &e.EntryRef, &e.Service, &e.ResourceType, &e.ResourceID,
			&e.ActorID, &e.ActorRole, &e.Action, &e.Detail, &e.IPAddress, &e.Signature, &e.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (q *Queries) GetAuditEntry(ctx context.Context, id uuid.UUID) (*models.AuditEntry, error) {
	var e models.AuditEntry
	err := q.pool.QueryRow(ctx,
		`SELECT id,entry_ref,service,resource_type,resource_id,actor_id,actor_role,action,detail,ip_address,signature,created_at
		 FROM audit_entries WHERE id=$1`, id).
		Scan(&e.ID, &e.EntryRef, &e.Service, &e.ResourceType, &e.ResourceID,
			&e.ActorID, &e.ActorRole, &e.Action, &e.Detail, &e.IPAddress, &e.Signature, &e.CreatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &e, nil
}

func (q *Queries) RecordBreachAttempt(ctx context.Context, b models.BreachAttempt) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO breach_attempts (id,service,actor_id,ip_address,reason,detected_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		b.ID, b.Service, b.ActorID, b.IPAddress, b.Reason, b.DetectedAt)
	return err
}

func (q *Queries) ListBreachAttempts(ctx context.Context, unresolvedOnly bool) ([]models.BreachAttempt, error) {
	query := `SELECT id,service,actor_id,ip_address,reason,detected_at,resolved,resolved_at
	          FROM breach_attempts`
	if unresolvedOnly {
		query += " WHERE resolved=false"
	}
	query += " ORDER BY detected_at DESC LIMIT 100"
	rows, err := q.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.BreachAttempt
	for rows.Next() {
		var b models.BreachAttempt
		if err := rows.Scan(&b.ID, &b.Service, &b.ActorID, &b.IPAddress, &b.Reason, &b.DetectedAt, &b.Resolved, &b.ResolvedAt); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, nil
}

func (q *Queries) UpsertEncryptionStatus(ctx context.Context, s models.EncryptionStatus) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO encryption_status (id,service,total_fields,encrypted_fields,algorithm,last_verified_at,is_compliant,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$6)
		 ON CONFLICT (service) DO UPDATE SET
		   total_fields=$3, encrypted_fields=$4, last_verified_at=$6, is_compliant=$7, updated_at=$6`,
		uuid.New(), s.Service, s.TotalFields, s.EncryptedFields, s.Algorithm, s.LastVerifiedAt, s.IsCompliant)
	return err
}

func (q *Queries) ListEncryptionStatus(ctx context.Context) ([]models.EncryptionStatus, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT service,total_fields,encrypted_fields,algorithm,last_verified_at,is_compliant FROM encryption_status ORDER BY service`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EncryptionStatus
	for rows.Next() {
		var s models.EncryptionStatus
		if err := rows.Scan(&s.Service, &s.TotalFields, &s.EncryptedFields, &s.Algorithm, &s.LastVerifiedAt, &s.IsCompliant); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func (q *Queries) SaveComplianceReport(ctx context.Context, r models.ComplianceReport) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO compliance_reports (id,report_ref,standard,country,period_start,period_end,total_events,breach_count,is_compliant,findings,generated_at,generated_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		r.ID, r.ReportRef, r.Standard, r.Country, r.PeriodStart, r.PeriodEnd,
		r.TotalEvents, r.BreachCount, r.IsCompliant, r.Findings, r.GeneratedAt, r.GeneratedBy)
	return err
}

func (q *Queries) CountAuditEvents(ctx context.Context, since, until time.Time) (int, error) {
	var count int
	_ = q.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_entries WHERE created_at BETWEEN $1 AND $2`, since, until).
		Scan(&count)
	return count, nil
}

func (q *Queries) CountBreaches(ctx context.Context, since, until time.Time) (int, error) {
	var count int
	_ = q.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM breach_attempts WHERE detected_at BETWEEN $1 AND $2`, since, until).
		Scan(&count)
	return count, nil
}
