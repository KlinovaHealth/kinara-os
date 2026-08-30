-- =============================================================================
-- Kinara OS — Payment Service
-- Migration : V202609010040__Payment_Service__Init.sql
-- Database  : kinara_payment
-- Description: Initialises the Payment Service schema: multi-currency wallets,
--              double-entry transactions, FX rate table with seed data,
--              and an immutable financial audit log.
--
-- CRITICAL: The payment_audit_log is a financial record. Its immutability
--           rules (no_update / no_delete) must never be dropped.
-- =============================================================================

\c kinara_payment;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- wallets
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS wallets (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id         UUID           NOT NULL,
    owner_type       TEXT           NOT NULL
                     CHECK (owner_type IN (
                         'patient','farmer','cooperative','clinic','merchant'
                     )),
    currency         TEXT           NOT NULL DEFAULT 'USD',
    balance          NUMERIC(18,4)  NOT NULL DEFAULT 0
                     CHECK (balance >= 0),
    reserved_amount  NUMERIC(18,4)  NOT NULL DEFAULT 0
                     CHECK (reserved_amount >= 0),
    status           TEXT           NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active','suspended','closed')),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (owner_id, currency)
);

COMMENT ON TABLE wallets IS
    'Multi-currency digital wallets for all Kinara OS participants. '
    'Each owner may hold one wallet per currency. The balance constraint enforces '
    'non-negativity at the database layer; application logic must also hold '
    'reserved_amount before deducting committed funds.';

-- ---------------------------------------------------------------------------
-- transactions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transactions (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    txn_ref          TEXT           UNIQUE NOT NULL,
    wallet_id        UUID           REFERENCES wallets(id),
    txn_type         TEXT           NOT NULL
                     CHECK (txn_type IN (
                         'credit','debit','transfer','refund','fee','reversal'
                     )),
    amount           NUMERIC(18,4)  NOT NULL
                     CHECK (amount > 0),
    currency         TEXT           NOT NULL,
    balance_before   NUMERIC(18,4)  NOT NULL,
    balance_after    NUMERIC(18,4)  NOT NULL,
    reference_type   TEXT
                     CHECK (reference_type IN (
                         'appointment','lab_order','prescription',
                         'shipment','telemedicine','loan'
                     )),
    reference_id     UUID,
    description      TEXT,
    status           TEXT           NOT NULL DEFAULT 'completed'
                     CHECK (status IN ('pending','completed','failed','reversed')),
    initiated_by     UUID,
    tenant_id        TEXT,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transactions IS
    'Immutable ledger of all wallet movements across the Kinara OS platform. '
    'balance_before and balance_after provide a running balance trail that enables '
    'reconciliation without relying solely on the audit log. '
    'Rows must never be updated or deleted after creation.';

-- ---------------------------------------------------------------------------
-- currency_rates
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS currency_rates (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    from_currency  TEXT           NOT NULL,
    to_currency    TEXT           NOT NULL,
    rate           NUMERIC(18,8)  NOT NULL
                   CHECK (rate > 0),
    source         TEXT           NOT NULL DEFAULT 'kinara',
    effective_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (from_currency, to_currency)
);

COMMENT ON TABLE currency_rates IS
    'Spot FX rates used by the Kinara OS payment engine for cross-currency transactions. '
    'The UNIQUE constraint on (from_currency, to_currency) means each pair has one '
    'live rate; historical rates should be archived before upserting a new rate. '
    'Seed data covers the principal African and global currencies used on the platform.';

-- Seed FX rates (USD as base)
INSERT INTO currency_rates (from_currency, to_currency, rate, source) VALUES
    ('USD', 'XOF',  600.00000000,   'kinara_seed'),
    ('USD', 'GHS',   14.00000000,   'kinara_seed'),
    ('USD', 'KES',  130.00000000,   'kinara_seed'),
    ('USD', 'NGN', 1500.00000000,   'kinara_seed'),
    ('USD', 'ETB',   56.00000000,   'kinara_seed'),
    ('USD', 'TZS', 2600.00000000,   'kinara_seed'),
    ('USD', 'RWF', 1300.00000000,   'kinara_seed'),
    ('USD', 'EUR',    0.93000000,   'kinara_seed'),
    ('USD', 'GBP',    0.80000000,   'kinara_seed')
ON CONFLICT (from_currency, to_currency) DO NOTHING;

-- ---------------------------------------------------------------------------
-- payment_audit_log  (immutable — CRITICAL financial record)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_audit_log (
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

COMMENT ON TABLE payment_audit_log IS
    'CRITICAL — Immutable financial audit trail for all Payment Service write operations. '
    'Satisfies PCI-DSS and Central Bank of West Africa (BCEAO) record-keeping requirements. '
    'UPDATE and DELETE are permanently blocked by rules. '
    'Any attempt to drop these rules must go through a security change-control process.';

CREATE RULE no_update_payment_audit AS ON UPDATE TO payment_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_payment_audit AS ON DELETE TO payment_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_payment_audit_entity
    ON payment_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_payment_audit_actor
    ON payment_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_wallets_owner
    ON wallets(owner_id);

CREATE INDEX IF NOT EXISTS idx_transactions_wallet_time
    ON transactions(wallet_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_reference
    ON transactions(reference_id);

CREATE INDEX IF NOT EXISTS idx_transactions_type_time
    ON transactions(txn_type, created_at);

CREATE INDEX IF NOT EXISTS idx_currency_rates_pair
    ON currency_rates(from_currency, to_currency);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- NOTE: Dropping these tables removes financial records. Ensure regulatory
--       sign-off before executing in any non-development environment.
-- DROP TABLE IF EXISTS payment_audit_log CASCADE;
-- DROP TABLE IF EXISTS currency_rates CASCADE;
-- DROP TABLE IF EXISTS transactions CASCADE;
-- DROP TABLE IF EXISTS wallets CASCADE;
