-- Irrigation Service — Farm irrigation systems, schedules, soil moisture, and watering history
-- Database: kinara_irrigation
-- Manages smart irrigation for smallholder farms — sensor data, scheduling, and event logging

\c kinara_irrigation;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS irrigation_systems (
    id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id           TEXT          NOT NULL UNIQUE,
    system_type       TEXT          NOT NULL
                      CHECK (system_type IN ('drip','sprinkler','flood','manual')),
    capacity_liters   NUMERIC(10,2) NOT NULL DEFAULT 0,
    sensor_id         TEXT,
    tenant_id         TEXT          NOT NULL,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  irrigation_systems                IS 'Farm irrigation system registry — one record per farm';
COMMENT ON COLUMN irrigation_systems.farm_id        IS 'Unique farm identifier — FK to agri farm registry (cross-service)';
COMMENT ON COLUMN irrigation_systems.system_type    IS 'Delivery method: drip (efficient), sprinkler (medium), flood (high-volume), manual';
COMMENT ON COLUMN irrigation_systems.capacity_liters IS 'Maximum daily water delivery capacity in litres';
COMMENT ON COLUMN irrigation_systems.sensor_id      IS 'IoT sensor device ID for soil moisture readings — null if no sensor';

CREATE INDEX IF NOT EXISTS idx_irr_sys_tenant ON irrigation_systems(tenant_id);

CREATE TABLE IF NOT EXISTS watering_schedules (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id          TEXT        NOT NULL,
    cron_expression  TEXT        NOT NULL,
    duration_min     INT         NOT NULL DEFAULT 30 CHECK (duration_min > 0),
    crop_type        TEXT,
    tenant_id        TEXT        NOT NULL,
    active           BOOLEAN     NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  watering_schedules                  IS 'Automated irrigation schedules per farm — cron-based triggers';
COMMENT ON COLUMN watering_schedules.cron_expression  IS 'Standard cron expression, e.g. "0 6 * * *" for 06:00 daily';
COMMENT ON COLUMN watering_schedules.duration_min     IS 'Scheduled irrigation duration in minutes';
COMMENT ON COLUMN watering_schedules.crop_type        IS 'Primary crop being irrigated — used to apply crop-specific duration adjustments';

CREATE INDEX IF NOT EXISTS idx_ws_farm   ON watering_schedules(farm_id);
CREATE INDEX IF NOT EXISTS idx_ws_active ON watering_schedules(active) WHERE active = true;

CREATE TABLE IF NOT EXISTS soil_moisture_readings (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id       TEXT          NOT NULL,
    moisture_pct  NUMERIC(5,2)  NOT NULL CHECK (moisture_pct BETWEEN 0 AND 100),
    sensor_id     TEXT          NOT NULL,
    recorded_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  soil_moisture_readings              IS 'Time-series soil moisture sensor readings — drives auto-irrigation logic';
COMMENT ON COLUMN soil_moisture_readings.moisture_pct IS 'Volumetric water content as percentage (0–100%)';
COMMENT ON COLUMN soil_moisture_readings.sensor_id    IS 'IoT device that produced this reading — matches irrigation_systems.sensor_id';

CREATE INDEX IF NOT EXISTS idx_smr_farm_time ON soil_moisture_readings(farm_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_smr_sensor    ON soil_moisture_readings(sensor_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS watering_history (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id         TEXT          NOT NULL,
    duration_min    INT           NOT NULL,
    amount_liters   NUMERIC(10,2),
    trigger_type    TEXT          NOT NULL DEFAULT 'manual'
                    CHECK (trigger_type IN ('manual','scheduled','auto')),
    irrigated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  watering_history              IS 'Actual irrigation events — source of truth for water usage reporting';
COMMENT ON COLUMN watering_history.trigger_type IS 'What initiated the event: manual (operator), scheduled (cron), auto (sensor threshold)';
COMMENT ON COLUMN watering_history.amount_liters IS 'Estimated or metered water volume delivered; null if unmetered';

CREATE INDEX IF NOT EXISTS idx_wh_farm_time ON watering_history(farm_id, irrigated_at DESC);
CREATE INDEX IF NOT EXISTS idx_wh_trigger   ON watering_history(trigger_type);

CREATE TABLE IF NOT EXISTS irrigation_alerts (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id     TEXT        NOT NULL,
    message     TEXT        NOT NULL,
    alert_type  TEXT        NOT NULL
                CHECK (alert_type IN ('low_moisture','overdue','system_fault')),
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  irrigation_alerts            IS 'Farmer notifications for irrigation anomalies — low moisture, missed schedules, sensor faults';
COMMENT ON COLUMN irrigation_alerts.alert_type IS 'Category: low_moisture (water now), overdue (schedule missed), system_fault (hardware issue)';

CREATE INDEX IF NOT EXISTS idx_irr_alert_farm  ON irrigation_alerts(farm_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_irr_alert_type  ON irrigation_alerts(alert_type);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS irrigation_audit_log (
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

COMMENT ON TABLE irrigation_audit_log IS 'Append-only audit trail for irrigation system configuration and schedule changes';

CREATE RULE no_update_irrigation_audit AS ON UPDATE TO irrigation_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_irrigation_audit AS ON DELETE TO irrigation_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_irrigation_audit_entity ON irrigation_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_irrigation_audit_actor  ON irrigation_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS irrigation_audit_log CASCADE;
-- DROP TABLE IF EXISTS irrigation_alerts CASCADE;
-- DROP TABLE IF EXISTS watering_history CASCADE;
-- DROP TABLE IF EXISTS soil_moisture_readings CASCADE;
-- DROP TABLE IF EXISTS watering_schedules CASCADE;
-- DROP TABLE IF EXISTS irrigation_systems CASCADE;
