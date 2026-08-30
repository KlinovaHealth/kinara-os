-- Outbreak Service — Disease case reporting, outbreak detection, and notification dispatch
-- Database: kinara_outbreak
-- Tracks suspected cases, triggers confirmed outbreak alerts, and logs notifications sent

\c kinara_outbreak;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS suspected_cases (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_ref      TEXT        NOT NULL UNIQUE,
    patient_id    UUID        NOT NULL,
    disease_code  TEXT        NOT NULL,
    disease_name  TEXT        NOT NULL,
    clinic_id     TEXT        NOT NULL,
    location      TEXT,
    symptoms      TEXT,
    reported_by   UUID        NOT NULL,
    tenant_id     TEXT        NOT NULL,
    reported_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  suspected_cases              IS 'Individual disease case reports — threshold breach triggers confirmed_outbreaks entry';
COMMENT ON COLUMN suspected_cases.case_ref     IS 'Human-readable unique reference, e.g. CASE-20260901-0001';
COMMENT ON COLUMN suspected_cases.disease_code IS 'ICD-10 or WHO surveillance code for the suspected disease';
COMMENT ON COLUMN suspected_cases.location     IS 'Free-text location or GPS coordinates of the case — used for geo-clustering';
COMMENT ON COLUMN suspected_cases.symptoms     IS 'Reported symptom list — unstructured free text for rapid entry';

CREATE INDEX IF NOT EXISTS idx_sc_disease    ON suspected_cases(disease_code, reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_sc_clinic     ON suspected_cases(clinic_id, reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_sc_patient    ON suspected_cases(patient_id);

CREATE TABLE IF NOT EXISTS confirmed_outbreaks (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_ref     TEXT        NOT NULL UNIQUE,
    disease_code  TEXT        NOT NULL,
    disease_name  TEXT        NOT NULL,
    clinic_id     TEXT        NOT NULL,
    case_count    INT         NOT NULL DEFAULT 0 CHECK (case_count >= 0),
    status        TEXT        NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','confirmed','contained')),
    tenant_id     TEXT        NOT NULL,
    detected_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    contained_at  TIMESTAMPTZ
);

COMMENT ON TABLE  confirmed_outbreaks            IS 'Active outbreak alerts — one per disease/clinic while not contained';
COMMENT ON COLUMN confirmed_outbreaks.alert_ref  IS 'Human-readable alert reference, e.g. OB-20260901-0001';
COMMENT ON COLUMN confirmed_outbreaks.case_count IS 'Running count of linked suspected cases; incremented by trigger or service';
COMMENT ON COLUMN confirmed_outbreaks.status     IS 'Alert lifecycle: active (monitoring), confirmed (WHO notified), contained (resolved)';
COMMENT ON COLUMN confirmed_outbreaks.detected_at IS 'Timestamp when case count first exceeded epidemic threshold';

-- Partial unique index: only one active/confirmed outbreak per disease per clinic at a time
CREATE UNIQUE INDEX IF NOT EXISTS uidx_outbreak_active
    ON confirmed_outbreaks(disease_code, clinic_id)
    WHERE status != 'contained';

CREATE INDEX IF NOT EXISTS idx_ob_status      ON confirmed_outbreaks(status);
CREATE INDEX IF NOT EXISTS idx_ob_disease     ON confirmed_outbreaks(disease_code, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_ob_tenant      ON confirmed_outbreaks(tenant_id, detected_at DESC);

CREATE TABLE IF NOT EXISTS outbreak_notifications (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    outbreak_id  UUID        NOT NULL REFERENCES confirmed_outbreaks(id),
    message      TEXT        NOT NULL,
    recipients   TEXT        NOT NULL,
    sent_by      UUID        NOT NULL,
    sent_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  outbreak_notifications            IS 'Notifications dispatched for a confirmed outbreak — SMS, email, or internal alert';
COMMENT ON COLUMN outbreak_notifications.recipients IS 'Comma-separated list or role name of notification targets';
COMMENT ON COLUMN outbreak_notifications.message    IS 'Full text of the notification as sent';

CREATE INDEX IF NOT EXISTS idx_obnotif_outbreak ON outbreak_notifications(outbreak_id, sent_at DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS outbreak_audit_log (
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

COMMENT ON TABLE outbreak_audit_log IS 'Append-only audit trail for outbreak cases, alerts, and notification events';

CREATE RULE no_update_outbreak_audit AS ON UPDATE TO outbreak_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_outbreak_audit AS ON DELETE TO outbreak_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_outbreak_audit_entity ON outbreak_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_outbreak_audit_actor  ON outbreak_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS outbreak_audit_log CASCADE;
-- DROP TABLE IF EXISTS outbreak_notifications CASCADE;
-- DROP TABLE IF EXISTS confirmed_outbreaks CASCADE;
-- DROP TABLE IF EXISTS suspected_cases CASCADE;
