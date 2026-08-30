-- =============================================================================
-- V202609010028__Fleet_Service__Init.sql
-- Kinara OS — Fleet Service
-- =============================================================================
\c kinara_fleet;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- vehicles
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS vehicles (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    registration_no       TEXT        UNIQUE NOT NULL,
    vehicle_type          TEXT        CHECK (vehicle_type IN (
                              'truck', 'pickup', 'motorcycle', 'van', 'bus',
                              'ambulance', 'refrigerated', 'tanker')),
    make                  TEXT        DEFAULT '',
    model                 TEXT        DEFAULT '',
    year                  INT         DEFAULT 2020,
    fuel_type             TEXT        DEFAULT 'diesel'
                          CHECK (fuel_type IN ('petrol', 'diesel', 'cng', 'electric', 'hybrid')),
    payload_capacity_kg   DOUBLE PRECISION DEFAULT 0,
    volume_capacity_m3    DOUBLE PRECISION DEFAULT 0,
    status                TEXT        DEFAULT 'available'
                          CHECK (status IN ('active', 'in_repair', 'retired', 'available', 'assigned', 'in_transit')),
    country               TEXT        NOT NULL,
    base_location         TEXT        DEFAULT '',
    current_odometer_km   DOUBLE PRECISION DEFAULT 0,
    last_service_km       DOUBLE PRECISION DEFAULT 0,
    next_service_km       DOUBLE PRECISION DEFAULT 0,
    insurance_expiry      TIMESTAMPTZ,
    inspection_expiry     TIMESTAMPTZ,
    assigned_driver_id    UUID,
    notes                 TEXT        DEFAULT '',
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE vehicles IS 'Fleet vehicle registry covering all Kinara-operated and partner transport assets';
COMMENT ON COLUMN vehicles.registration_no     IS 'Government-issued vehicle registration plate number';
COMMENT ON COLUMN vehicles.vehicle_type        IS 'Vehicle classification determining cargo and route eligibility';
COMMENT ON COLUMN vehicles.payload_capacity_kg IS 'Maximum authorised payload in kilograms';
COMMENT ON COLUMN vehicles.volume_capacity_m3  IS 'Cargo bay volume in cubic metres';
COMMENT ON COLUMN vehicles.current_odometer_km IS 'Latest recorded odometer reading in kilometres';
COMMENT ON COLUMN vehicles.next_service_km     IS 'Odometer reading at which next scheduled service is due';
COMMENT ON COLUMN vehicles.insurance_expiry    IS 'Date and time the vehicle insurance cover expires';
COMMENT ON COLUMN vehicles.inspection_expiry   IS 'Date and time the vehicle roadworthiness certificate expires';
COMMENT ON COLUMN vehicles.assigned_driver_id  IS 'UUID of the primary driver currently assigned to this vehicle';

-- ---------------------------------------------------------------------------
-- maintenance_records
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS maintenance_records (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id       UUID        NOT NULL REFERENCES vehicles(id),
    maintenance_type TEXT        CHECK (maintenance_type IN (
                         'service', 'repair', 'inspection', 'tyre_change', 'oil_change')),
    description      TEXT,
    cost             NUMERIC(12,2),
    currency         TEXT        DEFAULT 'XOF',
    odometer_km      DOUBLE PRECISION,
    performed_by     TEXT,
    workshop         TEXT,
    performed_at     DATE,
    next_due_km      DOUBLE PRECISION,
    next_due_date    DATE,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE maintenance_records IS 'Full maintenance history for each fleet vehicle';
COMMENT ON COLUMN maintenance_records.maintenance_type IS 'Category of maintenance work performed';
COMMENT ON COLUMN maintenance_records.odometer_km      IS 'Odometer reading at time of maintenance';
COMMENT ON COLUMN maintenance_records.performed_by     IS 'Name or ID of the mechanic or technician who did the work';
COMMENT ON COLUMN maintenance_records.workshop         IS 'Name or address of the workshop or service centre';
COMMENT ON COLUMN maintenance_records.next_due_km      IS 'Odometer reading at which next maintenance is due';
COMMENT ON COLUMN maintenance_records.next_due_date    IS 'Calendar date by which next maintenance must be completed';

-- ---------------------------------------------------------------------------
-- fleet_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS fleet_audit_log (
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

COMMENT ON TABLE fleet_audit_log IS 'Immutable audit trail for all fleet-service mutations';

CREATE RULE no_update_fleet_audit AS ON UPDATE TO fleet_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_fleet_audit AS ON DELETE TO fleet_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_vehicles_country_status    ON vehicles(country, status);
CREATE INDEX IF NOT EXISTS idx_vehicles_driver            ON vehicles(assigned_driver_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_vehicle        ON maintenance_records(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_performed_at   ON maintenance_records(performed_at);
CREATE INDEX IF NOT EXISTS idx_fleet_audit_entity         ON fleet_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_fleet_audit_actor          ON fleet_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_fleet_audit_actor;
-- DROP INDEX IF EXISTS idx_fleet_audit_entity;
-- DROP INDEX IF EXISTS idx_maintenance_performed_at;
-- DROP INDEX IF EXISTS idx_maintenance_vehicle;
-- DROP INDEX IF EXISTS idx_vehicles_driver;
-- DROP INDEX IF EXISTS idx_vehicles_country_status;
-- DROP RULE IF EXISTS no_delete_fleet_audit ON fleet_audit_log;
-- DROP RULE IF EXISTS no_update_fleet_audit ON fleet_audit_log;
-- DROP TABLE IF EXISTS fleet_audit_log;
-- DROP TABLE IF EXISTS maintenance_records;
-- DROP TABLE IF EXISTS vehicles;
-- =============================================================================
