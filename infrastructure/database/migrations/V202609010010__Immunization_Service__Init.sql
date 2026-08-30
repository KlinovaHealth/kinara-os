-- Immunization Service — Vaccination records, dose tracking, and overdue alerts
-- Database: kinara_immunization
-- Stores administered vaccine records and drives reminder notifications for missed doses

\c kinara_immunization;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS immunization_records (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    record_ref       TEXT        NOT NULL UNIQUE,
    patient_id       UUID        NOT NULL,
    vaccine_code     TEXT        NOT NULL,
    vaccine_name     TEXT        NOT NULL,
    dose_number      INT         NOT NULL DEFAULT 1 CHECK (dose_number >= 1),
    administered_by  UUID        NOT NULL,
    administered_at  TIMESTAMPTZ NOT NULL,
    lot_number       TEXT,
    expiry_date      DATE,
    site_of_injection TEXT,
    clinic_id        TEXT        NOT NULL,
    next_dose_date   DATE,
    status           TEXT        NOT NULL DEFAULT 'completed'
                     CHECK (status IN ('completed','missed','deferred')),
    tenant_id        TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  immunization_records                  IS 'Vaccination records per patient — supports multi-dose schedules via dose_number';
COMMENT ON COLUMN immunization_records.record_ref       IS 'Human-readable unique reference, e.g. IMM-20260901-0001';
COMMENT ON COLUMN immunization_records.vaccine_code     IS 'WHO/national vaccine code, e.g. BCG, OPV1, MCV1, PENTA3';
COMMENT ON COLUMN immunization_records.dose_number      IS 'Which dose in the schedule (1, 2, 3 …)';
COMMENT ON COLUMN immunization_records.lot_number       IS 'Vaccine lot/batch number for adverse event traceability';
COMMENT ON COLUMN immunization_records.expiry_date      IS 'Expiry date of the vaccine vial used';
COMMENT ON COLUMN immunization_records.site_of_injection IS 'Anatomical site, e.g. left_deltoid, right_thigh';
COMMENT ON COLUMN immunization_records.next_dose_date   IS 'Calculated due date for the next dose; used by alert engine';
COMMENT ON COLUMN immunization_records.status           IS 'Outcome: completed (given), missed (not given), deferred (postponed)';

CREATE INDEX IF NOT EXISTS idx_imm_patient     ON immunization_records(patient_id);
CREATE INDEX IF NOT EXISTS idx_imm_clinic      ON immunization_records(clinic_id);
CREATE INDEX IF NOT EXISTS idx_imm_vaccine     ON immunization_records(vaccine_code);
CREATE INDEX IF NOT EXISTS idx_imm_admin_date  ON immunization_records(administered_at DESC);
CREATE INDEX IF NOT EXISTS idx_imm_next_dose   ON immunization_records(next_dose_date) WHERE next_dose_date IS NOT NULL;

CREATE TABLE IF NOT EXISTS immunization_alerts (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID        NOT NULL,
    message     TEXT        NOT NULL,
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  immunization_alerts           IS 'Overdue vaccine reminder notifications dispatched to patients or caregivers';
COMMENT ON COLUMN immunization_alerts.message   IS 'Rendered alert text in patient locale — e.g. "Your child is due for MCV1 on 2026-10-01"';

CREATE INDEX IF NOT EXISTS idx_imm_alert_patient ON immunization_alerts(patient_id, sent_at DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS immunization_audit_log (
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

COMMENT ON TABLE immunization_audit_log IS 'Append-only audit trail for immunization records and alerts';

CREATE RULE no_update_immunization_audit AS ON UPDATE TO immunization_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_immunization_audit AS ON DELETE TO immunization_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_immunization_audit_entity ON immunization_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_immunization_audit_actor  ON immunization_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS immunization_audit_log CASCADE;
-- DROP TABLE IF EXISTS immunization_alerts CASCADE;
-- DROP TABLE IF EXISTS immunization_records CASCADE;
