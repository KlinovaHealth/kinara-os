-- =============================================================================
-- Kinara OS — Audit Service
-- Migration : V202609010041__Audit_Service__Init.sql
-- Database  : kinara_audit
--
-- NOTE: This database may not have been created by V001. If running against
--       a fresh cluster, ensure `kinara_audit` has been created before applying
--       this migration (e.g. via the cluster bootstrap script).
--
-- Description: Initialises the centralised Audit Service schema. This service
--              IS the platform-wide audit log; it intentionally has NO separate
--              service-level audit_log table of its own — adding one would create
--              an infinite regress. Service-level audit logs within each bounded
--              context (e.g. port_audit_log, payment_audit_log) remain in their
--              own databases as before.
-- =============================================================================

\c kinara_audit;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- audit_events  (this table IS the audit log — no separate audit_audit_log)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_ref       TEXT        UNIQUE
                    DEFAULT ('AUD-' || UPPER(SUBSTRING(gen_random_uuid()::TEXT, 1, 8))),
    service_name    TEXT        NOT NULL,
    entity_type     TEXT        NOT NULL,
    entity_id       TEXT        NOT NULL,
    action          TEXT        NOT NULL
                    CHECK (action IN (
                        'create','read','update','delete',
                        'login','logout','export','share'
                    )),
    actor_id        TEXT        NOT NULL,
    actor_role      TEXT,
    ip_address      INET,
    user_agent      TEXT,
    old_data        JSONB,
    new_data        JSONB,
    outcome         TEXT        NOT NULL DEFAULT 'success'
                    CHECK (outcome IN ('success','failure','partial')),
    signature_hash  TEXT        NOT NULL,
    tenant_id       TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE audit_events IS
    'Central platform-wide audit event store for Kinara OS. '
    'Every service pushes structured audit events here in addition to maintaining '
    'its own local audit log table. This table is the single authoritative source '
    'for cross-service compliance reporting, security investigations, and data-export audits. '
    'Rows are append-only: UPDATE and DELETE are blocked by rules below. '
    'There is deliberately no separate audit_audit_log — this table is self-auditing.';

-- Immutability rules — this table cannot be mutated after insert
CREATE RULE no_update_audit_events AS ON UPDATE TO audit_events DO INSTEAD NOTHING;
CREATE RULE no_delete_audit_events AS ON DELETE TO audit_events DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- audit_reports
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_reports (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    report_ref      TEXT        UNIQUE,
    report_type     TEXT
                    CHECK (report_type IN ('compliance','security','access','data_change')),
    period_start    TIMESTAMPTZ,
    period_end      TIMESTAMPTZ,
    total_events    INT         NOT NULL DEFAULT 0,
    generated_by    UUID,
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    file_url        TEXT,
    tenant_id       TEXT
);

COMMENT ON TABLE audit_reports IS
    'Generated compliance and security reports produced from audit_events. '
    'Each report summarises events within a time window and is stored as a '
    'signed file (file_url) alongside its metadata for regulatory submission.';

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_audit_events_service_time
    ON audit_events(service_name, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_events_entity
    ON audit_events(entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_audit_events_actor_time
    ON audit_events(actor_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_time
    ON audit_events(tenant_id, occurred_at DESC);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- NOTE: Dropping audit_events destroys the platform-wide compliance record.
--       Requires approval from the Data Protection Officer before execution
--       in any non-development environment.
-- DROP TABLE IF EXISTS audit_reports CASCADE;
-- DROP TABLE IF EXISTS audit_events CASCADE;
