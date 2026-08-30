-- =============================================================================
-- V202609010021__Weather_Service__Init.sql
-- Kinara OS — Weather Service
-- =============================================================================
\c kinara_weather;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- weather_observations
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS weather_observations (
    id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    station_id          TEXT          NOT NULL,
    country             TEXT          NOT NULL,
    region              TEXT,
    latitude            NUMERIC(10,7),
    longitude           NUMERIC(10,7),
    temperature_c       NUMERIC(5,2),
    humidity_pct        NUMERIC(5,2)  CHECK (humidity_pct BETWEEN 0 AND 100),
    rainfall_mm         NUMERIC(8,3)  DEFAULT 0,
    wind_speed_kmh      NUMERIC(6,2),
    wind_direction_deg  NUMERIC(5,1),
    pressure_hpa        NUMERIC(7,2),
    weather_code        TEXT,
    description         TEXT,
    recorded_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE weather_observations IS 'Raw meteorological readings from ground stations and remote sensors';
COMMENT ON COLUMN weather_observations.station_id         IS 'WMO or internal station identifier';
COMMENT ON COLUMN weather_observations.humidity_pct       IS 'Relative humidity 0–100%';
COMMENT ON COLUMN weather_observations.rainfall_mm        IS 'Accumulated rainfall in millimetres for the observation period';
COMMENT ON COLUMN weather_observations.wind_direction_deg IS 'Wind direction in degrees (0 = North, 90 = East)';
COMMENT ON COLUMN weather_observations.weather_code       IS 'WMO or OWM weather condition code';

-- ---------------------------------------------------------------------------
-- weather_forecasts
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS weather_forecasts (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    station_id               TEXT        NOT NULL,
    country                  TEXT        NOT NULL,
    region                   TEXT,
    forecast_date            DATE        NOT NULL,
    temp_high_c              NUMERIC(5,2),
    temp_low_c               NUMERIC(5,2),
    rainfall_probability_pct INT         CHECK (rainfall_probability_pct BETWEEN 0 AND 100),
    expected_rainfall_mm     NUMERIC(8,3),
    weather_code             TEXT,
    description              TEXT,
    created_at               TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE weather_forecasts IS 'Daily forecast records per station for agricultural planning';
COMMENT ON COLUMN weather_forecasts.rainfall_probability_pct IS 'Probability of precipitation (0–100%)';
COMMENT ON COLUMN weather_forecasts.expected_rainfall_mm     IS 'Forecasted rainfall in millimetres';
COMMENT ON COLUMN weather_forecasts.forecast_date            IS 'Calendar date the forecast applies to';

-- ---------------------------------------------------------------------------
-- agricultural_advisories
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agricultural_advisories (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    country        TEXT        NOT NULL,
    region         TEXT,
    crop_type      TEXT,
    advisory_type  TEXT        CHECK (advisory_type IN ('planting', 'irrigation', 'pest', 'harvest', 'storage')),
    advisory_text  TEXT        NOT NULL,
    valid_from     DATE,
    valid_until    DATE,
    created_by     UUID,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE agricultural_advisories IS 'Weather-driven crop advisories issued to farmers by region and commodity';
COMMENT ON COLUMN agricultural_advisories.advisory_type IS 'Category of agricultural guidance';
COMMENT ON COLUMN agricultural_advisories.advisory_text IS 'Human-readable advisory message in the target language';
COMMENT ON COLUMN agricultural_advisories.valid_from    IS 'First date the advisory is relevant';
COMMENT ON COLUMN agricultural_advisories.valid_until   IS 'Last date the advisory is relevant — NULL means open-ended';
COMMENT ON COLUMN agricultural_advisories.created_by    IS 'UUID of the agronomist or system that created the advisory';

-- ---------------------------------------------------------------------------
-- weather_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS weather_audit_log (
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

COMMENT ON TABLE weather_audit_log IS 'Immutable audit trail for all weather-service mutations';

CREATE RULE no_update_weather_audit AS ON UPDATE TO weather_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_weather_audit AS ON DELETE TO weather_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_weather_obs_station_time      ON weather_observations(station_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_weather_obs_country_time      ON weather_observations(country, recorded_at);
CREATE INDEX IF NOT EXISTS idx_weather_forecasts_station_date ON weather_forecasts(station_id, forecast_date);
CREATE INDEX IF NOT EXISTS idx_agri_advisories_country_crop  ON agricultural_advisories(country, crop_type, valid_from);
CREATE INDEX IF NOT EXISTS idx_weather_audit_entity          ON weather_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_weather_audit_actor           ON weather_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_weather_audit_actor;
-- DROP INDEX IF EXISTS idx_weather_audit_entity;
-- DROP INDEX IF EXISTS idx_agri_advisories_country_crop;
-- DROP INDEX IF EXISTS idx_weather_forecasts_station_date;
-- DROP INDEX IF EXISTS idx_weather_obs_country_time;
-- DROP INDEX IF EXISTS idx_weather_obs_station_time;
-- DROP RULE IF EXISTS no_delete_weather_audit ON weather_audit_log;
-- DROP RULE IF EXISTS no_update_weather_audit ON weather_audit_log;
-- DROP TABLE IF EXISTS weather_audit_log;
-- DROP TABLE IF EXISTS agricultural_advisories;
-- DROP TABLE IF EXISTS weather_forecasts;
-- DROP TABLE IF EXISTS weather_observations;
-- =============================================================================
