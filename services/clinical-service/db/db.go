package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/clinical-service/models"
)

// Queries wraps a pgxpool.Pool and exposes typed query methods.
type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// ─── Consultations ────────────────────────────────────────────────────────────

type CreateConsultationParams struct {
	PatientID           uuid.UUID
	DoctorID            uuid.UUID
	ConsultationType    models.ConsultationType
	ChiefComplaintEnc   string
	ScheduledAt         *time.Time
	Country             string
	Region              *string
	FacilityID          *uuid.UUID
	CreatedBy           uuid.UUID
}

func (q *Queries) CreateConsultation(ctx context.Context, p CreateConsultationParams) (*models.ConsultationRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO consultations
			(patient_id, doctor_id, consultation_type, chief_complaint_enc,
			 scheduled_at, country, region, facility_id, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, patient_id, doctor_id, status, consultation_type,
		          chief_complaint_enc, scheduled_at, started_at, ended_at,
		          country, region, facility_id, created_by, created_at, updated_at`,
		p.PatientID, p.DoctorID, p.ConsultationType, p.ChiefComplaintEnc,
		p.ScheduledAt, p.Country, p.Region, p.FacilityID, p.CreatedBy,
	)
	return scanConsultationRow(row)
}

func (q *Queries) GetConsultationByID(ctx context.Context, id uuid.UUID) (*models.ConsultationRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, patient_id, doctor_id, status, consultation_type,
		       chief_complaint_enc, scheduled_at, started_at, ended_at,
		       country, region, facility_id, created_by, created_at, updated_at
		FROM consultations WHERE id = $1`, id)
	return scanConsultationRow(row)
}

type ListConsultationsParams struct {
	PatientID *uuid.UUID
	DoctorID  *uuid.UUID
	Status    *models.ConsultationStatus
	Country   string
	Page      int
	Limit     int
}

func (q *Queries) ListConsultations(ctx context.Context, p ListConsultationsParams) ([]*models.ConsultationRow, error) {
	offset := (p.Page - 1) * p.Limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, patient_id, doctor_id, status, consultation_type,
		       chief_complaint_enc, scheduled_at, started_at, ended_at,
		       country, region, facility_id, created_by, created_at, updated_at
		FROM consultations
		WHERE ($1::UUID IS NULL OR patient_id = $1)
		  AND ($2::UUID IS NULL OR doctor_id = $2)
		  AND ($3::TEXT IS NULL OR status = $3)
		  AND ($4 = '' OR country = $4)
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6`,
		p.PatientID, p.DoctorID, p.Status, p.Country, p.Limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.ConsultationRow
	for rows.Next() {
		row, err := scanConsultationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (q *Queries) CountConsultations(ctx context.Context, p ListConsultationsParams) (int64, error) {
	var count int64
	err := q.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM consultations
		WHERE ($1::UUID IS NULL OR patient_id = $1)
		  AND ($2::UUID IS NULL OR doctor_id = $2)
		  AND ($3::TEXT IS NULL OR status = $3)
		  AND ($4 = '' OR country = $4)`,
		p.PatientID, p.DoctorID, p.Status, p.Country,
	).Scan(&count)
	return count, err
}

type UpdateConsultationParams struct {
	ID             uuid.UUID
	Status         *models.ConsultationStatus
	ChiefComplaint *string
	StartedAt      *time.Time
	EndedAt        *time.Time
}

func (q *Queries) UpdateConsultation(ctx context.Context, p UpdateConsultationParams) (*models.ConsultationRow, error) {
	row := q.pool.QueryRow(ctx, `
		UPDATE consultations SET
			status           = COALESCE($2, status),
			chief_complaint_enc = COALESCE($3, chief_complaint_enc),
			started_at       = COALESCE($4, started_at),
			ended_at         = COALESCE($5, ended_at),
			updated_at       = NOW()
		WHERE id = $1
		RETURNING id, patient_id, doctor_id, status, consultation_type,
		          chief_complaint_enc, scheduled_at, started_at, ended_at,
		          country, region, facility_id, created_by, created_at, updated_at`,
		p.ID, p.Status, p.ChiefComplaint, p.StartedAt, p.EndedAt,
	)
	return scanConsultationRow(row)
}

func scanConsultationRow(scanner interface{ Scan(...any) error }) (*models.ConsultationRow, error) {
	var c models.ConsultationRow
	err := scanner.Scan(
		&c.ID, &c.PatientID, &c.DoctorID, &c.Status, &c.ConsultationType,
		&c.ChiefComplaintEnc, &c.ScheduledAt, &c.StartedAt, &c.EndedAt,
		&c.Country, &c.Region, &c.FacilityID, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("consultation not found")
	}
	return &c, err
}

// ─── Diagnoses ────────────────────────────────────────────────────────────────

type CreateDiagnosisParams struct {
	ConsultationID   uuid.UUID
	PatientID        uuid.UUID
	DoctorID         uuid.UUID
	ICD10Code        string
	ICD10Desc        string
	ClinicalNotesEnc string
	Severity         models.Severity
	IsPrimary        bool
}

