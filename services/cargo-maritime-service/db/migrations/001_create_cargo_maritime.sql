CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE containers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    container_no VARCHAR(20) NOT NULL UNIQUE,
    container_type VARCHAR(20) NOT NULL,
    owner_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'empty',
    current_port_id UUID,
    vessel_id UUID,
    weight_kg NUMERIC(10,2) NOT NULL DEFAULT 0,
    tare_weight_kg NUMERIC(8,2) NOT NULL DEFAULT 0,
    payload_kg NUMERIC(10,2) GENERATED ALWAYS AS (weight_kg - tare_weight_kg) STORED,
    seal_no VARCHAR(50),
    temperature_c NUMERIC(5,2),
    is_hazmat BOOLEAN NOT NULL DEFAULT FALSE,
    hazmat_class VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE cargo_manifests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manifest_no VARCHAR(20) NOT NULL UNIQUE,
    voyage_id UUID NOT NULL,
    vessel_id UUID NOT NULL,
    port_of_loading UUID NOT NULL,
    port_of_discharge UUID NOT NULL,
    shipper_name VARCHAR(200) NOT NULL,
    consignee_name VARCHAR(200) NOT NULL,
    total_containers INT NOT NULL DEFAULT 0,
    total_weight_kg NUMERIC(12,2) NOT NULL DEFAULT 0,
    commodity VARCHAR(200) NOT NULL,
    is_finalized BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE manifest_containers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manifest_id UUID NOT NULL REFERENCES cargo_manifests(id),
    container_id UUID NOT NULL REFERENCES containers(id),
    container_no VARCHAR(20) NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(manifest_id, container_id)
);

CREATE RULE no_update_manifest_containers AS ON UPDATE TO manifest_containers DO INSTEAD NOTHING;
CREATE RULE no_delete_manifest_containers AS ON DELETE TO manifest_containers DO INSTEAD NOTHING;

CREATE TABLE damage_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    container_id UUID NOT NULL REFERENCES containers(id),
    container_no VARCHAR(20) NOT NULL,
    damage_level VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    photo_url TEXT,
    reported_by TEXT NOT NULL,
    estimated_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    port_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_damage_reports AS ON UPDATE TO damage_reports DO INSTEAD NOTHING;
CREATE RULE no_delete_damage_reports AS ON DELETE TO damage_reports DO INSTEAD NOTHING;

CREATE TABLE cargo_maritime_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_cargo_maritime_audit AS ON UPDATE TO cargo_maritime_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_cargo_maritime_audit AS ON DELETE TO cargo_maritime_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_containers_status ON containers(status);
CREATE INDEX idx_containers_vessel_id ON containers(vessel_id);
CREATE INDEX idx_manifest_containers_manifest_id ON manifest_containers(manifest_id);
CREATE INDEX idx_damage_reports_container_id ON damage_reports(container_id);
