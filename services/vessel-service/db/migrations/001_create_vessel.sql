CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE vessels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    imo_number VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    vessel_type VARCHAR(50) NOT NULL,
    flag VARCHAR(3) NOT NULL,
    owner VARCHAR(200) NOT NULL,
    operator_id UUID NOT NULL,
    year_built INT NOT NULL,
    gross_tonnage_t NUMERIC(12,2) NOT NULL DEFAULT 0,
    deadweight_t NUMERIC(12,2) NOT NULL DEFAULT 0,
    length_m NUMERIC(7,2) NOT NULL DEFAULT 0,
    beam_m NUMERIC(7,2) NOT NULL DEFAULT 0,
    max_draft_m NUMERIC(5,2) NOT NULL DEFAULT 0,
    max_speed_knots NUMERIC(5,2) NOT NULL DEFAULT 0,
    condition VARCHAR(20) NOT NULL DEFAULT 'good',
    current_port_id UUID,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE voyage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vessel_id UUID NOT NULL REFERENCES vessels(id),
    voyage_code VARCHAR(20) NOT NULL UNIQUE,
    departure_port_id UUID NOT NULL,
    arrival_port_id UUID NOT NULL,
    departed_at TIMESTAMPTZ,
    arrived_at TIMESTAMPTZ,
    distance_nm NUMERIC(8,2) NOT NULL DEFAULT 0,
    cargo_tonnage_t NUMERIC(10,2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_voyage_records AS ON UPDATE TO voyage_records DO INSTEAD NOTHING;
CREATE RULE no_delete_voyage_records AS ON DELETE TO voyage_records DO INSTEAD NOTHING;

CREATE TABLE maintenance_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vessel_id UUID NOT NULL REFERENCES vessels(id),
    maintenance_type VARCHAR(30) NOT NULL,
    description TEXT NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    vendor VARCHAR(200),
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vessel_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vessel_id UUID NOT NULL,
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_vessel_audit AS ON UPDATE TO vessel_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_vessel_audit AS ON DELETE TO vessel_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_vessel_flag ON vessels(flag);
CREATE INDEX idx_vessel_is_active ON vessels(is_active);
CREATE INDEX idx_voyage_vessel_id ON voyage_records(vessel_id);
CREATE INDEX idx_maintenance_vessel_id ON maintenance_records(vessel_id);
