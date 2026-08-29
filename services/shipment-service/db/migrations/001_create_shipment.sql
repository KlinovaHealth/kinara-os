CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE shipments (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tracking_code    TEXT NOT NULL UNIQUE,
  sender_id        UUID NOT NULL,
  recipient_name   TEXT NOT NULL,
  recipient_phone  TEXT NOT NULL,
  origin_address   TEXT NOT NULL,
  origin_country   TEXT NOT NULL,
  dest_address     TEXT NOT NULL,
  dest_country     TEXT NOT NULL,
  weight_kg        DOUBLE PRECISION NOT NULL DEFAULT 0,
  length_cm        DOUBLE PRECISION NOT NULL DEFAULT 0,
  width_cm         DOUBLE PRECISION NOT NULL DEFAULT 0,
  height_cm        DOUBLE PRECISION NOT NULL DEFAULT 0,
  declared_value   DOUBLE PRECISION NOT NULL DEFAULT 0,
  currency         TEXT NOT NULL DEFAULT 'USD',
  service_level    TEXT NOT NULL DEFAULT 'standard',
  status           TEXT NOT NULL DEFAULT 'created',
  freight_charge   DOUBLE PRECISION NOT NULL DEFAULT 0,
  insurance_charge DOUBLE PRECISION NOT NULL DEFAULT 0,
  total_charge     DOUBLE PRECISION NOT NULL DEFAULT 0,
  picked_at        TIMESTAMPTZ,
  delivered_at     TIMESTAMPTZ,
  est_delivery     TIMESTAMPTZ,
  notes            TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE shipment_events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shipment_id UUID NOT NULL REFERENCES shipments(id),
  status      TEXT NOT NULL,
  location    TEXT NOT NULL DEFAULT '',
  notes       TEXT,
  event_time  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE shipment_audit_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id  UUID,
  user_id    UUID NOT NULL,
  action     TEXT NOT NULL,
  resource   TEXT NOT NULL,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_shipment_events AS ON UPDATE TO shipment_events DO INSTEAD NOTHING;
CREATE RULE no_delete_shipment_events AS ON DELETE TO shipment_events DO INSTEAD NOTHING;
CREATE RULE no_update_shipment_audit AS ON UPDATE TO shipment_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_shipment_audit AS ON DELETE TO shipment_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_shipments_status ON shipments(status);
CREATE INDEX idx_shipments_sender ON shipments(sender_id);
CREATE INDEX idx_shipment_events ON shipment_events(shipment_id, event_time DESC);
