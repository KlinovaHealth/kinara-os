-- =============================================================================
-- V202609010022__Farmer_Finance_Service__Init.sql
-- Kinara OS — Farmer Finance Service
-- NOTE: kinara_farmer_finance may not have been created in V001 if the
--       bootstrap script only provisioned the original four core databases.
--       Ensure the DBA or infra pipeline creates this database before running
--       this migration: CREATE DATABASE kinara_farmer_finance;
-- =============================================================================
\c kinara_farmer_finance;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- farmer_loans
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS farmer_loans (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_ref          TEXT        UNIQUE NOT NULL,
    farmer_id         UUID        NOT NULL,
    cooperative_id    UUID,
    loan_type         TEXT        CHECK (loan_type IN ('input', 'equipment', 'emergency', 'seasonal')),
    amount            NUMERIC(14,4) NOT NULL,
    currency          TEXT        DEFAULT 'XOF',
    interest_rate_pct NUMERIC(5,2) DEFAULT 0,
    term_months       INT,
    status            TEXT        DEFAULT 'pending'
                      CHECK (status IN ('pending', 'approved', 'disbursed', 'active', 'paid', 'defaulted', 'rejected')),
    disbursed_at      TIMESTAMPTZ,
    due_date          DATE,
    notes             TEXT,
    approved_by       UUID,
    tenant_id         TEXT,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE farmer_loans IS 'Loan origination records for agricultural finance extended to farmers';
COMMENT ON COLUMN farmer_loans.loan_ref          IS 'Human-readable unique loan reference (e.g. LOAN-2026-00001)';
COMMENT ON COLUMN farmer_loans.farmer_id         IS 'Cross-DB reference to farmers.id in kinara_farmer';
COMMENT ON COLUMN farmer_loans.cooperative_id    IS 'Optional cross-DB reference to cooperatives.id in kinara_cooperative';
COMMENT ON COLUMN farmer_loans.loan_type         IS 'Purpose classification — input (seeds/fertiliser), equipment, emergency, seasonal';
COMMENT ON COLUMN farmer_loans.interest_rate_pct IS 'Annual interest rate as a percentage';
COMMENT ON COLUMN farmer_loans.term_months       IS 'Loan repayment term in calendar months';
COMMENT ON COLUMN farmer_loans.tenant_id         IS 'Multi-tenant partition key (MFI or cooperative tenant)';

-- ---------------------------------------------------------------------------
-- loan_repayments
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS loan_repayments (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_id         UUID        NOT NULL REFERENCES farmer_loans(id),
    amount          NUMERIC(14,4) NOT NULL,
    currency        TEXT        DEFAULT 'XOF',
    payment_method  TEXT        CHECK (payment_method IN ('mobile_money', 'bank', 'cash', 'cooperative')),
    reference_no    TEXT,
    paid_at         TIMESTAMPTZ DEFAULT NOW(),
    recorded_by     UUID
);

COMMENT ON TABLE loan_repayments IS 'Individual repayment instalments recorded against farmer loans';
COMMENT ON COLUMN loan_repayments.payment_method IS 'Channel through which the repayment was made';
COMMENT ON COLUMN loan_repayments.reference_no   IS 'External payment reference (mobile-money TxID, bank ref, etc.)';
COMMENT ON COLUMN loan_repayments.recorded_by    IS 'UUID of the field agent or system that logged the repayment';

-- ---------------------------------------------------------------------------
-- farmer_income_records
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS farmer_income_records (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id    UUID        NOT NULL,
    source       TEXT        CHECK (source IN ('crop_sale', 'livestock', 'cooperative_dividend', 'remittance', 'other')),
    amount       NUMERIC(14,4),
    currency     TEXT        DEFAULT 'XOF',
    income_date  DATE,
    description  TEXT,
    recorded_at  TIMESTAMPTZ DEFAULT NOW(),
    tenant_id    TEXT
);

COMMENT ON TABLE farmer_income_records IS 'Self-reported and system-derived income records for farmer financial profiles';
COMMENT ON COLUMN farmer_income_records.farmer_id   IS 'Cross-DB reference to farmers.id in kinara_farmer';
COMMENT ON COLUMN farmer_income_records.source      IS 'Income stream classification for credit-scoring and analytics';
COMMENT ON COLUMN farmer_income_records.income_date IS 'Calendar date the income was received';
COMMENT ON COLUMN farmer_income_records.tenant_id   IS 'Multi-tenant partition key';

-- ---------------------------------------------------------------------------
-- farmer_finance_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS farmer_finance_audit_log (
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

COMMENT ON TABLE farmer_finance_audit_log IS 'Immutable audit trail for all farmer-finance-service mutations';

CREATE RULE no_update_ff_audit AS ON UPDATE TO farmer_finance_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_ff_audit AS ON DELETE TO farmer_finance_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_farmer_loans_farmer      ON farmer_loans(farmer_id);
CREATE INDEX IF NOT EXISTS idx_farmer_loans_status      ON farmer_loans(status);
CREATE INDEX IF NOT EXISTS idx_loan_repayments_loan     ON loan_repayments(loan_id);
CREATE INDEX IF NOT EXISTS idx_farmer_income_farmer_date ON farmer_income_records(farmer_id, income_date);
CREATE INDEX IF NOT EXISTS idx_ff_audit_entity          ON farmer_finance_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_ff_audit_actor           ON farmer_finance_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_ff_audit_actor;
-- DROP INDEX IF EXISTS idx_ff_audit_entity;
-- DROP INDEX IF EXISTS idx_farmer_income_farmer_date;
-- DROP INDEX IF EXISTS idx_loan_repayments_loan;
-- DROP INDEX IF EXISTS idx_farmer_loans_status;
-- DROP INDEX IF EXISTS idx_farmer_loans_farmer;
-- DROP RULE IF EXISTS no_delete_ff_audit ON farmer_finance_audit_log;
-- DROP RULE IF EXISTS no_update_ff_audit ON farmer_finance_audit_log;
-- DROP TABLE IF EXISTS farmer_finance_audit_log;
-- DROP TABLE IF EXISTS farmer_income_records;
-- DROP TABLE IF EXISTS loan_repayments;
-- DROP TABLE IF EXISTS farmer_loans;
-- =============================================================================
