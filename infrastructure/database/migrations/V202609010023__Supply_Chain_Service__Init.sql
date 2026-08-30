-- =============================================================================
-- V202609010023__Supply_Chain_Service__Init.sql
-- Kinara OS — Supply Chain Service
-- NOTE: kinara_supply_chain may not have been created in V001 if the
--       bootstrap script only provisioned the original four core databases.
--       Ensure the DBA or infra pipeline creates this database before running
--       this migration: CREATE DATABASE kinara_supply_chain;
-- =============================================================================
\c kinara_supply_chain;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- supply_chain_nodes
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS supply_chain_nodes (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    node_name        TEXT        NOT NULL,
    node_type        TEXT        CHECK (node_type IN (
                         'farm', 'collection_centre', 'processor',
                         'wholesaler', 'retailer', 'port', 'warehouse')),
    country          TEXT        NOT NULL,
    region           TEXT,
    gps_lat          DOUBLE PRECISION,
    gps_lng          DOUBLE PRECISION,
    capacity_tonnes  DOUBLE PRECISION,
    status           TEXT        DEFAULT 'active',
    tenant_id        TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE supply_chain_nodes IS 'Physical or logical nodes in the agricultural supply chain network';
COMMENT ON COLUMN supply_chain_nodes.node_type       IS 'Classification of the node in the supply chain topology';
COMMENT ON COLUMN supply_chain_nodes.capacity_tonnes IS 'Maximum throughput or storage capacity in metric tonnes';
COMMENT ON COLUMN supply_chain_nodes.tenant_id       IS 'Multi-tenant partition key';

-- ---------------------------------------------------------------------------
-- supply_chain_transfers
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS supply_chain_transfers (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_ref     TEXT        UNIQUE,
    from_node_id     UUID        REFERENCES supply_chain_nodes(id),
    to_node_id       UUID        REFERENCES supply_chain_nodes(id),
    commodity        TEXT        NOT NULL,
    quantity_kg      DOUBLE PRECISION        CHECK (quantity_kg > 0),
    grade            TEXT,
    transport_method TEXT        CHECK (transport_method IN ('truck', 'rail', 'boat', 'motorcycle', 'manual')),
    departure_at     TIMESTAMPTZ,
    arrival_at       TIMESTAMPTZ,
    status           TEXT        DEFAULT 'pending'
                     CHECK (status IN ('pending', 'in_transit', 'delivered', 'failed')),
    notes            TEXT,
    created_by       UUID,
    tenant_id        TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE supply_chain_transfers IS 'Commodity movement records between supply-chain nodes';
COMMENT ON COLUMN supply_chain_transfers.transfer_ref     IS 'Human-readable transfer reference number';
COMMENT ON COLUMN supply_chain_transfers.commodity        IS 'Agricultural product being transferred (e.g. maize, cocoa)';
COMMENT ON COLUMN supply_chain_transfers.quantity_kg      IS 'Net weight of the transferred commodity in kilograms';
COMMENT ON COLUMN supply_chain_transfers.transport_method IS 'Mode of transport used for this leg';
COMMENT ON COLUMN supply_chain_transfers.grade            IS 'Quality grade assigned at dispatch (A/B/C or custom)';

-- ---------------------------------------------------------------------------
-- quality_checks
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS quality_checks (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_id         UUID        NOT NULL REFERENCES supply_chain_transfers(id),
    inspector_id        UUID,
    grade               TEXT,
    moisture_pct        NUMERIC(5,2),
    foreign_matter_pct  NUMERIC(5,2),
    result              TEXT        CHECK (result IN ('pass', 'fail', 'conditional')),
    notes               TEXT,
    checked_at          TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE quality_checks IS 'Quality inspection results attached to supply-chain transfers';
COMMENT ON COLUMN quality_checks.inspector_id       IS 'UUID of the certified quality inspector';
COMMENT ON COLUMN quality_checks.moisture_pct       IS 'Moisture content percentage — critical for grain storage';
COMMENT ON COLUMN quality_checks.foreign_matter_pct IS 'Percentage of non-commodity material (stones, chaff, etc.)';
COMMENT ON COLUMN quality_checks.result             IS 'Pass = accepted; Fail = rejected; Conditional = accept with caveats';

-- ---------------------------------------------------------------------------
-- supply_chain_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS supply_chain_audit_log (
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

COMMENT ON TABLE supply_chain_audit_log IS 'Immutable audit trail for all supply-chain-service mutations';

CREATE RULE no_update_sc_audit AS ON UPDATE TO supply_chain_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_sc_audit AS ON DELETE TO supply_chain_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_sc_nodes_country         ON supply_chain_nodes(country);
CREATE INDEX IF NOT EXISTS idx_sc_transfers_from_node   ON supply_chain_transfers(from_node_id);
CREATE INDEX IF NOT EXISTS idx_sc_transfers_to_node     ON supply_chain_transfers(to_node_id);
CREATE INDEX IF NOT EXISTS idx_sc_transfers_status      ON supply_chain_transfers(status);
CREATE INDEX IF NOT EXISTS idx_quality_checks_transfer  ON quality_checks(transfer_id);
CREATE INDEX IF NOT EXISTS idx_sc_audit_entity          ON supply_chain_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_sc_audit_actor           ON supply_chain_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_sc_audit_actor;
-- DROP INDEX IF EXISTS idx_sc_audit_entity;
-- DROP INDEX IF EXISTS idx_quality_checks_transfer;
-- DROP INDEX IF EXISTS idx_sc_transfers_status;
-- DROP INDEX IF EXISTS idx_sc_transfers_to_node;
-- DROP INDEX IF EXISTS idx_sc_transfers_from_node;
-- DROP INDEX IF EXISTS idx_sc_nodes_country;
-- DROP RULE IF EXISTS no_delete_sc_audit ON supply_chain_audit_log;
-- DROP RULE IF EXISTS no_update_sc_audit ON supply_chain_audit_log;
-- DROP TABLE IF EXISTS supply_chain_audit_log;
-- DROP TABLE IF EXISTS quality_checks;
-- DROP TABLE IF EXISTS supply_chain_transfers;
-- DROP TABLE IF EXISTS supply_chain_nodes;
-- =============================================================================
