CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE transport_trips (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_code           TEXT NOT NULL UNIQUE,
  route_id            UUID,
  vehicle_id          UUID NOT NULL,
  driver_id           UUID NOT NULL,
  cargo_id            UUID,
  status              TEXT NOT NULL DEFAULT 'scheduled',
  country             TEXT NOT NULL,
  origin_address      TEXT NOT NULL,
  origin_lat          DOUBLE PRECISION NOT NULL DEFAULT 0,
  origin_lng          DOUBLE PRECISION NOT NULL DEFAULT 0,
  dest_address        TEXT NOT NULL,
  dest_lat            DOUBLE PRECISION NOT NULL DEFAULT 0,
  dest_lng            DOUBLE PRECISION NOT NULL DEFAULT 0,
  scheduled_pickup    TIMESTAMPTZ NOT NULL,
  scheduled_delivery  TIMESTAMPTZ,
  actual_pickup       TIMESTAMPTZ,
  actual_delivery     TIMESTAMPTZ,
  distance_km         DOUBLE PRECISION NOT NULL DEFAULT 0,
  cost_per_km         DOUBLE PRECISION NOT NULL DEFAULT 0,
  total_cost          DOUBLE PRECISION NOT NULL DEFAULT 0,
  currency            TEXT NOT NULL DEFAULT 'USD',
  fuel_cost           DOUBLE PRECISION NOT NULL DEFAULT 0,
  current_lat         DOUBLE PRECISION,
  current_lng         DOUBLE PRECISION,
  last_gps_update     TIMESTAMPTZ,
  delay_reason_code   TEXT,
  notes               TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE gps_updates (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id     UUID NOT NULL REFERENCES transport_trips(id),
  latitude    DOUBLE PRECISION NOT NULL,
  longitude   DOUBLE PRECISION NOT NULL,
  speed_kph   DOUBLE PRECISION NOT NULL DEFAULT 0,
  heading     DOUBLE PRECISION NOT NULL DEFAULT 0,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transport_audit_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id  UUID,
  user_id    UUID NOT NULL,
  action     TEXT NOT NULL,
  resource   TEXT NOT NULL,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_gps AS ON UPDATE TO gps_updates DO INSTEAD NOTHING;
CREATE RULE no_delete_gps AS ON DELETE TO gps_updates DO INSTEAD NOTHING;
CREATE RULE no_update_transport_audit AS ON UPDATE TO transport_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_transport_audit AS ON DELETE TO transport_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_transport_trips_status ON transport_trips(status);
CREATE INDEX idx_transport_trips_driver ON transport_trips(driver_id);
CREATE INDEX idx_transport_trips_vehicle ON transport_trips(vehicle_id);
CREATE INDEX idx_gps_trip ON gps_updates(trip_id, recorded_at DESC);
