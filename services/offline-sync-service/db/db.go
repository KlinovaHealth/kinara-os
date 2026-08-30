package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaxCachedRecords = 200
	CacheWindowHours = 72 * time.Hour
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// SyncRecord is a pending write from a device.
type SyncRecord struct {
	ID             uuid.UUID  `json:"id"`
	DeviceID       uuid.UUID  `json:"device_id"`
	IdempotencyKey string     `json:"idempotency_key"`
	PayloadType    string     `json:"payload_type"` // consultation|prescription|referral|vital_signs
	Payload        []byte     `json:"payload"`
	PatientID      uuid.UUID  `json:"patient_id"`
	ClinicID       uuid.UUID  `json:"clinic_id"`
	ReceivedAt     time.Time  `json:"received_at"`
	AppliedAt      *time.Time `json:"applied_at,omitempty"`
	RejectedAt     *time.Time `json:"rejected_at,omitempty"`
	RejectReason   *string    `json:"reject_reason,omitempty"`
}

// PatientRecord is a patient visible to a scoped device pull.
type PatientRecord struct {
	PatientID   uuid.UUID `json:"patient_id"`
	ClinicID    uuid.UUID `json:"clinic_id"`
	LastVisitAt time.Time `json:"last_visit_at"`
	ExpiresAt   time.Time `json:"expires_at"` // now + 72h, informational
}

// DeviceStatus holds revocation/staleness state.
type DeviceStatus struct {
	Revoked      bool
	RevokedAt    *time.Time
	LastSeenAt   *time.Time
}

// GetDeviceStatus returns revocation and last_seen for a device.
func (q *Queries) GetDeviceStatus(ctx context.Context, deviceID uuid.UUID) (*DeviceStatus, error) {
	var s DeviceStatus
	err := q.pool.QueryRow(ctx,
		`SELECT revoked_at IS NOT NULL, revoked_at, last_seen_at
		 FROM devices WHERE id = $1`, deviceID).
		Scan(&s.Revoked, &s.RevokedAt, &s.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// PullPatients returns patients for the clinic seen within the last 72h,
// capped at MaxCachedRecords records. This is the server-enforced cache scope.
func (q *Queries) PullPatients(ctx context.Context, clinicID uuid.UUID) ([]PatientRecord, error) {
	cutoff := time.Now().UTC().Add(-CacheWindowHours)
	expiresAt := time.Now().UTC().Add(CacheWindowHours)

	rows, err := q.pool.Query(ctx, `
		SELECT DISTINCT p.id, v.clinic_id, MAX(v.visit_date) AS last_visit_at
		FROM patients p
		JOIN visits v ON v.patient_id = p.id
		WHERE v.clinic_id = $1
		  AND v.visit_date >= $2
		GROUP BY p.id, v.clinic_id
		ORDER BY last_visit_at DESC
		LIMIT $3`,
		clinicID, cutoff, MaxCachedRecords)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PatientRecord
	for rows.Next() {
		var rec PatientRecord
		if err := rows.Scan(&rec.PatientID, &rec.ClinicID, &rec.LastVisitAt); err != nil {
			return nil, err
		}
		rec.ExpiresAt = expiresAt
		result = append(result, rec)
	}
	return result, rows.Err()
}

// PushExists checks idempotency: returns true if this (device_id, idempotency_key) was already received.
func (q *Queries) PushExists(ctx context.Context, deviceID uuid.UUID, idempotencyKey string) (bool, error) {
	var exists bool
	err := q.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sync_queue WHERE device_id = $1 AND idempotency_key = $2)`,
		deviceID, idempotencyKey).Scan(&exists)
	return exists, err
}

// PatientInClinic checks that patient_id belongs to the given clinic (server-side scope guard).
func (q *Queries) PatientInClinic(ctx context.Context, patientID, clinicID uuid.UUID) (bool, error) {
	var exists bool
	err := q.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM visits WHERE patient_id = $1 AND clinic_id = $2 LIMIT 1
		)`, patientID, clinicID).Scan(&exists)
	return exists, err
}

// InsertSyncRecord inserts a new push record. The sync_queue table has an immutable
// RULE so updates/deletes are silently ignored — idempotency_key prevents re-inserts.
func (q *Queries) InsertSyncRecord(ctx context.Context, rec SyncRecord) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO sync_queue
			(id, device_id, idempotency_key, payload_type, payload, patient_id, clinic_id, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (device_id, idempotency_key) DO NOTHING`,
		rec.ID, rec.DeviceID, rec.IdempotencyKey, rec.PayloadType,
		rec.Payload, rec.PatientID, rec.ClinicID, rec.ReceivedAt)
	return err
}

// MarkApplied marks a sync record as applied.
func (q *Queries) MarkApplied(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE sync_queue SET applied_at = $1 WHERE id = $2 AND applied_at IS NULL`,
		now, id)
	return err
}

// MarkRejected marks a sync record as rejected with a reason.
func (q *Queries) MarkRejected(ctx context.Context, id uuid.UUID, reason string, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE sync_queue SET rejected_at = $1, reject_reason = $2 WHERE id = $3 AND rejected_at IS NULL`,
		now, reason, id)
	return err
}

// GetSyncStatus returns pending, applied, and rejected counts for a device.
func (q *Queries) GetSyncStatus(ctx context.Context, deviceID uuid.UUID) (pending, applied, rejected int, err error) {
	err = q.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE applied_at IS NULL AND rejected_at IS NULL),
			COUNT(*) FILTER (WHERE applied_at IS NOT NULL),
			COUNT(*) FILTER (WHERE rejected_at IS NOT NULL)
		FROM sync_queue WHERE device_id = $1`, deviceID).
		Scan(&pending, &applied, &rejected)
	return
}

// UpdateLastSeen bumps last_seen_at for a device after a successful sync.
func (q *Queries) UpdateLastSeen(ctx context.Context, deviceID uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE devices SET last_seen_at = $1 WHERE id = $2`, now, deviceID)
	return err
}
