-- Farmer Service Schema
-- Agriculture Pillar: farmer registry, plots, crop records

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── farmers ─────────────────────────────────────────────────────────────────
CREATE TABLE farmers (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        UNIQUE,             -- links to auth-service
    full_name_enc       TEXT        NOT NULL,
    phone_enc           TEXT        NOT NULL,
    national_id_enc     TEXT        NOT NULL DEFAULT '',
    country             TEXT        NOT NULL,
    region              TEXT        NOT NULL DEFAULT '',
    district            TEXT        NOT NULL DEFAULT '',
    gps_lat             DOUBLE PRECISION,
    gps_lng             DOUBLE PRECISION,
    farm_size_ha        DOUBLE PRECISION NOT NULL DEFAULT 0,
    farm_size           TEXT        NOT NULL DEFAULT 'smallholder'
                            CHECK (farm_size IN ('smallholder','small','medium','large')),
    primary_language    TEXT        NOT NULL DEFAULT 'en',
    is_verified         BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    cooperative_id      UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_farmers_country      ON farmers(country);
CREATE INDEX idx_farmers_region       ON farmers(region);
CREATE INDEX idx_farmers_cooperative  ON farmers(cooperative_id);
CREATE INDEX idx_farmers_active       ON farmers(is_active);

-- ─── farm_plots ───────────────────────────────────────────────────────────────
CREATE TABLE farm_plots (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id       UUID        NOT NULL REFERENCES farmers(id) ON DELETE RESTRICT,
    name            TEXT        NOT NULL,
    size_ha         DOUBLE PRECISION NOT NULL DEFAULT 0,
    soil_type       TEXT        NOT NULL DEFAULT '',
    irrigation      BOOLEAN     NOT NULL DEFAULT FALSE,
    gps_polygon     TEXT        NOT NULL DEFAULT '', -- GeoJSON
    current_crop    TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plots_farmer ON farm_plots(farmer_id);

-- ─── crop_records ─────────────────────────────────────────────────────────────
CREATE TABLE crop_records (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id           UUID        NOT NULL REFERENCES farmers(id) ON DELETE RESTRICT,
    plot_id             UUID        REFERENCES farm_plots(id),
    crop_type           TEXT        NOT NULL,
    variety             TEXT        NOT NULL DEFAULT '',
    area_ha             DOUBLE PRECISION NOT NULL,
    planted_at          TIMESTAMPTZ NOT NULL,
    expected_harvest    TIMESTAMPTZ NOT NULL,
    actual_harvest      TIMESTAMPTZ,
    yield_kg            DOUBLE PRECISION,
    status              TEXT        NOT NULL DEFAULT 'planted'
                            CHECK (status IN ('planted','growing','harvested','failed')),
    notes               TEXT        NOT NULL DEFAULT '',
    season              TEXT        NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crops_farmer  ON crop_records(farmer_id);
CREATE INDEX idx_crops_status  ON crop_records(status);
CREATE INDEX idx_crops_type    ON crop_records(crop_type);
CREATE INDEX idx_crops_season  ON crop_records(season);

-- ─── farmer_audit_log ────────────────────────────────────────────────────────
CREATE TABLE farmer_audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id   UUID,
    user_id     UUID        NOT NULL,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    ip_address  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_farmer_audit_farmer ON farmer_audit_log(farmer_id);
CREATE INDEX idx_farmer_audit_user   ON farmer_audit_log(user_id);

CREATE RULE farmer_audit_log_no_update AS
    ON UPDATE TO farmer_audit_log DO INSTEAD NOTHING;

CREATE RULE farmer_audit_log_no_delete AS
    ON DELETE TO farmer_audit_log DO INSTEAD NOTHING;
