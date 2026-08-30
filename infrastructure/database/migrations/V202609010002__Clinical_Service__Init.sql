-- Clinical Service — Consultations, SOAP notes, diagnoses, and prescriptions
-- Database: kinara_clinical
-- Stores all clinical encounter data; sensitive fields encrypted at application layer

\c kinara_clinical;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS consultations (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id          UUID        NOT NULL,
    doctor_id           UUID        NOT NULL,
    status              TEXT        NOT NULL DEFAULT 'scheduled'
                        CHECK (status IN ('scheduled','in_progress','completed','cancelled','no_show')),
    consultation_type   TEXT        NOT NULL
                        CHECK (consultation_type IN ('video','audio','chat','in_person','whatsapp')),
    chief_complaint_enc TEXT,
    scheduled_at        TIMESTAMPTZ NOT NULL,
    started_at          TIMESTAMPTZ,
    ended_at            TIMESTAMPTZ,
    country             TEXT        NOT NULL,
    region              TEXT,
    facility_id         TEXT,
    created_by          UUID        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  consultations                      IS 'Patient consultation lifecycle — all encounter types including telemedicine';
COMMENT ON COLUMN consultations.chief_complaint_enc  IS 'Patient-reported chief complaint, AES-256-GCM encrypted';
COMMENT ON COLUMN consultations.consultation_type    IS 'Channel: video, audio, chat, in_person, or whatsapp';
COMMENT ON COLUMN consultations.facility_id          IS 'Originating clinic or hospital identifier';

CREATE INDEX IF NOT EXISTS idx_consult_patient    ON consultations(patient_id);
CREATE INDEX IF NOT EXISTS idx_consult_doctor     ON consultations(doctor_id);
CREATE INDEX IF NOT EXISTS idx_consult_status     ON consultations(status);
CREATE INDEX IF NOT EXISTS idx_consult_created    ON consultations(created_at DESC);

CREATE TABLE IF NOT EXISTS soap_notes (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id  UUID        NOT NULL,
    subjective_enc   TEXT,
    objective_enc    TEXT,
    assessment_enc   TEXT,
    plan_enc         TEXT,
    created_by       UUID        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  soap_notes              IS 'Structured SOAP clinical notes per consultation — all sections encrypted';
COMMENT ON COLUMN soap_notes.subjective_enc IS 'Patient subjective description, encrypted';
COMMENT ON COLUMN soap_notes.assessment_enc IS 'Clinician assessment / differential, encrypted';

CREATE INDEX IF NOT EXISTS idx_soap_consultation ON soap_notes(consultation_id);

CREATE TABLE IF NOT EXISTS diagnoses (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id  UUID        NOT NULL,
    icd10_code       TEXT        NOT NULL,
    description      TEXT,
    severity         TEXT,
    is_primary       BOOLEAN     NOT NULL DEFAULT false,
    created_by       UUID        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  diagnoses            IS 'ICD-10 diagnoses attached to a consultation';
COMMENT ON COLUMN diagnoses.icd10_code IS 'International Classification of Diseases, 10th revision code';
COMMENT ON COLUMN diagnoses.is_primary IS 'True if this is the primary (principal) diagnosis for the encounter';

CREATE INDEX IF NOT EXISTS idx_diagnoses_consultation ON diagnoses(consultation_id);
CREATE INDEX IF NOT EXISTS idx_diagnoses_icd10        ON diagnoses(icd10_code);

CREATE TABLE IF NOT EXISTS prescriptions (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id     UUID        NOT NULL,
    medication_name     TEXT        NOT NULL,
    dosage              TEXT        NOT NULL,
    frequency           TEXT        NOT NULL,
    duration_days       INT,
    instructions_enc    TEXT,
    status              TEXT        NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','dispensed','cancelled','expired')),
    created_by          UUID        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  prescriptions                 IS 'Medications prescribed at a consultation — linked to pharmacy service for dispensing';
COMMENT ON COLUMN prescriptions.instructions_enc IS 'Special dispensing or administration instructions, encrypted';
COMMENT ON COLUMN prescriptions.status           IS 'Lifecycle: active, dispensed, cancelled, expired';

CREATE INDEX IF NOT EXISTS idx_rx_consultation ON prescriptions(consultation_id);
CREATE INDEX IF NOT EXISTS idx_rx_status       ON prescriptions(status);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS clinical_audit_log (
    id             BIGSERIAL   PRIMARY KEY,
    entity_id      UUID        NOT NULL,
    action         TEXT        NOT NULL,  -- 'create','update','delete','read'
    actor_id       TEXT        NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    signature_hash TEXT,
    ip_address     INET,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE clinical_audit_log IS 'Append-only audit trail for all clinical service entities';

CREATE RULE no_update_clinical_audit AS ON UPDATE TO clinical_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_clinical_audit AS ON DELETE TO clinical_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_clinical_audit_entity ON clinical_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_clinical_audit_actor  ON clinical_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS clinical_audit_log CASCADE;
-- DROP TABLE IF EXISTS prescriptions CASCADE;
-- DROP TABLE IF EXISTS diagnoses CASCADE;
-- DROP TABLE IF EXISTS soap_notes CASCADE;
-- DROP TABLE IF EXISTS consultations CASCADE;
