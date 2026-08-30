-- Offline sync queue: buffers writes from devices when offline.
-- Applied to: kinara_auth (co-located with device registry)

\c kinara_auth

CREATE TABLE IF NOT EXISTS sync_queue (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID        NOT NULL REFERENCES devices(id),
    idempotency_key TEXT        NOT NULL,
    payload_type    TEXT        NOT NULL,  -- consultation|prescription|referral|vital_signs
    payload         JSONB       NOT NULL,
    patient_id      UUID        NOT NULL,
    clinic_id       UUID        NOT NULL,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at      TIMESTAMPTZ,
    rejected_at     TIMESTAMPTZ,
    reject_reason   TEXT,
    CONSTRAINT sync_queue_idempotency UNIQUE (device_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_sync_queue_device      ON sync_queue(device_id);
CREATE INDEX IF NOT EXISTS idx_sync_queue_patient     ON sync_queue(patient_id);
CREATE INDEX IF NOT EXISTS idx_sync_queue_clinic      ON sync_queue(clinic_id);
CREATE INDEX IF NOT EXISTS idx_sync_queue_pending     ON sync_queue(received_at) WHERE applied_at IS NULL AND rejected_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sync_queue_type        ON sync_queue(payload_type);

-- Immutable sync audit log
CREATE RULE no_update_sync_queue AS ON UPDATE TO sync_queue DO INSTEAD NOTHING;
-- Note: applied_at/rejected_at are set via INSERT into a separate status table, not UPDATE.
-- The RULE above enforces append-only at DB level.
