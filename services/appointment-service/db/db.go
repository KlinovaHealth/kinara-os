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

func (q *Queries) CreateAppointment(ctx context.Context, a models.Appointment) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO appointments (id,appointment_ref,patient_id,doctor_id,clinic_id,scheduled_at,duration_min,type,status,notes,tenant_id,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		a.ID, a.AppointmentRef, a.PatientID, a.DoctorID, a.ClinicID,
		a.ScheduledAt, a.DurationMin, a.Type, a.Status, a.Notes,
		a.TenantID, a.CreatedAt, a.UpdatedAt)
	return err
}

func (q *Queries) GetAppointment(ctx context.Context, id uuid.UUID) (*models.Appointment, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT id,appointment_ref,patient_id,doctor_id,clinic_id,scheduled_at,duration_min,type,status,notes,tenant_id,created_at,updated_at
		 FROM appointments WHERE id=$1`, id)
	var a models.Appointment
	err := row.Scan(&a.ID, &a.AppointmentRef, &a.PatientID, &a.DoctorID, &a.ClinicID,
		&a.ScheduledAt, &a.DurationMin, &a.Type, &a.Status, &a.Notes,
		&a.TenantID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (q *Queries) ListAppointments(ctx context.Context, patientID, doctorID *uuid.UUID, tenantID string, limit int) ([]models.Appointment, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,appointment_ref,patient_id,doctor_id,clinic_id,scheduled_at,duration_min,type,status,notes,tenant_id,created_at,updated_at
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
	var out []models.Appointment
	for rows.Next() {
		var a models.Appointment
		if err := rows.Scan(&a.ID, &a.AppointmentRef, &a.PatientID, &a.DoctorID, &a.ClinicID,
			&a.ScheduledAt, &a.DurationMin, &a.Type, &a.Status, &a.Notes,
			&a.TenantID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (q *Queries) UpdateStatus(ctx context.Context, id uuid.UUID, status models.AppointmentStatus, notes string, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE appointments SET status=$1, notes=COALESCE(NULLIF($2,''), notes), updated_at=$3 WHERE id=$4`,
		status, notes, now, id)
	return err
}
