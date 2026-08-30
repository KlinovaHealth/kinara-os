-- V048: SMS Gateway audit log
-- Immutable record of every inbound SMS and its resolved response.
\c kinara_sms;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS sms_audit_logs (
    log_id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    phone           VARCHAR(20)   NOT NULL,
    message_in      TEXT          NOT NULL,
    intent          VARCHAR(100),
    intent_confidence FLOAT       CHECK (intent_confidence IS NULL OR (intent_confidence >= 0 AND intent_confidence <= 1)),
    parameters      JSONB,
    service_called  VARCHAR(100),
    service_request JSONB,
    service_response JSONB,
    message_out     TEXT          NOT NULL,
    status          VARCHAR(50)   NOT NULL DEFAULT 'ok',
    error_message   TEXT,
    provider        VARCHAR(50)   NOT NULL DEFAULT 'twilio',
    country_code    CHAR(2),
    duration_ms     INTEGER       CHECK (duration_ms IS NULL OR duration_ms >= 0),
    timestamp       TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    signature_hash  VARCHAR(256)
);

-- Immutable audit log — no row may ever be modified or removed.
CREATE RULE sms_audit_no_update AS ON UPDATE TO sms_audit_logs DO INSTEAD NOTHING;
CREATE RULE sms_audit_no_delete AS ON DELETE TO sms_audit_logs DO INSTEAD NOTHING;

-- Indexes for operational queries.
CREATE INDEX IF NOT EXISTS idx_sms_audit_phone     ON sms_audit_logs (phone);
CREATE INDEX IF NOT EXISTS idx_sms_audit_timestamp ON sms_audit_logs (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_sms_audit_intent    ON sms_audit_logs (intent);
CREATE INDEX IF NOT EXISTS idx_sms_audit_status    ON sms_audit_logs (status);
CREATE INDEX IF NOT EXISTS idx_sms_audit_provider  ON sms_audit_logs (provider);

COMMENT ON TABLE  sms_audit_logs IS 'Immutable audit trail of all SMS interactions with Kinara OS.';
COMMENT ON COLUMN sms_audit_logs.signature_hash    IS 'HMAC-SHA256 of (phone||timestamp||message_in) for tamper detection.';
COMMENT ON COLUMN sms_audit_logs.intent_confidence IS 'Parser confidence score 0.0–1.0 for intent classification.';
COMMENT ON COLUMN sms_audit_logs.parameters        IS 'Extracted SMS parameters (crop, patient_ref, amount, etc.).';
COMMENT ON COLUMN sms_audit_logs.service_called    IS 'Downstream microservice contacted to resolve the intent.';
COMMENT ON COLUMN sms_audit_logs.duration_ms       IS 'Total processing time from receipt to SMS send, in milliseconds.';

-- DOWN: DROP TABLE sms_audit_logs;