func (q *Queries) CreateDiagnosis(ctx context.Context, p CreateDiagnosisParams) (*models.DiagnosisRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO diagnoses
			(consultation_id, patient_id, doctor_id, icd10_code, icd10_description,
			 clinical_notes_enc, severity, is_primary)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, consultation_id, patient_id, doctor_id, icd10_code,
		          icd10_description, clinical_notes_enc, severity, is_primary,
		          created_at, updated_at`,
		p.ConsultationID, p.PatientID, p.DoctorID, p.ICD10Code, p.ICD10Desc,
		p.ClinicalNotesEnc, p.Severity, p.IsPrimary,
	)
	return scanDiagnosisRow(row)
}

func (q *Queries) ListDiagnoses(ctx context.Context, consultationID uuid.UUID) ([]*models.DiagnosisRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, consultation_id, patient_id, doctor_id, icd10_code,
		       icd10_description, clinical_notes_enc, severity, is_primary,
		       created_at, updated_at
		FROM diagnoses WHERE consultation_id = $1 ORDER BY is_primary DESC, created_at ASC`,
		consultationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.DiagnosisRow
	for rows.Next() {
		r, err := scanDiagnosisRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func scanDiagnosisRow(scanner interface{ Scan(...any) error }) (*models.DiagnosisRow, error) {
	var d models.DiagnosisRow
	err := scanner.Scan(
		&d.ID, &d.ConsultationID, &d.PatientID, &d.DoctorID,
		&d.ICD10Code, &d.ICD10Desc, &d.ClinicalNotesEnc,
		&d.Severity, &d.IsPrimary, &d.CreatedAt, &d.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("diagnosis not found")
	}
	return &d, err
}

// ─── Treatments ───────────────────────────────────────────────────────────────

type CreateTreatmentParams struct {
	ConsultationID  uuid.UUID
	PatientID       uuid.UUID
	DoctorID        uuid.UUID
	TreatmentType   models.TreatmentType
	InstructionsEnc string
	DurationDays    int
	FollowUpDate    *time.Time
}

func (q *Queries) CreateTreatment(ctx context.Context, p CreateTreatmentParams) (*models.TreatmentRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO treatments
			(consultation_id, patient_id, doctor_id, treatment_type,
			 instructions_enc, duration_days, follow_up_date)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, consultation_id, patient_id, doctor_id, treatment_type,
		          instructions_enc, duration_days, follow_up_date, status,
		          created_at, updated_at`,
		p.ConsultationID, p.PatientID, p.DoctorID, p.TreatmentType,
		p.InstructionsEnc, p.DurationDays, p.FollowUpDate,
	)
	return scanTreatmentRow(row)
}

func (q *Queries) ListTreatments(ctx context.Context, consultationID uuid.UUID) ([]*models.TreatmentRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, consultation_id, patient_id, doctor_id, treatment_type,
		       instructions_enc, duration_days, follow_up_date, status,
		       created_at, updated_at
		FROM treatments WHERE consultation_id = $1 ORDER BY created_at ASC`,
		consultationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.TreatmentRow
	for rows.Next() {
		r, err := scanTreatmentRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateTreatmentStatus(ctx context.Context, id uuid.UUID, status models.TreatmentStatus) (*models.TreatmentRow, error) {
	row := q.pool.QueryRow(ctx, `
		UPDATE treatments SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, consultation_id, patient_id, doctor_id, treatment_type,
		          instructions_enc, duration_days, follow_up_date, status,
		          created_at, updated_at`, id, status)
	return scanTreatmentRow(row)
}

func scanTreatmentRow(scanner interface{ Scan(...any) error }) (*models.TreatmentRow, error) {
	var t models.TreatmentRow
	err := scanner.Scan(
		&t.ID, &t.ConsultationID, &t.PatientID, &t.DoctorID,
		&t.TreatmentType, &t.InstructionsEnc, &t.DurationDays,
		&t.FollowUpDate, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("treatment not found")
	}
	return &t, err
}

// ─── Clinical Notes ───────────────────────────────────────────────────────────

type CreateNoteParams struct {
	ConsultationID uuid.UUID
	PatientID      uuid.UUID
	AuthorID       uuid.UUID
	NoteType       models.NoteType
	ContentEnc     string
}

func (q *Queries) CreateNote(ctx context.Context, p CreateNoteParams) (*models.ClinicalNoteRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO clinical_notes
			(consultation_id, patient_id, author_id, note_type, content_enc)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, consultation_id, patient_id, author_id, note_type, content_enc, created_at`,
		p.ConsultationID, p.PatientID, p.AuthorID, p.NoteType, p.ContentEnc,
	)
	return scanNoteRow(row)
}

func (q *Queries) ListNotes(ctx context.Context, consultationID uuid.UUID) ([]*models.ClinicalNoteRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, consultation_id, patient_id, author_id, note_type, content_enc, created_at
		FROM clinical_notes WHERE consultation_id = $1 ORDER BY created_at ASC`,
		consultationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.ClinicalNoteRow
	for rows.Next() {
		r, err := scanNoteRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func scanNoteRow(scanner interface{ Scan(...any) error }) (*models.ClinicalNoteRow, error) {
	var n models.ClinicalNoteRow
	err := scanner.Scan(
		&n.ID, &n.ConsultationID, &n.PatientID, &n.AuthorID,
		&n.NoteType, &n.ContentEnc, &n.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("note not found")
	}
	return &n, err
}

// ─── Prescriptions ────────────────────────────────────────────────────────────

type CreatePrescriptionParams struct {
	ConsultationID uuid.UUID
	PatientID      uuid.UUID
	DoctorID       uuid.UUID
	PharmacyID     *uuid.UUID
	MedicationsEnc string
	NotesEnc       *string
}

func (q *Queries) CreatePrescription(ctx context.Context, p CreatePrescriptionParams) (*models.PrescriptionRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO prescriptions
			(consultation_id, patient_id, doctor_id, pharmacy_id, medications_enc, notes_enc)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, consultation_id, patient_id, doctor_id, pharmacy_id,
		          medications_enc, notes_enc, status, dispensed_at, created_at, updated_at`,
		p.ConsultationID, p.PatientID, p.DoctorID, p.PharmacyID, p.MedicationsEnc, p.NotesEnc,
	)
	return scanPrescriptionRow(row)
}

func (q *Queries) GetPrescriptionByID(ctx context.Context, id uuid.UUID) (*models.PrescriptionRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, consultation_id, patient_id, doctor_id, pharmacy_id,
		       medications_enc, notes_enc, status, dispensed_at, created_at, updated_at
		FROM prescriptions WHERE id = $1`, id)
	return scanPrescriptionRow(row)
}

func (q *Queries) ListPrescriptions(ctx context.Context, consultationID uuid.UUID) ([]*models.PrescriptionRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, consultation_id, patient_id, doctor_id, pharmacy_id,
		       medications_enc, notes_enc, status, dispensed_at, created_at, updated_at
		FROM prescriptions WHERE consultation_id = $1 ORDER BY created_at ASC`,
		consultationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.PrescriptionRow
	for rows.Next() {
		r, err := scanPrescriptionRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (q *Queries) UpdatePrescriptionStatus(ctx context.Context, id uuid.UUID, status models.PrescriptionStatus) (*models.PrescriptionRow, error) {
	var dispensedAt *time.Time
	if status == models.PrescriptionDispensed {
		now := time.Now()
		dispensedAt = &now
	}
	row := q.pool.QueryRow(ctx, `
		UPDATE prescriptions SET status = $2, dispensed_at = COALESCE($3, dispensed_at), updated_at = NOW()
		WHERE id = $1
		RETURNING id, consultation_id, patient_id, doctor_id, pharmacy_id,
		          medications_enc, notes_enc, status, dispensed_at, created_at, updated_at`,
		id, status, dispensedAt,
	)
	return scanPrescriptionRow(row)
}

func scanPrescriptionRow(scanner interface{ Scan(...any) error }) (*models.PrescriptionRow, error) {
	var p models.PrescriptionRow
	err := scanner.Scan(
		&p.ID, &p.ConsultationID, &p.PatientID, &p.DoctorID, &p.PharmacyID,
		&p.MedicationsEnc, &p.NotesEnc, &p.Status, &p.DispensedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("prescription not found")
	}
	return &p, err
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

type InsertAuditParams struct {
	ResourceType string
	ResourceID   uuid.UUID
	PatientID    uuid.UUID
	Action       models.AuditAction
	AccessorID   uuid.UUID
	AccessorRole string
	IPAddress    string
	RequestID    string
	Changes      interface{}
}

func (q *Queries) InsertAuditLog(ctx context.Context, p InsertAuditParams) error {
	var changesJSON []byte
	if p.Changes != nil {
		var err error
		changesJSON, err = json.Marshal(p.Changes)
		if err != nil {
			return err
		}
	}
	_, err := q.pool.Exec(ctx, `
		INSERT INTO clinical_audit_log
			(resource_type, resource_id, patient_id, action,
			 accessor_id, accessor_role, ip_address, request_id, changes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ResourceType, p.ResourceID, p.PatientID, p.Action,
		p.AccessorID, p.AccessorRole, p.IPAddress, p.RequestID, changesJSON,
	)
	return err
}

func (q *Queries) GetAuditLog(ctx context.Context, resourceID uuid.UUID, limit, offset int) ([]*models.ClinicalAuditLog, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, resource_type, resource_id, patient_id, action,
		       accessor_id, accessor_role, ip_address, request_id, changes, created_at
		FROM clinical_audit_log
		WHERE resource_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		resourceID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.ClinicalAuditLog
	for rows.Next() {
		var a models.ClinicalAuditLog
		var changesJSON []byte
		err := rows.Scan(
			&a.ID, &a.ResourceType, &a.ResourceID, &a.PatientID, &a.Action,
			&a.AccessorID, &a.AccessorRole, &a.IPAddress, &a.RequestID,
			&changesJSON, &a.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if changesJSON != nil {
			json.Unmarshal(changesJSON, &a.Changes)
		}
		result = append(result, &a)
	}
	return result, rows.Err()
}
