-- =============================================================================
-- Kinara OS — Cargo Service
-- Migration : V202609010032__Cargo_Service__Init.sql
-- Database  : kinara_cargo
-- Description: Initialises the Cargo Service schema: cargo bookings, tracking
--              events, and an immutable audit log.
-- =============================================================================

\c kinara_cargo;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- cargo_bookings
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cargo_bookings (
    id                    UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_ref           TEXT           UNIQUE NOT NULL
                          DEFAULT ('KN-' || UPPER(SUBSTRING(gen_random_uuid()::TEXT, 1, 8))),
    shipper_id            UUID           NOT NULL,
    consignee_id          UUID,
    cargo_type            TEXT           NOT NULL DEFAULT 'general'
                          CHECK (cargo_type IN (
                              'general','perishable','fragile','hazardous',
                              'livestock','bulk_grain','medical','refrigerated'
                          )),
    description           TEXT           NOT NULL DEFAULT '',
    weight_kg             DOUBLE PRECISION
                          CHECK (weight_kg > 0),
    volume_m3             DOUBLE PRECISION NOT NULL DEFAULT 0,
    status                TEXT           NOT NULL DEFAULT 'pending'
                          CHECK (status IN (
                              'pending','picked_up','in_transit',
                              'delivered','cancelled','failed'
                          )),
    origin_address        TEXT           NOT NULL,
    origin_lat            DOUBLE PRECISION NOT NULL DEFAULT 0,
    origin_lng            DOUBLE PRECISION NOT NULL DEFAULT 0,
    destination_address   TEXT           NOT NULL,
    destination_lat       DOUBLE PRECISION NOT NULL DEFAULT 0,
    destination_lng       DOUBLE PRECISION NOT NULL DEFAULT 0,
    pickup_at             TIMESTAMPTZ,
    delivered_at          TIMESTAMPTZ,
    assigned_vehicle_id   UUID,
    assigned_driver_id    UUID,
    estimated_delivery    TIMESTAMPTZ,
    freight_cost          NUMERIC(12,2)  NOT NULL DEFAULT 0,
    currency              TEXT           NOT NULL DEFAULT 'XOF',
    tenant_id             TEXT,
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE cargo_bookings IS
    'Core cargo booking record for the Kinara OS logistics pillar. '
    'Covers origin-to-destination movement for all cargo types including perishables, '
    'hazardous goods, livestock, and medical supplies. Supports multi-tenant deployments.';

-- ---------------------------------------------------------------------------
-- cargo_tracking_events
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cargo_tracking_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id   UUID        NOT NULL REFERENCES cargo_bookings(id) ON DELETE CASCADE,
    event_type   TEXT        NOT NULL,
    description  TEXT,
    location     TEXT,
    gps_lat      DOUBLE PRECISION,
    gps_lng      DOUBLE PRECISION,
    recorded_by  UUID,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE cargo_tracking_events IS
    'Granular tracking events appended as cargo moves through the supply chain. '
    'Each event records GPS position, human-readable location, and the reporting actor.';

-- ---------------------------------------------------------------------------
-- cargo_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cargo_audit_log (
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

COMMENT ON TABLE cargo_audit_log IS
    'Immutable append-only audit trail for all Cargo Service write operations. '
    'UPDATE and DELETE are blocked by rules to preserve chain-of-custody integrity.';

CREATE RULE no_update_cargo_audit AS ON UPDATE TO cargo_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_cargo_audit AS ON DELETE TO cargo_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_cargo_audit_entity
    ON cargo_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_cargo_audit_actor
    ON cargo_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_cargo_bookings_shipper
    ON cargo_bookings(shipper_id);

CREATE INDEX IF NOT EXISTS idx_cargo_bookings_status
    ON cargo_bookings(status);

CREATE INDEX IF NOT EXISTS idx_cargo_bookings_driver
    ON cargo_bookings(assigned_driver_id);

CREATE INDEX IF NOT EXISTS idx_cargo_tracking_booking_time
    ON cargo_tracking_events(booking_id, occurred_at);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS cargo_audit_log CASCADE;
-- DROP TABLE IF EXISTS cargo_tracking_events CASCADE;
-- DROP TABLE IF EXISTS cargo_bookings CASCADE;
