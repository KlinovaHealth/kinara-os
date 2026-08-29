-- Farmer finance service schema
-- Income records and savings transactions are immutable.

CREATE TABLE IF NOT EXISTS income_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id   UUID NOT NULL,
    source      TEXT NOT NULL DEFAULT 'crop_sale',
    amount      NUMERIC(14,4) NOT NULL,
    currency    TEXT NOT NULL DEFAULT 'XOF',
    description TEXT NOT NULL DEFAULT '',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_income AS ON UPDATE TO income_records DO INSTEAD NOTHING;
CREATE RULE no_delete_income AS ON DELETE TO income_records DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS loans (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_ref         TEXT NOT NULL UNIQUE,
    farmer_id        UUID NOT NULL,
    principal_amount NUMERIC(14,4) NOT NULL,
    interest_rate    NUMERIC(5,2) NOT NULL DEFAULT 5.0,
    currency         TEXT NOT NULL DEFAULT 'XOF',
    status           TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','approved','disbursed','repaid','defaulted')),
    due_date         TIMESTAMPTZ NOT NULL,
    disbursed_at     TIMESTAMPTZ,
    repaid_at        TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS savings_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id       UUID NOT NULL UNIQUE,
    balance         NUMERIC(14,4) NOT NULL DEFAULT 0,
    currency        TEXT NOT NULL DEFAULT 'XOF',
    total_saved     NUMERIC(14,4) NOT NULL DEFAULT 0,
    total_withdrawn NUMERIC(14,4) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS savings_transactions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES savings_accounts(id),
    farmer_id  UUID NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('credit','debit')),
    amount     NUMERIC(14,4) NOT NULL,
    balance    NUMERIC(14,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_savings_txn AS ON UPDATE TO savings_transactions DO INSTEAD NOTHING;
CREATE RULE no_delete_savings_txn AS ON DELETE TO savings_transactions DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_income_farmer    ON income_records(farmer_id);
CREATE INDEX IF NOT EXISTS idx_income_recorded  ON income_records(recorded_at);
CREATE INDEX IF NOT EXISTS idx_loans_farmer     ON loans(farmer_id);
CREATE INDEX IF NOT EXISTS idx_loans_status     ON loans(status);
CREATE INDEX IF NOT EXISTS idx_savings_farmer   ON savings_accounts(farmer_id);
