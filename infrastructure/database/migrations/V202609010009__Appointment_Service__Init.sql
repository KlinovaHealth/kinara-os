-- Appointment Service — Clinic appointment bookings and full lifecycle tracking
-- Database: kinara_appointment
-- Manages scheduled, confirmed, completed, cancelled, and no-show appointment states

\c kinara_appointment;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS appointments (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    appointment_ref  TEXT        NOT NULL UNIQUE,
    patient_id       UUID        NOT NULL,
    doctor_id        UUID        NOT NULL,
    clinic_id        TEXT        NOT NULL,
    scheduled_at     TIMESTAMPTZ NOT NULL,
    duration_min     INT         NOT NULL DEFAULT 30 CHECK (duration_min > 0),
    type             TEXT        NOT NULL DEFAULT 'consultation'
                     CHECK (type IN ('consultation','follow_up','procedure','emergency')),
    status           TEXT        NOT NULL DEFAULT 'scheduled'
                     CHECK (status IN ('scheduled','confirmed','completed','cancelled','no_show')),
    notes            TEXT,
    reason           TEXT,
    cancelled_by     TEXT,
    completed_by     TEXT,
    tenant_id        TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  appointments                  IS 'Clinic appointment bookings — full lifecycle tracking from scheduling to outcome';
COMMENT ON COLUMN appointments.appointment_ref  IS 'Human-readable unique reference, e.g. APT-20260915-0042';
COMMENT ON COLUMN appointments.duration_min     IS 'Planned slot length in minutes; must be positive';
COMMENT ON COLUMN appointments.type             IS 'Visit type: consultation, follow_up, procedure, emergency';
COMMENT ON COLUMN appointments.status           IS 'Current state: scheduled → confirmed → completed | cancelled | no_show';
COMMENT ON COLUMN appointments.cancelled_by     IS 'Actor who cancelled: patient, doctor, clinic, system';
COMMENT ON COLUMN appointments.completed_by     IS 'UUID or identifier of the clinician who marked completion';

CREATE INDEX IF NOT EXISTS idx_apt_patient        ON appointments(patient_id);
CREATE INDEX IF NOT EXISTS idx_apt_doctor         ON appointments(doctor_id);
CREATE INDEX IF NOT EXISTS idx_apt_clinic_sched   ON appointments(clinic_id, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_apt_status         ON appointments(status);
CREATE INDEX IF NOT EXISTS idx_apt_tenant         ON appointments(tenant_id, scheduled_at DESC);

-- Immutable audit log with status-transition columns for compliance reporting
CREATE TABLE IF NOT EXISTS appointment_audit_log (
    id             BIGSERIAL   PRIMARY KEY,
    entity_id      UUID        NOT NULL,
    action         TEXT        NOT NULL,  -- 'create','update','delete','read'
    actor_id       TEXT        NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    old_status     TEXT,
    new_status     TEXT,
    signature_hash TEXT,
    ip_address     INET,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  appointment_audit_log           IS 'Append-only audit trail for appointment changes — includes status transition columns for reporting';
COMMENT ON COLUMN appointment_audit_log.old_status IS 'Status before the update — denormalized for fast transition-frequency queries';
COMMENT ON COLUMN appointment_audit_log.new_status IS 'Status after the update — denormalized for fast transition-frequency queries';

CREATE RULE no_update_appointment_audit AS ON UPDATE TO appointment_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_appointment_audit AS ON DELETE TO appointment_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_appointment_audit_entity ON appointment_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_appointment_audit_actor  ON appointment_audit_log(actor_id,  occurred_at);
CREATE INDEX IF NOT EXISTS idx_appointment_audit_status ON appointment_audit_log(new_status, occurred_at DESC);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS appointment_audit_log CASCADE;
-- DROP TABLE IF EXISTS appointments CASCADE;
