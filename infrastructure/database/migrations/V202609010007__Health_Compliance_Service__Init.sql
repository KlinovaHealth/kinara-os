-- Health Compliance Service — Regulatory compliance checks, violations, and summary reports
-- Database: kinara_compliance
-- Tracks regulatory check results, violations, resolution, and compliance rate over time

\c kinara_compliance;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS compliance_checks (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    check_type      TEXT          NOT NULL,
    entity_id       UUID          NOT NULL,
    entity_type     TEXT          NOT NULL,
    status          TEXT          NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('compliant','non_compliant','pending')),
    score           NUMERIC(5,2)  CHECK (score BETWEEN 0 AND 100),
    checked_by      UUID          NOT NULL,
    checked_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    next_check_due  TIMESTAMPTZ,
    notes           TEXT,
    tenant_id       TEXT          NOT NULL,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  compliance_checks             IS 'Individual regulatory and operational compliance check results per entity';
COMMENT ON COLUMN compliance_checks.check_type  IS 'Type of check: facility_license, staff_credential, infection_control, data_privacy, drug_storage';
COMMENT ON COLUMN compliance_checks.entity_type IS 'What was checked: facility, clinician, department, service';
COMMENT ON COLUMN compliance_checks.score       IS 'Percentage compliance score (0–100); null if check is binary pass/fail';
COMMENT ON COLUMN compliance_checks.next_check_due IS 'Scheduled date for the next mandatory review cycle';

CREATE INDEX IF NOT EXISTS idx_cc_entity    ON compliance_checks(entity_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_cc_status    ON compliance_checks(status);
CREATE INDEX IF NOT EXISTS idx_cc_tenant    ON compliance_checks(tenant_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_cc_due       ON compliance_checks(next_check_due) WHERE next_check_due IS NOT NULL;

CREATE TABLE IF NOT EXISTS compliance_violations (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id         UUID        NOT NULL,
    violation_code   TEXT        NOT NULL,
    description      TEXT        NOT NULL,
    severity         TEXT        NOT NULL
                     CHECK (severity IN ('low','medium','high','critical')),
    resolved_at      TIMESTAMPTZ,
    resolved_by      UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  compliance_violations               IS 'Individual violations identified during a compliance check';
COMMENT ON COLUMN compliance_violations.violation_code IS 'Regulatory code or internal policy reference for the violation';
COMMENT ON COLUMN compliance_violations.severity       IS 'Impact level: low, medium, high, critical — drives escalation rules';
COMMENT ON COLUMN compliance_violations.resolved_at    IS 'Timestamp when violation was remediated; null means open';

CREATE INDEX IF NOT EXISTS idx_cv_check     ON compliance_violations(check_id);
CREATE INDEX IF NOT EXISTS idx_cv_severity  ON compliance_violations(severity);
CREATE INDEX IF NOT EXISTS idx_cv_open      ON compliance_violations(resolved_at) WHERE resolved_at IS NULL;

CREATE TABLE IF NOT EXISTS compliance_reports (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    report_period    DATE          NOT NULL,
    total_checks     INT           NOT NULL DEFAULT 0,
    compliant_count  INT           NOT NULL DEFAULT 0,
    compliance_rate  NUMERIC(5,2)  CHECK (compliance_rate BETWEEN 0 AND 100),
    generated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    tenant_id        TEXT          NOT NULL
);

COMMENT ON TABLE  compliance_reports                IS 'Periodic compliance summary reports — one row per period per tenant';
COMMENT ON COLUMN compliance_reports.report_period  IS 'First day of the reporting period (month or quarter start)';
COMMENT ON COLUMN compliance_reports.compliance_rate IS 'Percentage of checks that returned compliant status';

CREATE INDEX IF NOT EXISTS idx_crep_tenant_period ON compliance_reports(tenant_id, report_period DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS health_compliance_audit_log (
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

COMMENT ON TABLE health_compliance_audit_log IS 'Append-only audit trail for compliance checks and violation records';

CREATE RULE no_update_health_compliance_audit AS ON UPDATE TO health_compliance_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_health_compliance_audit AS ON DELETE TO health_compliance_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_hca_audit_entity ON health_compliance_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_hca_audit_actor  ON health_compliance_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS health_compliance_audit_log CASCADE;
-- DROP TABLE IF EXISTS compliance_reports CASCADE;
-- DROP TABLE IF EXISTS compliance_violations CASCADE;
-- DROP TABLE IF EXISTS compliance_checks CASCADE;
