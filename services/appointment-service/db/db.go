package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/appointment-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

// CreateAppointment inserts a new appointment row.
func (q *Queries) CreateAppointment(ctx context.Context, a models.Appointment) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO appointments
		 (id,appointment_ref,patient_id,doctor_id,clinic_id,scheduled_at,duration_min,
		  type,status,notes,reason,tenant_id,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		a.ID, a.AppointmentRef, a.PatientID, a.DoctorID, a.ClinicID,
		a.ScheduledAt, a.DurationMin, a.Type, a.Status, a.Notes, a.Reason,
		a.TenantID, a.CreatedAt, a.UpdatedAt)
	return err
}

// GetAppointment fetches a single appointment by primary key.
func (q *Queries) GetAppointment(ctx context.Context, id uuid.UUID) (*models.Appointment, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id,appointment_ref,patient_id,doctor_id,clinic_id,scheduled_at,duration_min,
		        type,status,notes,COALESCE(reason,''),COALESCE(cancelled_by,''),COALESCE(completed_by,''),
		        tenant_id,created_at,updated_at
		 FROM appointments WHERE id=$1`, id)
	var a models.Appointment
	err := row.Scan(
		&a.ID, &a.AppointmentRef, &a.PatientID, &a.DoctorID, &a.ClinicID,
		&a.ScheduledAt, &a.DurationMin, &a.Type, &a.Status, &a.Notes,
		&a.Reason, &a.CancelledBy, &a.CompletedBy,
		&a.TenantID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAppointments is the original general-purpose list (kept for backwards compat).
func (q *Queries) ListAppointments(ctx context.Context, patientID, doctorID *uuid.UUID, tenantID string, limit int) ([]models.Appointment, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,appointment_ref,patient_id,doctor_id,clinic_id,scheduled_at,duration_min,
		        type,status,notes,COALESCE(reason,''),COALESCE(cancelled_by,''),COALESCE(completed_by,''),
		        tenant_id,created_at,updated_at
		 FROM appointments
		 WHERE tenant_id=$1
		   AND ($2::uuid IS NULL OR patient_id=$2)
		   AND ($3::uuid IS NULL OR doctor_id=$3)
		 ORDER BY scheduled_at DESC LIMIT $4`,
		tenantID, patientID, doctorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAppointments(rows)
}

// UpdateStatus is the original status-only update (kept for backwards compat).
func (q *Queries) UpdateStatus(ctx context.Context, id uuid.UUID, status models.AppointmentStatus, notes string, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE appointments SET status=$1, notes=COALESCE(NULLIF($2,''), notes), updated_at=$3 WHERE id=$4`,
		status, notes, now, id)
	return err
}

// RescheduleAppointment updates scheduled_at and duration_min.
func (q *Queries) RescheduleAppointment(ctx context.Context, id uuid.UUID, newTime time.Time, durationMin int, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE appointments SET scheduled_at=$1, duration_min=$2, updated_at=$3 WHERE id=$4`,
		newTime, durationMin, now, id)
	return err
}

// CancelAppointment sets status to cancelled and records reason/actor.
func (q *Queries) CancelAppointment(ctx context.Context, id uuid.UUID, reason, actorID string, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE appointments
		 SET status=$1, reason=COALESCE(NULLIF($2,''), reason), cancelled_by=$3, updated_at=$4
		 WHERE id=$5`,
		models.StatusCancelled, reason, actorID, now, id)
	return err
}

// CompleteAppointment sets status to completed and records notes/actor.
func (q *Queries) CompleteAppointment(ctx context.Context, id uuid.UUID, notes, actorID string, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE appointments
		 SET status=$1, notes=COALESCE(NULLIF($2,''), notes), completed_by=$3, updated_at=$4
		 WHERE id=$5`,
		models.StatusCompleted, notes, actorID, now, id)
	return err
}

// ListByClinic returns appointments for a clinic, optionally filtered by date and status.
func (q *Queries) ListByClinic(ctx context.Context, clinicID, tenantID string, date *time.Time, status *string, limit int) ([]models.Appointment, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,appointment_ref,patient_id,doctor_id,clinic_id,scheduled_at,duration_min,
		        type,status,notes,COALESCE(reason,''),COALESCE(cancelled_by,''),COALESCE(completed_by,''),
		        tenant_id,created_at,updated_at
		 FROM appointments
		 WHERE clinic_id=$1 AND tenant_id=$2
		   AND ($3::date IS NULL OR scheduled_at::date=$3)
		   AND ($4::text IS NULL OR status=$4)
		 ORDER BY scheduled_at ASC LIMIT $5`,
		clinicID, tenantID, date, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAppointments(rows)
}

// ListByPatient returns all appointments for a patient within a tenant.
func (q *Queries) ListByPatient(ctx context.Context, patientID uuid.UUID, tenantID string, limit int) ([]models.Appointment, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,appointment_ref,patient_id,doctor_id,clinic_id,scheduled_at,duration_min,
		        type,status,notes,COALESCE(reason,''),COALESCE(cancelled_by,''),COALESCE(completed_by,''),
		        tenant_id,created_at,updated_at
		 FROM appointments
		 WHERE patient_id=$1 AND tenant_id=$2
		 ORDER BY scheduled_at DESC LIMIT $3`,
		patientID, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAppointments(rows)
}

// InsertAudit appends an immutable audit log entry.
func (q *Queries) InsertAudit(ctx context.Context, entry models.AuditEntry) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO appointment_audit_log (appointment_id, action, actor_id, old_status, new_status, occurred_at)
		 VALUES ($1,$2,$3,$4,$5,NOW())`,
		entry.AppointmentID, entry.Action, entry.ActorID, entry.OldStatus, entry.NewStatus)
	return err
}

// GetAuditHistory returns the full audit trail for an appointment in chronological order.
func (q *Queries) GetAuditHistory(ctx context.Context, apptID uuid.UUID) ([]models.AuditEntry, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, appointment_id, action, actor_id,
		        COALESCE(old_status,''), COALESCE(new_status,''), occurred_at
		 FROM appointment_audit_log
		 WHERE appointment_id=$1
		 ORDER BY occurred_at ASC, id ASC`,
		apptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		if err := rows.Scan(&e.ID, &e.AppointmentID, &e.Action, &e.ActorID, &e.OldStatus, &e.NewStatus, &e.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// scanAppointments is a shared row-scanning helper.
func scanAppointments(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]models.Appointment, error) {
	var out []models.Appointment
	for rows.Next() {
		var a models.Appointment
		if err := rows.Scan(
			&a.ID, &a.AppointmentRef, &a.PatientID, &a.DoctorID, &a.ClinicID,
			&a.ScheduledAt, &a.DurationMin, &a.Type, &a.Status, &a.Notes,
			&a.Reason, &a.CancelledBy, &a.CompletedBy,
			&a.TenantID, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
