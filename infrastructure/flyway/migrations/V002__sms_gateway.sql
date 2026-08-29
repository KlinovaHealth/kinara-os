\c kinara_sms;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE sms_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(20) NOT NULL,
    direction VARCHAR(10) NOT NULL DEFAULT 'inbound',
    from_number VARCHAR(30) NOT NULL,
    to_number VARCHAR(30),
    body TEXT NOT NULL,
    response TEXT,
    command VARCHAR(30),
    success BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_sms_logs AS ON UPDATE TO sms_logs DO INSTEAD NOTHING;
CREATE RULE no_delete_sms_logs AS ON DELETE TO sms_logs DO INSTEAD NOTHING;

CREATE INDEX idx_sms_logs_from ON sms_logs(from_number);
CREATE INDEX idx_sms_logs_created ON sms_logs(created_at DESC);
CREATE INDEX idx_sms_logs_command ON sms_logs(command);
