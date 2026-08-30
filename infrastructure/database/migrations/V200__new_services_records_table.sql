-- Phase 5: New services base schema
-- Applied to each of the 100 new service databases

-- Generic records table for all new services (stub; replaced per-service in Phase 6)
CREATE TABLE IF NOT EXISTS records (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    data_enc    TEXT        NOT NULL,
    created_by  UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Immutable audit log (DO INSTEAD NOTHING blocks direct UPDATE/DELETE)
CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    record_id   UUID,
    user_id     UUID        NOT NULL,
    action      TEXT        NOT NULL,
    ip_address  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_audit AS ON UPDATE TO audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_audit AS ON DELETE TO audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_records_created_at ON records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_created_by ON records(created_by);
CREATE INDEX IF NOT EXISTS idx_audit_log_record_id ON audit_log(record_id);
