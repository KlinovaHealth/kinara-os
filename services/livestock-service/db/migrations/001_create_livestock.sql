CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS animals (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tag_ref       TEXT NOT NULL UNIQUE,
    farmer_id     UUID NOT NULL,
    species       TEXT NOT NULL,
    breed         TEXT,
    birth_date    DATE,
    weight_kg     NUMERIC(8,2) NOT NULL DEFAULT 0,
    health_status TEXT NOT NULL DEFAULT 'healthy',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    tenant_id     TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS production_records (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    animal_id    UUID NOT NULL REFERENCES animals(id),
    farmer_id    UUID NOT NULL,
    product_type TEXT NOT NULL,
    quantity_kg  NUMERIC(10,3) NOT NULL,
    recorded_at  TIMESTAMPTZ NOT NULL,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_animals_farmer ON animals(farmer_id);
CREATE INDEX IF NOT EXISTS idx_animals_tenant ON animals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_production_animal ON production_records(animal_id);

CREATE RULE no_delete_production AS ON DELETE TO production_records DO INSTEAD NOTHING;
