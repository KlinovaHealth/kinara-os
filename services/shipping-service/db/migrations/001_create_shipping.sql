CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE freight_bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_ref VARCHAR(20) NOT NULL UNIQUE,
    shipper_id UUID NOT NULL,
    shipper_name VARCHAR(200) NOT NULL,
    consignee_name VARCHAR(200) NOT NULL,
    shipment_type VARCHAR(10) NOT NULL DEFAULT 'fcl',
    port_of_loading UUID NOT NULL,
    port_of_discharge UUID NOT NULL,
    vessel_id UUID,
    commodity_description TEXT NOT NULL,
    container_count INT NOT NULL DEFAULT 1,
    weight_kg NUMERIC(12,2) NOT NULL DEFAULT 0,
    freight_rate_usd NUMERIC(10,2) NOT NULL DEFAULT 0,
    insurance_pct NUMERIC(5,3) NOT NULL DEFAULT 0,
    insurance_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    declared_value NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_freight NUMERIC(14,2) NOT NULL DEFAULT 0,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE bills_of_lading (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bol_number VARCHAR(20) NOT NULL UNIQUE,
    booking_id UUID NOT NULL REFERENCES freight_bookings(id),
    vessel_name VARCHAR(200) NOT NULL,
    voyage_no VARCHAR(50) NOT NULL,
    shipper_name VARCHAR(200) NOT NULL,
    consignee_name VARCHAR(200) NOT NULL,
    notify_party VARCHAR(200),
    port_of_loading TEXT NOT NULL,
    port_of_discharge TEXT NOT NULL,
    commodity_description TEXT NOT NULL,
    container_count INT NOT NULL DEFAULT 1,
    gross_weight_kg NUMERIC(12,2) NOT NULL DEFAULT 0,
    freight_prepaid BOOLEAN NOT NULL DEFAULT TRUE,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    issued_at TIMESTAMPTZ,
    surrendered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE demurrage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES freight_bookings(id),
    container_no VARCHAR(20) NOT NULL,
    free_days INT NOT NULL DEFAULT 0,
    used_days INT NOT NULL DEFAULT 0,
    daily_rate_usd NUMERIC(8,2) NOT NULL DEFAULT 0,
    total_charge NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    port_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_demurrage AS ON UPDATE TO demurrage_records DO INSTEAD NOTHING;
CREATE RULE no_delete_demurrage AS ON DELETE TO demurrage_records DO INSTEAD NOTHING;

CREATE TABLE shipping_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_shipping_audit AS ON UPDATE TO shipping_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_shipping_audit AS ON DELETE TO shipping_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_bookings_shipper_id ON freight_bookings(shipper_id);
CREATE INDEX idx_bookings_status ON freight_bookings(status);
CREATE INDEX idx_bol_booking_id ON bills_of_lading(booking_id);
CREATE INDEX idx_demurrage_booking_id ON demurrage_records(booking_id);
