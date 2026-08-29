-- Cargo Service Schema — Logistics Pillar
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE cargo_bookings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_ref         TEXT NOT NULL UNIQUE DEFAULT 'KN-' || UPPER(SUBSTRING(gen_random_uuid()::TEXT,1,8)),
    shipper_id          UUID NOT NULL,
    consignee_id        UUID,
    cargo_type          TEXT NOT NULL DEFAULT 'general' CHECK (cargo_type IN ('general','perishable','fragile','hazardous','livestock','bulk_grain','medical','refrigerated')),
    description         TEXT NOT NULL DEFAULT '',
    weight_kg           DOUBLE PRECISION NOT NULL CHECK (weight_kg > 0),
    volume_m3           DOUBLE PRECISION NOT NULL DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','picked_up','in_transit','delivered','cancelled','failed')),
    origin_address      TEXT NOT NULL,
    origin_lat          DOUBLE PRECISION NOT NULL DEFAULT 0,
    origin_lng          DOUBLE PRECISION NOT NULL DEFAULT 0,
    destination_address TEXT NOT NULL,
    destination_lat     DOUBLE PRECISION NOT NULL DEFAULT 0,
    destination_lng     DOUBLE PRECISION NOT NULL DEFAULT 0,
    pickup_at           TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    assigned_vehicle_id UUID,
    assigned_driver_id  UUID,
    estimated_delivery  TIMESTAMPTZ,
    freight_cost        NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency            TEXT NOT NULL DEFAULT 'USD',
    notes               TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cargo_shipper ON cargo_bookings(shipper_id);
CREATE INDEX idx_cargo_status  ON cargo_bookings(status);
CREATE INDEX idx_cargo_date    ON cargo_bookings(created_at DESC);

CREATE TABLE tracking_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cargo_id   UUID NOT NULL REFERENCES cargo_bookings(id),
    status     TEXT NOT NULL,
    location   TEXT NOT NULL DEFAULT '',
    latitude   DOUBLE PRECISION NOT NULL DEFAULT 0,
    longitude  DOUBLE PRECISION NOT NULL DEFAULT 0,
    notes      TEXT NOT NULL DEFAULT '',
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE tracking_no_update AS ON UPDATE TO tracking_events DO INSTEAD NOTHING;
CREATE RULE tracking_no_delete AS ON DELETE TO tracking_events DO INSTEAD NOTHING;
CREATE INDEX idx_tracking_cargo ON tracking_events(cargo_id);
CREATE INDEX idx_tracking_time  ON tracking_events(event_time DESC);

CREATE TABLE cargo_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), entity_id UUID, user_id UUID NOT NULL,
    action TEXT NOT NULL, resource TEXT NOT NULL, ip_address TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE cargo_audit_no_update AS ON UPDATE TO cargo_audit_log DO INSTEAD NOTHING;
CREATE RULE cargo_audit_no_delete AS ON DELETE TO cargo_audit_log DO INSTEAD NOTHING;
