-- =============================================================================
-- V202609010024__Transport_Service__Init.sql
-- Kinara OS — Transport Service
-- =============================================================================
\c kinara_transport;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- transport_requests
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transport_requests (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    request_ref     TEXT        UNIQUE,
    requester_id    UUID        NOT NULL,
    cargo_type      TEXT,
    origin_address  TEXT        NOT NULL,
    origin_lat      DOUBLE PRECISION,
    origin_lng      DOUBLE PRECISION,
    dest_address    TEXT        NOT NULL,
    dest_lat        DOUBLE PRECISION,
    dest_lng        DOUBLE PRECISION,
    weight_kg       DOUBLE PRECISION,
    volume_m3       DOUBLE PRECISION,
    required_by     TIMESTAMPTZ,
    status          TEXT        DEFAULT 'pending'
                    CHECK (status IN ('pending', 'matched', 'in_transit', 'delivered', 'cancelled')),
    tenant_id       TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE transport_requests IS 'Freight transport requests raised by farmers, cooperatives, or supply-chain actors';
COMMENT ON COLUMN transport_requests.request_ref    IS 'Human-readable transport request number';
COMMENT ON COLUMN transport_requests.requester_id   IS 'UUID of the user or entity that placed the request';
COMMENT ON COLUMN transport_requests.cargo_type     IS 'Description of the goods to be transported';
COMMENT ON COLUMN transport_requests.required_by    IS 'Deadline by which the cargo must be delivered';
COMMENT ON COLUMN transport_requests.tenant_id      IS 'Multi-tenant partition key';

-- ---------------------------------------------------------------------------
-- transport_assignments
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transport_assignments (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id    UUID        NOT NULL REFERENCES transport_requests(id),
    driver_id     UUID,
    vehicle_id    UUID,
    assigned_at   TIMESTAMPTZ DEFAULT NOW(),
    picked_up_at  TIMESTAMPTZ,
    delivered_at  TIMESTAMPTZ,
    distance_km   NUMERIC(10,3),
    cost          NUMERIC(12,2),
    currency      TEXT        DEFAULT 'XOF',
    rating        INT         CHECK (rating BETWEEN 1 AND 5)
);

COMMENT ON TABLE transport_assignments IS 'Driver and vehicle assignments fulfilling transport requests';
COMMENT ON COLUMN transport_assignments.driver_id   IS 'Cross-DB reference to driver user record in kinara_fleet';
COMMENT ON COLUMN transport_assignments.vehicle_id  IS 'Cross-DB reference to vehicles.id in kinara_fleet';
COMMENT ON COLUMN transport_assignments.distance_km IS 'Actual distance driven as recorded by the driver app or GPS';
COMMENT ON COLUMN transport_assignments.rating      IS 'Requester rating of the delivery (1 = poor, 5 = excellent)';

-- ---------------------------------------------------------------------------
-- routes
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS routes (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    origin_address TEXT,
    dest_address   TEXT,
    distance_km    NUMERIC(10,3),
    duration_min   INT,
    waypoints      JSONB,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE routes IS 'Cached route geometries and metrics between common origin-destination pairs';
COMMENT ON COLUMN routes.waypoints    IS 'GeoJSON LineString or ordered array of lat/lng waypoints';
COMMENT ON COLUMN routes.duration_min IS 'Estimated travel time in minutes under normal conditions';

-- ---------------------------------------------------------------------------
-- transport_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transport_audit_log (
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

COMMENT ON TABLE transport_audit_log IS 'Immutable audit trail for all transport-service mutations';

CREATE RULE no_update_transport_audit AS ON UPDATE TO transport_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_transport_audit AS ON DELETE TO transport_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_transport_requests_requester ON transport_requests(requester_id);
CREATE INDEX IF NOT EXISTS idx_transport_requests_status    ON transport_requests(status);
CREATE INDEX IF NOT EXISTS idx_transport_assignments_request ON transport_assignments(request_id);
CREATE INDEX IF NOT EXISTS idx_transport_assignments_driver  ON transport_assignments(driver_id);
CREATE INDEX IF NOT EXISTS idx_transport_assignments_vehicle ON transport_assignments(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_transport_audit_entity       ON transport_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_transport_audit_actor        ON transport_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_transport_audit_actor;
-- DROP INDEX IF EXISTS idx_transport_audit_entity;
-- DROP INDEX IF EXISTS idx_transport_assignments_vehicle;
-- DROP INDEX IF EXISTS idx_transport_assignments_driver;
-- DROP INDEX IF EXISTS idx_transport_assignments_request;
-- DROP INDEX IF EXISTS idx_transport_requests_status;
-- DROP INDEX IF EXISTS idx_transport_requests_requester;
-- DROP RULE IF EXISTS no_delete_transport_audit ON transport_audit_log;
-- DROP RULE IF EXISTS no_update_transport_audit ON transport_audit_log;
-- DROP TABLE IF EXISTS transport_audit_log;
-- DROP TABLE IF EXISTS routes;
-- DROP TABLE IF EXISTS transport_assignments;
-- DROP TABLE IF EXISTS transport_requests;
-- =============================================================================
