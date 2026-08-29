-- Migration: 001_create_patients
-- Kinara OS — Patient Service
-- All PHI columns store AES-256-GCM ciphertext (base64-encoded).
-- The audit log is insert-only; UPDATE and DELETE are blocked by rules.

BEGIN;

-- ─── patients ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS patients (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- PHI — encrypted at rest with AES-256-GCM
    national_id_enc             TEXT        NOT NULL,
    full_name_enc               TEXT        NOT NULL,
    date_of_birth_enc           TEXT        NOT NULL,
    phone_number_enc            TEXT        NOT NULL,
    email_enc                   TEXT,
    address_enc                 TEXT,
    blood_type_enc              TEXT,
    allergies_enc               TEXT,           -- encrypted JSON array
    emergency_contact_name_enc  TEXT,
    emergency_contact_phone_enc TEXT,
    emergency_contact_rel_enc   TEXT,

    -- Non-PHI indexable fields
    gender      TEXT        NOT NULL CHECK (gender IN ('male','female','other','prefer_not_to_say')),
    country     TEXT        NOT NULL,
    region      TEXT,
    status      TEXT        NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','inactive','deceased','transferred')),

    created_by  UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ             -- soft delete; NULL = active
);

-- ─── indexes ─────────────────────────────────────────────────────────────────

CREATE INDEX IF NOT EXISTS idx_patients_country    ON patients (country);
CREATE INDEX IF NOT EXISTS idx_patients_region     ON patients (region)  WHERE region IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_patients_status     ON patients (status);
CREATE INDEX IF NOT EXISTS idx_patients_created_by ON patients (created_by);
CREATE INDEX IF NOT EXISTS idx_patients_active     ON patients (id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_patients_created_at ON patients (created_at DESC);

-- ─── updated_at trigger ───────────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_patients_updated_at
    BEFORE UPDATE ON patients
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ─── patient_audit_log (immutable) ───────────────────────────────────────────

CREATE TABLE IF NOT EXISTS patient_audit_log (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id    UUID        NOT NULL REFERENCES patients (id),
    action        TEXT        NOT NULL CHECK (action IN ('create','read','update','delete','search')),
    accessor_id   UUID        NOT NULL,
    accessor_role TEXT        NOT NULL,
    ip_address    TEXT,
    request_id    TEXT,
    changes       JSONB,                  -- diff for update operations
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_patient_id  ON patient_audit_log (patient_id);
CREATE INDEX IF NOT EXISTS idx_audit_accessor_id ON patient_audit_log (accessor_id);
CREATE INDEX IF NOT EXISTS idx_audit_created_at  ON patient_audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action      ON patient_audit_log (action);

-- Enforce immutability: no UPDATE or DELETE on the audit log.
CREATE RULE audit_no_update AS ON UPDATE TO patient_audit_log DO INSTEAD NOTHING;
CREATE RULE audit_no_delete AS ON DELETE TO patient_audit_log DO INSTEAD NOTHING;

COMMIT;
