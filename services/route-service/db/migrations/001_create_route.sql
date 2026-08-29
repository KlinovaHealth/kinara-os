-- Route Service Schema — Logistics Pillar
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE routes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    route_code    TEXT NOT NULL UNIQUE,
    route_type    TEXT NOT NULL DEFAULT 'fixed' CHECK (route_type IN ('fixed','dynamic','circular')),
    status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive','archived')),
    country       TEXT NOT NULL,
    origin_name   TEXT NOT NULL,
    origin_lat    DOUBLE PRECISION NOT NULL DEFAULT 0,
    origin_lng    DOUBLE PRECISION NOT NULL DEFAULT 0,
    dest_name     TEXT NOT NULL,
    dest_lat      DOUBLE PRECISION NOT NULL DEFAULT 0,
    dest_lng      DOUBLE PRECISION NOT NULL DEFAULT 0,
    distance_km   DOUBLE PRECISION NOT NULL DEFAULT 0,
    est_hours     DOUBLE PRECISION NOT NULL DEFAULT 0,
    waypoints     JSONB NOT NULL DEFAULT '[]',
    freight_class TEXT NOT NULL DEFAULT '',
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_routes_country ON routes(country);
CREATE INDEX idx_routes_status  ON routes(status);

CREATE TABLE route_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id        UUID NOT NULL REFERENCES routes(id),
    vehicle_id      UUID,
    driver_id       UUID,
    departure_time  TIMESTAMPTZ NOT NULL,
    arrival_time    TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled','departed','completed','cancelled','delayed')),
    actual_dept_at  TIMESTAMPTZ,
    actual_arr_at   TIMESTAMPTZ,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_schedules_route      ON route_schedules(route_id);
CREATE INDEX idx_schedules_vehicle    ON route_schedules(vehicle_id);
CREATE INDEX idx_schedules_departure  ON route_schedules(departure_time);

CREATE TABLE route_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), entity_id UUID, user_id UUID NOT NULL,
    action TEXT NOT NULL, resource TEXT NOT NULL, ip_address TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE route_audit_no_update AS ON UPDATE TO route_audit_log DO INSTEAD NOTHING;
CREATE RULE route_audit_no_delete AS ON DELETE TO route_audit_log DO INSTEAD NOTHING;
