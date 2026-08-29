CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE letters_of_credit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lc_number VARCHAR(20) NOT NULL UNIQUE,
    lc_type VARCHAR(20) NOT NULL DEFAULT 'documentary',
    applicant_id UUID NOT NULL,
    applicant_name VARCHAR(200) NOT NULL,
    beneficiary_name VARCHAR(200) NOT NULL,
    issuing_bank VARCHAR(200) NOT NULL,
    advising_bank VARCHAR(200),
    amount NUMERIC(14,2) NOT NULL,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    expiry_date TIMESTAMPTZ NOT NULL,
    shipment_pol TEXT NOT NULL,
    shipment_pod TEXT NOT NULL,
    goods_description TEXT NOT NULL,
    documents_required JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    issued_at TIMESTAMPTZ,
    realized_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE financing_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_no VARCHAR(20) NOT NULL UNIQUE,
    applicant_id UUID NOT NULL,
    booking_id UUID,
    lc_id UUID REFERENCES letters_of_credit(id),
    requested_amount NUMERIC(14,2) NOT NULL,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    payment_terms VARCHAR(30) NOT NULL,
    interest_rate_pct NUMERIC(5,3) NOT NULL DEFAULT 0,
    interest_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_repayable NUMERIC(14,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approved_at TIMESTAMPTZ,
    disbursed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE trade_finance_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_tf_audit AS ON UPDATE TO trade_finance_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_tf_audit AS ON DELETE TO trade_finance_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_lc_applicant_id ON letters_of_credit(applicant_id);
CREATE INDEX idx_lc_status ON letters_of_credit(status);
CREATE INDEX idx_financing_applicant ON financing_requests(applicant_id);
CREATE INDEX idx_financing_status ON financing_requests(status);
