package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/referral-service/models"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// ─── Create referral ──────────────────────────────────────────────────────────

func (q *Queries) CreateReferral(ctx context.Context, row models.ReferralRow) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO referrals
			(id, patient_id, from_clinic_id, to_clinic_id, from_clinician_id,
			 reason_enc, patient_name_enc, urgency, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		row.ID, row.PatientID, row.FromClinicID, row.ToClinicID, row.FromClinicianID,
		row.ReasonEnc, row.PatientNameEnc, row.Urgency, row.Status,
		row.CreatedAt, row.UpdatedAt,
	)
	return err
}

// ─── Get referral ─────────────────────────────────────────────────────────────

func (q *Queries) GetReferral(ctx context.Context, id uuid.UUID) (*models.ReferralRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, patient_id, from_clinic_id, to_clinic_id, from_clinician_id,
		       to_clinician_id, reason_enc, patient_name_enc, urgency, status,
		       follow_up_date, follow_up_notes_enc, accepted_at, completed_at,
		       rejected_at, rejection_reason_enc, created_at, updated_at
		FROM referrals WHERE id = $1`, id)
	return scanReferralRow(row)
}

// ─── List referrals ───────────────────────────────────────────────────────────

type ListReferralsParams struct {
	PatientID *uuid.UUID
	ClinicID  *uuid.UUID // matches from_clinic_id OR to_clinic_id
	Status    *models.ReferralStatus
	Page      int
	Limit     int
}

func (q *Queries) ListReferrals(ctx context.Context, p ListReferralsParams) ([]models.ReferralRow, error) {
	where, args := buildReferralWhere(p)
	offset := (p.Page - 1) * p.Limit
	args = append(args, p.Limit, offset)

	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, patient_id, from_clinic_id, to_clinic_id, from_clinician_id,
		       to_clinician_id, reason_enc, patient_name_enc, urgency, status,
		       follow_up_date, follow_up_notes_enc, accepted_at, completed_at,
		       rejected_at, rejection_reason_enc, created_at, updated_at
		FROM referrals %s
		ORDER BY
			CASE urgency
				WHEN 'emergency'  THEN 1
				WHEN 'urgent'     THEN 2
				WHEN 'semi_urgent' THEN 3
				ELSE 4
			END,
			created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ReferralRow
	for rows.Next() {
		r, err := scanReferralRowFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *r)
	}
	return result, rows.Err()
}

func (q *Queries) CountReferrals(ctx context.Context, p ListReferralsParams) (int, error) {
	where, args := buildReferralWhere(p)
	var total int
	err := q.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM referrals %s`, where), args...,
	).Scan(&total)
	return total, err
}

func buildReferralWhere(p ListReferralsParams) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}
	n := 1
	if p.PatientID != nil {
		where += fmt.Sprintf(" AND patient_id = $%d", n)
		args = append(args, *p.PatientID)
		n++
	}
	if p.ClinicID != nil {
		where += fmt.Sprintf(" AND (from_clinic_id = $%d OR to_clinic_id = $%d)", n, n+1)
		args = append(args, *p.ClinicID, *p.ClinicID)
		n += 2
	}
	if p.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, *p.Status)
	}
	return where, args
}

// ─── Update referral status ───────────────────────────────────────────────────

type UpdateStatusParams struct {
	ID              uuid.UUID
	Status          models.ReferralStatus
	ToClinicianID   *uuid.UUID
	RejectionReason *string
	Now             time.Time
}

func (q *Queries) UpdateReferralStatus(ctx context.Context, p UpdateStatusParams) error {
	var acceptedAt, completedAt, rejectedAt *time.Time
	switch p.Status {
	case models.ReferralAccepted:
		acceptedAt = &p.Now
	case models.ReferralCompleted:
		completedAt = &p.Now
	case models.ReferralRejected:
		rejectedAt = &p.Now
	}
	_, err := q.pool.Exec(ctx, `
		UPDATE referrals SET
			status = $1,
			to_clinician_id = COALESCE($2, to_clinician_id),
			rejection_reason_enc = COALESCE($3, rejection_reason_enc),
			accepted_at = COALESCE($4, accepted_at),
			completed_at = COALESCE($5, completed_at),
			rejected_at = COALESCE($6, rejected_at),
			updated_at = $7
		WHERE id = $8`,
		p.Status, p.ToClinicianID, p.RejectionReason,
		acceptedAt, completedAt, rejectedAt,
		p.Now, p.ID,
	)
	return err
}

