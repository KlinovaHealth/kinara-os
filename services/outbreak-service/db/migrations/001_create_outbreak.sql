CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS suspected_cases (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_ref     TEXT NOT NULL UNIQUE,
    patient_id   UUID NOT NULL,
    disease_code TEXT NOT NULL,
    disease_name TEXT NOT NULL,
    clinic_id    TEXT NOT NULL,
    location     TEXT,
    symptoms     TEXT,
    reported_by  UUID NOT NULL,
    tenant_id    TEXT NOT NULL,
    reported_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cases_disease_clinic ON suspected_cases(disease_code, clinic_id, reported_at);

CREATE TABLE IF NOT EXISTS confirmed_outbreaks (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_ref    TEXT NOT NULL UNIQUE,
    disease_code TEXT NOT NULL,
    disease_name TEXT NOT NULL,
    clinic_id    TEXT NOT NULL,
    case_count   INT NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'active',
    tenant_id    TEXT NOT NULL,
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    contained_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_outbreak_unique ON confirmed_outbreaks(disease_code, clinic_id) WHERE status != 'contained';

CREATE TABLE IF NOT EXISTS outbreak_notifications (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    outbreak_id UUID NOT NULL REFERENCES confirmed_outbreaks(id),
    message     TEXT NOT NULL,
    recipients  TEXT,
    sent_by     UUID NOT NULL,
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS outbreak_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    outbreak_id UUID,
    action      TEXT NOT NULL,
    actor_id    TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_outbreak_audit AS ON UPDATE TO outbreak_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_outbreak_audit AS ON DELETE TO outbreak_audit_log DO INSTEAD NOTHING;
