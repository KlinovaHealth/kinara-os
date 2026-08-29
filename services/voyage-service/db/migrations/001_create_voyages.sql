CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS voyages (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    voyage_ref        TEXT NOT NULL UNIQUE,
    vessel_id         UUID NOT NULL,
    origin_port       TEXT NOT NULL,
    destination_port  TEXT NOT NULL,
    cargo_type        TEXT,
    cargo_tons        NUMERIC(10,2) NOT NULL DEFAULT 0,
    status            TEXT NOT NULL DEFAULT 'planned',
    departure_at      TIMESTAMPTZ,
    est_arrival_at    TIMESTAMPTZ,
    actual_arrival_at TIMESTAMPTZ,
    distance_nm       NUMERIC(10,2) NOT NULL DEFAULT 0,
    fuel_tons         NUMERIC(8,3) NOT NULL DEFAULT 0,
    tenant_id         TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS voyage_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    voyage_id   UUID NOT NULL REFERENCES voyages(id),
    event_type  TEXT NOT NULL,
    description TEXT NOT NULL,
    latitude    NUMERIC(9,6),
    longitude   NUMERIC(9,6),
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_voyages_vessel ON voyages(vessel_id);
CREATE INDEX IF NOT EXISTS idx_voyages_status ON voyages(status);
CREATE INDEX IF NOT EXISTS idx_voyages_tenant ON voyages(tenant_id);
CREATE INDEX IF NOT EXISTS idx_events_voyage ON voyage_events(voyage_id);

CREATE RULE no_delete_events AS ON DELETE TO voyage_events DO INSTEAD NOTHING;
