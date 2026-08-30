-- Governance Service — Organizational policies, reviews, and board decisions
-- Database: kinara_governance
-- Stores the full policy lifecycle, scheduled review records, and authoritative decision register

\c kinara_governance;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS governance_policies (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_code     TEXT        NOT NULL UNIQUE,
    title           TEXT        NOT NULL,
    description     TEXT,
    version         INT         NOT NULL DEFAULT 1,
    status          TEXT        NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','active','deprecated')),
    effective_date  DATE,
    expiry_date     DATE,
    created_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  governance_policies             IS 'Versioned organizational policy register — each new version is a new row';
COMMENT ON COLUMN governance_policies.policy_code IS 'Human-readable unique code, e.g. GOV-001, HR-LEAVE-002';
COMMENT ON COLUMN governance_policies.version     IS 'Sequential version number; increment on substantive amendments';
COMMENT ON COLUMN governance_policies.status      IS 'Lifecycle: draft (under review), active (in force), deprecated (superseded)';
COMMENT ON COLUMN governance_policies.expiry_date IS 'Mandatory review date; null means indefinite until next amendment';

CREATE INDEX IF NOT EXISTS idx_gov_policy_status ON governance_policies(status);
CREATE INDEX IF NOT EXISTS idx_gov_policy_code   ON governance_policies(policy_code);
CREATE INDEX IF NOT EXISTS idx_gov_policy_expiry ON governance_policies(expiry_date) WHERE expiry_date IS NOT NULL;

CREATE TABLE IF NOT EXISTS policy_reviews (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id    UUID        NOT NULL REFERENCES governance_policies(id),
    reviewer_id  UUID        NOT NULL,
    review_date  DATE        NOT NULL,
    outcome      TEXT,
    comments     TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  policy_reviews            IS 'Scheduled and ad-hoc policy review records — linked to active policy version';
COMMENT ON COLUMN policy_reviews.outcome    IS 'Result: approved, approved_with_changes, deferred, deprecated';
COMMENT ON COLUMN policy_reviews.review_date IS 'Date the review was conducted (not necessarily when record was created)';

CREATE INDEX IF NOT EXISTS idx_pr_policy  ON policy_reviews(policy_id, review_date DESC);
CREATE INDEX IF NOT EXISTS idx_pr_reviewer ON policy_reviews(reviewer_id);

CREATE TABLE IF NOT EXISTS governance_decisions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_ref   TEXT        NOT NULL UNIQUE,
    title          TEXT        NOT NULL,
    description    TEXT,
    decision_type  TEXT        NOT NULL,
    status         TEXT        NOT NULL DEFAULT 'proposed'
                   CHECK (status IN ('proposed','approved','rejected','implemented')),
    approved_by    UUID,
    approved_at    TIMESTAMPTZ,
    tenant_id      TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  governance_decisions              IS 'Board and committee decision register — authoritative record of formal decisions';
COMMENT ON COLUMN governance_decisions.decision_ref IS 'Unique decision reference, e.g. BOARD-2026-001';
COMMENT ON COLUMN governance_decisions.decision_type IS 'Category: strategic, operational, financial, clinical, regulatory';
COMMENT ON COLUMN governance_decisions.status        IS 'Workflow state: proposed → approved/rejected → implemented';
COMMENT ON COLUMN governance_decisions.approved_by   IS 'UUID of the authorizing officer or committee chair';

CREATE INDEX IF NOT EXISTS idx_gd_status  ON governance_decisions(status);
CREATE INDEX IF NOT EXISTS idx_gd_tenant  ON governance_decisions(tenant_id, created_at DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS governance_audit_log (
    id             BIGSERIAL   PRIMARY KEY,
    entity_id      UUID        NOT NULL,
    action         TEXT        NOT NULL,  -- 'create','update','delete','read'
    actor_id       TEXT        NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    signature_hash TEXT,
    ip_address     INET,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE governance_audit_log IS 'Append-only audit trail for policies, reviews, and governance decisions';

CREATE RULE no_update_governance_audit AS ON UPDATE TO governance_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_governance_audit AS ON DELETE TO governance_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_governance_audit_entity ON governance_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_governance_audit_actor  ON governance_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS governance_audit_log CASCADE;
-- DROP TABLE IF EXISTS governance_decisions CASCADE;
-- DROP TABLE IF EXISTS policy_reviews CASCADE;
-- DROP TABLE IF EXISTS governance_policies CASCADE;
