CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE warehouses (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name           TEXT NOT NULL,
  code           TEXT NOT NULL UNIQUE,
  country        TEXT NOT NULL,
  region         TEXT NOT NULL DEFAULT '',
  address        TEXT NOT NULL,
  latitude       DOUBLE PRECISION NOT NULL DEFAULT 0,
  longitude      DOUBLE PRECISION NOT NULL DEFAULT 0,
  capacity_m3    DOUBLE PRECISION NOT NULL DEFAULT 0,
  used_m3        DOUBLE PRECISION NOT NULL DEFAULT 0,
  status         TEXT NOT NULL DEFAULT 'active',
  manager_name   TEXT NOT NULL DEFAULT '',
  contact_phone  TEXT NOT NULL DEFAULT '',
  notes          TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE stock_items (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  warehouse_id     UUID NOT NULL REFERENCES warehouses(id),
  sku              TEXT NOT NULL,
  product_name     TEXT NOT NULL,
  category         TEXT NOT NULL DEFAULT '',
  bin_location     TEXT NOT NULL DEFAULT '',
  quantity_on_hand DOUBLE PRECISION NOT NULL DEFAULT 0,
  unit             TEXT NOT NULL DEFAULT 'kg',
  unit_weight_kg   DOUBLE PRECISION NOT NULL DEFAULT 0,
  unit_volume_m3   DOUBLE PRECISION NOT NULL DEFAULT 0,
  reorder_level    DOUBLE PRECISION NOT NULL DEFAULT 0,
  supplier_id      UUID,
  last_received_at TIMESTAMPTZ,
  last_dispatched_at TIMESTAMPTZ,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(warehouse_id, sku)
);

CREATE TABLE stock_movements (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  warehouse_id  UUID NOT NULL REFERENCES warehouses(id),
  stock_item_id UUID NOT NULL REFERENCES stock_items(id),
  movement_type TEXT NOT NULL,
  quantity      DOUBLE PRECISION NOT NULL,
  ref_id        UUID,
  ref_type      TEXT,
  notes         TEXT,
  recorded_by   UUID NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE warehouse_audit_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id  UUID,
  user_id    UUID NOT NULL,
  action     TEXT NOT NULL,
  resource   TEXT NOT NULL,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_movements AS ON UPDATE TO stock_movements DO INSTEAD NOTHING;
CREATE RULE no_delete_movements AS ON DELETE TO stock_movements DO INSTEAD NOTHING;
CREATE RULE no_update_warehouse_audit AS ON UPDATE TO warehouse_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_warehouse_audit AS ON DELETE TO warehouse_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_stock_items_warehouse ON stock_items(warehouse_id);
CREATE INDEX idx_stock_movements_item ON stock_movements(stock_item_id, created_at DESC);
CREATE INDEX idx_low_stock ON stock_items(warehouse_id) WHERE quantity_on_hand <= reorder_level;
