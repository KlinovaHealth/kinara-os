-- Notification Service Schema
-- Cross-pillar alerts: SMS, Push, WhatsApp, Email, In-App

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── notification_templates ───────────────────────────────────────────────────
CREATE TABLE notification_templates (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type            TEXT        NOT NULL,
    channel         TEXT        NOT NULL
                        CHECK (channel IN ('sms','push','whatsapp','email','in_app')),
    name            TEXT        NOT NULL,
    subject_template TEXT       NOT NULL DEFAULT '',
    body_template   TEXT        NOT NULL,
    variables       TEXT[]      NOT NULL DEFAULT '{}',
    language        TEXT        NOT NULL DEFAULT 'en',
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_templates_type_channel ON notification_templates(type, channel);
CREATE INDEX idx_templates_language     ON notification_templates(language);

-- ─── user_preferences ────────────────────────────────────────────────────────
CREATE TABLE user_preferences (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL UNIQUE,
    sms_enabled         BOOLEAN     NOT NULL DEFAULT TRUE,
    push_enabled        BOOLEAN     NOT NULL DEFAULT TRUE,
    whatsapp_enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    email_enabled       BOOLEAN     NOT NULL DEFAULT TRUE,
    in_app_enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    quiet_hours_start   TEXT,       -- HH:MM format e.g. "22:00"
    quiet_hours_end     TEXT,       -- HH:MM format e.g. "07:00"
    timezone            TEXT        NOT NULL DEFAULT 'Africa/Accra',
    language            TEXT        NOT NULL DEFAULT 'en',
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_preferences_user ON user_preferences(user_id);

-- ─── notifications ────────────────────────────────────────────────────────────
-- message and recipient encrypted end-to-end (PHI/PII in message bodies and phone numbers)
CREATE TABLE notifications (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    type            TEXT        NOT NULL,
    channel         TEXT        NOT NULL
                        CHECK (channel IN ('sms','push','whatsapp','email','in_app')),
    priority        TEXT        NOT NULL DEFAULT 'normal'
                        CHECK (priority IN ('low','normal','high','critical')),
    message_enc     TEXT        NOT NULL,
    subject_enc     TEXT        NOT NULL DEFAULT '',
    recipient_enc   TEXT        NOT NULL,
    template_id     UUID        REFERENCES notification_templates(id),
    status          TEXT        NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','queued','sent','delivered','failed','cancelled')),
    retry_count     INTEGER     NOT NULL DEFAULT 0,
    scheduled_at    TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    failure_reason  TEXT        NOT NULL DEFAULT '',
    external_id     TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user       ON notifications(user_id);
CREATE INDEX idx_notifications_status     ON notifications(status);
CREATE INDEX idx_notifications_type       ON notifications(type);
CREATE INDEX idx_notifications_channel    ON notifications(channel);
CREATE INDEX idx_notifications_priority   ON notifications(priority);
CREATE INDEX idx_notifications_scheduled  ON notifications(scheduled_at) WHERE scheduled_at IS NOT NULL;

-- ─── notification_audit_log ───────────────────────────────────────────────────
CREATE TABLE notification_audit_log (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id     UUID,
    user_id             UUID        NOT NULL,
    action              TEXT        NOT NULL,
    resource            TEXT        NOT NULL,
    ip_address          TEXT        NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notif_audit_user ON notification_audit_log(user_id);

CREATE RULE notification_audit_log_no_update AS
    ON UPDATE TO notification_audit_log DO INSTEAD NOTHING;

CREATE RULE notification_audit_log_no_delete AS
    ON DELETE TO notification_audit_log DO INSTEAD NOTHING;

-- ─── Seed templates for all four pillars ─────────────────────────────────────

INSERT INTO notification_templates (type, channel, name, subject_template, body_template, variables, language) VALUES
-- Health — SMS
('appointment_reminder', 'sms', 'Appointment Reminder SMS',
 '',
 'Reminder: You have an appointment at {{clinic_name}} on {{date}} at {{time}}. Reply CONFIRM or CANCEL.',
 ARRAY['clinic_name','date','time'], 'en'),

('appointment_reminder', 'whatsapp', 'Appointment Reminder WhatsApp',
 '',
 '👋 Hello {{patient_name}}, this is a reminder of your appointment at {{clinic_name}} on {{date}} at {{time}}.',
 ARRAY['patient_name','clinic_name','date','time'], 'en'),

('prescription_alert', 'sms', 'Prescription Ready SMS',
 '',
 'Your prescription for {{medication}} is ready for pickup at {{pharmacy_name}}.',
 ARRAY['medication','pharmacy_name'], 'en'),

('referral_status', 'sms', 'Referral Status Update SMS',
 '',
 'Your referral to {{clinic_name}} has been {{status}}. Ref: {{referral_id}}.',
 ARRAY['clinic_name','status','referral_id'], 'en'),

-- Agriculture — SMS
('price_alert', 'sms', 'Market Price Alert SMS',
 '',
 'Price alert: {{commodity}} is now trading at {{price}} {{currency}} per {{unit}} at {{market}}.',
 ARRAY['commodity','price','currency','unit','market'], 'en'),

('weather_alert', 'sms', 'Weather Alert SMS',
 '',
 'WEATHER ALERT for {{region}}: {{alert_type}}. {{message}} Take necessary precautions.',
 ARRAY['region','alert_type','message'], 'en'),

-- Logistics — Push
('delivery_status', 'push', 'Delivery Status Push',
 'Delivery Update',
 'Your shipment {{shipment_id}} is now {{status}}. ETA: {{eta}}.',
 ARRAY['shipment_id','status','eta'], 'en'),

('fleet_alert', 'push', 'Fleet Alert Push',
 'Fleet Alert',
 'Vehicle {{vehicle_id}} alert: {{alert_type}}. Location: {{location}}.',
 ARRAY['vehicle_id','alert_type','location'], 'en'),

-- Maritime — Email
('port_alert', 'email', 'Port Alert Email',
 'Port Alert: {{port_name}} — {{alert_type}}',
 'Dear {{recipient_name}},\n\nAn alert has been issued for {{port_name}}:\n\n{{message}}\n\nPlease take appropriate action.',
 ARRAY['recipient_name','port_name','alert_type','message'], 'en'),

-- System
('system_alert', 'email', 'System Alert Email',
 'System Alert: {{alert_type}}',
 'A system alert has been triggered:\n\nType: {{alert_type}}\nMessage: {{message}}\nTime: {{timestamp}}',
 ARRAY['alert_type','message','timestamp'], 'en');
