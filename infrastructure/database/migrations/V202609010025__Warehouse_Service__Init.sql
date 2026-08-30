-- =============================================================================
-- V202609010025__Warehouse_Service__Init.sql
-- Kinara OS — Warehouse Service
-- =============================================================================
\c kinara_warehouse;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- warehouses
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS warehouses (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT        NOT NULL,
    location              TEXT        NOT NULL,
    country               TEXT        NOT NULL,
    region                TEXT,
    gps_lat               DOUBLE PRECISION,
    gps_lng               DOUBLE PRECISION,
    total_capacity_m3     DOUBLE PRECISION NOT NULL,
    available_capacity_m3 DOUBLE PRECISION NOT NULL,
    warehouse_type        TEXT        DEFAULT 'general'
                          CHECK (warehouse_type IN ('general', 'cold_chain', 'hazmat', 'grain', 'pharmaceutical')),
    status                TEXT        DEFAULT 'active',
    manager_id            UUID,
    tenant_id             TEXT,
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE warehouses IS 'Warehouse registry for physical storage facilities across the Kinara network';
COMMENT ON COLUMN warehouses.total_capacity_m3     IS 'Total designed storage volume in cubic metres';
COMMENT ON COLUMN warehouses.available_capacity_m3 IS 'Real-time available capacity — updated on each movement';
COMMENT ON COLUMN warehouses.warehouse_type        IS 'Facility type determining permitted commodities and temperature range';
COMMENT ON COLUMN warehouses.manager_id            IS 'UUID of the facility manager responsible for operations';
COMMENT ON COLUMN warehouses.tenant_id             IS 'Multi-tenant partition key';

-- ---------------------------------------------------------------------------
-- storage_slots
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS storage_slots (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id     UUID        NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    slot_code        TEXT        NOT NULL,
    slot_type        TEXT,
    capacity_kg      DOUBLE PRECISION,
    temperature_zone TEXT        CHECK (temperature_zone IN ('ambient', 'cool', 'cold', 'frozen')),
    status           TEXT        DEFAULT 'available'
                     CHECK (status IN ('available', 'occupied', 'reserved', 'maintenance')),
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE storage_slots IS 'Individual storage positions (bins, bays, racks) within a warehouse';
COMMENT ON COLUMN storage_slots.slot_code        IS 'Human-readable slot identifier (e.g. A-01-03)';
COMMENT ON COLUMN storage_slots.temperature_zone IS 'Temperature band required for commodities stored in this slot';
COMMENT ON COLUMN storage_slots.capacity_kg      IS 'Maximum weight load in kilograms for this slot';

-- ---------------------------------------------------------------------------
-- inventory_items
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS inventory_items (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID        NOT NULL REFERENCES warehouses(id),
    slot_id      UUID        REFERENCES storage_slots(id),
    owner_id     UUID        NOT NULL,
    commodity    TEXT        NOT NULL,
    quantity_kg  DOUBLE PRECISION,
    volume_m3    DOUBLE PRECISION,
    stored_at    TIMESTAMPTZ DEFAULT NOW(),
    expires_at   TIMESTAMPTZ,
    condition    TEXT        DEFAULT 'good'
                 CHECK (condition IN ('good', 'fair', 'damaged', 'expired')),
    notes        TEXT
);

COMMENT ON TABLE inventory_items IS 'Commodity inventory records held within warehouse storage slots';
COMMENT ON COLUMN inventory_items.owner_id   IS 'UUID of the farmer, cooperative, or trader that owns the goods';
COMMENT ON COLUMN inventory_items.commodity  IS 'Agricultural product name (e.g. maize, rice, groundnuts)';
COMMENT ON COLUMN inventory_items.expires_at IS 'Expected expiry or best-before date for the stored commodity';
COMMENT ON COLUMN inventory_items.condition  IS 'Physical state of the goods at last inspection';

-- ---------------------------------------------------------------------------
-- warehouse_movements
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS warehouse_movements (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id       UUID        NOT NULL REFERENCES warehouses(id),
    inventory_item_id  UUID,
    movement_type      TEXT        CHECK (movement_type IN ('in', 'out', 'transfer', 'adjustment')),
    quantity_kg        DOUBLE PRECISION,
    reference_id       UUID,
    performed_by       UUID,
    occurred_at        TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE warehouse_movements IS 'Ledger of all stock movements (in, out, transfer, adjustment) within warehouses';
COMMENT ON COLUMN warehouse_movements.movement_type     IS 'Direction or nature of the stock movement';
COMMENT ON COLUMN warehouse_movements.reference_id      IS 'FK to the originating transfer, order, or adjustment document';
COMMENT ON COLUMN warehouse_movements.performed_by      IS 'UUID of the warehouse operative who executed the movement';

-- ---------------------------------------------------------------------------
-- warehouse_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS warehouse_audit_log (
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

COMMENT ON TABLE warehouse_audit_log IS 'Immutable audit trail for all warehouse-service mutations';

CREATE RULE no_update_warehouse_audit AS ON UPDATE TO warehouse_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_warehouse_audit AS ON DELETE TO warehouse_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_warehouses_country_status    ON warehouses(country, status);
CREATE INDEX IF NOT EXISTS idx_storage_slots_warehouse_status ON storage_slots(warehouse_id, status);
CREATE INDEX IF NOT EXISTS idx_inventory_items_warehouse     ON inventory_items(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inventory_items_owner        ON inventory_items(owner_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_movements_warehouse ON warehouse_movements(warehouse_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_warehouse_audit_entity       ON warehouse_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_warehouse_audit_actor        ON warehouse_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_warehouse_audit_actor;
-- DROP INDEX IF EXISTS idx_warehouse_audit_entity;
-- DROP INDEX IF EXISTS idx_warehouse_movements_warehouse;
-- DROP INDEX IF EXISTS idx_inventory_items_owner;
-- DROP INDEX IF EXISTS idx_inventory_items_warehouse;
-- DROP INDEX IF EXISTS idx_storage_slots_warehouse_status;
-- DROP INDEX IF EXISTS idx_warehouses_country_status;
-- DROP RULE IF EXISTS no_delete_warehouse_audit ON warehouse_audit_log;
-- DROP RULE IF EXISTS no_update_warehouse_audit ON warehouse_audit_log;
-- DROP TABLE IF EXISTS warehouse_audit_log;
-- DROP TABLE IF EXISTS warehouse_movements;
-- DROP TABLE IF EXISTS inventory_items;
-- DROP TABLE IF EXISTS storage_slots;
-- DROP TABLE IF EXISTS warehouses;
-- =============================================================================
