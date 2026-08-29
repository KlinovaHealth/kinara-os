package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/audit-service/models"
)

var ErrNotFound = errors.New("db: record not found")

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) InsertEvent(ctx context.Context, e models.AuditEvent) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO audit_events
		 (id,event_ref,service,pillar,event_type,actor_id,actor_role,resource_id,resource_type,detail,ip_address,tenant_id,signature,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.ID, e.EventRef, e.Service, e.Pillar, e.EventType, e.ActorID, e.ActorRole,
		e.ResourceID, e.ResourceType, e.Detail, e.IPAddress, e.TenantID, e.Signature, e.CreatedAt)
	return err
}

type ListEventsParams struct {
	Service  *string
	Pillar   *string
	ActorID  *uuid.UUID
	TenantID *string
	Since    *time.Time
	Page     int
	Limit    int
}

func (q *Queries) ListEvents(ctx context.Context, p ListEventsParams) ([]models.AuditEvent, error) {
	query := `SELECT id,event_ref,service,pillar,event_type,actor_id,actor_role,resource_id,resource_type,detail,ip_address,tenant_id,signature,created_at
	          FROM audit_events WHERE 1=1`
	args := []interface{}{}
	i := 1
	if p.Service != nil {
		query += fmt.Sprintf(" AND service=$%d", i); args = append(args, *p.Service); i++
	}
	if p.Pillar != nil {
		query += fmt.Sprintf(" AND pillar=$%d", i); args = append(args, *p.Pillar); i++
	}
	if p.ActorID != nil {
		query += fmt.Sprintf(" AND actor_id=$%d", i); args = append(args, *p.ActorID); i++
	}
	if p.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id=$%d", i); args = append(args, *p.TenantID); i++
	}
	if p.Since != nil {
		query += fmt.Sprintf(" AND created_at>=$%d", i); args = append(args, *p.Since); i++
	}
	if p.Limit == 0 { p.Limit = 100 }
	if p.Page < 1 { p.Page = 1 }
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.AuditEvent
	for rows.Next() {
		var e models.AuditEvent
		if err := rows.Scan(&e.ID, &e.EventRef, &e.Service, &e.Pillar, &e.EventType, &e.ActorID, &e.ActorRole,
			&e.ResourceID, &e.ResourceType, &e.Detail, &e.IPAddress, &e.TenantID, &e.Signature, &e.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (q *Queries) CountByPillar(ctx context.Context, since, until time.Time) (map[string]int, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT pillar, COUNT(*) FROM audit_events WHERE created_at BETWEEN $1 AND $2 GROUP BY pillar`, since, until)
	if err != nil { return nil, err }
	defer rows.Close()
	result := map[string]int{}
	for rows.Next() {
		var pillar string
		var count int
		if err := rows.Scan(&pillar, &count); err != nil { return nil, err }
		result[pillar] = count
	}
	return result, nil
}

func (q *Queries) CountByService(ctx context.Context, since, until time.Time) (map[string]int, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT service, COUNT(*) FROM audit_events WHERE created_at BETWEEN $1 AND $2 GROUP BY service`, since, until)
	if err != nil { return nil, err }
	defer rows.Close()
	result := map[string]int{}
	for rows.Next() {
		var service string
		var count int
		if err := rows.Scan(&service, &count); err != nil { return nil, err }
		result[service] = count
	}
	return result, nil
}
