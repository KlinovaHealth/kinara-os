CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS irrigation_schedules (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_ref   TEXT NOT NULL UNIQUE,
    farmer_id      UUID NOT NULL,
    field_id       TEXT NOT NULL,
    crop_type      TEXT NOT NULL,
    method         TEXT NOT NULL,
    frequency_days INT NOT NULL DEFAULT 7,
    duration_min   INT NOT NULL DEFAULT 60,
    water_liters   NUMERIC(10,2) NOT NULL,
    is_active      BOOLEAN NOT NULL DEFAULT true,
    tenant_id      TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS irrigation_events (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id  UUID NOT NULL REFERENCES irrigation_schedules(id),
    farmer_id    UUID NOT NULL,
    field_id     TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    water_used_l NUMERIC(10,2) NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'scheduled',
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schedules_farmer ON irrigation_schedules(farmer_id);
CREATE INDEX IF NOT EXISTS idx_events_schedule ON irrigation_events(schedule_id);
CREATE INDEX IF NOT EXISTS idx_events_farmer ON irrigation_events(farmer_id);
