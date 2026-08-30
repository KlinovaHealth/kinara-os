-- Patient Service — Core patient registry initialization
-- Database: kinara_patient
-- Stores patient demographic records with PHI fields encrypted at the application layer (AES-256-GCM)

\c kinara_patient;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS patients (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- PHI — all encrypted with AES-256-GCM at application layer
    national_id_enc             TEXT        NOT NULL,
    full_name_enc               TEXT        NOT NULL,
    date_of_birth_enc           TEXT        NOT NULL,
    phone_number_enc            TEXT        NOT NULL,
    email_enc                   TEXT,
    address_enc                 TEXT,
    blood_type_enc              TEXT,
    allergies_enc               TEXT,
    emergency_contact_name_enc  TEXT,
    emergency_contact_phone_enc TEXT,
    -- Non-PHI indexable fields
    gender      TEXT        NOT NULL CHECK (gender IN ('male','female','other','prefer_not_to_say')),
    country     TEXT        NOT NULL,
    region      TEXT,
    status      TEXT        NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','inactive','deceased','transferred')),
    tenant_id   TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  patients                              IS 'Patient registry — PHI fields AES-256-GCM encrypted at application layer';
COMMENT ON COLUMN patients.national_id_enc             IS 'National ID ciphertext — NEVER returned in list views';
COMMENT ON COLUMN patients.full_name_enc               IS 'Patient full name, encrypted';
COMMENT ON COLUMN patients.date_of_birth_enc           IS 'Date of birth, encrypted — derive age bands for analytics only';
COMMENT ON COLUMN patients.phone_number_enc            IS 'Primary contact number, encrypted';
COMMENT ON COLUMN patients.tenant_id                   IS 'Clinic / facility tenant identifier for multi-tenant isolation';
COMMENT ON COLUMN patients.status                      IS 'Lifecycle status: active, inactive, deceased, or transferred';

CREATE INDEX IF NOT EXISTS idx_patients_country    ON patients(country);
CREATE INDEX IF NOT EXISTS idx_patients_tenant     ON patients(tenant_id);
CREATE INDEX IF NOT EXISTS idx_patients_status     ON patients(status);
CREATE INDEX IF NOT EXISTS idx_patients_created    ON patients(created_at DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS patient_audit_log (
    id             BIGSERIAL   PRIMARY KEY,
    entity_id      UUID        NOT NULL,
    action         TEXT        NOT NULL,  -- 'create','update','delete','read'
    actor_id       TEXT        NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    signature_hash TEXT,                 -- ed25519 tamper detection
    ip_address     INET,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  patient_audit_log               IS 'Append-only audit trail for patient record changes — immutable via rules';
COMMENT ON COLUMN patient_audit_log.signature_hash IS 'ed25519 signature over (entity_id||action||actor_id||occurred_at) for tamper detection';
COMMENT ON COLUMN patient_audit_log.action         IS 'One of: create, update, delete, read';

CREATE RULE no_update_patient_audit AS ON UPDATE TO patient_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_patient_audit AS ON DELETE TO patient_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_patient_audit_entity ON patient_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_patient_audit_actor  ON patient_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS patient_audit_log CASCADE;
-- DROP TABLE IF EXISTS patients CASCADE;
