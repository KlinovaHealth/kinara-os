-- =============================================================================
-- Kinara OS — SMS Gateway Service
-- Migration : V202609010046__SMS_Gateway__Init.sql
-- Database  : kinara_sms
--
-- NOTE: The earlier V202609010002__sms_gateway.sql (if it exists in your cluster)
--       may have created a table named `sms_logs`. This migration creates a
--       distinct, extended schema (sms_messages, sms_sessions, sms_commands).
--       If V002 is present, verify that there is no column or table name clash
--       before applying. This migration does NOT drop or alter V002 objects.
--
-- Description: Extended SMS Gateway schema: inbound/outbound messages with
--              command parsing, stateful USSD-style sessions, command registry
--              with bilingual seed data, and an immutable audit log.
-- =============================================================================

\c kinara_sms;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- sms_messages
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sms_messages (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    message_ref     TEXT        UNIQUE
                    DEFAULT ('SMS-' || UPPER(SUBSTRING(gen_random_uuid()::TEXT, 1, 8))),
    provider        TEXT        NOT NULL
                    CHECK (provider IN ('africastalking','twilio','vonage','infobip')),
    direction       TEXT        NOT NULL DEFAULT 'inbound'
                    CHECK (direction IN ('inbound','outbound')),
    from_number     TEXT        NOT NULL,
    to_number       TEXT,
    body            TEXT        NOT NULL,
    response        TEXT,
    command         TEXT
                    CHECK (command IN (
                        'REGISTER','REPORT','PRICE','HELP','AIDE',
                        'PATIENT','MALADE','SYMPTOM','SYMPTOME',
                        'APPT','RDV','LAB','LABO'
                    )),
    command_result  JSONB,
    success         BOOLEAN     NOT NULL DEFAULT true,
    error_code      TEXT,
    session_id      TEXT,
    tenant_id       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE sms_messages IS
    'Full SMS message log for the Kinara OS gateway. Covers inbound messages '
    'from patients and farmers (low-connectivity areas without smartphones) and '
    'outbound replies from the platform. Parsed command and command_result enable '
    'asynchronous processing and response tracking. Bilingual commands (English + French) '
    'are supported to serve Francophone West Africa.';

-- ---------------------------------------------------------------------------
-- sms_sessions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sms_sessions (
    session_id       TEXT        PRIMARY KEY,
    phone_number     TEXT        NOT NULL,
    session_state    TEXT,
    session_data     JSONB       NOT NULL DEFAULT '{}',
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at       TIMESTAMPTZ
);

COMMENT ON TABLE sms_sessions IS
    'Stateful SMS sessions enabling multi-step conversational flows via plain SMS '
    '(analogous to USSD menus). session_state is a free-form state machine key; '
    'session_data holds collected values between messages. Sessions expire after '
    'inactivity (expires_at is set by the application layer).';

-- ---------------------------------------------------------------------------
-- sms_commands
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sms_commands (
    command_name   TEXT        PRIMARY KEY,
    aliases        TEXT[],
    description    TEXT,
    allowed_roles  TEXT[],
    handler_url    TEXT,
    is_active      BOOLEAN     NOT NULL DEFAULT true
);

COMMENT ON TABLE sms_commands IS
    'Registry of recognised SMS commands and their French aliases. '
    'handler_url points to the internal microservice endpoint that processes '
    'the command. allowed_roles restricts which user roles may invoke each command.';

-- Seed: English commands and French aliases
INSERT INTO sms_commands (command_name, aliases, description, allowed_roles, handler_url, is_active) VALUES
    ('REGISTER',  ARRAY['INSCRIRE'],        'Register a new user or facility',
     ARRAY['patient','farmer','frontdesk'], '/sms/handler/register',  true),
    ('REPORT',    ARRAY['RAPPORT'],          'Submit a health or field report',
     ARRAY['patient','farmer','vet'],        '/sms/handler/report',    true),
    ('PRICE',     ARRAY['PRIX'],             'Query commodity market price',
     ARRAY['farmer','cooperative'],          '/sms/handler/price',     true),
    ('HELP',      ARRAY['AIDE'],             'Display available commands',
     ARRAY['patient','farmer','driver'],     '/sms/handler/help',      true),
    ('PATIENT',   ARRAY['MALADE'],           'Look up or create a patient record',
     ARRAY['doctor','nurse','frontdesk'],    '/sms/handler/patient',   true),
    ('SYMPTOM',   ARRAY['SYMPTOME'],         'Report or look up symptoms',
     ARRAY['patient','doctor','nurse'],      '/sms/handler/symptom',   true),
    ('APPT',      ARRAY['RDV'],              'Book or query an appointment',
     ARRAY['patient','frontdesk'],           '/sms/handler/appt',      true),
    ('LAB',       ARRAY['LABO'],             'Submit or retrieve lab order status',
     ARRAY['doctor','nurse','patient'],      '/sms/handler/lab',       true)
ON CONFLICT (command_name) DO NOTHING;

-- ---------------------------------------------------------------------------
-- sms_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sms_audit_log (
    id              BIGSERIAL   PRIMARY KEY,
    entity_id       UUID,
    action          TEXT        NOT NULL,
    actor_id        TEXT        NOT NULL,
    old_data        JSONB,
    new_data        JSONB,
    signature_hash  TEXT,
    ip_address      INET,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE sms_audit_log IS
    'Immutable append-only audit trail for all SMS Gateway write operations. '
    'UPDATE and DELETE are blocked by rules to preserve message and session records '
    'for patient-safety and regulatory review.';

CREATE RULE no_update_sms_audit AS ON UPDATE TO sms_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_sms_audit AS ON DELETE TO sms_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_sms_audit_entity
    ON sms_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_sms_audit_actor
    ON sms_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_sms_messages_from_number
    ON sms_messages(from_number);

CREATE INDEX IF NOT EXISTS idx_sms_messages_command_time
    ON sms_messages(command, created_at);

CREATE INDEX IF NOT EXISTS idx_sms_messages_created_desc
    ON sms_messages(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sms_sessions_phone
    ON sms_sessions(phone_number);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS sms_audit_log CASCADE;
-- DROP TABLE IF EXISTS sms_commands CASCADE;
-- DROP TABLE IF EXISTS sms_sessions CASCADE;
-- DROP TABLE IF EXISTS sms_messages CASCADE;
