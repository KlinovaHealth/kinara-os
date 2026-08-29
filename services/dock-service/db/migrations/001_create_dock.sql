CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE dock_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id UUID NOT NULL,
    berth_id UUID NOT NULL,
    vessel_id UUID NOT NULL,
    operation_type VARCHAR(30) NOT NULL,
    cargo_type VARCHAR(100) NOT NULL,
    tonnage_t NUMERIC(12,2) NOT NULL DEFAULT 0,
    unit_count INT NOT NULL DEFAULT 0,
    stevedore_team VARCHAR(200),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    planned_duration_hrs NUMERIC(6,2) NOT NULL DEFAULT 0,
    actual_duration_hrs NUMERIC(6,2),
    safety_incident BOOLEAN NOT NULL DEFAULT FALSE,
    incident_details TEXT,
    billing_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE dock_equipment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id UUID NOT NULL,
    equipment_code VARCHAR(20) NOT NULL UNIQUE,
    equipment_type VARCHAR(30) NOT NULL,
    model VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    capacity_t NUMERIC(8,2) NOT NULL DEFAULT 0,
    last_service_at TIMESTAMPTZ,
    next_service_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE safety_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation_id UUID NOT NULL REFERENCES dock_operations(id),
    port_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    injured_count INT NOT NULL DEFAULT 0,
    reported_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_safety_events AS ON UPDATE TO safety_events DO INSTEAD NOTHING;
CREATE RULE no_delete_safety_events AS ON DELETE TO safety_events DO INSTEAD NOTHING;

CREATE TABLE dock_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id UUID NOT NULL,
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_dock_audit AS ON UPDATE TO dock_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_dock_audit AS ON DELETE TO dock_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_dock_operations_port_id ON dock_operations(port_id);
CREATE INDEX idx_dock_operations_vessel_id ON dock_operations(vessel_id);
CREATE INDEX idx_dock_equipment_port_id ON dock_equipment(port_id);
CREATE INDEX idx_safety_events_port_id ON safety_events(port_id);
