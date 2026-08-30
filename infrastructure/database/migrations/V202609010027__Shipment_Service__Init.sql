-- =============================================================================
-- V202609010027__Shipment_Service__Init.sql
-- Kinara OS — Shipment Service
-- =============================================================================
\c kinara_shipment;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- shipments
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS shipments (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tracking_code     TEXT        UNIQUE NOT NULL,
    sender_id         UUID        NOT NULL,
    recipient_name    TEXT        NOT NULL,
    recipient_phone   TEXT        NOT NULL,
    origin_address    TEXT        NOT NULL,
    origin_country    TEXT        NOT NULL,
    dest_address      TEXT        NOT NULL,
    dest_country      TEXT        NOT NULL,
    weight_kg         DOUBLE PRECISION    DEFAULT 0,
    length_cm         DOUBLE PRECISION    DEFAULT 0,
    width_cm          DOUBLE PRECISION    DEFAULT 0,
    height_cm         DOUBLE PRECISION    DEFAULT 0,
    declared_value    DOUBLE PRECISION    DEFAULT 0,
    currency          TEXT        DEFAULT 'USD',
    service_level     TEXT        DEFAULT 'standard'
                      CHECK (service_level IN ('standard', 'express', 'economy')),
    status            TEXT        DEFAULT 'created'
                      CHECK (status IN ('created', 'collected', 'in_transit', 'out_for_delivery', 'delivered', 'returned', 'lost')),
    freight_charge    DOUBLE PRECISION    DEFAULT 0,
    insurance_charge  DOUBLE PRECISION    DEFAULT 0,
    total_charge      DOUBLE PRECISION    DEFAULT 0,
    picked_at         TIMESTAMPTZ,
    delivered_at      TIMESTAMPTZ,
    notes             TEXT,
    tenant_id         TEXT,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE shipments IS 'End-to-end shipment records covering domestic and cross-border consignments';
COMMENT ON COLUMN shipments.tracking_code    IS 'Public-facing tracking number (e.g. KIN-2026-0000001)';
COMMENT ON COLUMN shipments.sender_id        IS 'UUID of the user or merchant sending the shipment';
COMMENT ON COLUMN shipments.declared_value   IS 'Customer-declared value in the specified currency — used for insurance';
COMMENT ON COLUMN shipments.service_level    IS 'Delivery speed tier chosen by the sender';
COMMENT ON COLUMN shipments.freight_charge   IS 'Base freight cost computed at booking';
COMMENT ON COLUMN shipments.insurance_charge IS 'Optional insurance premium on the declared value';
COMMENT ON COLUMN shipments.total_charge     IS 'Derived total: freight_charge + insurance_charge + surcharges';
COMMENT ON COLUMN shipments.tenant_id        IS 'Multi-tenant partition key (logistics operator or reseller)';

-- ---------------------------------------------------------------------------
-- shipment_events
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS shipment_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id  UUID        NOT NULL REFERENCES shipments(id),
    event_type   TEXT        NOT NULL,
    location     TEXT,
    description  TEXT,
    occurred_at  TIMESTAMPTZ DEFAULT NOW(),
    recorded_by  UUID
);

COMMENT ON TABLE shipment_events IS 'Tracking event timeline for each shipment — immutable append-only log';
COMMENT ON COLUMN shipment_events.event_type  IS 'Event classification (e.g. collected, hub_arrived, out_for_delivery)';
COMMENT ON COLUMN shipment_events.location    IS 'Human-readable location name or facility code where the event occurred';
COMMENT ON COLUMN shipment_events.recorded_by IS 'UUID of the agent, scanner, or system that generated the event';

-- ---------------------------------------------------------------------------
-- shipment_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS shipment_audit_log (
    id             BIGSERIAL    PRIMARY KEY,
    entity_id      UUID         NOT NULL,
    action         TEXT         NOT NULL,
    actor_id       TEXT         NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    signature_hash TEXT,
    ip_address     INET,
    occurred_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE shipment_audit_log IS 'Immutable audit trail for all shipment-service mutations';

CREATE RULE no_update_shipment_audit AS ON UPDATE TO shipment_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_shipment_audit AS ON DELETE TO shipment_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_shipments_sender         ON shipments(sender_id);
CREATE INDEX IF NOT EXISTS idx_shipments_status         ON shipments(status);
CREATE INDEX IF NOT EXISTS idx_shipments_origin_country ON shipments(origin_country);
CREATE INDEX IF NOT EXISTS idx_shipments_dest_country   ON shipments(dest_country);
CREATE INDEX IF NOT EXISTS idx_shipment_events_shipment ON shipment_events(shipment_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_shipment_audit_entity    ON shipment_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_shipment_audit_actor     ON shipment_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_shipment_audit_actor;
-- DROP INDEX IF EXISTS idx_shipment_audit_entity;
-- DROP INDEX IF EXISTS idx_shipment_events_shipment;
-- DROP INDEX IF EXISTS idx_shipments_dest_country;
-- DROP INDEX IF EXISTS idx_shipments_origin_country;
-- DROP INDEX IF EXISTS idx_shipments_status;
-- DROP INDEX IF EXISTS idx_shipments_sender;
-- DROP RULE IF EXISTS no_delete_shipment_audit ON shipment_audit_log;
-- DROP RULE IF EXISTS no_update_shipment_audit ON shipment_audit_log;
-- DROP TABLE IF EXISTS shipment_audit_log;
-- DROP TABLE IF EXISTS shipment_events;
-- DROP TABLE IF EXISTS shipments;
-- =============================================================================
