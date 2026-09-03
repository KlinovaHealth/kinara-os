-- Migration 002: Add entity_type and tenant_id to users for multi-tenancy
--
-- entity_type: which entity the user belongs to ('klinova' or 'vha').
--              Set at registration, signed into every JWT, never overridden by client.
-- tenant_id:   UUID of the owning entity row in the tenants table.
--              Used for DB-level row scoping on shared tables.

-- Tenants table (one row per entity)
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,   -- 'klinova' | 'vha'
    display_name TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed the two live tenants
INSERT INTO tenants (id, name, display_name) VALUES
    ('00000000-0000-0000-0000-000000000001', 'klinova', 'Klinova'),
    ('00000000-0000-0000-0000-000000000002', 'vha',     'Village Health Access')
ON CONFLICT (name) DO NOTHING;

-- Add tenant columns to users
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS entity_type TEXT NOT NULL DEFAULT 'klinova'
        CHECK (entity_type IN ('klinova', 'vha')),
    ADD COLUMN IF NOT EXISTS tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
        REFERENCES tenants(id);

CREATE INDEX IF NOT EXISTS idx_users_tenant_id    ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_entity_type  ON users(entity_type);

-- entity_type is nullable on access_log: pre-auth failures (rate limits, unknown usernames)
-- have no user context and cannot be tenant-scoped.
ALTER TABLE access_log
    ADD COLUMN IF NOT EXISTS entity_type TEXT CHECK (entity_type IN ('klinova', 'vha')),
    ADD COLUMN IF NOT EXISTS tenant_id   UUID REFERENCES tenants(id);

CREATE INDEX IF NOT EXISTS idx_access_log_entity_type ON access_log(entity_type);
CREATE INDEX IF NOT EXISTS idx_access_log_tenant_id   ON access_log(tenant_id);

-- Backfill sessions for cross-tenant scoping
ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS entity_type TEXT NOT NULL DEFAULT 'klinova'
        CHECK (entity_type IN ('klinova', 'vha')),
    ADD COLUMN IF NOT EXISTS tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

CREATE INDEX IF NOT EXISTS idx_sessions_entity_type ON sessions(entity_type);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant_id   ON sessions(tenant_id);
