package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/input-service/models"
)

// Store is the interface over all DB operations, enabling mock injection in tests.
type Store interface {
	GetForm(ctx context.Context, formType string) (*models.Form, error)
	CreateSubmission(ctx context.Context, s models.FormSubmission) error
	GetSubmission(ctx context.Context, id uuid.UUID) (*models.FormSubmission, error)
	ListByPatient(ctx context.Context, patientID uuid.UUID) ([]models.FormSubmission, error)
	UpdateSubmission(ctx context.Context, id uuid.UUID, data []byte, updatedAt time.Time) error
	InsertAudit(ctx context.Context, submissionID, actorID, action string) error
}

// Queries is the concrete postgres implementation of Store.
type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

// Ensure Queries satisfies Store at compile time.
var _ Store = (*Queries)(nil)

func (q *Queries) GetForm(ctx context.Context, formType string) (*models.Form, error) {
	var f models.Form
	err := q.pool.QueryRow(ctx,
		`SELECT id, form_type, title, schema, version, active, created_at
		 FROM forms WHERE form_type=$1 AND active=true`,
		formType).Scan(&f.ID, &f.FormType, &f.Title, &f.Schema, &f.Version, &f.Active, &f.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Fall back to hardcoded forms so the service works without seed data.
			return fallbackForm(formType)
		}
		return nil, err
	}
	return &f, nil
}

func fallbackForm(formType string) (*models.Form, error) {
	switch formType {
	case "patient-intake":
		return defaultPatientIntakeForm(), nil
	case "child-health":
		schema := json.RawMessage(`{"required":["child_name","date_of_birth","weight_kg","height_cm"],"properties":{"child_name":{"type":"string"},"date_of_birth":{"type":"string","format":"date"},"weight_kg":{"type":"number"},"height_cm":{"type":"number"}}}`)
		return &models.Form{FormType: "child-health", Title: "Child Health Assessment", Schema: schema, Version: 1, Active: true}, nil
	case "antenatal":
		schema := json.RawMessage(`{"required":["patient_id","gestational_age_weeks","blood_pressure"],"properties":{"patient_id":{"type":"string"},"gestational_age_weeks":{"type":"integer"},"blood_pressure":{"type":"string"}}}`)
		return &models.Form{FormType: "antenatal", Title: "Antenatal Care Form", Schema: schema, Version: 1, Active: true}, nil
	default:
		return nil, pgx.ErrNoRows
	}
}

func defaultPatientIntakeForm() *models.Form {
	schema := json.RawMessage(`{"required":["full_name","date_of_birth","sex"],"properties":{"full_name":{"type":"string"},"date_of_birth":{"type":"string","format":"date"},"sex":{"type":"string","enum":["M","F"]},"blood_type":{"type":"string"},"allergies":{"type":"string"},"chief_complaint":{"type":"string"}}}`)
	return &models.Form{FormType: "patient-intake", Title: "Patient Intake Form", Schema: schema, Version: 1, Active: true}
}

func (q *Queries) CreateSubmission(ctx context.Context, s models.FormSubmission) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO form_submissions
		 (id, submission_ref, patient_id, form_type, form_version, data, submitted_by, tenant_id, submitted_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.ID, s.SubmissionRef, s.PatientID, s.FormType, s.FormVersion,
		s.Data, s.SubmittedBy, s.TenantID, s.SubmittedAt, s.UpdatedAt)
	return err
}

func (q *Queries) GetSubmission(ctx context.Context, id uuid.UUID) (*models.FormSubmission, error) {
	var s models.FormSubmission
	err := q.pool.QueryRow(ctx,
		`SELECT id, submission_ref, patient_id, form_type, form_version, data, submitted_by, tenant_id, submitted_at, updated_at
		 FROM form_submissions WHERE id=$1`, id).
		Scan(&s.ID, &s.SubmissionRef, &s.PatientID, &s.FormType, &s.FormVersion,
			&s.Data, &s.SubmittedBy, &s.TenantID, &s.SubmittedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (q *Queries) ListByPatient(ctx context.Context, patientID uuid.UUID) ([]models.FormSubmission, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, submission_ref, patient_id, form_type, form_version, data, submitted_by, tenant_id, submitted_at, updated_at
		 FROM form_submissions WHERE patient_id=$1 ORDER BY submitted_at DESC`,
		patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.FormSubmission
	for rows.Next() {
		var s models.FormSubmission
		if err := rows.Scan(&s.ID, &s.SubmissionRef, &s.PatientID, &s.FormType, &s.FormVersion,
			&s.Data, &s.SubmittedBy, &s.TenantID, &s.SubmittedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (q *Queries) UpdateSubmission(ctx context.Context, id uuid.UUID, data []byte, updatedAt time.Time) error {
	// First, archive current data into history.
	var oldData []byte
	var changedBy uuid.UUID
	err := q.pool.QueryRow(ctx,
		`SELECT data, submitted_by FROM form_submissions WHERE id=$1`, id).
		Scan(&oldData, &changedBy)
	if err != nil {
		return err
	}
	_, err = q.pool.Exec(ctx,
		`INSERT INTO form_submission_history (submission_id, old_data, changed_by, changed_at)
		 VALUES ($1,$2,$3,$4)`,
		id, oldData, changedBy, updatedAt)
	if err != nil {
		return err
	}
	_, err = q.pool.Exec(ctx,
		`UPDATE form_submissions SET data=$1, updated_at=$2 WHERE id=$3`,
		data, updatedAt, id)
	return err
}

func (q *Queries) InsertAudit(ctx context.Context, submissionID, actorID, action string) error {
	var subIDVal interface{}
	if submissionID != "" {
		parsed, err := uuid.Parse(submissionID)
		if err == nil {
			subIDVal = parsed
		}
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO input_audit_log (submission_id, action, actor_id, occurred_at)
		 VALUES ($1,$2,$3,NOW())`,
		subIDVal, action, actorID)
	return err
}
