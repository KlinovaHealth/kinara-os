-- Weather Service Schema
-- Agriculture + Health Pillars: forecasts, alerts, pest/disease advisories

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── weather_forecasts ────────────────────────────────────────────────────────
CREATE TABLE weather_forecasts (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    country         TEXT            NOT NULL,
    region          TEXT            NOT NULL DEFAULT '',
    district        TEXT            NOT NULL DEFAULT '',
    latitude        DOUBLE PRECISION NOT NULL,
    longitude       DOUBLE PRECISION NOT NULL,
    forecast_type   TEXT            NOT NULL DEFAULT 'daily'
                        CHECK (forecast_type IN ('daily','hourly','seasonal')),
    forecast_date   TIMESTAMPTZ     NOT NULL,
    condition       TEXT            NOT NULL
                        CHECK (condition IN (
                            'sunny','partly_cloudy','cloudy','rainy',
                            'heavy_rain','thunderstorm','drizzle','foggy','windy','haze'
                        )),
    temp_min_c      DOUBLE PRECISION NOT NULL DEFAULT 0,
    temp_max_c      DOUBLE PRECISION NOT NULL DEFAULT 0,
    temp_avg_c      DOUBLE PRECISION NOT NULL DEFAULT 0,
    humidity_pct    DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (humidity_pct BETWEEN 0 AND 100),
    wind_speed_kmh  DOUBLE PRECISION NOT NULL DEFAULT 0,
    wind_direction  TEXT            NOT NULL DEFAULT '',
    rainfall_mm     DOUBLE PRECISION NOT NULL DEFAULT 0,
    rainfall_prob   DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (rainfall_prob BETWEEN 0 AND 100),
    uv_index        DOUBLE PRECISION NOT NULL DEFAULT 0,
    data_source     TEXT            NOT NULL DEFAULT 'manual',
    valid_until     TIMESTAMPTZ     NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_forecasts_country  ON weather_forecasts(country);
CREATE INDEX idx_forecasts_region   ON weather_forecasts(region);
CREATE INDEX idx_forecasts_date     ON weather_forecasts(forecast_date);
CREATE INDEX idx_forecasts_location ON weather_forecasts(latitude, longitude);

-- ─── weather_alerts ───────────────────────────────────────────────────────────
CREATE TABLE weather_alerts (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_type      TEXT        NOT NULL
                        CHECK (alert_type IN (
                            'flood','drought','frost','heat_wave','high_wind',
                            'heavy_rain','pest_risk','disease_risk','locust','fire_risk'
                        )),
    severity        TEXT        NOT NULL DEFAULT 'warning'
                        CHECK (severity IN ('info','watch','warning','emergency')),
    country         TEXT        NOT NULL,
    region          TEXT        NOT NULL DEFAULT '',
    district        TEXT        NOT NULL DEFAULT '',
    title           TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    instructions    TEXT        NOT NULL DEFAULT '',
    affected_crops  TEXT[]      NOT NULL DEFAULT '{}',
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    active          BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_country  ON weather_alerts(country);
CREATE INDEX idx_alerts_region   ON weather_alerts(region);
CREATE INDEX idx_alerts_type     ON weather_alerts(alert_type);
CREATE INDEX idx_alerts_severity ON weather_alerts(severity);
CREATE INDEX idx_alerts_active   ON weather_alerts(active) WHERE active = true;

-- ─── pest_advisories ──────────────────────────────────────────────────────────
CREATE TABLE pest_advisories (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    pest_name       TEXT        NOT NULL,
    pest_type       TEXT        NOT NULL DEFAULT 'pest'
                        CHECK (pest_type IN ('pest','disease')),
    affected_crops  TEXT[]      NOT NULL DEFAULT '{}',
    country         TEXT        NOT NULL,
    region          TEXT        NOT NULL DEFAULT '',
    risk_level      TEXT        NOT NULL DEFAULT 'moderate'
                        CHECK (risk_level IN ('low','moderate','high','critical')),
    description     TEXT        NOT NULL DEFAULT '',
    symptoms        TEXT        NOT NULL DEFAULT '',
    prevention      TEXT        NOT NULL DEFAULT '',
    treatment       TEXT        NOT NULL DEFAULT '',
    reported_cases  INT         NOT NULL DEFAULT 0,
    valid_from      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_advisories_country    ON pest_advisories(country);
CREATE INDEX idx_advisories_region     ON pest_advisories(region);
CREATE INDEX idx_advisories_pest       ON pest_advisories(pest_name);
CREATE INDEX idx_advisories_risk       ON pest_advisories(risk_level);

-- ─── weather_observations ─────────────────────────────────────────────────────
-- Ground-truth reports from farmers/field agents
CREATE TABLE weather_observations (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id     UUID            NOT NULL,
    country         TEXT            NOT NULL,
    region          TEXT            NOT NULL DEFAULT '',
    district        TEXT            NOT NULL DEFAULT '',
    latitude        DOUBLE PRECISION NOT NULL,
    longitude       DOUBLE PRECISION NOT NULL,
    observed_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    temp_c          DOUBLE PRECISION,
    rainfall_mm     DOUBLE PRECISION,
    humidity_pct    DOUBLE PRECISION,
    wind_speed_kmh  DOUBLE PRECISION,
    condition       TEXT            NOT NULL
                        CHECK (condition IN (
                            'sunny','partly_cloudy','cloudy','rainy',
                            'heavy_rain','thunderstorm','drizzle','foggy','windy','haze'
                        )),
    notes           TEXT            NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_observations_reporter ON weather_observations(reporter_id);
CREATE INDEX idx_observations_country  ON weather_observations(country);
CREATE INDEX idx_observations_region   ON weather_observations(region);
CREATE INDEX idx_observations_date     ON weather_observations(observed_at DESC);

-- Immutable: observations are ground truth records, never edited
CREATE RULE weather_observations_no_update AS
    ON UPDATE TO weather_observations DO INSTEAD NOTHING;

CREATE RULE weather_observations_no_delete AS
    ON DELETE TO weather_observations DO INSTEAD NOTHING;

-- ─── weather_audit_log ────────────────────────────────────────────────────────
CREATE TABLE weather_audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   UUID,
    user_id     UUID        NOT NULL,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    ip_address  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE weather_audit_log_no_update AS
    ON UPDATE TO weather_audit_log DO INSTEAD NOTHING;

CREATE RULE weather_audit_log_no_delete AS
    ON DELETE TO weather_audit_log DO INSTEAD NOTHING;
