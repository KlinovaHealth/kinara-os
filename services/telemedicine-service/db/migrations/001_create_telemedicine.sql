-- Telemedicine service schema
-- Notes encrypted AES-256-GCM; prescriptions and audit_log are immutable.

CREATE TABLE IF NOT EXISTS doctors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id       UUID NOT NULL,
    full_name       TEXT NOT NULL,
    specialization  TEXT NOT NULL DEFAULT 'general',
    license_number  TEXT NOT NULL,
    is_available    BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS consultations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consult_ref      TEXT NOT NULL UNIQUE,
    patient_id       UUID NOT NULL,
    doctor_id        UUID NOT NULL REFERENCES doctors(id),
    clinic_id        UUID NOT NULL,
    type             TEXT NOT NULL CHECK (type IN ('video','audio','chat','in_person')),
    status           TEXT NOT NULL DEFAULT 'scheduled'
                        CHECK (status IN ('scheduled','in_progress','completed','cancelled','no_show')),
    chief_complaint  TEXT NOT NULL,
    scheduled_at     TIMESTAMPTZ NOT NULL,
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    duration_minutes INTEGER,
    cost_usd         NUMERIC(10,2) NOT NULL DEFAULT 5.00,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Prescriptions: append-only once issued
CREATE TABLE IF NOT EXISTS prescriptions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id  UUID NOT NULL REFERENCES consultations(id),
    patient_id       UUID NOT NULL,
    doctor_id        UUID NOT NULL,
    medication_name  TEXT NOT NULL,
    dosage           TEXT NOT NULL,
    frequency_days   INTEGER NOT NULL DEFAULT 7,
    instructions_enc TEXT NOT NULL,
    issued_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_prescriptions AS ON UPDATE TO prescriptions DO INSTEAD NOTHING;
CREATE RULE no_delete_prescriptions AS ON DELETE TO prescriptions DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS recording_metadata (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id  UUID NOT NULL REFERENCES consultations(id),
    storage_path     TEXT NOT NULL,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_recordings AS ON UPDATE TO recording_metadata DO INSTEAD NOTHING;
CREATE RULE no_delete_recordings AS ON DELETE TO recording_metadata DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS telemedicine_audit_log (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id  UUID,
    actor_id         UUID NOT NULL,
    action           TEXT NOT NULL,
    detail           TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_tele_audit AS ON UPDATE TO telemedicine_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_tele_audit AS ON DELETE TO telemedicine_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_consultations_patient   ON consultations(patient_id);
CREATE INDEX IF NOT EXISTS idx_consultations_doctor    ON consultations(doctor_id);
CREATE INDEX IF NOT EXISTS idx_consultations_status    ON consultations(status);
CREATE INDEX IF NOT EXISTS idx_consultations_scheduled ON consultations(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_doctors_available       ON doctors(is_available) WHERE is_available = true;
