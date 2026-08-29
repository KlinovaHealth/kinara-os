package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/telemedicine-service/models"
)

var ErrNotFound = errors.New("db: record not found")

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RegisterDoctor(ctx context.Context, d models.Doctor) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO doctors (id,clinic_id,full_name,specialization,license_number,is_available,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$7)`,
		d.ID, d.ClinicID, d.FullName, d.Specialization, d.LicenseNumber, d.IsAvailable, d.CreatedAt)
	return err
}

func (q *Queries) ListAvailableDoctors(ctx context.Context) ([]models.Doctor, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,clinic_id,full_name,specialization,license_number,is_available,created_at
		 FROM doctors WHERE is_available=true ORDER BY full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []models.Doctor
	for rows.Next() {
		var d models.Doctor
		if err := rows.Scan(&d.ID, &d.ClinicID, &d.FullName, &d.Specialization, &d.LicenseNumber, &d.IsAvailable, &d.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (q *Queries) SetDoctorAvailability(ctx context.Context, doctorID uuid.UUID, available bool) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE doctors SET is_available=$1, updated_at=$2 WHERE id=$3`,
		available, time.Now().UTC(), doctorID)
	return err
}

func (q *Queries) CreateConsultation(ctx context.Context, c models.Consultation) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO consultations
		 (id,consult_ref,patient_id,doctor_id,clinic_id,type,status,chief_complaint,scheduled_at,cost_usd,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)`,
		c.ID, c.ConsultRef, c.PatientID, c.DoctorID, c.ClinicID, c.Type,
		c.Status, c.ChiefComplaint, c.ScheduledAt, c.CostUSD, c.CreatedAt)
	return err
}

func (q *Queries) GetConsultation(ctx context.Context, id uuid.UUID) (*models.Consultation, error) {
	var c models.Consultation
	err := q.pool.QueryRow(ctx,
		`SELECT id,consult_ref,patient_id,doctor_id,clinic_id,type,status,chief_complaint,
		        scheduled_at,started_at,completed_at,duration_minutes,cost_usd,created_at,updated_at
		 FROM consultations WHERE id=$1`, id).
		Scan(&c.ID, &c.ConsultRef, &c.PatientID, &c.DoctorID, &c.ClinicID, &c.Type,
			&c.Status, &c.ChiefComplaint, &c.ScheduledAt, &c.StartedAt, &c.CompletedAt,
			&c.DurationMinutes, &c.CostUSD, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &c, nil
}

func (q *Queries) ListConsultations(ctx context.Context, patientID *uuid.UUID, doctorID *uuid.UUID, limit int) ([]models.Consultation, error) {
	query := `SELECT id,consult_ref,patient_id,doctor_id,clinic_id,type,status,chief_complaint,
	                 scheduled_at,started_at,completed_at,duration_minutes,cost_usd,created_at,updated_at
	          FROM consultations WHERE 1=1`
	args := []interface{}{}
	i := 1
	if patientID != nil {
		query += fmt.Sprintf(" AND patient_id=$%d", i)
		args = append(args, *patientID)
		i++
	}
	if doctorID != nil {
		query += fmt.Sprintf(" AND doctor_id=$%d", i)
		args = append(args, *doctorID)
		i++
	}
	query += fmt.Sprintf(" ORDER BY scheduled_at DESC LIMIT $%d", i)
	args = append(args, limit)

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cs []models.Consultation
	for rows.Next() {
		var c models.Consultation
		if err := rows.Scan(&c.ID, &c.ConsultRef, &c.PatientID, &c.DoctorID, &c.ClinicID, &c.Type,
			&c.Status, &c.ChiefComplaint, &c.ScheduledAt, &c.StartedAt, &c.CompletedAt,
			&c.DurationMinutes, &c.CostUSD, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cs = append(cs, c)
	}
	return cs, nil
}

func (q *Queries) StartConsultation(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE consultations SET status='in_progress', started_at=COALESCE(started_at,$1), updated_at=$2 WHERE id=$3`,
		now, now, id)
	return err
}

func (q *Queries) CompleteConsultation(ctx context.Context, id uuid.UUID, durationMinutes int, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE consultations SET status='completed', completed_at=COALESCE(completed_at,$1),
		 duration_minutes=COALESCE(duration_minutes,$2), updated_at=$3 WHERE id=$4`,
		now, durationMinutes, now, id)
	return err
}

func (q *Queries) IssuePrescription(ctx context.Context, p models.Prescription, instructionsEnc string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO prescriptions (id,consultation_id,patient_id,doctor_id,medication_name,dosage,frequency_days,instructions_enc,issued_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.ConsultationID, p.PatientID, p.DoctorID,
		p.Medication, p.Dosage, p.FrequencyDays, instructionsEnc, p.IssuedAt)
	return err
}

func (q *Queries) GetPrescription(ctx context.Context, consultationID uuid.UUID) (*models.Prescription, string, error) {
	var p models.Prescription
	var instEnc string
	err := q.pool.QueryRow(ctx,
		`SELECT id,consultation_id,patient_id,doctor_id,medication_name,dosage,frequency_days,instructions_enc,issued_at
		 FROM prescriptions WHERE consultation_id=$1 ORDER BY issued_at DESC LIMIT 1`, consultationID).
		Scan(&p.ID, &p.ConsultationID, &p.PatientID, &p.DoctorID,
			&p.Medication, &p.Dosage, &p.FrequencyDays, &instEnc, &p.IssuedAt)
	if err != nil {
		return nil, "", ErrNotFound
	}
	return &p, instEnc, nil
}

func (q *Queries) SaveRecording(ctx context.Context, r models.RecordingMetadata) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO recording_metadata (id,consultation_id,storage_path,duration_seconds,created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		r.ID, r.ConsultationID, r.StoragePath, r.DurationSeconds, r.CreatedAt)
	return err
}

func (q *Queries) GetRecording(ctx context.Context, consultationID uuid.UUID) (*models.RecordingMetadata, error) {
	var r models.RecordingMetadata
	err := q.pool.QueryRow(ctx,
		`SELECT id,consultation_id,storage_path,duration_seconds,created_at
		 FROM recording_metadata WHERE consultation_id=$1`, consultationID).
		Scan(&r.ID, &r.ConsultationID, &r.StoragePath, &r.DurationSeconds, &r.CreatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &r, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.TelemedicineAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO telemedicine_audit_log (id,consultation_id,actor_id,action,detail,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.ConsultationID, l.ActorID, l.Action, l.Detail, l.CreatedAt)
	return err
}