// ─── Follow-up ────────────────────────────────────────────────────────────────

type FollowUpParams struct {
	ID           uuid.UUID
	FollowUpDate time.Time
	NotesEnc     *string
	Now          time.Time
}

func (q *Queries) ScheduleFollowUp(ctx context.Context, p FollowUpParams) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE referrals SET
			follow_up_date = $1,
			follow_up_notes_enc = COALESCE($2, follow_up_notes_enc),
			updated_at = $3
		WHERE id = $4`,
		p.FollowUpDate, p.NotesEnc, p.Now, p.ID,
	)
	return err
}

// ─── Notes ────────────────────────────────────────────────────────────────────

func (q *Queries) CreateNote(ctx context.Context, row models.ReferralNoteRow) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO referral_notes (id, referral_id, note_enc, created_by_user_id, created_at)
		VALUES ($1,$2,$3,$4,$5)`,
		row.ID, row.ReferralID, row.NoteEnc, row.CreatedByUserID, row.CreatedAt,
	)
	return err
}

func (q *Queries) ListNotes(ctx context.Context, referralID uuid.UUID) ([]models.ReferralNoteRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, referral_id, note_enc, created_by_user_id, created_at
		FROM referral_notes WHERE referral_id = $1
		ORDER BY created_at ASC`, referralID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ReferralNoteRow
	for rows.Next() {
		var n models.ReferralNoteRow
		if err := rows.Scan(&n.ID, &n.ReferralID, &n.NoteEnc, &n.CreatedByUserID, &n.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

// ─── History ──────────────────────────────────────────────────────────────────

func (q *Queries) InsertHistory(ctx context.Context, h models.ReferralHistory) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO referral_history
			(id, referral_id, status_before, status_after, changed_by_user_id, changed_by_role, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		h.ID, h.ReferralID, h.StatusBefore, h.StatusAfter,
		h.ChangedByUserID, h.ChangedByRole, h.Notes, h.CreatedAt,
	)
	return err
}

func (q *Queries) ListHistory(ctx context.Context, referralID uuid.UUID) ([]models.ReferralHistory, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, referral_id, status_before, status_after, changed_by_user_id, changed_by_role, notes, created_at
		FROM referral_history WHERE referral_id = $1
		ORDER BY created_at ASC`, referralID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ReferralHistory
	for rows.Next() {
		var h models.ReferralHistory
		if err := rows.Scan(&h.ID, &h.ReferralID, &h.StatusBefore, &h.StatusAfter,
			&h.ChangedByUserID, &h.ChangedByRole, &h.Notes, &h.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (q *Queries) InsertAuditLog(ctx context.Context, log models.ReferralAuditLog) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO referral_audit_log (id, referral_id, user_id, action, resource, ip_address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.ID, log.ReferralID, log.UserID, log.Action, log.Resource, log.IPAddress, log.CreatedAt,
	)
	return err
}

// ─── Row scanners ─────────────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...any) error
}

func scanReferralRow(row scannable) (*models.ReferralRow, error) {
	var r models.ReferralRow
	err := row.Scan(
		&r.ID, &r.PatientID, &r.FromClinicID, &r.ToClinicID, &r.FromClinicianID,
		&r.ToClinicianID, &r.ReasonEnc, &r.PatientNameEnc, &r.Urgency, &r.Status,
		&r.FollowUpDate, &r.FollowUpNotesEnc, &r.AcceptedAt, &r.CompletedAt,
		&r.RejectedAt, &r.RejectionReasonEnc, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

type pgxRows interface {
	Scan(dest ...any) error
}

func scanReferralRowFromRows(rows pgxRows) (*models.ReferralRow, error) {
	return scanReferralRow(rows)
}
