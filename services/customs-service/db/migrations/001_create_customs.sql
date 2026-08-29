CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tariff_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hs_code VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(30) NOT NULL,
    duty_rate_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
    vat_rate_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
    country VARCHAR(3) NOT NULL,
    is_restricted BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(hs_code, country)
);

CREATE TABLE clearance_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_no VARCHAR(20) NOT NULL UNIQUE,
    importer_name VARCHAR(200) NOT NULL,
    importer_id VARCHAR(100) NOT NULL,
    manifest_id UUID NOT NULL,
    vessel_id UUID NOT NULL,
    port_id UUID NOT NULL,
    hs_code VARCHAR(20) NOT NULL,
    goods_description TEXT NOT NULL,
    declared_value NUMERIC(14,2) NOT NULL DEFAULT 0,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    weight_kg NUMERIC(10,2) NOT NULL DEFAULT 0,
    duty_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    vat_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_due NUMERIC(14,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by TEXT,
    reviewed_at TIMESTAMPTZ,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE customs_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id UUID NOT NULL,
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_customs_audit AS ON UPDATE TO customs_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_customs_audit AS ON DELETE TO customs_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_tariff_hs_country ON tariff_codes(hs_code, country);
CREATE INDEX idx_clearance_status ON clearance_requests(status);
CREATE INDEX idx_clearance_port_id ON clearance_requests(port_id);
CREATE INDEX idx_clearance_manifest_id ON clearance_requests(manifest_id);
