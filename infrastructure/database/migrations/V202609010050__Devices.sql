-- Device Registry: tracks enrolled tablets/phones that can cache PHI offline.
-- Applied to: kinara_auth (device enrollment coupled with identity)

\c kinara_auth

CREATE TABLE IF NOT EXISTS clinics (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    region      TEXT        NOT NULL,
    country     TEXT        NOT NULL DEFAULT 'TG',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS devices (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    device_name         TEXT        NOT NULL,
    clinic_id           UUID        NOT NULL REFERENCES clinics(id),
    assigned_staff_id   UUID,
    device_secret_hash  TEXT        NOT NULL,
    enrolled_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ,
    revoked_at          TIMESTAMPTZ,
    revoked_reason      TEXT,
    CONSTRAINT device_name_clinic_unique UNIQUE (device_name, clinic_id)
);

CREATE INDEX IF NOT EXISTS idx_devices_clinic        ON devices(clinic_id);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen     ON devices(last_seen_at);
CREATE INDEX IF NOT EXISTS idx_devices_revoked       ON devices(revoked_at) WHERE revoked_at IS NULL;

-- Immutable audit log for device events
CREATE TABLE IF NOT EXISTS device_audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID        NOT NULL REFERENCES devices(id),
    event       TEXT        NOT NULL,  -- enrolled|heartbeat|revoked|wipe_triggered
    actor_id    UUID,
    ip_address  TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_device_audit AS ON UPDATE TO device_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_device_audit AS ON DELETE TO device_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_device_audit_device   ON device_audit_log(device_id);
CREATE INDEX IF NOT EXISTS idx_device_audit_event    ON device_audit_log(event);

-- Seed Togo pilot clinic
INSERT INTO clinics (id, name, region, country)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Lomé Health Centre 1',
    'Maritime',
    'TG'
) ON CONFLICT DO NOTHING;
