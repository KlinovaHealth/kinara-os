CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS forms (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_type  TEXT NOT NULL UNIQUE,
    title      TEXT NOT NULL,
    schema     JSONB NOT NULL,
    version    INT NOT NULL DEFAULT 1,
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO forms (form_type, title, schema) VALUES
('patient-intake', 'Patient Intake Form',    '{"required":["full_name","date_of_birth","sex"]}'::jsonb),
('child-health',   'Child Health Assessment','{"required":["child_name","date_of_birth","weight_kg","height_cm"]}'::jsonb),
('antenatal',      'Antenatal Care Form',    '{"required":["patient_id","gestational_age_weeks","blood_pressure"]}'::jsonb)
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS form_submissions (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_ref TEXT NOT NULL UNIQUE,
    patient_id     UUID NOT NULL,
    form_type      TEXT NOT NULL,
    form_version   INT NOT NULL DEFAULT 1,
    data           JSONB NOT NULL,
    submitted_by   UUID NOT NULL,
    tenant_id      TEXT NOT NULL,
    submitted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submissions_patient ON form_submissions(patient_id);

CREATE TABLE IF NOT EXISTS form_submission_history (
    id            BIGSERIAL PRIMARY KEY,
    submission_id UUID NOT NULL,
    old_data      JSONB NOT NULL,
    changed_by    UUID NOT NULL,
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS input_audit_log (
    id            BIGSERIAL PRIMARY KEY,
    submission_id UUID,
    action        TEXT NOT NULL,
    actor_id      TEXT NOT NULL,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_input_audit AS ON UPDATE TO input_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_input_audit AS ON DELETE TO input_audit_log DO INSTEAD NOTHING;
