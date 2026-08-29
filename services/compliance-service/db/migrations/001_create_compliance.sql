CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE transit_permits (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  permit_no         TEXT NOT NULL UNIQUE,
  vehicle_id        UUID NOT NULL,
  driver_id         UUID,
  permit_type       TEXT NOT NULL,
  status            TEXT NOT NULL DEFAULT 'pending',
  issued_by         TEXT NOT NULL,
  country           TEXT NOT NULL,
  route_restriction TEXT,
  max_weight_kg     DOUBLE PRECISION NOT NULL DEFAULT 0,
  valid_from        TIMESTAMPTZ NOT NULL,
  valid_until       TIMESTAMPTZ NOT NULL,
  notes             TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE border_crossings (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vehicle_id     UUID NOT NULL,
  driver_id      UUID NOT NULL,
  from_country   TEXT NOT NULL,
  to_country     TEXT NOT NULL,
  border_post    TEXT NOT NULL,
  cargo_desc     TEXT NOT NULL DEFAULT '',
  gross_weight_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  crossed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  exit_permit_no TEXT NOT NULL DEFAULT '',
  entry_permit_no TEXT NOT NULL DEFAULT '',
  notes          TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE weight_checks (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vehicle_id      UUID NOT NULL,
  country         TEXT NOT NULL,
  check_station   TEXT NOT NULL,
  gross_weight_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
  legal_limit_kg  DOUBLE PRECISION NOT NULL DEFAULT 0,
  is_compliant    BOOLEAN NOT NULL DEFAULT TRUE,
  fine_amount     DOUBLE PRECISION NOT NULL DEFAULT 0,
  currency        TEXT NOT NULL DEFAULT 'USD',
  checked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  notes           TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE compliance_audit_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id  UUID,
  user_id    UUID NOT NULL,
  action     TEXT NOT NULL,
  resource   TEXT NOT NULL,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_border_crossings AS ON UPDATE TO border_crossings DO INSTEAD NOTHING;
CREATE RULE no_delete_border_crossings AS ON DELETE TO border_crossings DO INSTEAD NOTHING;
CREATE RULE no_update_weight_checks AS ON UPDATE TO weight_checks DO INSTEAD NOTHING;
CREATE RULE no_delete_weight_checks AS ON DELETE TO weight_checks DO INSTEAD NOTHING;
CREATE RULE no_update_compliance_audit AS ON UPDATE TO compliance_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_compliance_audit AS ON DELETE TO compliance_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_permits_vehicle ON transit_permits(vehicle_id);
CREATE INDEX idx_permits_status ON transit_permits(status);
CREATE INDEX idx_permits_expiry ON transit_permits(valid_until);
CREATE INDEX idx_crossings_vehicle ON border_crossings(vehicle_id, crossed_at DESC);
CREATE INDEX idx_weight_checks_vehicle ON weight_checks(vehicle_id, checked_at DESC);
