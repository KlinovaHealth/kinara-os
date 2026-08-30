CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS irrigation_systems (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id TEXT NOT NULL UNIQUE,
    system_type TEXT NOT NULL,
    capacity_liters NUMERIC(10,2) NOT NULL DEFAULT 0,
    sensor_id TEXT,
    tenant_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS watering_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id TEXT NOT NULL,
    cron_expression TEXT NOT NULL,
    duration_min INT NOT NULL DEFAULT 30,
    crop_type TEXT,
    tenant_id TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_schedules_farm ON watering_schedules(farm_id);
CREATE TABLE IF NOT EXISTS soil_moisture_readings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id TEXT NOT NULL,
    moisture_pct NUMERIC(5,2) NOT NULL,
    sensor_id TEXT,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_moisture_farm ON soil_moisture_readings(farm_id, recorded_at);
CREATE TABLE IF NOT EXISTS watering_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id TEXT NOT NULL,
    duration_min INT NOT NULL,
    amount_liters NUMERIC(10,2),
    trigger_type TEXT NOT NULL DEFAULT 'manual',
    irrigated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_history_farm ON watering_history(farm_id, irrigated_at);
CREATE TABLE IF NOT EXISTS irrigation_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    farm_id TEXT NOT NULL,
    message TEXT NOT NULL,
    alert_type TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS irrigation_audit_log (
    id BIGSERIAL PRIMARY KEY,
    farm_id TEXT NOT NULL,
    action TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE RULE no_update_irr_audit AS ON UPDATE TO irrigation_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_irr_audit AS ON DELETE TO irrigation_audit_log DO INSTEAD NOTHING;
