-- =============================================================================
-- Kinara OS — Voyage Service
-- Migration : V202609010047__Voyage_Service__Init.sql
-- Database  : kinara_voyage
-- Description: Initialises the Voyage Service schema: voyage records, cargo
--              manifests, manifest line items, voyage events, and an immutable
--              audit log. This is the final migration in the V31-V47 batch.
-- =============================================================================

\c kinara_voyage;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- voyages
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS voyages (
    id                    UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_ref            TEXT           UNIQUE NOT NULL,
    vessel_id             UUID           NOT NULL,
    departure_port_code   TEXT           NOT NULL,
    arrival_port_code     TEXT           NOT NULL,
    departure_at          TIMESTAMPTZ,
    arrival_at            TIMESTAMPTZ,
    estimated_arrival     TIMESTAMPTZ,
    status                TEXT           NOT NULL DEFAULT 'planned'
                          CHECK (status IN (
                              'planned','underway','arrived','cancelled','diverted'
                          )),
    captain_id            UUID,
    cargo_manifest_id     UUID,          -- denormalised FK; FK constraint added below
    waypoints             JSONB          NOT NULL DEFAULT '[]',
    distance_nm           NUMERIC(10,3),
    fuel_consumed_tonnes  NUMERIC(10,3),
    notes                 TEXT,
    tenant_id             TEXT,
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE voyages IS
    'Master voyage records for the Kinara OS maritime pillar. '
    'Each voyage links a vessel to a departure and arrival port pair, '
    'carries a JSONB waypoints array for intermediate stops, and tracks '
    'fuel consumption and nautical distance for environmental and cost reporting. '
    'cargo_manifest_id is denormalised here for fast lookups; the authoritative '
    'FK runs from cargo_manifests.voyage_id back to this table.';

-- ---------------------------------------------------------------------------
-- cargo_manifests
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cargo_manifests (
    id                    UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id             UUID           REFERENCES voyages(id),
    manifest_ref          TEXT           UNIQUE,
    total_units           INT            NOT NULL DEFAULT 0,
    total_teu             NUMERIC(8,2)   NOT NULL DEFAULT 0,
    total_weight_tonnes   NUMERIC(12,3)  NOT NULL DEFAULT 0,
    total_value_usd       NUMERIC(14,4)  NOT NULL DEFAULT 0,
    status                TEXT           NOT NULL DEFAULT 'draft'
                          CHECK (status IN (
                              'draft','submitted','approved','amended','closed'
                          )),
    submitted_at          TIMESTAMPTZ,
    approved_at           TIMESTAMPTZ,
    approved_by           UUID
);

COMMENT ON TABLE cargo_manifests IS
    'Cargo manifests attached to a voyage. A manifest aggregates all cargo carried '
    'on a single voyage and must be approved by the port authority before departure. '
    'Aggregate totals (TEU, weight, value) are maintained here for fast reporting; '
    'line-level detail lives in manifest_items.';

-- Add forward-reference FK from voyages.cargo_manifest_id now that cargo_manifests exists
ALTER TABLE voyages
    ADD CONSTRAINT fk_voyages_cargo_manifest
    FOREIGN KEY (cargo_manifest_id) REFERENCES cargo_manifests(id)
    NOT VALID;  -- validated lazily to avoid locking on large tables

-- ---------------------------------------------------------------------------
-- manifest_items
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS manifest_items (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    manifest_id      UUID           NOT NULL REFERENCES cargo_manifests(id) ON DELETE CASCADE,
    item_ref         TEXT,
    description      TEXT           NOT NULL,
    hs_code          TEXT,
    quantity         INT            NOT NULL DEFAULT 1,
    weight_kg        NUMERIC(10,3),
    volume_m3        NUMERIC(8,3),
    shipper_id       UUID,
    consignee_id     UUID,
    origin_country   TEXT,
    dest_country     TEXT,
    declared_value   NUMERIC(12,4),
    currency         TEXT           NOT NULL DEFAULT 'USD'
);

COMMENT ON TABLE manifest_items IS
    'Individual cargo line items within a manifest. '
    'hs_code stores the Harmonised System commodity code for customs purposes. '
    'Each item tracks its own declared value, weight, volume, and shipper/consignee '
    'identifiers to support itemised customs duties and cargo insurance claims.';

-- ---------------------------------------------------------------------------
-- voyage_events
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS voyage_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id    UUID        NOT NULL REFERENCES voyages(id),
    event_type   TEXT
                 CHECK (event_type IN (
                     'departure','waypoint','arrival','diversion',
                     'delay','emergency','inspection'
                 )),
    location     TEXT,
    latitude     NUMERIC(9,6),
    longitude    NUMERIC(9,6),
    description  TEXT,
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_by  UUID
);

COMMENT ON TABLE voyage_events IS
    'Time-ordered events appended as a voyage progresses. '
    'Supports position reporting at waypoints, diversion notifications, '
    'delay records, emergency declarations, and port-state control inspections. '
    'Combined with vessel_positions (kinara_vessel) this provides a full track log.';

-- ---------------------------------------------------------------------------
-- voyage_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS voyage_audit_log (
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

COMMENT ON TABLE voyage_audit_log IS
    'Immutable append-only audit trail for all Voyage Service write operations. '
    'UPDATE and DELETE are blocked by rules to preserve voyage and manifest records '
    'for maritime authority inspection and insurance purposes.';

CREATE RULE no_update_voyage_audit AS ON UPDATE TO voyage_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_voyage_audit AS ON DELETE TO voyage_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_voyage_audit_entity
    ON voyage_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_voyage_audit_actor
    ON voyage_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_voyages_vessel_departure
    ON voyages(vessel_id, departure_at);

CREATE INDEX IF NOT EXISTS idx_voyages_status
    ON voyages(status);

CREATE INDEX IF NOT EXISTS idx_cargo_manifests_voyage
    ON cargo_manifests(voyage_id);

CREATE INDEX IF NOT EXISTS idx_manifest_items_manifest
    ON manifest_items(manifest_id);

CREATE INDEX IF NOT EXISTS idx_voyage_events_voyage_time
    ON voyage_events(voyage_id, recorded_at);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- ALTER TABLE voyages DROP CONSTRAINT IF EXISTS fk_voyages_cargo_manifest;
-- DROP TABLE IF EXISTS voyage_audit_log CASCADE;
-- DROP TABLE IF EXISTS voyage_events CASCADE;
-- DROP TABLE IF EXISTS manifest_items CASCADE;
-- DROP TABLE IF EXISTS cargo_manifests CASCADE;
-- DROP TABLE IF EXISTS voyages CASCADE;
