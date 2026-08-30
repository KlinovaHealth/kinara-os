-- Input Service — Dynamic form templates, submissions, and change history
-- Database: kinara_input
-- Stores JSON schema-driven form definitions and all patient/clinician submissions with full edit history

\c kinara_input;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS forms (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    form_type   TEXT        NOT NULL UNIQUE,
    title       TEXT        NOT NULL,
    schema      JSONB       NOT NULL,
    version     INT         NOT NULL DEFAULT 1,
    active      BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  forms            IS 'Dynamic form template registry — schema-driven, versioned per form_type';
COMMENT ON COLUMN forms.form_type  IS 'Unique slug for the form, e.g. patient-intake, child-health, antenatal';
COMMENT ON COLUMN forms.schema     IS 'JSON Schema v7 definition of fields, validations, and conditional logic';
COMMENT ON COLUMN forms.version    IS 'Monotonically increasing version number; submissions record the version used';
COMMENT ON COLUMN forms.active     IS 'False means archived — submissions still readable, no new submissions accepted';

INSERT INTO forms (form_type, title, schema, version, active) VALUES
    ('patient-intake',  'Patient Intake Form',    '{"type":"object","properties":{"full_name":{"type":"string"},"dob":{"type":"string","format":"date"},"gender":{"type":"string"},"phone":{"type":"string"},"chief_complaint":{"type":"string"}}}'::jsonb, 1, true),
    ('child-health',    'Child Health Assessment', '{"type":"object","properties":{"patient_id":{"type":"string"},"age_months":{"type":"integer"},"weight_kg":{"type":"number"},"height_cm":{"type":"number"},"muac_cm":{"type":"number"},"vaccination_status":{"type":"string"}}}'::jsonb, 1, true),
    ('antenatal',       'Antenatal Care Visit',   '{"type":"object","properties":{"patient_id":{"type":"string"},"gestational_age_weeks":{"type":"integer"},"bp_systolic":{"type":"integer"},"bp_diastolic":{"type":"integer"},"weight_kg":{"type":"number"},"fundal_height_cm":{"type":"number"},"fetal_heartbeat":{"type":"string"}}}'::jsonb, 1, true)
ON CONFLICT (form_type) DO NOTHING;

CREATE TABLE IF NOT EXISTS form_submissions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_ref  TEXT        NOT NULL UNIQUE,
    patient_id      UUID        NOT NULL,
    form_type       TEXT        NOT NULL,
    form_version    INT         NOT NULL DEFAULT 1,
    data            JSONB       NOT NULL,
    submitted_by    UUID        NOT NULL,
    tenant_id       TEXT        NOT NULL,
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  form_submissions               IS 'Completed form submissions — data stored as JSONB conforming to forms.schema';
COMMENT ON COLUMN form_submissions.submission_ref IS 'Human-readable unique reference, e.g. SUB-20260901-0001';
COMMENT ON COLUMN form_submissions.form_version   IS 'Version of the form schema at time of submission — must match forms.version';
COMMENT ON COLUMN form_submissions.data           IS 'Submitted field values as JSONB — validated against form schema at API layer';

CREATE INDEX IF NOT EXISTS idx_sub_patient   ON form_submissions(patient_id, submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_sub_form_type ON form_submissions(form_type, submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_sub_tenant    ON form_submissions(tenant_id, submitted_at DESC);

CREATE TABLE IF NOT EXISTS form_submission_history (
    id             BIGSERIAL   PRIMARY KEY,
    submission_id  UUID        NOT NULL,
    old_data       JSONB       NOT NULL,
    changed_by     UUID        NOT NULL,
    changed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  form_submission_history               IS 'Complete edit history for form submissions — one row per save operation';
COMMENT ON COLUMN form_submission_history.submission_id IS 'References form_submissions(id) — no FK to allow history after submission delete';
COMMENT ON COLUMN form_submission_history.old_data      IS 'Full JSONB snapshot of submission data before this change';

CREATE INDEX IF NOT EXISTS idx_fsh_submission ON form_submission_history(submission_id, changed_at DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS input_audit_log (
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

COMMENT ON TABLE input_audit_log IS 'Append-only audit trail for form definitions and submission operations';

CREATE RULE no_update_input_audit AS ON UPDATE TO input_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_input_audit AS ON DELETE TO input_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_input_audit_entity ON input_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_input_audit_actor  ON input_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS input_audit_log CASCADE;
-- DROP TABLE IF EXISTS form_submission_history CASCADE;
-- DROP TABLE IF EXISTS form_submissions CASCADE;
-- DROP TABLE IF EXISTS forms CASCADE;
