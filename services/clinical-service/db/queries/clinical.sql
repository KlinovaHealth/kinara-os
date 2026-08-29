-- Consultation queries

-- name: CreateConsultation :one
INSERT INTO consultations
    (patient_id, doctor_id, consultation_type, chief_complaint_enc,
     scheduled_at, country, region, facility_id, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING *;

-- name: GetConsultationByID :one
SELECT * FROM consultations WHERE id = $1;

-- name: ListConsultations :many
SELECT * FROM consultations
WHERE ($1::UUID IS NULL OR patient_id = $1)
  AND ($2::UUID IS NULL OR doctor_id = $2)
  AND ($3::TEXT IS NULL OR status = $3)
  AND ($4 = '' OR country = $4)
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountConsultations :one
SELECT COUNT(*) FROM consultations
WHERE ($1::UUID IS NULL OR patient_id = $1)
  AND ($2::UUID IS NULL OR doctor_id = $2)
  AND ($3::TEXT IS NULL OR status = $3)
  AND ($4 = '' OR country = $4);

-- name: UpdateConsultation :one
UPDATE consultations SET
    status              = COALESCE($2, status),
    chief_complaint_enc = COALESCE($3, chief_complaint_enc),
    started_at          = COALESCE($4, started_at),
    ended_at            = COALESCE($5, ended_at),
    updated_at          = NOW()
WHERE id = $1
RETURNING *;

-- Diagnosis queries

-- name: CreateDiagnosis :one
INSERT INTO diagnoses
    (consultation_id, patient_id, doctor_id, icd10_code, icd10_description,
     clinical_notes_enc, severity, is_primary)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING *;

-- name: ListDiagnoses :many
SELECT * FROM diagnoses
WHERE consultation_id = $1
ORDER BY is_primary DESC, created_at ASC;

-- Treatment queries

-- name: CreateTreatment :one
INSERT INTO treatments
    (consultation_id, patient_id, doctor_id, treatment_type,
     instructions_enc, duration_days, follow_up_date)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING *;

-- name: ListTreatments :many
SELECT * FROM treatments WHERE consultation_id = $1 ORDER BY created_at ASC;

-- name: UpdateTreatmentStatus :one
UPDATE treatments SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- Clinical note queries

-- name: CreateNote :one
INSERT INTO clinical_notes
    (consultation_id, patient_id, author_id, note_type, content_enc)
VALUES ($1,$2,$3,$4,$5)
RETURNING *;

-- name: ListNotes :many
SELECT * FROM clinical_notes
WHERE consultation_id = $1
ORDER BY created_at ASC;

-- Prescription queries

-- name: CreatePrescription :one
INSERT INTO prescriptions
    (consultation_id, patient_id, doctor_id, pharmacy_id, medications_enc, notes_enc)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING *;

-- name: GetPrescriptionByID :one
SELECT * FROM prescriptions WHERE id = $1;

-- name: ListPrescriptions :many
SELECT * FROM prescriptions WHERE consultation_id = $1 ORDER BY created_at ASC;

-- name: UpdatePrescriptionStatus :one
UPDATE prescriptions SET status = $2, dispensed_at = COALESCE($3, dispensed_at), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- Audit log queries

-- name: InsertAuditLog :exec
INSERT INTO clinical_audit_log
    (resource_type, resource_id, patient_id, action,
     accessor_id, accessor_role, ip_address, request_id, changes)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9);

-- name: GetAuditLog :many
SELECT * FROM clinical_audit_log
WHERE resource_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
