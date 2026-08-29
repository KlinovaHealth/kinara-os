CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS input_purchases (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    purchase_ref TEXT NOT NULL UNIQUE,
    farmer_id    UUID NOT NULL,
    coop_id      UUID,
    input_type   TEXT NOT NULL,
    input_name   TEXT NOT NULL,
    quantity     NUMERIC(12,3) NOT NULL,
    unit         TEXT NOT NULL,
    cost_xof     NUMERIC(14,2) NOT NULL,
    supplier     TEXT,
    purchased_at TIMESTAMPTZ NOT NULL,
    tenant_id    TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS input_usages (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    purchase_id UUID NOT NULL REFERENCES input_purchases(id),
    farmer_id   UUID NOT NULL,
    field_id    TEXT NOT NULL,
    quantity    NUMERIC(12,3) NOT NULL,
    used_at     TIMESTAMPTZ NOT NULL,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchases_farmer ON input_purchases(farmer_id);
CREATE INDEX IF NOT EXISTS idx_purchases_tenant ON input_purchases(tenant_id);
CREATE INDEX IF NOT EXISTS idx_usages_purchase ON input_usages(purchase_id);
