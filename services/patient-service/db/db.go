// Package db provides the database access layer for the patient service.
// Queries use pgx/v5 with parameterized statements — no raw string interpolation.
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/patient-service/models"
)

// Queries wraps a pgx connection pool and exposes type-safe query methods.
type Queries struct {
	pool *pgxpool.Pool
}

// New returns a Queries instance backed by the given pool.
func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// ─── Patient CRUD ─────────────────────────────────────────────────────────────

// CreatePatientParams holds the encrypted values ready to insert.
type CreatePatientParams struct {
	NationalIDEnc            string
	FullNameEnc              string
	DateOfBirthEnc           string
	Gender                   models.Gender
	PhoneNumberEnc           string
	EmailEnc                 *string
	AddressEnc               *string
	Country                  string
	Region                   *string
	BloodTypeEnc             *string
	AllergiesEnc             *string
	EmergencyContactNameEnc  *string
	EmergencyContactPhoneEnc *string
	EmergencyContactRelEnc   *string
	Status                   models.PatientStatus
	CreatedBy                uuid.UUID
}

// CreatePatient inserts a new patient row and returns the full row.
func (q *Queries) CreatePatient(ctx context.Context, p CreatePatientParams) (*models.PatientRow, error) {
	const query = `
		INSERT INTO patients (
			national_id_enc, full_name_enc, date_of_birth_enc, gender,
			phone_number_enc, email_enc, address_enc, country, region,
			blood_type_enc, allergies_enc,
			emergency_contact_name_enc, emergency_contact_phone_enc, emergency_contact_rel_enc,
			status, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16
		) RETURNING
			id, national_id_enc, full_name_enc, date_of_birth_enc, gender,
			phone_number_enc, email_enc, address_enc, country, region,
			blood_type_enc, allergies_enc,
			emergency_contact_name_enc, emergency_contact_phone_enc, emergency_contact_rel_enc,
			status, created_by, created_at, updated_at, deleted_at`

	row := q.pool.QueryRow(ctx, query,
		p.NationalIDEnc, p.FullNameEnc, p.DateOfBirthEnc, p.Gender,
		p.PhoneNumberEnc, p.EmailEnc, p.AddressEnc, p.Country, p.Region,
		p.BloodTypeEnc, p.AllergiesEnc,
		p.EmergencyContactNameEnc, p.EmergencyContactPhoneEnc, p.EmergencyContactRelEnc,
		p.Status, p.CreatedBy,
	)

	return scanPatientRow(row)
}

