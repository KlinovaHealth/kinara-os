-- =============================================================================
-- Kinara OS — Event Bus
-- Migration : V202609110052__Event_Bus__Init.sql
-- Database  : kinara_events
-- Prereq    : CREATE DATABASE kinara_events; (V003__event_bus_database.sql)
-- Description:
--   Append-only ledger of all domain events published across Kinara OS pillars.
--   Scope (tenant_id, entity_type, clinic_id) is always derived from the
--   publishing service's JWT claims — callers cannot supply these fields.
--   Events form chains via origin_event_id; hop depth is capped at 5 in
--   the application layer (pkg/events).
-- =============================================================================

\c kinara_events

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- events
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS events (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type             TEXT        NOT NULL,           -- dot-namespaced, e.g. clinical.visit.completed
    tenant_id        UUID        NOT NULL,
    entity_type      TEXT        NOT NULL,           -- "klinova" | "vha"
    clinic_id        UUID,                           -- null for non-clinic-scoped events
    origin_user_id   UUID,                           -- null for machine/agent-published events
    origin_event_id  UUID        REFERENCES events(id),  -- null for root events
    hop              INT         NOT NULL DEFAULT 0,
    payload          JSONB       NOT NULL,
    published_by     TEXT        NOT NULL,           -- service name or agent identifier
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------

-- Tenant-scoped time-range queries (primary consumer pattern)
CREATE INDEX IF NOT EXISTS idx_events_tenant_time
    ON events(tenant_id, created_at);

-- Type-based fan-out queries
CREATE INDEX IF NOT EXISTS idx_events_type_time
    ON events(type, created_at);

-- Event chain traversal; sparse — only rows with a parent need this index
CREATE INDEX IF NOT EXISTS idx_events_origin
    ON events(origin_event_id)
    WHERE origin_event_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Append-only enforcement
-- ---------------------------------------------------------------------------
CREATE RULE no_update_events
    AS ON UPDATE TO events DO INSTEAD NOTHING;

CREATE RULE no_delete_events
    AS ON DELETE TO events DO INSTEAD NOTHING;
