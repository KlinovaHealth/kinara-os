CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE ports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    code VARCHAR(10) NOT NULL UNIQUE,
    country VARCHAR(3) NOT NULL,
    city VARCHAR(100) NOT NULL,
    latitude NUMERIC(9,6) NOT NULL,
    longitude NUMERIC(9,6) NOT NULL,
    max_draft_m NUMERIC(5,2) DEFAULT 0,
    total_berths INT NOT NULL DEFAULT 0,
    alert_level VARCHAR(20) NOT NULL DEFAULT 'normal',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE berths (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id UUID NOT NULL REFERENCES ports(id),
    berth_number VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    max_length_m NUMERIC(7,2) DEFAULT 0,
    max_draft_m NUMERIC(5,2) DEFAULT 0,
    max_tonnage_t NUMERIC(10,2) DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(port_id, berth_number)
);

CREATE TABLE berth_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    berth_id UUID NOT NULL REFERENCES berths(id),
    vessel_id UUID NOT NULL,
    vessel_name VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'expected',
    eta TIMESTAMPTZ NOT NULL,
    etd TIMESTAMPTZ NOT NULL,
    actual_arrival TIMESTAMPTZ,
    actual_departure TIMESTAMPTZ,
    cargo_type VARCHAR(100) NOT NULL,
    tonnage_t NUMERIC(10,2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE congestion_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id UUID NOT NULL REFERENCES ports(id),
    alert_level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    occupied_berths INT NOT NULL DEFAULT 0,
    total_berths INT NOT NULL DEFAULT 0,
    occupancy_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_congestion_alerts AS ON UPDATE TO congestion_alerts DO INSTEAD NOTHING;
CREATE RULE no_delete_congestion_alerts AS ON DELETE TO congestion_alerts DO INSTEAD NOTHING;

CREATE TABLE port_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id UUID NOT NULL,
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_port_audit AS ON UPDATE TO port_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_port_audit AS ON DELETE TO port_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_berths_port_id ON berths(port_id);
CREATE INDEX idx_berth_schedules_berth_id ON berth_schedules(berth_id);
CREATE INDEX idx_berth_schedules_vessel_id ON berth_schedules(vessel_id);
CREATE INDEX idx_berth_schedules_eta ON berth_schedules(eta);
CREATE INDEX idx_congestion_alerts_port_id ON congestion_alerts(port_id);
