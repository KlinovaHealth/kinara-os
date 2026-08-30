-- =============================================================================
-- Kinara OS — Trade Finance Service
-- Migration : V202609010035__Trade_Finance_Service__Init.sql
-- Database  : kinara_trade_finance
-- Description: Initialises the Trade Finance Service schema: letters of credit,
--              trade guarantees, documentary collections, and an immutable audit log.
-- =============================================================================

\c kinara_trade_finance;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- letters_of_credit
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS letters_of_credit (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    lc_ref           TEXT           UNIQUE NOT NULL,
    applicant_id     UUID           NOT NULL,
    beneficiary_id   UUID           NOT NULL,
    issuing_bank     TEXT,
    advising_bank    TEXT,
    lc_type          TEXT           NOT NULL DEFAULT 'irrevocable'
                     CHECK (lc_type IN ('revocable','irrevocable','standby','transferable')),
    amount           NUMERIC(14,4)  NOT NULL,
    currency         TEXT           NOT NULL DEFAULT 'USD',
    commodity        TEXT,
    origin_country   TEXT,
    dest_country     TEXT,
    expiry_date      DATE,
    status           TEXT           NOT NULL DEFAULT 'draft'
                     CHECK (status IN (
                         'draft','issued','advised','presented',
                         'accepted','paid','expired','cancelled'
                     )),
    issued_at        TIMESTAMPTZ,
    paid_at          TIMESTAMPTZ,
    tenant_id        TEXT,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE letters_of_credit IS
    'Letters of credit issued or advised through Kinara OS trade finance. '
    'Supports revocable, irrevocable, standby, and transferable LC types '
    'denominated in any currency for intra-African and international trade.';

-- ---------------------------------------------------------------------------
-- trade_guarantees
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS trade_guarantees (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    guarantee_ref    TEXT           UNIQUE,
    applicant_id     UUID,
    beneficiary_id   UUID,
    guarantee_type   TEXT
                     CHECK (guarantee_type IN (
                         'performance','advance_payment','bid_bond','customs'
                     )),
    amount           NUMERIC(14,4),
    currency         TEXT           NOT NULL DEFAULT 'USD',
    issuing_bank     TEXT,
    validity_date    DATE,
    status           TEXT           NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active','called','expired','cancelled')),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE trade_guarantees IS
    'Bank-issued guarantees supporting trade transactions on Kinara OS. '
    'Covers performance bonds, advance payment guarantees, bid bonds, and customs bonds.';

-- ---------------------------------------------------------------------------
-- documentary_collections
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS documentary_collections (
    id                   UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_ref       TEXT           UNIQUE,
    exporter_id          UUID,
    importer_id          UUID,
    collection_type      TEXT
                         CHECK (collection_type IN ('D/P','D/A','clean')),
    amount               NUMERIC(14,4),
    currency             TEXT           NOT NULL DEFAULT 'USD',
    documents_received   BOOLEAN        NOT NULL DEFAULT false,
    status               TEXT           NOT NULL DEFAULT 'pending',
    presented_at         TIMESTAMPTZ,
    settled_at           TIMESTAMPTZ,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE documentary_collections IS
    'Documentary collection records (D/P, D/A, clean) managed via Kinara OS. '
    'Tracks document receipt, presentation to the importer bank, and final settlement.';

-- ---------------------------------------------------------------------------
-- trade_finance_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS trade_finance_audit_log (
    id              BIGSERIAL   PRIMARY KEY,
    entity_id       UUID,
    action          TEXT        NOT NULL,
    actor_id        TEXT        NOT NULL,
    old_data        JSONB,
    new_data        JSONB,
    signature_hash  TEXT,
    ip_address      INET,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE trade_finance_audit_log IS
    'Immutable append-only audit trail for all Trade Finance Service write operations. '
    'UPDATE and DELETE are blocked by rules to satisfy financial regulatory requirements.';

CREATE RULE no_update_trade_finance_audit AS ON UPDATE TO trade_finance_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_trade_finance_audit AS ON DELETE TO trade_finance_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_trade_finance_audit_entity
    ON trade_finance_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_trade_finance_audit_actor
    ON trade_finance_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_lc_applicant
    ON letters_of_credit(applicant_id);

CREATE INDEX IF NOT EXISTS idx_lc_status
    ON letters_of_credit(status);

CREATE INDEX IF NOT EXISTS idx_lc_expiry
    ON letters_of_credit(expiry_date);

CREATE INDEX IF NOT EXISTS idx_trade_guarantees_applicant
    ON trade_guarantees(applicant_id);

CREATE INDEX IF NOT EXISTS idx_documentary_collections_exporter
    ON documentary_collections(exporter_id);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS trade_finance_audit_log CASCADE;
-- DROP TABLE IF EXISTS documentary_collections CASCADE;
-- DROP TABLE IF EXISTS trade_guarantees CASCADE;
-- DROP TABLE IF EXISTS letters_of_credit CASCADE;
