CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE deliveries (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  delivery_code    TEXT NOT NULL UNIQUE,
  cargo_id         UUID,
  driver_id        UUID,
  recipient_name   TEXT NOT NULL,
  recipient_phone  TEXT NOT NULL,
  delivery_address TEXT NOT NULL,
  delivery_lat     DOUBLE PRECISION NOT NULL DEFAULT 0,
  delivery_lng     DOUBLE PRECISION NOT NULL DEFAULT 0,
  status           TEXT NOT NULL DEFAULT 'pending',
  window_start     TIMESTAMPTZ,
  window_end       TIMESTAMPTZ,
  attempt_count    INT NOT NULL DEFAULT 0,
  delivered_at     TIMESTAMPTZ,
  proof_photo_url  TEXT,
  signature_url    TEXT,
  failure_reason   TEXT,
  next_attempt_at  TIMESTAMPTZ,
  sms_notified     BOOLEAN NOT NULL DEFAULT FALSE,
  country          TEXT NOT NULL DEFAULT '',
  notes            TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lastmile_audit_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id  UUID,
  user_id    UUID NOT NULL,
  action     TEXT NOT NULL,
  resource   TEXT NOT NULL,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_lastmile_audit AS ON UPDATE TO lastmile_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_lastmile_audit AS ON DELETE TO lastmile_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_deliveries_status ON deliveries(status);
CREATE INDEX idx_deliveries_driver ON deliveries(driver_id);
CREATE INDEX idx_deliveries_cargo ON deliveries(cargo_id);
