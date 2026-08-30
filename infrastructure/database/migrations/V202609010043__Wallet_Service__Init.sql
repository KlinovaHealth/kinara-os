-- =============================================================================
-- Kinara OS — Wallet Service (extended wallet features)
-- Migration : V202609010043__Wallet_Service__Init.sql
-- Database  : kinara_payment
--
-- Wallet Service: Extended wallet features for Kinara OS payment pillar.
-- Shares the kinara_payment database with payment-service (V202609010040).
-- Tables in this migration reference the `wallets` table created in V40.
-- Run V202609010040 BEFORE this migration.
-- =============================================================================

\c kinara_payment;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- wallet_limits
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS wallet_limits (
    id              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id       UUID           NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    limit_type      TEXT           NOT NULL
                    CHECK (limit_type IN (
                        'daily_debit','daily_credit','single_txn','monthly'
                    )),
    limit_amount    NUMERIC(18,4)  NOT NULL,
    currency        TEXT           NOT NULL DEFAULT 'USD',
    effective_from  DATE           NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE wallet_limits IS
    'Configurable spending and receiving limits applied to individual wallets. '
    'Central bank regulations for e-money in ECOWAS member states require transaction '
    'and daily volume ceilings; this table makes those limits auditable and adjustable '
    'without code changes. Multiple limit types may coexist per wallet.';

-- ---------------------------------------------------------------------------
-- wallet_beneficiaries
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS wallet_beneficiaries (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id             UUID        NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    beneficiary_name      TEXT        NOT NULL,
    beneficiary_account   TEXT        NOT NULL,
    beneficiary_type      TEXT
                          CHECK (beneficiary_type IN (
                              'mobile_money','bank','cooperative','wallet'
                          )),
    is_verified           BOOLEAN     NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE wallet_beneficiaries IS
    'Saved beneficiaries for outbound transfers from a wallet. '
    'Verification status (is_verified) gates whether a beneficiary can receive '
    'funds above the single-transaction limit defined in wallet_limits.';

-- ---------------------------------------------------------------------------
-- mobile_money_transfers
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mobile_money_transfers (
    id                  UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    txn_ref             TEXT           UNIQUE,
    sender_wallet_id    UUID           REFERENCES wallets(id),
    recipient_msisdn    TEXT           NOT NULL,
    recipient_name      TEXT,
    amount              NUMERIC(14,4)  NOT NULL,
    currency            TEXT           NOT NULL DEFAULT 'XOF',
    provider            TEXT
                        CHECK (provider IN ('mtn','orange','airtel','mpesa','wave')),
    status              TEXT           NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','processing','success','failed')),
    provider_ref        TEXT,
    initiated_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ,
    tenant_id           TEXT
);

COMMENT ON TABLE mobile_money_transfers IS
    'Mobile money push transfers originating from a Kinara OS wallet to an external MSISDN. '
    'Covers the five dominant African mobile money providers: MTN, Orange, Airtel, '
    'M-Pesa, and Wave. provider_ref stores the operator''s confirmation reference for '
    'reconciliation and dispute resolution.';

-- ---------------------------------------------------------------------------
-- wallet_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS wallet_audit_log (
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

COMMENT ON TABLE wallet_audit_log IS
    'Immutable append-only audit trail for Wallet Service write operations. '
    'Supplements the payment_audit_log (V40) with wallet-extension-specific events. '
    'UPDATE and DELETE are blocked by rules to satisfy financial record-keeping obligations.';

CREATE RULE no_update_wallet_audit AS ON UPDATE TO wallet_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_wallet_audit AS ON DELETE TO wallet_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_wallet_audit_entity
    ON wallet_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_wallet_audit_actor
    ON wallet_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_wallet_limits_wallet
    ON wallet_limits(wallet_id);

CREATE INDEX IF NOT EXISTS idx_wallet_beneficiaries_wallet
    ON wallet_beneficiaries(wallet_id);

CREATE INDEX IF NOT EXISTS idx_mobile_money_sender
    ON mobile_money_transfers(sender_wallet_id);

CREATE INDEX IF NOT EXISTS idx_mobile_money_msisdn
    ON mobile_money_transfers(recipient_msisdn);

CREATE INDEX IF NOT EXISTS idx_mobile_money_status
    ON mobile_money_transfers(status, initiated_at);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS wallet_audit_log CASCADE;
-- DROP TABLE IF EXISTS mobile_money_transfers CASCADE;
-- DROP TABLE IF EXISTS wallet_beneficiaries CASCADE;
-- DROP TABLE IF EXISTS wallet_limits CASCADE;
