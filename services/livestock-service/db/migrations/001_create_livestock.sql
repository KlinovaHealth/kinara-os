CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS animals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    animal_ref TEXT NOT NULL UNIQUE,
    farmer_id UUID NOT NULL,
    animal_type TEXT NOT NULL,
    breed TEXT,
    age_months INT,
    sex TEXT,
    ear_tag TEXT,
    tenant_id TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_animals_farmer ON animals(farmer_id);
CREATE TABLE IF NOT EXISTS health_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    animal_id UUID NOT NULL REFERENCES animals(id),
    event_type TEXT NOT NULL,
    description TEXT,
    treatment TEXT,
    veterinarian_id UUID,
    event_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_health_animal ON health_events(animal_id, event_date);
CREATE TABLE IF NOT EXISTS production_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    animal_id UUID NOT NULL REFERENCES animals(id),
    production_type TEXT NOT NULL,
    quantity NUMERIC(10,3) NOT NULL,
    unit TEXT NOT NULL,
    recorded_date DATE NOT NULL,
    recorded_by UUID NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_production_animal ON production_records(animal_id, recorded_date);
CREATE TABLE IF NOT EXISTS veterinary_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    animal_id UUID NOT NULL,
    alert_type TEXT NOT NULL,
    priority TEXT NOT NULL DEFAULT 'medium',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS livestock_audit_log (
    id BIGSERIAL PRIMARY KEY,
    animal_id TEXT,
    action TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE no_update_livestock_audit AS ON UPDATE TO livestock_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_livestock_audit AS ON DELETE TO livestock_audit_log DO INSTEAD NOTHING;
