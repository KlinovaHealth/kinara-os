-- =============================================================================
-- Kinara OS — Shipping Service
-- Migration : V202609010034__Shipping_Service__Init.sql
-- Database  : kinara_shipping
-- Description: Initialises the Shipping Service schema: shipping lines,
--              vessel schedules, container bookings, and an immutable audit log.
-- =============================================================================

\c kinara_shipping;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- shipping_lines
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS shipping_lines (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    code        TEXT        UNIQUE NOT NULL,
    country     TEXT,
    status      TEXT        NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','suspended','inactive')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE shipping_lines IS
    'Registry of shipping line operators whose vessels and schedules are managed '
    'within Kinara OS. Includes African and international carriers serving AU trade routes.';

-- ---------------------------------------------------------------------------
-- shipping_schedules
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS shipping_schedules (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    shipping_line_id    UUID        REFERENCES shipping_lines(id),
    vessel_id           UUID,
    origin_port_code    TEXT        NOT NULL,
    dest_port_code      TEXT        NOT NULL,
    departure_date      DATE        NOT NULL,
    arrival_date        DATE,
    service_name        TEXT,
    voyage_number       TEXT,
    available_teu       INT         NOT NULL DEFAULT 0,
    booked_teu          INT         NOT NULL DEFAULT 0,
    freight_rate_usd    NUMERIC(12,4),
    status              TEXT        NOT NULL DEFAULT 'open'
                        CHECK (status IN ('open','full','departed','arrived','cancelled')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE shipping_schedules IS
    'Published vessel sailing schedules between port pairs. '
    'Tracks TEU capacity, bookings, and freight rates to support container booking workflows.';

-- ---------------------------------------------------------------------------
-- container_bookings
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS container_bookings (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_ref      TEXT           UNIQUE,
    schedule_id      UUID           REFERENCES shipping_schedules(id),
    shipper_id       UUID           NOT NULL,
    cargo_type       TEXT
                     CHECK (cargo_type IN ('general','reefer','hazmat','out_of_gauge')),
    container_size   TEXT
                     CHECK (container_size IN ('20ft','40ft','40hc','45hc')),
    container_count  INT            NOT NULL DEFAULT 1,
    weight_tonnes    NUMERIC(10,3),
    status           TEXT           NOT NULL DEFAULT 'pending'
                     CHECK (status IN (
                         'pending','confirmed','loaded','shipped','delivered','cancelled'
                     )),
    bl_number        TEXT,
    freight_total    NUMERIC(14,4),
    currency         TEXT           NOT NULL DEFAULT 'USD',
    booked_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE container_bookings IS
    'Container slot bookings placed against a published shipping schedule. '
    'Tracks container type, count, weight, bill of lading number, and booking lifecycle status.';

-- ---------------------------------------------------------------------------
-- shipping_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS shipping_audit_log (
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

COMMENT ON TABLE shipping_audit_log IS
    'Immutable append-only audit trail for all Shipping Service write operations. '
    'UPDATE and DELETE are blocked by rules to preserve booking and schedule history.';

CREATE RULE no_update_shipping_audit AS ON UPDATE TO shipping_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_shipping_audit AS ON DELETE TO shipping_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_shipping_audit_entity
    ON shipping_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_shipping_audit_actor
    ON shipping_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_shipping_schedules_origin_departure
    ON shipping_schedules(origin_port_code, departure_date);

CREATE INDEX IF NOT EXISTS idx_shipping_schedules_dest
    ON shipping_schedules(dest_port_code);

CREATE INDEX IF NOT EXISTS idx_container_bookings_shipper
    ON container_bookings(shipper_id);

CREATE INDEX IF NOT EXISTS idx_container_bookings_schedule_status
    ON container_bookings(schedule_id, status);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS shipping_audit_log CASCADE;
-- DROP TABLE IF EXISTS container_bookings CASCADE;
-- DROP TABLE IF EXISTS shipping_schedules CASCADE;
-- DROP TABLE IF EXISTS shipping_lines CASCADE;
