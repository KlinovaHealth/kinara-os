CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS immunization_records (
    id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    record_ref       TEXT        NOT NULL UNIQUE,
    patient_id       UUID        NOT NULL,
    vaccine_code     TEXT        NOT NULL,
    vaccine_name     TEXT        NOT NULL,
    dose_number      INT         NOT NULL DEFAULT 1,
    administered_by  UUID,
    administered_at  TIMESTAMPTZ NOT NULL,
    lot_number       TEXT,
    expiry_date      DATE,
    site_of_injection TEXT,
    clinic_id        TEXT        NOT NULL,
    next_dose_date   DATE,
    status           TEXT        NOT NULL DEFAULT 'completed',
    tenant_id        TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_imm_patient ON immunization_records(patient_id);
CREATE INDEX IF NOT EXISTS idx_imm_clinic  ON immunization_records(clinic_id);

CREATE TABLE IF NOT EXISTS immunization_alerts (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id UUID        NOT NULL,
    message    TEXT        NOT NULL,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS immunization_audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    record_id   UUID        NOT NULL,
    action      TEXT        NOT NULL,
    actor_id    TEXT        NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_imm_audit AS ON UPDATE TO immunization_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_imm_audit AS ON DELETE TO immunization_audit_log DO INSTEAD NOTHING;