// GetPatientByID fetches a single non-deleted patient by primary key.
func (q *Queries) GetPatientByID(ctx context.Context, id uuid.UUID) (*models.PatientRow, error) {
	const query = `
		SELECT id, national_id_enc, full_name_enc, date_of_birth_enc, gender,
			phone_number_enc, email_enc, address_enc, country, region,
			blood_type_enc, allergies_enc,
			emergency_contact_name_enc, emergency_contact_phone_enc, emergency_contact_rel_enc,
			status, created_by, created_at, updated_at, deleted_at
		FROM patients
		WHERE id = $1 AND deleted_at IS NULL`

	row := q.pool.QueryRow(ctx, query, id)
	p, err := scanPatientRow(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// ListPatientsParams holds filter + pagination values for ListPatients.
type ListPatientsParams struct {
	Country string
	Region  string
	Status  string
	Limit   int
	Offset  int
}

// ListPatients returns a paginated, filtered list of non-deleted patients.
func (q *Queries) ListPatients(ctx context.Context, p ListPatientsParams) ([]*models.PatientRow, error) {
	const query = `
		SELECT id, national_id_enc, full_name_enc, date_of_birth_enc, gender,
			phone_number_enc, email_enc, address_enc, country, region,
			blood_type_enc, allergies_enc,
			emergency_contact_name_enc, emergency_contact_phone_enc, emergency_contact_rel_enc,
			status, created_by, created_at, updated_at, deleted_at
		FROM patients
		WHERE deleted_at IS NULL
		  AND ($1::text = '' OR country = $1)
		  AND ($2::text = '' OR region  = $2)
		  AND ($3::text = '' OR status  = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`

	rows, err := q.pool.Query(ctx, query, p.Country, p.Region, p.Status, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("db: ListPatients query: %w", err)
	}
	defer rows.Close()

	var patients []*models.PatientRow
	for rows.Next() {
		pat, err := scanPatientRow(rows)
		if err != nil {
			return nil, fmt.Errorf("db: ListPatients scan: %w", err)
		}
		patients = append(patients, pat)
	}
	return patients, rows.Err()
}

// CountPatients returns the total row count matching the same filters as ListPatients.
func (q *Queries) CountPatients(ctx context.Context, p ListPatientsParams) (int64, error) {
	const query = `
		SELECT COUNT(*) FROM patients
		WHERE deleted_at IS NULL
		  AND ($1::text = '' OR country = $1)
		  AND ($2::text = '' OR region  = $2)
		  AND ($3::text = '' OR status  = $3)`

	var count int64
	err := q.pool.QueryRow(ctx, query, p.Country, p.Region, p.Status).Scan(&count)
	return count, err
}

// UpdatePatientParams holds nullable update values (nil = keep existing).
type UpdatePatientParams struct {
	ID                       uuid.UUID
	PhoneNumberEnc           *string
	EmailEnc                 *string
	AddressEnc               *string
	Region                   *string
	BloodTypeEnc             *string
	AllergiesEnc             *string
	EmergencyContactNameEnc  *string
	EmergencyContactPhoneEnc *string
	EmergencyContactRelEnc   *string
	Status                   *string
}

// UpdatePatient performs a partial update and returns the updated row.
// Returns nil, nil when no matching row exists.
func (q *Queries) UpdatePatient(ctx context.Context, p UpdatePatientParams) (*models.PatientRow, error) {
	const query = `
		UPDATE patients SET
			phone_number_enc            = COALESCE($2,  phone_number_enc),
			email_enc                   = COALESCE($3,  email_enc),
			address_enc                 = COALESCE($4,  address_enc),
			region                      = COALESCE($5,  region),
			blood_type_enc              = COALESCE($6,  blood_type_enc),
			allergies_enc               = COALESCE($7,  allergies_enc),
			emergency_contact_name_enc  = COALESCE($8,  emergency_contact_name_enc),
			emergency_contact_phone_enc = COALESCE($9,  emergency_contact_phone_enc),
			emergency_contact_rel_enc   = COALESCE($10, emergency_contact_rel_enc),
			status                      = COALESCE($11, status)
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING
			id, national_id_enc, full_name_enc, date_of_birth_enc, gender,
			phone_number_enc, email_enc, address_enc, country, region,
			blood_type_enc, allergies_enc,
			emergency_contact_name_enc, emergency_contact_phone_enc, emergency_contact_rel_enc,
			status, created_by, created_at, updated_at, deleted_at`

	row := q.pool.QueryRow(ctx, query,
		p.ID,
		p.PhoneNumberEnc, p.EmailEnc, p.AddressEnc, p.Region,
		p.BloodTypeEnc, p.AllergiesEnc,
		p.EmergencyContactNameEnc, p.EmergencyContactPhoneEnc, p.EmergencyContactRelEnc,
		p.Status,
	)

	pat, err := scanPatientRow(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return pat, err
}

// SoftDeletePatient sets deleted_at and returns the patient ID.
// Returns uuid.Nil when no matching row exists.
func (q *Queries) SoftDeletePatient(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	const query = `
		UPDATE patients SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id`

	var deletedID uuid.UUID
	err := q.pool.QueryRow(ctx, query, id).Scan(&deletedID)
	if err == pgx.ErrNoRows {
		return uuid.Nil, nil
	}
	return deletedID, err
}

// ─── Audit log ────────────────────────────────────────────────────────────────

// InsertAuditLogParams holds one audit event.
type InsertAuditLogParams struct {
	PatientID    uuid.UUID
	Action       models.AuditAction
	AccessorID   uuid.UUID
	AccessorRole string
	IPAddress    string
	RequestID    string
	Changes      map[string]interface{}
}

// InsertAuditLog writes one immutable audit event.
func (q *Queries) InsertAuditLog(ctx context.Context, p InsertAuditLogParams) error {
	var changesJSON []byte
	if p.Changes != nil {
		var err error
		changesJSON, err = json.Marshal(p.Changes)
		if err != nil {
			return fmt.Errorf("db: marshal audit changes: %w", err)
		}
	}

	const query = `
		INSERT INTO patient_audit_log
			(patient_id, action, accessor_id, accessor_role, ip_address, request_id, changes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := q.pool.Exec(ctx, query,
		p.PatientID, p.Action, p.AccessorID, p.AccessorRole,
		p.IPAddress, p.RequestID, changesJSON,
	)
	return err
}

// GetAuditLog returns paginated audit events for a patient.
func (q *Queries) GetAuditLog(ctx context.Context, patientID uuid.UUID, limit, offset int) ([]*models.PatientAuditLog, error) {
	const query = `
		SELECT id, patient_id, action, accessor_id, accessor_role,
			ip_address, request_id, changes, created_at
		FROM patient_audit_log
		WHERE patient_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := q.pool.Query(ctx, query, patientID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("db: GetAuditLog query: %w", err)
	}
	defer rows.Close()

	var logs []*models.PatientAuditLog
	for rows.Next() {
		var l models.PatientAuditLog
		var changesRaw []byte
		err := rows.Scan(
			&l.ID, &l.PatientID, &l.Action, &l.AccessorID, &l.AccessorRole,
			&l.IPAddress, &l.RequestID, &changesRaw, &l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("db: GetAuditLog scan: %w", err)
		}
		if changesRaw != nil {
			_ = json.Unmarshal(changesRaw, &l.Changes)
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}

// ─── scanner ──────────────────────────────────────────────────────────────────

// scanPatientRow is shared by all single-row patient queries.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanPatientRow(row scanner) (*models.PatientRow, error) {
	var p models.PatientRow
	var dob string // stored as encrypted string, not native timestamptz
	err := row.Scan(
		&p.ID,
		&p.NationalIDEnc,
		&p.FullNameEnc,
		&dob, // date_of_birth_enc is TEXT (encrypted)
		&p.Gender,
		&p.PhoneNumberEnc,
		&p.EmailEnc,
		&p.AddressEnc,
		&p.Country,
		&p.Region,
		&p.BloodTypeEnc,
		&p.AllergiesEnc,
		&p.EmergencyContactNameEnc,
		&p.EmergencyContactPhoneEnc,
		&p.EmergencyContactRelEnc,
		&p.Status,
		&p.CreatedBy,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	// Store dob back into the encrypted field for the handler to decrypt.
	p.DateOfBirthEnc = dob
	_ = time.Now() // keep time import
	return &p, nil
}
