-- Health compliance service schema
-- All tables are immutable. Signatures use ed25519 to detect tampering.

CREATE TABLE IF NOT EXISTS audit_entries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_ref     TEXT NOT NULL UNIQUE,
    service       TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   UUID NOT NULL,
    actor_id      UUID NOT NULL,
    actor_role    TEXT NOT NULL,
    action        TEXT NOT NULL,
    detail        TEXT,
    ip_address    TEXT NOT NULL DEFAULT '',
    signature     TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_audit_entries AS ON UPDATE TO audit_entries DO INSTEAD NOTHING;
CREATE RULE no_delete_audit_entries AS ON DELETE TO audit_entries DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS breach_attempts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service     TEXT NOT NULL,
    actor_id    UUID,
    ip_address  TEXT NOT NULL DEFAULT '',
    reason      TEXT NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved    BOOLEAN NOT NULL DEFAULT false,
    resolved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS encryption_status (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service          TEXT NOT NULL UNIQUE,
    total_fields     INTEGER NOT NULL DEFAULT 0,
    encrypted_fields INTEGER NOT NULL DEFAULT 0,
    algorithm        TEXT NOT NULL DEFAULT 'AES-256-GCM',
    last_verified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_compliant     BOOLEAN NOT NULL DEFAULT true,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS compliance_reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_ref   TEXT NOT NULL UNIQUE,
    standard     TEXT NOT NULL,
    country      CHAR(2) NOT NULL DEFAULT '',
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    total_events INTEGER NOT NULL DEFAULT 0,
    breach_count INTEGER NOT NULL DEFAULT 0,
    is_compliant BOOLEAN NOT NULL DEFAULT true,
    findings     TEXT NOT NULL DEFAULT '',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    generated_by UUID NOT NULL
);

CREATE RULE no_update_compliance_reports AS ON UPDATE TO compliance_reports DO INSTEAD NOTHING;
CREATE RULE no_delete_compliance_reports AS ON DELETE TO compliance_reports DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_audit_entries_resource   ON audit_entries(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_actor      ON audit_entries(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_service    ON audit_entries(service);
CREATE INDEX IF NOT EXISTS idx_audit_entries_created    ON audit_entries(created_at);
CREATE INDEX IF NOT EXISTS idx_breach_attempts_resolved ON breach_attempts(resolved) WHERE resolved = false;
