CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    owner_type VARCHAR(50) NOT NULL,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    balance NUMERIC(18,4) NOT NULL DEFAULT 0,
    reserved_amount NUMERIC(18,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, currency)
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    txn_ref VARCHAR(25) NOT NULL UNIQUE,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    txn_type VARCHAR(20) NOT NULL,
    amount NUMERIC(18,4) NOT NULL,
    currency VARCHAR(5) NOT NULL,
    balance_before NUMERIC(18,4) NOT NULL,
    balance_after NUMERIC(18,4) NOT NULL,
    reference_type VARCHAR(50),
    reference_id VARCHAR(100),
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE currency_conversions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_wallet_id UUID NOT NULL REFERENCES wallets(id),
    to_wallet_id UUID NOT NULL REFERENCES wallets(id),
    from_currency VARCHAR(5) NOT NULL,
    to_currency VARCHAR(5) NOT NULL,
    from_amount NUMERIC(18,4) NOT NULL,
    to_amount NUMERIC(18,4) NOT NULL,
    exchange_rate NUMERIC(14,6) NOT NULL,
    fee_amount NUMERIC(18,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE settlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    settlement_ref VARCHAR(25) NOT NULL UNIQUE,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    amount NUMERIC(18,4) NOT NULL,
    currency VARCHAR(5) NOT NULL,
    bank_account_no VARCHAR(50),
    bank_code VARCHAR(20),
    mobile_money_no VARCHAR(30),
    provider VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payment_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_txn AS ON UPDATE TO transactions DO INSTEAD NOTHING;
CREATE RULE no_delete_txn AS ON DELETE TO transactions DO INSTEAD NOTHING;
CREATE RULE no_update_conv AS ON UPDATE TO currency_conversions DO INSTEAD NOTHING;
CREATE RULE no_delete_conv AS ON DELETE TO currency_conversions DO INSTEAD NOTHING;
CREATE RULE no_update_payment_audit AS ON UPDATE TO payment_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_payment_audit AS ON DELETE TO payment_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_wallets_owner ON wallets(owner_id);
CREATE INDEX idx_transactions_wallet ON transactions(wallet_id);
CREATE INDEX idx_settlements_wallet ON settlements(wallet_id);
