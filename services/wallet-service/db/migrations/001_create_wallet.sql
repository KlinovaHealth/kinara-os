-- Wallet service: multi-currency balance tracking and reconciliation.

CREATE TABLE IF NOT EXISTS wallet_balances (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    currency   TEXT NOT NULL,
    balance    NUMERIC(18,8) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, currency)
);

CREATE TABLE IF NOT EXISTS reconciliation_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    is_balanced     BOOLEAN NOT NULL DEFAULT true,
    discrepancy_usd NUMERIC(18,8) NOT NULL DEFAULT 0,
    detail          TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_recon_logs AS ON UPDATE TO reconciliation_logs DO INSTEAD NOTHING;
CREATE RULE no_delete_recon_logs AS ON DELETE TO reconciliation_logs DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_wallet_balances_user ON wallet_balances(user_id);
CREATE INDEX IF NOT EXISTS idx_recon_logs_user      ON reconciliation_logs(user_id);
