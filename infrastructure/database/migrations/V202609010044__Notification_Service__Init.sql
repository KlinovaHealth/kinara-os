-- =============================================================================
-- Kinara OS — Notification Service
-- Migration : V202609010044__Notification_Service__Init.sql
-- Database  : kinara_notification
-- Description: Initialises the Notification Service schema: multi-channel
--              outbound notifications, localised templates, user channel
--              preferences, and an immutable audit log.
-- =============================================================================

\c kinara_notification;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- notifications
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_ref TEXT        UNIQUE,
    recipient_id     UUID        NOT NULL,
    recipient_type   TEXT
                     CHECK (recipient_type IN (
                         'patient','farmer','clinic','driver','admin'
                     )),
    channel          TEXT        NOT NULL
                     CHECK (channel IN ('sms','push','email','whatsapp','in_app')),
    template_key     TEXT,
    title            TEXT,
    body             TEXT        NOT NULL,
    data             JSONB       NOT NULL DEFAULT '{}',
    priority         TEXT        NOT NULL DEFAULT 'normal'
                     CHECK (priority IN ('low','normal','high','critical')),
    status           TEXT        NOT NULL DEFAULT 'pending'
                     CHECK (status IN (
                         'pending','sent','delivered','failed','cancelled'
                     )),
    sent_at          TIMESTAMPTZ,
    delivered_at     TIMESTAMPTZ,
    failure_reason   TEXT,
    retry_count      INT         NOT NULL DEFAULT 0,
    tenant_id        TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE notifications IS
    'Outbound notification records for all Kinara OS participants. '
    'Supports five delivery channels: SMS (critical for low-connectivity areas), '
    'push, email, WhatsApp, and in-app. Priority tiers allow critical health '
    'alerts (e.g. epidemic reports) to pre-empt normal traffic. '
    'Retry logic is tracked via retry_count; failure reasons aid debugging.';

-- ---------------------------------------------------------------------------
-- notification_templates
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_templates (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    template_key   TEXT        UNIQUE NOT NULL,
    channel        TEXT        NOT NULL,
    language       TEXT        NOT NULL DEFAULT 'en'
                   CHECK (language IN ('en','fr','ha','sw')),
    subject        TEXT,
    body_template  TEXT        NOT NULL,
    variables      JSONB       NOT NULL DEFAULT '{}',
    is_active      BOOLEAN     NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE notification_templates IS
    'Localised notification templates used by the Kinara OS notification engine. '
    'Supports English, French, Hausa, and Swahili to cover the platform''s '
    'primary operating regions. The variables JSONB documents the substitution '
    'tokens expected in body_template (e.g. {{patient_name}}, {{appointment_time}}).';

-- ---------------------------------------------------------------------------
-- notification_preferences
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id           UUID        NOT NULL,
    channel           TEXT        NOT NULL,
    enabled           BOOLEAN     NOT NULL DEFAULT true,
    quiet_hours_start TIME,
    quiet_hours_end   TIME,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, channel)
);

COMMENT ON TABLE notification_preferences IS
    'Per-user, per-channel notification opt-in/out preferences. '
    'quiet_hours_start / quiet_hours_end suppress non-critical notifications '
    'during user-defined sleep windows, respecting local time zones '
    '(time-zone handling is performed at the application layer).';

-- ---------------------------------------------------------------------------
-- notification_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_audit_log (
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

COMMENT ON TABLE notification_audit_log IS
    'Immutable append-only audit trail for all Notification Service write operations. '
    'UPDATE and DELETE are blocked by rules to preserve evidence of message delivery '
    'for regulatory and patient-safety investigations.';

CREATE RULE no_update_notification_audit AS ON UPDATE TO notification_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_notification_audit AS ON DELETE TO notification_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_notification_audit_entity
    ON notification_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_notification_audit_actor
    ON notification_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_notifications_recipient_time
    ON notifications(recipient_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_status_time
    ON notifications(status, created_at);

CREATE INDEX IF NOT EXISTS idx_notifications_tenant_time
    ON notifications(tenant_id, created_at);

CREATE INDEX IF NOT EXISTS idx_notification_templates_key
    ON notification_templates(template_key);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS notification_audit_log CASCADE;
-- DROP TABLE IF EXISTS notification_preferences CASCADE;
-- DROP TABLE IF EXISTS notification_templates CASCADE;
-- DROP TABLE IF EXISTS notifications CASCADE;
