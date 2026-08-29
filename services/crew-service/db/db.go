package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/crew-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RegisterCrew(ctx context.Context, c models.CrewMember) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO crew_members (id,crew_ref,full_name,nationality,passport_number,rank,vessel_id,is_active,tenant_id,joined_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		c.ID, c.CrewRef, c.FullName, c.Nationality, c.PassportNumber, c.Rank,
		c.VesselID, c.IsActive, c.TenantID, c.JoinedAt, c.UpdatedAt)
	return err
}

func (q *Queries) GetCrew(ctx context.Context, id uuid.UUID) (*models.CrewMember, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id,crew_ref,full_name,nationality,passport_number,rank,vessel_id,is_active,tenant_id,joined_at,updated_at
		 FROM crew_members WHERE id=$1`, id)
	var c models.CrewMember
	err := row.Scan(&c.ID, &c.CrewRef, &c.FullName, &c.Nationality, &c.PassportNumber, &c.Rank,
		&c.VesselID, &c.IsActive, &c.TenantID, &c.JoinedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (q *Queries) ListByVessel(ctx context.Context, vesselID uuid.UUID) ([]models.CrewMember, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,crew_ref,full_name,nationality,passport_number,rank,vessel_id,is_active,tenant_id,joined_at,updated_at
		 FROM crew_members WHERE vessel_id=$1 AND is_active=true ORDER BY rank`, vesselID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CrewMember
	for rows.Next() {
		var c models.CrewMember
		if err := rows.Scan(&c.ID, &c.CrewRef, &c.FullName, &c.Nationality, &c.PassportNumber, &c.Rank,
			&c.VesselID, &c.IsActive, &c.TenantID, &c.JoinedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (q *Queries) AssignVessel(ctx context.Context, crewID, vesselID uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE crew_members SET vessel_id=$1, updated_at=$2 WHERE id=$3`, vesselID, now, crewID)
	return err
}

func (q *Queries) AddCertification(ctx context.Context, cert models.CrewCertification) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO crew_certifications (id,crew_id,cert_type,cert_number,issued_by,issued_at,expires_at,status,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		cert.ID, cert.CrewID, cert.CertType, cert.CertNumber, cert.IssuedBy,
		cert.IssuedAt, cert.ExpiresAt, cert.Status, time.Now().UTC())
	return err
}

func (q *Queries) ListCertifications(ctx context.Context, crewID uuid.UUID) ([]models.CrewCertification, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,crew_id,cert_type,cert_number,issued_by,issued_at,expires_at,status,created_at
		 FROM crew_certifications WHERE crew_id=$1 ORDER BY expires_at`, crewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CrewCertification
	for rows.Next() {
		var c models.CrewCertification
		if err := rows.Scan(&c.ID, &c.CrewID, &c.CertType, &c.CertNumber, &c.IssuedBy,
			&c.IssuedAt, &c.ExpiresAt, &c.Status, new(time.Time)); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
