-- Telemedicine Service — Video/audio/chat consultation sessions and payments
-- Database: kinara_telemedicine
-- Stores session lifecycle, room metadata, ratings, and payment records for remote consultations
-- Note: run 'CREATE DATABASE kinara_telemedicine;' as superuser if not exists

\c kinara_telemedicine;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS telemedicine_sessions (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id       UUID          NOT NULL,
    doctor_id        UUID          NOT NULL,
    session_type     TEXT          NOT NULL
                     CHECK (session_type IN ('video','audio','chat')),
    status           TEXT          NOT NULL DEFAULT 'scheduled'
                     CHECK (status IN ('scheduled','active','completed','failed','cancelled')),
    scheduled_at     TIMESTAMPTZ   NOT NULL,
    started_at       TIMESTAMPTZ,
    ended_at         TIMESTAMPTZ,
    duration_seconds INT,
    room_url         TEXT,
    recording_url    TEXT,
    cost_usd         NUMERIC(10,2),
    currency         TEXT          NOT NULL DEFAULT 'XOF',
    cost_local       NUMERIC(12,2),
    platform         TEXT          NOT NULL DEFAULT 'kinara',
    patient_rating   INT           CHECK (patient_rating BETWEEN 1 AND 5),
    doctor_rating    INT           CHECK (doctor_rating  BETWEEN 1 AND 5),
    notes_enc        TEXT,
    tenant_id        TEXT          NOT NULL,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  telemedicine_sessions              IS 'Remote consultation sessions — video, audio, and chat channels';
COMMENT ON COLUMN telemedicine_sessions.room_url     IS 'Video provider room URL — short-lived token; rotate before each session';
COMMENT ON COLUMN telemedicine_sessions.recording_url IS 'Encrypted recording storage URL; null if recording disabled';
COMMENT ON COLUMN telemedicine_sessions.cost_usd     IS 'Session cost in USD for cross-currency reporting';
COMMENT ON COLUMN telemedicine_sessions.currency     IS 'Local billing currency; defaults to West African CFA Franc (XOF)';
COMMENT ON COLUMN telemedicine_sessions.platform     IS 'Delivery platform — kinara native or third-party';
COMMENT ON COLUMN telemedicine_sessions.notes_enc    IS 'Post-session clinical notes, AES-256-GCM encrypted';

CREATE INDEX IF NOT EXISTS idx_tele_patient    ON telemedicine_sessions(patient_id);
CREATE INDEX IF NOT EXISTS idx_tele_doctor     ON telemedicine_sessions(doctor_id);
CREATE INDEX IF NOT EXISTS idx_tele_status     ON telemedicine_sessions(status);
CREATE INDEX IF NOT EXISTS idx_tele_scheduled  ON telemedicine_sessions(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_tele_tenant     ON telemedicine_sessions(tenant_id);

CREATE TABLE IF NOT EXISTS telemedicine_payments (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID          NOT NULL,
    amount      NUMERIC(12,2) NOT NULL,
    currency    TEXT          NOT NULL,
    txn_ref     TEXT          NOT NULL UNIQUE,
    status      TEXT          NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','completed','failed','refunded')),
    paid_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  telemedicine_payments         IS 'Payment records for telemedicine sessions — linked to session by session_id';
COMMENT ON COLUMN telemedicine_payments.txn_ref IS 'External payment gateway transaction reference — unique per payment';
COMMENT ON COLUMN telemedicine_payments.status  IS 'Payment state: pending, completed, failed, refunded';

CREATE INDEX IF NOT EXISTS idx_tele_pay_session ON telemedicine_payments(session_id);
CREATE INDEX IF NOT EXISTS idx_tele_pay_status  ON telemedicine_payments(status);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS telemedicine_audit_log (
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

COMMENT ON TABLE telemedicine_audit_log IS 'Append-only audit trail for telemedicine sessions and payments';

CREATE RULE no_update_telemedicine_audit AS ON UPDATE TO telemedicine_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_telemedicine_audit AS ON DELETE TO telemedicine_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_tele_audit_entity ON telemedicine_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_tele_audit_actor  ON telemedicine_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS telemedicine_audit_log CASCADE;
-- DROP TABLE IF EXISTS telemedicine_payments CASCADE;
-- DROP TABLE IF EXISTS telemedicine_sessions CASCADE;
