-- =============================================================================
-- Kinara OS — Analytics Service
-- Migration : V202609010042__Analytics_Service__Init.sql
-- Database  : kinara_analytics
-- Description: Initialises the Analytics Service schema: raw event ingestion,
--              KPI snapshots, configurable dashboards, and an immutable audit log.
-- =============================================================================

\c kinara_analytics;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- analytics_events
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS analytics_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type    TEXT        NOT NULL,
    service_name  TEXT        NOT NULL,
    entity_type   TEXT,
    entity_id     UUID,
    properties    JSONB       NOT NULL DEFAULT '{}',
    country       TEXT,
    region        TEXT,
    user_id       UUID,
    session_id    TEXT,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id     TEXT
);

COMMENT ON TABLE analytics_events IS
    'Raw analytics event stream ingested from all Kinara OS services and pillars. '
    'Events are schema-flexible (properties JSONB) to accommodate diverse service domains. '
    'This table is the foundation for KPI calculations, trend analysis, and Gates Foundation '
    'impact reporting across health, agri, logistics, and maritime pillars.';

-- ---------------------------------------------------------------------------
-- kpi_snapshots
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS kpi_snapshots (
    id              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    kpi_name        TEXT           NOT NULL,
    pillar          TEXT
                    CHECK (pillar IN ('health','agri','logistics','maritime','governance')),
    value           NUMERIC(18,4)  NOT NULL,
    unit            TEXT,
    period_start    TIMESTAMPTZ,
    period_end      TIMESTAMPTZ,
    country         TEXT,
    region          TEXT,
    calculated_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    metadata        JSONB          NOT NULL DEFAULT '{}'
);

COMMENT ON TABLE kpi_snapshots IS
    'Pre-computed KPI snapshots calculated from analytics_events and service databases. '
    'Each row captures a KPI value for a specific pillar, geography, and time window. '
    'Snapshots support the Kinara OS dashboard layer and external reporting to the '
    'Gates Foundation (target: 150 services across Africa by January 2027).';

-- ---------------------------------------------------------------------------
-- dashboards
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS dashboards (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT        NOT NULL,
    description  TEXT,
    pillar       TEXT,
    config       JSONB       NOT NULL DEFAULT '{}',
    created_by   UUID,
    is_public    BOOLEAN     NOT NULL DEFAULT false,
    tenant_id    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE dashboards IS
    'Configurable dashboard definitions for Kinara OS analytics. '
    'The config JSONB field stores widget layout, KPI selections, filters, and chart types. '
    'Dashboards may be scoped to a pillar and tenant, or marked public for cross-tenant '
    'impact reporting.';

-- ---------------------------------------------------------------------------
-- analytics_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS analytics_audit_log (
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

COMMENT ON TABLE analytics_audit_log IS
    'Immutable append-only audit trail for all Analytics Service write operations. '
    'Tracks who created or modified dashboards and KPI configurations. '
    'UPDATE and DELETE are blocked by rules to ensure data lineage integrity.';

CREATE RULE no_update_analytics_audit AS ON UPDATE TO analytics_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_analytics_audit AS ON DELETE TO analytics_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_analytics_audit_entity
    ON analytics_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_analytics_audit_actor
    ON analytics_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_analytics_events_type_time
    ON analytics_events(event_type, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_events_service_time
    ON analytics_events(service_name, occurred_at);

CREATE INDEX IF NOT EXISTS idx_kpi_snapshots_pillar_name_period
    ON kpi_snapshots(pillar, kpi_name, period_start);

CREATE INDEX IF NOT EXISTS idx_kpi_snapshots_country_time
    ON kpi_snapshots(country, calculated_at);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS analytics_audit_log CASCADE;
-- DROP TABLE IF EXISTS dashboards CASCADE;
-- DROP TABLE IF EXISTS kpi_snapshots CASCADE;
-- DROP TABLE IF EXISTS analytics_events CASCADE;
