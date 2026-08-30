-- =============================================================================
-- V202609010026__Last_Mile_Service__Init.sql
-- Kinara OS — Last Mile Delivery Service
-- =============================================================================
\c kinara_lastmile;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- delivery_orders
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS delivery_orders (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_ref          TEXT        UNIQUE,
    shipper_id         UUID        NOT NULL,
    recipient_name     TEXT        NOT NULL,
    recipient_phone    TEXT        NOT NULL,
    recipient_address  TEXT        NOT NULL,
    recipient_lat      DOUBLE PRECISION,
    recipient_lng      DOUBLE PRECISION,
    package_type       TEXT        DEFAULT 'parcel'
                       CHECK (package_type IN ('parcel', 'document', 'fragile', 'cold_chain')),
    weight_kg          DOUBLE PRECISION,
    status             TEXT        DEFAULT 'pending'
                       CHECK (status IN ('pending', 'assigned', 'picked_up', 'in_transit', 'delivered', 'failed', 'returned')),
    courier_id         UUID,
    picked_up_at       TIMESTAMPTZ,
    delivered_at       TIMESTAMPTZ,
    signature_url      TEXT,
    proof_photo_url    TEXT,
    tenant_id          TEXT,
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE delivery_orders IS 'Last-mile delivery orders from shippers to end recipients';
COMMENT ON COLUMN delivery_orders.shipper_id       IS 'UUID of the farmer, cooperative, or merchant dispatching the package';
COMMENT ON COLUMN delivery_orders.recipient_phone  IS 'Contact number for the recipient — used for delivery notifications';
COMMENT ON COLUMN delivery_orders.package_type     IS 'Handling classification that determines courier and vehicle requirements';
COMMENT ON COLUMN delivery_orders.courier_id       IS 'Cross-DB reference to the assigned courier in kinara_fleet';
COMMENT ON COLUMN delivery_orders.signature_url    IS 'Cloud URL to the digital proof-of-delivery signature image';
COMMENT ON COLUMN delivery_orders.proof_photo_url  IS 'Cloud URL to the delivery confirmation photograph';
COMMENT ON COLUMN delivery_orders.tenant_id        IS 'Multi-tenant partition key';

-- ---------------------------------------------------------------------------
-- delivery_attempts
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS delivery_attempts (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_order_id UUID        NOT NULL REFERENCES delivery_orders(id),
    attempt_number    INT         DEFAULT 1,
    outcome           TEXT        CHECK (outcome IN ('success', 'failure', 'no_answer', 'wrong_address')),
    notes             TEXT,
    gps_lat           DOUBLE PRECISION,
    gps_lng           DOUBLE PRECISION,
    attempted_at      TIMESTAMPTZ DEFAULT NOW(),
    courier_id        UUID
);

COMMENT ON TABLE delivery_attempts IS 'Individual delivery attempt records for tracking failed and retried deliveries';
COMMENT ON COLUMN delivery_attempts.attempt_number IS 'Sequential attempt counter for this delivery order (starts at 1)';
COMMENT ON COLUMN delivery_attempts.outcome        IS 'Result of the delivery attempt';
COMMENT ON COLUMN delivery_attempts.gps_lat        IS 'Courier GPS latitude at time of attempt';
COMMENT ON COLUMN delivery_attempts.gps_lng        IS 'Courier GPS longitude at time of attempt';
COMMENT ON COLUMN delivery_attempts.courier_id     IS 'UUID of the courier who made this specific attempt';

-- ---------------------------------------------------------------------------
-- last_mile_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS last_mile_audit_log (
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

COMMENT ON TABLE last_mile_audit_log IS 'Immutable audit trail for all last-mile-service mutations';

CREATE RULE no_update_lm_audit AS ON UPDATE TO last_mile_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_lm_audit AS ON DELETE TO last_mile_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_delivery_orders_shipper     ON delivery_orders(shipper_id);
CREATE INDEX IF NOT EXISTS idx_delivery_orders_status      ON delivery_orders(status);
CREATE INDEX IF NOT EXISTS idx_delivery_orders_courier     ON delivery_orders(courier_id);
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_order     ON delivery_attempts(delivery_order_id);
CREATE INDEX IF NOT EXISTS idx_lm_audit_entity             ON last_mile_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_lm_audit_actor              ON last_mile_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_lm_audit_actor;
-- DROP INDEX IF EXISTS idx_lm_audit_entity;
-- DROP INDEX IF EXISTS idx_delivery_attempts_order;
-- DROP INDEX IF EXISTS idx_delivery_orders_courier;
-- DROP INDEX IF EXISTS idx_delivery_orders_status;
-- DROP INDEX IF EXISTS idx_delivery_orders_shipper;
-- DROP RULE IF EXISTS no_delete_lm_audit ON last_mile_audit_log;
-- DROP RULE IF EXISTS no_update_lm_audit ON last_mile_audit_log;
-- DROP TABLE IF EXISTS last_mile_audit_log;
-- DROP TABLE IF EXISTS delivery_attempts;
-- DROP TABLE IF EXISTS delivery_orders;
-- =============================================================================
