-- =============================================================================
-- V202609010020__Cooperative_Service__Init.sql
-- Kinara OS — Cooperative Service
-- =============================================================================
\c kinara_cooperative;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- cooperatives
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cooperatives (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT        NOT NULL,
    registration_no  TEXT        UNIQUE NOT NULL,
    coop_type        TEXT        DEFAULT 'multi_purpose'
                     CHECK (coop_type IN ('production', 'marketing', 'credit', 'multi_purpose')),
    status           TEXT        DEFAULT 'active'
                     CHECK (status IN ('active', 'suspended', 'dissolved')),
    country          TEXT        NOT NULL,
    region           TEXT        DEFAULT '',
    district         TEXT        DEFAULT '',
    total_members    INT         DEFAULT 0 CHECK (total_members >= 0),
    total_farm_ha    DOUBLE PRECISION DEFAULT 0,
    description      TEXT        DEFAULT '',
    contact_phone    TEXT        DEFAULT '',
    contact_email    TEXT        DEFAULT '',
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE cooperatives IS 'Registry of farmer cooperatives and collective organisations';
COMMENT ON COLUMN cooperatives.registration_no IS 'Government-issued cooperative registration number — unique per country';
COMMENT ON COLUMN cooperatives.coop_type       IS 'Operational classification of the cooperative';
COMMENT ON COLUMN cooperatives.total_members   IS 'Denormalised member count — updated by trigger or service layer';
COMMENT ON COLUMN cooperatives.total_farm_ha   IS 'Aggregate farmland (ha) across all active members';

-- ---------------------------------------------------------------------------
-- cooperative_members
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cooperative_members (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    cooperative_id  UUID        NOT NULL REFERENCES cooperatives(id) ON DELETE CASCADE,
    farmer_id       UUID        NOT NULL,
    role            TEXT        DEFAULT 'member'
                    CHECK (role IN ('member', 'officer', 'president', 'treasurer', 'secretary')),
    joined_at       TIMESTAMPTZ DEFAULT NOW(),
    left_at         TIMESTAMPTZ,
    share_count     INT         DEFAULT 1,
    status          TEXT        DEFAULT 'active'
                    CHECK (status IN ('active', 'suspended', 'left'))
);

COMMENT ON TABLE cooperative_members IS 'Membership roster linking farmers to their cooperatives';
COMMENT ON COLUMN cooperative_members.farmer_id   IS 'Cross-DB reference to farmers.id in kinara_farmer';
COMMENT ON COLUMN cooperative_members.role        IS 'Governance role within the cooperative';
COMMENT ON COLUMN cooperative_members.share_count IS 'Number of cooperative shares held by the member';

-- ---------------------------------------------------------------------------
-- cooperative_transactions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cooperative_transactions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    cooperative_id  UUID        NOT NULL REFERENCES cooperatives(id),
    txn_type        TEXT        CHECK (txn_type IN ('contribution', 'dividend', 'loan', 'repayment', 'expense')),
    amount          NUMERIC(14,4) NOT NULL,
    currency        TEXT        DEFAULT 'XOF',
    description     TEXT,
    reference_id    UUID,
    performed_by    UUID,
    occurred_at     TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE cooperative_transactions IS 'Financial transaction ledger for cooperative funds';
COMMENT ON COLUMN cooperative_transactions.currency     IS 'ISO 4217 currency code; defaults to West African CFA franc';
COMMENT ON COLUMN cooperative_transactions.reference_id IS 'Optional FK to external entity (loan_id, order_id, etc.)';
COMMENT ON COLUMN cooperative_transactions.performed_by IS 'User ID of the treasurer or officer who recorded the transaction';

-- ---------------------------------------------------------------------------
-- cooperative_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cooperative_audit_log (
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

COMMENT ON TABLE cooperative_audit_log IS 'Immutable audit trail for all cooperative-service mutations';

CREATE RULE no_update_cooperative_audit AS ON UPDATE TO cooperative_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_cooperative_audit AS ON DELETE TO cooperative_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_cooperatives_country_status  ON cooperatives(country, status);
CREATE INDEX IF NOT EXISTS idx_coop_members_cooperative     ON cooperative_members(cooperative_id);
CREATE INDEX IF NOT EXISTS idx_coop_members_farmer          ON cooperative_members(farmer_id);
CREATE INDEX IF NOT EXISTS idx_coop_txns_coop_date          ON cooperative_transactions(cooperative_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_cooperative_audit_entity     ON cooperative_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_cooperative_audit_actor      ON cooperative_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_cooperative_audit_actor;
-- DROP INDEX IF EXISTS idx_cooperative_audit_entity;
-- DROP INDEX IF EXISTS idx_coop_txns_coop_date;
-- DROP INDEX IF EXISTS idx_coop_members_farmer;
-- DROP INDEX IF EXISTS idx_coop_members_cooperative;
-- DROP INDEX IF EXISTS idx_cooperatives_country_status;
-- DROP RULE IF EXISTS no_delete_cooperative_audit ON cooperative_audit_log;
-- DROP RULE IF EXISTS no_update_cooperative_audit ON cooperative_audit_log;
-- DROP TABLE IF EXISTS cooperative_audit_log;
-- DROP TABLE IF EXISTS cooperative_transactions;
-- DROP TABLE IF EXISTS cooperative_members;
-- DROP TABLE IF EXISTS cooperatives;
-- =============================================================================
