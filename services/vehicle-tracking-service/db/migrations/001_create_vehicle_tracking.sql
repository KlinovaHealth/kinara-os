CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS vehicles (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_ref   TEXT NOT NULL UNIQUE,
    fleet_id      TEXT NOT NULL,
    vehicle_type  TEXT NOT NULL,
    capacity_kg   NUMERIC(10,2),
    driver_name   TEXT,
    driver_id     UUID,
    status        TEXT NOT NULL DEFAULT 'active',
    tenant_id     TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vehicles_fleet ON vehicles(fleet_id);

CREATE TABLE IF NOT EXISTS gps_locations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id  UUID NOT NULL REFERENCES vehicles(id),
    latitude    NUMERIC(10,7) NOT NULL,
    longitude   NUMERIC(10,7) NOT NULL,
    speed_kmh   NUMERIC(6,2) DEFAULT 0,
    heading_deg NUMERIC(5,2) DEFAULT 0,
    pinged_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gps_vehicle_time ON gps_locations(vehicle_id, pinged_at DESC);

CREATE TABLE IF NOT EXISTS vehicle_routes (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id  UUID NOT NULL REFERENCES vehicles(id),
    origin_lat  NUMERIC(10,7) NOT NULL,
    origin_lng  NUMERIC(10,7) NOT NULL,
    dest_lat    NUMERIC(10,7) NOT NULL,
    dest_lng    NUMERIC(10,7) NOT NULL,
    description TEXT,
    active      BOOLEAN NOT NULL DEFAULT true,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    eta         TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS vehicle_alerts (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id UUID NOT NULL,
    alert_type TEXT NOT NULL,
    message    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alerts_vehicle ON vehicle_alerts(vehicle_id, created_at);
