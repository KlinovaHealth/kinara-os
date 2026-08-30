CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS appointments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    appointment_ref TEXT NOT NULL UNIQUE,
    patient_id      UUID NOT NULL,
    doctor_id       UUID NOT NULL,
    clinic_id       TEXT NOT NULL,
    scheduled_at    TIMESTAMPTZ NOT NULL,
    duration_min    INT NOT NULL DEFAULT 30,
    type            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'scheduled',
    notes           TEXT,
    reason          TEXT,
    cancelled_by    TEXT,
    completed_by    TEXT,
    tenant_id       TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_appointments_patient    ON appointments(patient_id);
CREATE INDEX IF NOT EXISTS idx_appointments_doctor     ON appointments(doctor_id);
CREATE INDEX IF NOT EXISTS idx_appointments_scheduled  ON appointments(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_appointments_tenant     ON appointments(tenant_id);
-- Composite index for per-clinic queries ordered by time.
CREATE INDEX IF NOT EXISTS idx_appointments_clinic_at  ON appointments(clinic_id, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_appointments_status     ON appointments(status);

CREATE TABLE IF NOT EXISTS appointment_audit_log (
    id             BIGSERIAL PRIMARY KEY,
    appointment_id UUID NOT NULL,
    action         TEXT NOT NULL,
    actor_id       TEXT NOT NULL,
    old_status     TEXT,
    new_status     TEXT,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_appointment ON appointment_audit_log(appointment_id, occurred_at);

-- Immutability rules: audit rows may never be changed or removed.
CREATE RULE no_update_audit AS ON UPDATE TO appointment_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_audit AS ON DELETE TO appointment_audit_log DO INSTEAD NOTHING;
