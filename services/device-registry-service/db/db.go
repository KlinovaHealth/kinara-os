package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

type Device struct {
	ID                uuid.UUID  `db:"id"`
	DeviceName        string     `db:"device_name"`
	ClinicID          uuid.UUID  `db:"clinic_id"`
	AssignedStaffID   *uuid.UUID `db:"assigned_staff_id"`
	DeviceSecretHash  string     `db:"device_secret_hash"`
	EnrolledAt        time.Time  `db:"enrolled_at"`
	LastSeenAt        *time.Time `db:"last_seen_at"`
	RevokedAt         *time.Time `db:"revoked_at"`
	RevokedReason     *string    `db:"revoked_reason"`
}

func (q *Queries) EnrollDevice(ctx context.Context, d Device) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO devices
			(id, device_name, clinic_id, assigned_staff_id, device_secret_hash, enrolled_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		d.ID, d.DeviceName, d.ClinicID, d.AssignedStaffID, d.DeviceSecretHash, d.EnrolledAt)
	return err
}

func (q *Queries) GetDevice(ctx context.Context, id uuid.UUID) (*Device, error) {
	var d Device
	err := q.pool.QueryRow(ctx, `
		SELECT id, device_name, clinic_id, assigned_staff_id, device_secret_hash,
		       enrolled_at, last_seen_at, revoked_at, revoked_reason
		FROM devices WHERE id = $1`, id).
		Scan(&d.ID, &d.DeviceName, &d.ClinicID, &d.AssignedStaffID, &d.DeviceSecretHash,
			&d.EnrolledAt, &d.LastSeenAt, &d.RevokedAt, &d.RevokedReason)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (q *Queries) ListDevices(ctx context.Context, clinicID *uuid.UUID) ([]Device, error) {
	query := `SELECT id, device_name, clinic_id, assigned_staff_id, device_secret_hash,
		enrolled_at, last_seen_at, revoked_at, revoked_reason FROM devices`
	var args []interface{}
	if clinicID != nil {
		query += " WHERE clinic_id = $1"
		args = append(args, *clinicID)
	}
	query += " ORDER BY enrolled_at DESC"

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.DeviceName, &d.ClinicID, &d.AssignedStaffID, &d.DeviceSecretHash,
			&d.EnrolledAt, &d.LastSeenAt, &d.RevokedAt, &d.RevokedReason); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (q *Queries) RevokeDevice(ctx context.Context, id uuid.UUID, reason string, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE devices SET revoked_at = $1, revoked_reason = $2 WHERE id = $3 AND revoked_at IS NULL`,
		now, reason, id)
	return err
}

func (q *Queries) UpdateLastSeen(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE devices SET last_seen_at = $1 WHERE id = $2`, now, id)
	return err
}

func (q *Queries) IsRevoked(ctx context.Context, id uuid.UUID) (bool, error) {
	var revokedAt *time.Time
	err := q.pool.QueryRow(ctx, `SELECT revoked_at FROM devices WHERE id = $1`, id).Scan(&revokedAt)
	if err != nil {
		return false, err
	}
	return revokedAt != nil, nil
}

// StalenessThreshold is how long a device can go without syncing before its cache is wiped.
const StalenessThreshold = 7 * 24 * time.Hour

func (q *Queries) IsStale(ctx context.Context, id uuid.UUID) (bool, error) {
	var lastSeen *time.Time
	err := q.pool.QueryRow(ctx, `SELECT last_seen_at FROM devices WHERE id = $1`, id).Scan(&lastSeen)
	if err != nil {
		return false, err
	}
	if lastSeen == nil {
		return false, nil // never synced — not stale yet
	}
	return time.Since(*lastSeen) > StalenessThreshold, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, deviceID uuid.UUID, event string, actorID *uuid.UUID, ip string) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO device_audit_log (id, device_id, event, actor_id, ip_address, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())`,
		deviceID, event, actorID, ip)
	return err
}
