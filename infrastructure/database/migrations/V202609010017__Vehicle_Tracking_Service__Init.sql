-- Vehicle Tracking Service — Fleet registry, real-time GPS, route management, and alerts
-- Database: kinara_vehicle_tracking
-- Tracks logistics and ambulance fleets: registration, live location pings, planned routes, anomaly alerts
-- Note: run 'CREATE DATABASE kinara_vehicle_tracking;' as superuser if not exists

\c kinara_vehicle_tracking;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS vehicles (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_ref    TEXT        NOT NULL UNIQUE,
    fleet_id       TEXT        NOT NULL,
    vehicle_type   TEXT        NOT NULL
                   CHECK (vehicle_type IN ('truck','bike','van','motorcycle','ambulance')),
    capacity_kg    NUMERIC(10,2),
    driver_name    TEXT,
    driver_id      UUID,
    status         TEXT        NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active','idle','maintenance','offline')),
    tenant_id      TEXT        NOT NULL,
    registered_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  vehicles              IS 'Fleet vehicle registry — covers logistics trucks, motorcycles, vans, and ambulances';
COMMENT ON COLUMN vehicles.vehicle_ref  IS 'Human-readable unique reference, e.g. VEH-AMB-20260901-001';
COMMENT ON COLUMN vehicles.fleet_id     IS 'Fleet group identifier — e.g. health-east, agri-north, logistics-hub-1';
COMMENT ON COLUMN vehicles.vehicle_type IS 'Vehicle category affecting routing rules and load limits';
COMMENT ON COLUMN vehicles.capacity_kg  IS 'Maximum payload capacity in kilograms; null for passenger vehicles';
COMMENT ON COLUMN vehicles.driver_id    IS 'Current assigned driver UUID — may change between dispatches';
COMMENT ON COLUMN vehicles.status       IS 'Operational status: active (in service), idle, maintenance, offline (no signal)';

CREATE INDEX IF NOT EXISTS idx_veh_fleet   ON vehicles(fleet_id);
CREATE INDEX IF NOT EXISTS idx_veh_status  ON vehicles(status);
CREATE INDEX IF NOT EXISTS idx_veh_tenant  ON vehicles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_veh_driver  ON vehicles(driver_id) WHERE driver_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS gps_locations (
    id           BIGSERIAL     PRIMARY KEY,
    vehicle_id   UUID          NOT NULL REFERENCES vehicles(id),
    latitude     NUMERIC(10,7) NOT NULL CHECK (latitude  BETWEEN  -90 AND  90),
    longitude    NUMERIC(10,7) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    speed_kmh    NUMERIC(6,2)  NOT NULL DEFAULT 0 CHECK (speed_kmh >= 0),
    heading_deg  NUMERIC(5,2)  NOT NULL DEFAULT 0 CHECK (heading_deg BETWEEN 0 AND 360),
    pinged_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  gps_locations              IS 'Real-time GPS pings — high-volume append-only table; use BIGSERIAL PK for ingest speed';
COMMENT ON COLUMN gps_locations.latitude     IS 'WGS-84 latitude — 7 decimal places gives ~1 cm precision';
COMMENT ON COLUMN gps_locations.longitude    IS 'WGS-84 longitude — 7 decimal places gives ~1 cm precision';
COMMENT ON COLUMN gps_locations.speed_kmh    IS 'GPS-derived speed in km/h; 0 if stationary';
COMMENT ON COLUMN gps_locations.heading_deg  IS 'Bearing in degrees clockwise from true north (0–360)';
COMMENT ON COLUMN gps_locations.pinged_at    IS 'UTC timestamp of the GPS fix — index DESC for last-known-position queries';

CREATE INDEX IF NOT EXISTS idx_gps_vehicle_time ON gps_locations(vehicle_id, pinged_at DESC);
CREATE INDEX IF NOT EXISTS idx_gps_recent        ON gps_locations(pinged_at DESC);

CREATE TABLE IF NOT EXISTS vehicle_routes (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id   UUID          NOT NULL REFERENCES vehicles(id),
    origin_lat   NUMERIC(10,7) NOT NULL,
    origin_lng   NUMERIC(10,7) NOT NULL,
    dest_lat     NUMERIC(10,7) NOT NULL,
    dest_lng     NUMERIC(10,7) NOT NULL,
    description  TEXT,
    active       BOOLEAN       NOT NULL DEFAULT true,
    assigned_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    eta          TIMESTAMPTZ
);

COMMENT ON TABLE  vehicle_routes             IS 'Planned routes assigned to vehicles — active flag prevents duplicate active routes per vehicle';
COMMENT ON COLUMN vehicle_routes.origin_lat  IS 'WGS-84 latitude of route origin';
COMMENT ON COLUMN vehicle_routes.dest_lat    IS 'WGS-84 latitude of destination';
COMMENT ON COLUMN vehicle_routes.eta         IS 'Estimated time of arrival at destination — updated by routing engine';
COMMENT ON COLUMN vehicle_routes.active      IS 'True while route is in progress; set to false on completion or reassignment';

CREATE INDEX IF NOT EXISTS idx_vr_vehicle_active ON vehicle_routes(vehicle_id, active);
CREATE INDEX IF NOT EXISTS idx_vr_eta            ON vehicle_routes(eta) WHERE eta IS NOT NULL AND active = true;

CREATE TABLE IF NOT EXISTS vehicle_alerts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id   UUID        NOT NULL,
    alert_type   TEXT        NOT NULL
                 CHECK (alert_type IN ('off_route','speeding','no_signal','manual')),
    message      TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  vehicle_alerts            IS 'Anomaly alerts for fleet vehicles — off-route deviation, speed violations, signal loss';
COMMENT ON COLUMN vehicle_alerts.alert_type IS 'Alert category: off_route (geofence breach), speeding (> threshold), no_signal (timeout), manual';
COMMENT ON COLUMN vehicle_alerts.message    IS 'Human-readable alert description including context (speed value, deviation distance, etc.)';

CREATE INDEX IF NOT EXISTS idx_va_vehicle_time ON vehicle_alerts(vehicle_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_va_type         ON vehicle_alerts(alert_type, created_at DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS vehicle_audit_log (
    id             BIGSERIAL   PRIMARY KEY,
    entity_id      UUID        NOT NULL,
    action         TEXT        NOT NULL,  -- 'create','update','delete','read'
    actor_id       TEXT        NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    signature_hash TEXT,
    ip_address     INET,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE vehicle_audit_log IS 'Append-only audit trail for vehicle registration, route assignment, and status changes';

CREATE RULE no_update_vehicle_audit AS ON UPDATE TO vehicle_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_vehicle_audit AS ON DELETE TO vehicle_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_vehicle_audit_entity ON vehicle_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_vehicle_audit_actor  ON vehicle_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS vehicle_audit_log CASCADE;
-- DROP TABLE IF EXISTS vehicle_alerts CASCADE;
-- DROP TABLE IF EXISTS vehicle_routes CASCADE;
-- DROP TABLE IF EXISTS gps_locations CASCADE;
-- DROP TABLE IF EXISTS vehicles CASCADE;
