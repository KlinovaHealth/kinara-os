-- =============================================================================
-- V202609010018__Farmer_Service__Init.sql
-- Kinara OS — Farmer Service
-- =============================================================================
\c kinara_farmer;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- farmers
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS farmers (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        UNIQUE,
    full_name_enc    TEXT        NOT NULL,
    phone_enc        TEXT        NOT NULL,
    national_id_enc  TEXT        NOT NULL DEFAULT '',
    country          TEXT        NOT NULL,
    region           TEXT        NOT NULL DEFAULT '',
    district         TEXT        NOT NULL DEFAULT '',
    gps_lat          DOUBLE PRECISION,
    gps_lng          DOUBLE PRECISION,
    farm_size_ha     DOUBLE PRECISION        DEFAULT 0,
    farm_size        TEXT        DEFAULT 'smallholder'
                     CHECK (farm_size IN ('smallholder', 'small', 'medium', 'large')),
    primary_language TEXT        DEFAULT 'en',
    is_verified      BOOLEAN     DEFAULT false,
    is_active        BOOLEAN     DEFAULT true,
    cooperative_id   UUID,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE farmers IS 'Farmer registry — PHI fields AES-256-GCM encrypted';
COMMENT ON COLUMN farmers.full_name_enc   IS 'Full name ciphertext — AES-256-GCM encrypted';
COMMENT ON COLUMN farmers.phone_enc       IS 'Phone number ciphertext — AES-256-GCM encrypted';
COMMENT ON COLUMN farmers.national_id_enc IS 'National ID ciphertext — never returned in list views';
COMMENT ON COLUMN farmers.gps_lat         IS 'Latitude of primary farm location';
COMMENT ON COLUMN farmers.gps_lng         IS 'Longitude of primary farm location';
COMMENT ON COLUMN farmers.farm_size       IS 'Categorical farm size bucket';
COMMENT ON COLUMN farmers.cooperative_id  IS 'FK to cooperative in kinara_cooperative (cross-DB reference)';

-- ---------------------------------------------------------------------------
-- farm_plots
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS farm_plots (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id   UUID        NOT NULL REFERENCES farmers(id) ON DELETE CASCADE,
    plot_name   TEXT,
    area_ha     DOUBLE PRECISION CHECK (area_ha > 0),
    soil_type   TEXT        CHECK (soil_type IN ('clay', 'sandy', 'loam', 'silty', 'peaty')),
    gps_lat     DOUBLE PRECISION,
    gps_lng     DOUBLE PRECISION,
    is_irrigated BOOLEAN    DEFAULT false,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE farm_plots IS 'Individual plot records belonging to a farmer';
COMMENT ON COLUMN farm_plots.area_ha    IS 'Plot area in hectares — must be positive';
COMMENT ON COLUMN farm_plots.soil_type  IS 'Dominant soil classification for the plot';

-- ---------------------------------------------------------------------------
-- crop_records
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS crop_records (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id            UUID        NOT NULL REFERENCES farmers(id) ON DELETE CASCADE,
    plot_id              UUID        REFERENCES farm_plots(id),
    crop_type            TEXT        NOT NULL,
    variety              TEXT,
    season               TEXT,
    planting_date        DATE,
    expected_harvest_date DATE,
    actual_harvest_date  DATE,
    yield_kg             DOUBLE PRECISION,
    area_ha              DOUBLE PRECISION,
    status               TEXT        DEFAULT 'planted'
                         CHECK (status IN ('planted', 'growing', 'harvested', 'failed')),
    created_at           TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE crop_records IS 'Seasonal crop planting and harvest records per farmer plot';
COMMENT ON COLUMN crop_records.crop_type IS 'Commodity name (e.g. maize, cassava, rice)';
COMMENT ON COLUMN crop_records.status    IS 'Lifecycle state of the crop season';
COMMENT ON COLUMN crop_records.yield_kg  IS 'Actual harvested yield in kilograms';

-- ---------------------------------------------------------------------------
-- farmer_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS farmer_audit_log (
    id             BIGSERIAL    PRIMARY KEY,
    entity_id      UUID         NOT NULL,
    action         TEXT         NOT NULL,
    actor_id       TEXT         NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    signature_hash TEXT,
    ip_address     INET,
    occurred_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE farmer_audit_log IS 'Immutable audit trail for all farmer-service mutations';

CREATE RULE no_update_farmer_audit AS ON UPDATE TO farmer_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_farmer_audit AS ON DELETE TO farmer_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_farmers_country    ON farmers(country);
CREATE INDEX IF NOT EXISTS idx_farmers_region     ON farmers(region);
CREATE INDEX IF NOT EXISTS idx_farmers_is_active  ON farmers(is_active);
CREATE INDEX IF NOT EXISTS idx_farm_plots_farmer  ON farm_plots(farmer_id);
CREATE INDEX IF NOT EXISTS idx_crop_records_farmer ON crop_records(farmer_id);
CREATE INDEX IF NOT EXISTS idx_crop_records_type_season ON crop_records(crop_type, season);
CREATE INDEX IF NOT EXISTS idx_farmer_audit_entity ON farmer_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_farmer_audit_actor  ON farmer_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_farmer_audit_actor;
-- DROP INDEX IF EXISTS idx_farmer_audit_entity;
-- DROP INDEX IF EXISTS idx_crop_records_type_season;
-- DROP INDEX IF EXISTS idx_crop_records_farmer;
-- DROP INDEX IF EXISTS idx_farm_plots_farmer;
-- DROP INDEX IF EXISTS idx_farmers_is_active;
-- DROP INDEX IF EXISTS idx_farmers_region;
-- DROP INDEX IF EXISTS idx_farmers_country;
-- DROP RULE IF EXISTS no_delete_farmer_audit ON farmer_audit_log;
-- DROP RULE IF EXISTS no_update_farmer_audit ON farmer_audit_log;
-- DROP TABLE IF EXISTS farmer_audit_log;
-- DROP TABLE IF EXISTS crop_records;
-- DROP TABLE IF EXISTS farm_plots;
-- DROP TABLE IF EXISTS farmers;
-- =============================================================================
