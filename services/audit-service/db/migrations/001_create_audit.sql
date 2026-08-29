-- Cross-pillar audit service schema.
-- audit_events is append-only; signatures use ed25519.

CREATE TABLE IF NOT EXISTS audit_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_ref     TEXT NOT NULL UNIQUE,
    service       TEXT NOT NULL,
    pillar        TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    actor_id      UUID NOT NULL,
    actor_role    TEXT NOT NULL DEFAULT '',
    resource_id   TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '',
    detail        TEXT NOT NULL DEFAULT '',
    ip_address    TEXT NOT NULL DEFAULT '',
    tenant_id     TEXT NOT NULL DEFAULT '',
    signature     TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_audit_events AS ON UPDATE TO audit_events DO INSTEAD NOTHING;
CREATE RULE no_delete_audit_events AS ON DELETE TO audit_events DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_audit_events_service    ON audit_events(service);
CREATE INDEX IF NOT EXISTS idx_audit_events_pillar     ON audit_events(pillar);
CREATE INDEX IF NOT EXISTS idx_audit_events_actor      ON audit_events(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant     ON audit_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_created    ON audit_events(created_at);
