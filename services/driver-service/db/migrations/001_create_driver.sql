-- Driver Service Schema — Logistics Pillar
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE drivers (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name_enc        TEXT NOT NULL,
    phone_enc            TEXT NOT NULL,
    national_id_enc      TEXT NOT NULL,
    license_no           TEXT NOT NULL UNIQUE,
    license_class        TEXT NOT NULL CHECK (license_class IN ('A','B','C','D','E')),
    license_expiry       TIMESTAMPTZ NOT NULL,
    status               TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('active','suspended','inactive','available','on_duty','off_duty')),
    country              TEXT NOT NULL,
    base_location        TEXT NOT NULL DEFAULT '',
    total_trips          INT NOT NULL DEFAULT 0,
    total_km             DOUBLE PRECISION NOT NULL DEFAULT 0,
    rating               DOUBLE PRECISION NOT NULL DEFAULT 5.0 CHECK (rating BETWEEN 0 AND 5),
    assigned_vehicle_id  UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_drivers_country ON drivers(country);
CREATE INDEX idx_drivers_status  ON drivers(status);
CREATE INDEX idx_drivers_license ON drivers(license_no);

CREATE TABLE driver_trips (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id   UUID NOT NULL REFERENCES drivers(id),
    vehicle_id  UUID NOT NULL,
    route_id    UUID,
    distance_km DOUBLE PRECISION NOT NULL DEFAULT 0,
    start_time  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    end_time    TIMESTAMPTZ,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','cancelled')),
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE trips_no_delete AS ON DELETE TO driver_trips DO INSTEAD NOTHING;
CREATE INDEX idx_trips_driver ON driver_trips(driver_id);
CREATE INDEX idx_trips_status ON driver_trips(status);

CREATE TABLE driver_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), entity_id UUID, user_id UUID NOT NULL,
    action TEXT NOT NULL, resource TEXT NOT NULL, ip_address TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE driver_audit_no_update AS ON UPDATE TO driver_audit_log DO INSTEAD NOTHING;
CREATE RULE driver_audit_no_delete AS ON DELETE TO driver_audit_log DO INSTEAD NOTHING;
