-- Fleet Service Schema — Logistics Pillar
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE vehicles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    registration_no     TEXT NOT NULL UNIQUE,
    vehicle_type        TEXT NOT NULL CHECK (vehicle_type IN ('truck','pickup','motorcycle','van','bus','ambulance','refrigerated','tanker')),
    make                TEXT NOT NULL DEFAULT '',
    model               TEXT NOT NULL DEFAULT '',
    year                INT  NOT NULL DEFAULT 2020,
    fuel_type           TEXT NOT NULL DEFAULT 'diesel' CHECK (fuel_type IN ('petrol','diesel','cng','electric','hybrid')),
    payload_capacity_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
    volume_capacity_m3  DOUBLE PRECISION NOT NULL DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('active','in_repair','retired','available','assigned','in_transit')),
    country             TEXT NOT NULL,
    base_location       TEXT NOT NULL DEFAULT '',
    current_odometer_km DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_service_km     DOUBLE PRECISION NOT NULL DEFAULT 0,
    next_service_km     DOUBLE PRECISION NOT NULL DEFAULT 0,
    insurance_expiry    TIMESTAMPTZ,
    inspection_expiry   TIMESTAMPTZ,
    assigned_driver_id  UUID,
    notes               TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_vehicles_country ON vehicles(country);
CREATE INDEX idx_vehicles_status  ON vehicles(status);
CREATE INDEX idx_vehicles_type    ON vehicles(vehicle_type);

CREATE TABLE maintenance_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id      UUID NOT NULL REFERENCES vehicles(id),
    service_type    TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    odometer_km     DOUBLE PRECISION NOT NULL DEFAULT 0,
    cost            NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency        TEXT NOT NULL DEFAULT 'USD',
    serviced_by     TEXT NOT NULL DEFAULT '',
    serviced_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_service_km DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE maintenance_no_update AS ON UPDATE TO maintenance_records DO INSTEAD NOTHING;
CREATE RULE maintenance_no_delete AS ON DELETE TO maintenance_records DO INSTEAD NOTHING;
CREATE INDEX idx_maintenance_vehicle ON maintenance_records(vehicle_id);

CREATE TABLE fuel_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id     UUID NOT NULL REFERENCES vehicles(id),
    driver_id      UUID,
    litres_filled  DOUBLE PRECISION NOT NULL CHECK (litres_filled > 0),
    cost_per_litre NUMERIC(10,4) NOT NULL DEFAULT 0,
    total_cost     NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency       TEXT NOT NULL DEFAULT 'USD',
    odometer_km    DOUBLE PRECISION NOT NULL DEFAULT 0,
    station        TEXT NOT NULL DEFAULT '',
    filled_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE fuel_no_update AS ON UPDATE TO fuel_logs DO INSTEAD NOTHING;
CREATE RULE fuel_no_delete  AS ON DELETE TO fuel_logs DO INSTEAD NOTHING;
CREATE INDEX idx_fuel_vehicle ON fuel_logs(vehicle_id);
CREATE INDEX idx_fuel_date    ON fuel_logs(filled_at DESC);

CREATE TABLE fleet_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), entity_id UUID, user_id UUID NOT NULL,
    action TEXT NOT NULL, resource TEXT NOT NULL, ip_address TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE fleet_audit_no_update AS ON UPDATE TO fleet_audit_log DO INSTEAD NOTHING;
CREATE RULE fleet_audit_no_delete AS ON DELETE TO fleet_audit_log DO INSTEAD NOTHING;
