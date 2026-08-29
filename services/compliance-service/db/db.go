package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/compliance-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreatePermit(ctx context.Context, p models.TransitPermit) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO transit_permits(id,permit_no,vehicle_id,driver_id,permit_type,status,issued_by,country,route_restriction,max_weight_kg,valid_from,valid_until,notes,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		p.ID,p.PermitNo,p.VehicleID,p.DriverID,p.PermitType,p.Status,p.IssuedBy,p.Country,p.RouteRestriction,p.MaxWeightKg,p.ValidFrom,p.ValidUntil,p.Notes,p.CreatedAt,p.UpdatedAt)
	return err
}

func (q *Queries) GetPermit(ctx context.Context, id uuid.UUID) (*models.TransitPermit, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,permit_no,vehicle_id,driver_id,permit_type,status,issued_by,country,route_restriction,max_weight_kg,valid_from,valid_until,notes,created_at,updated_at FROM transit_permits WHERE id=$1`, id)
	var p models.TransitPermit
	err := row.Scan(&p.ID,&p.PermitNo,&p.VehicleID,&p.DriverID,&p.PermitType,&p.Status,&p.IssuedBy,&p.Country,&p.RouteRestriction,&p.MaxWeightKg,&p.ValidFrom,&p.ValidUntil,&p.Notes,&p.CreatedAt,&p.UpdatedAt)
	return &p, err
}

type ListPermitsParams struct{ VehicleID *uuid.UUID; Country *string; Status *models.PermitStatus; Page, Limit int }

func (q *Queries) ListPermits(ctx context.Context, p ListPermitsParams) ([]models.TransitPermit, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.VehicleID != nil { where += fmt.Sprintf(" AND vehicle_id=$%d",n); args=append(args,*p.VehicleID); n++ }
	if p.Country != nil { where += fmt.Sprintf(" AND country=$%d",n); args=append(args,*p.Country); n++ }
	if p.Status != nil { where += fmt.Sprintf(" AND status=$%d",n); args=append(args,*p.Status); n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT id,permit_no,vehicle_id,driver_id,permit_type,status,issued_by,country,route_restriction,max_weight_kg,valid_from,valid_until,notes,created_at,updated_at FROM transit_permits %s ORDER BY valid_until ASC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.TransitPermit
	for rows.Next() {
		var permit models.TransitPermit
		if err := rows.Scan(&permit.ID,&permit.PermitNo,&permit.VehicleID,&permit.DriverID,&permit.PermitType,&permit.Status,&permit.IssuedBy,&permit.Country,&permit.RouteRestriction,&permit.MaxWeightKg,&permit.ValidFrom,&permit.ValidUntil,&permit.Notes,&permit.CreatedAt,&permit.UpdatedAt); err != nil { return nil, err }
		result = append(result, permit)
	}
	return result, rows.Err()
}

func (q *Queries) UpdatePermitStatus(ctx context.Context, id uuid.UUID, status models.PermitStatus, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE transit_permits SET status=$1,updated_at=$2 WHERE id=$3`, status,now,id)
	return err
}

func (q *Queries) CreateBorderCrossing(ctx context.Context, b models.BorderCrossing) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO border_crossings(id,vehicle_id,driver_id,from_country,to_country,border_post,cargo_desc,gross_weight_kg,crossed_at,exit_permit_no,entry_permit_no,notes,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		b.ID,b.VehicleID,b.DriverID,b.FromCountry,b.ToCountry,b.BorderPost,b.CargoDesc,b.GrossWeightKg,b.CrossedAt,b.ExitPermitNo,b.EntryPermitNo,b.Notes,b.CreatedAt)
	return err
}

func (q *Queries) ListBorderCrossings(ctx context.Context, vehicleID uuid.UUID) ([]models.BorderCrossing, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,vehicle_id,driver_id,from_country,to_country,border_post,cargo_desc,gross_weight_kg,crossed_at,exit_permit_no,entry_permit_no,notes,created_at FROM border_crossings WHERE vehicle_id=$1 ORDER BY crossed_at DESC LIMIT 50`, vehicleID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.BorderCrossing
	for rows.Next() {
		var b models.BorderCrossing
		if err := rows.Scan(&b.ID,&b.VehicleID,&b.DriverID,&b.FromCountry,&b.ToCountry,&b.BorderPost,&b.CargoDesc,&b.GrossWeightKg,&b.CrossedAt,&b.ExitPermitNo,&b.EntryPermitNo,&b.Notes,&b.CreatedAt); err != nil { return nil, err }
		result = append(result, b)
	}
	return result, rows.Err()
}

func (q *Queries) CreateWeightCheck(ctx context.Context, w models.WeightCheck) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO weight_checks(id,vehicle_id,country,check_station,gross_weight_kg,legal_limit_kg,is_compliant,fine_amount,currency,checked_at,notes,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		w.ID,w.VehicleID,w.Country,w.CheckStation,w.GrossWeightKg,w.LegalLimitKg,w.IsCompliant,w.FineAmount,w.Currency,w.CheckedAt,w.Notes,w.CreatedAt)
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.ComplianceAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO compliance_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}
