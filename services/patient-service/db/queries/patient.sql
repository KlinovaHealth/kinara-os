-- sqlc queries for the patient service.
-- Run: sqlc generate (from db/ directory with sqlc.yaml present)

-- name: CreatePatient :one
INSERT INTO patients (
    national_id_enc,
    full_name_enc,
    date_of_birth_enc,
    gender,
    phone_number_enc,
    email_enc,
    address_enc,
    country,
    region,
    blood_type_enc,
    allergies_enc,
    emergency_contact_name_enc,
    emergency_contact_phone_enc,
    emergency_contact_rel_enc,
    status,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING *;

-- name: GetPatientByID :one
SELECT * FROM patients
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListPatients :many
SELECT * FROM patients
WHERE deleted_at IS NULL
  AND ($1::text = '' OR country = $1)
  AND ($2::text = '' OR region  = $2)
  AND ($3::text = '' OR status  = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountPatients :one
SELECT COUNT(*) FROM patients
WHERE deleted_at IS NULL
  AND ($1::text = '' OR country = $1)
  AND ($2::text = '' OR region  = $2)
  AND ($3::text = '' OR status  = $3);

-- name: UpdatePatient :one
UPDATE patients SET
    phone_number_enc            = COALESCE($2, phone_number_enc),
    email_enc                   = COALESCE($3, email_enc),
    address_enc                 = COALESCE($4, address_enc),
    region                      = COALESCE($5, region),
    blood_type_enc              = COALESCE($6, blood_type_enc),
    allergies_enc               = COALESCE($7, allergies_enc),
    emergency_contact_name_enc  = COALESCE($8, emergency_contact_name_enc),
    emergency_contact_phone_enc = COALESCE($9, emergency_contact_phone_enc),
    emergency_contact_rel_enc   = COALESCE($10, emergency_contact_rel_enc),
    status                      = COALESCE($11, status)
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePatient :one
UPDATE patients
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id;

-- name: InsertAuditLog :one
INSERT INTO patient_audit_log (
    patient_id, action, accessor_id, accessor_role,
    ip_address, request_id, changes
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetPatientAuditLog :many
SELECT * FROM patient_audit_log
WHERE patient_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
