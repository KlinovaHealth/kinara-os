-- Livestock Service — Animal registry, veterinary events, production records, and health alerts
-- Database: kinara_livestock
-- Manages smallholder and commercial livestock: registration, vet encounters, and output tracking

\c kinara_livestock;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS animals (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    animal_ref      TEXT        NOT NULL UNIQUE,
    farmer_id       UUID        NOT NULL,
    animal_type     TEXT        NOT NULL
                    CHECK (animal_type IN ('cattle','goat','sheep','pig','poultry','rabbit')),
    breed           TEXT,
    age_months      INT         CHECK (age_months >= 0),
    sex             TEXT        NOT NULL CHECK (sex IN ('M','F')),
    ear_tag         TEXT,
    tenant_id       TEXT        NOT NULL,
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  animals               IS 'Individual animal registry — unique per ear_tag/animal_ref within a farm';
COMMENT ON COLUMN animals.animal_ref    IS 'Human-readable unique reference, e.g. ANI-COW-20260901-001';
COMMENT ON COLUMN animals.animal_type   IS 'Species category for breed references and production tracking';
COMMENT ON COLUMN animals.ear_tag       IS 'Physical ear tag or electronic ID (RFID) for field identification';
COMMENT ON COLUMN animals.age_months    IS 'Age in months at time of registration — updated via periodic records if needed';
COMMENT ON COLUMN animals.sex           IS 'Biological sex: M (male) or F (female) — used for production and breeding logic';

CREATE INDEX IF NOT EXISTS idx_ani_farmer  ON animals(farmer_id);
CREATE INDEX IF NOT EXISTS idx_ani_type    ON animals(animal_type);
CREATE INDEX IF NOT EXISTS idx_ani_tenant  ON animals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ani_ear_tag ON animals(ear_tag) WHERE ear_tag IS NOT NULL;

CREATE TABLE IF NOT EXISTS health_events (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    animal_id        UUID        NOT NULL REFERENCES animals(id),
    event_type       TEXT        NOT NULL
                     CHECK (event_type IN ('vaccination','illness','treatment','checkup','death')),
    description      TEXT,
    treatment        TEXT,
    veterinarian_id  UUID,
    event_date       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by       UUID        NOT NULL
);

COMMENT ON TABLE  health_events                 IS 'Veterinary and health events per animal — full medical history';
COMMENT ON COLUMN health_events.event_type      IS 'Type: vaccination, illness (diagnosis), treatment, checkup (wellness), death';
COMMENT ON COLUMN health_events.treatment       IS 'Treatment administered including drug name, dose, and route if applicable';
COMMENT ON COLUMN health_events.veterinarian_id IS 'UUID of the veterinarian or extension officer who performed the event';

CREATE INDEX IF NOT EXISTS idx_he_animal    ON health_events(animal_id, event_date DESC);
CREATE INDEX IF NOT EXISTS idx_he_event_type ON health_events(event_type);
CREATE INDEX IF NOT EXISTS idx_he_vet       ON health_events(veterinarian_id);

CREATE TABLE IF NOT EXISTS production_records (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    animal_id        UUID          NOT NULL REFERENCES animals(id),
    production_type  TEXT          NOT NULL
                     CHECK (production_type IN ('milk','eggs','wool','meat')),
    quantity         NUMERIC(10,3) NOT NULL CHECK (quantity >= 0),
    unit             TEXT          NOT NULL,
    recorded_date    DATE          NOT NULL,
    recorded_by      UUID          NOT NULL
);

COMMENT ON TABLE  production_records                 IS 'Daily production output per animal — milk, eggs, wool, or meat yield';
COMMENT ON COLUMN production_records.production_type IS 'Output category: milk (L), eggs (count), wool (kg), meat (kg)';
COMMENT ON COLUMN production_records.quantity        IS 'Amount produced; unit specifies the measure (L, count, kg)';
COMMENT ON COLUMN production_records.recorded_date   IS 'Calendar date of production — one record per animal per type per day';

CREATE INDEX IF NOT EXISTS idx_pr_animal      ON production_records(animal_id, recorded_date DESC);
CREATE INDEX IF NOT EXISTS idx_pr_prod_type   ON production_records(production_type, recorded_date DESC);

CREATE TABLE IF NOT EXISTS veterinary_alerts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    animal_id    UUID        NOT NULL,
    alert_type   TEXT        NOT NULL,
    priority     TEXT        NOT NULL DEFAULT 'medium'
                 CHECK (priority IN ('low','medium','high')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  veterinary_alerts            IS 'Urgent health alerts for animals — disease suspicion, overdue vaccination, injury';
COMMENT ON COLUMN veterinary_alerts.alert_type IS 'Alert category, e.g. disease_suspicion, vaccine_overdue, injury, mortality_risk';
COMMENT ON COLUMN veterinary_alerts.priority   IS 'Urgency: low (informational), medium (within 48 h), high (immediate)';

CREATE INDEX IF NOT EXISTS idx_vet_alert_animal   ON veterinary_alerts(animal_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_vet_alert_priority ON veterinary_alerts(priority) WHERE priority = 'high';

-- Immutable audit log
CREATE TABLE IF NOT EXISTS livestock_audit_log (
    id             BIGSERIAL   PRIMARY KEY,
    entity_id      UUID        NOT NULL,
    action         TEXT        NOT NULL,  -- 'create','update','delete','read'
    actor_id       TEXT        NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    signature_hash TEXT,
    ip_address     INET,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE livestock_audit_log IS 'Append-only audit trail for animal registrations, health events, and production records';

CREATE RULE no_update_livestock_audit AS ON UPDATE TO livestock_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_livestock_audit AS ON DELETE TO livestock_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_livestock_audit_entity ON livestock_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_livestock_audit_actor  ON livestock_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS livestock_audit_log CASCADE;
-- DROP TABLE IF EXISTS veterinary_alerts CASCADE;
-- DROP TABLE IF EXISTS production_records CASCADE;
-- DROP TABLE IF EXISTS health_events CASCADE;
-- DROP TABLE IF EXISTS animals CASCADE;
