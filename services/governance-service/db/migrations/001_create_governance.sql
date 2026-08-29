-- Kinara Governance OS — Governance Service schema
-- Handles compliance reporting, epidemiology tracking, coordination rules, alerts.
-- Audit log is immutable via PostgreSQL rules.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── Compliance Reports ───────────────────────────────────────────────────────

CREATE TABLE compliance_reports (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    facility_id          UUID,
    ministry_id          UUID,
    report_type          TEXT         NOT NULL
                                      CHECK (report_type IN ('epidemiology','compliance','disease_burden','outbreak','mortality','immunization')),
    frequency            TEXT         NOT NULL
                                      CHECK (frequency IN ('daily','weekly','monthly','quarterly','annual','on_demand')),
    period_start         TIMESTAMPTZ  NOT NULL,
    period_end           TIMESTAMPTZ  NOT NULL,
    status               TEXT         NOT NULL DEFAULT 'pending'
                                      CHECK (status IN ('pending','compliant','violation','exempted')),
    country              TEXT         NOT NULL,
    region               TEXT,
    summary              TEXT         NOT NULL,
    data_payload_enc     TEXT         NOT NULL,
    submitted_by         UUID         NOT NULL,
    submitted_at         TIMESTAMPTZ,
    reviewed_by          UUID,
    reviewed_at          TIMESTAMPTZ,
    violation_notes_enc  TEXT,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_compliance_reports_country     ON compliance_reports(country);
CREATE INDEX idx_compliance_reports_status      ON compliance_reports(status);
CREATE INDEX idx_compliance_reports_report_type ON compliance_reports(report_type);
CREATE INDEX idx_compliance_reports_period      ON compliance_reports(period_start, period_end);
CREATE INDEX idx_compliance_reports_facility    ON compliance_reports(facility_id);
CREATE INDEX idx_compliance_reports_ministry    ON compliance_reports(ministry_id);

-- ─── Epidemiology Records ─────────────────────────────────────────────────────
-- No PHI — aggregate counts only.

CREATE TABLE epidemiology_records (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    icd10_code      TEXT         NOT NULL,
    icd10_description TEXT       NOT NULL,
    country         TEXT         NOT NULL,
    region          TEXT,
    district        TEXT,
    case_count      INT          NOT NULL DEFAULT 0,
    death_count     INT          NOT NULL DEFAULT 0,
    recovered_count INT          NOT NULL DEFAULT 0,
    period_start    TIMESTAMPTZ  NOT NULL,
    period_end      TIMESTAMPTZ  NOT NULL,
    age_group       TEXT,
    gender          TEXT         CHECK (gender IN ('male','female','all')),
    facility_id     UUID,
    reported_by     UUID         NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_epi_country      ON epidemiology_records(country);
CREATE INDEX idx_epi_icd10        ON epidemiology_records(icd10_code);
CREATE INDEX idx_epi_period       ON epidemiology_records(period_start, period_end);
CREATE INDEX idx_epi_case_count   ON epidemiology_records(case_count DESC);
CREATE INDEX idx_epi_facility     ON epidemiology_records(facility_id);

-- ─── Coordination Rules ───────────────────────────────────────────────────────

CREATE TABLE coordination_rules (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_type        TEXT        NOT NULL
                                 CHECK (rule_type IN ('reporting_threshold','outbreak_threshold','data_retention','access_policy','notification')),
    name             TEXT        NOT NULL,
    description      TEXT        NOT NULL,
    country          TEXT        NOT NULL,
    region           TEXT,
    parameters_enc   TEXT        NOT NULL,
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    effective_from   TIMESTAMPTZ NOT NULL,
    effective_until  TIMESTAMPTZ,
    created_by       UUID        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rules_country     ON coordination_rules(country);
CREATE INDEX idx_rules_rule_type   ON coordination_rules(rule_type);
CREATE INDEX idx_rules_is_active   ON coordination_rules(is_active);

-- ─── Governance Alerts ────────────────────────────────────────────────────────

CREATE TABLE governance_alerts (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id          UUID        REFERENCES coordination_rules(id),
    severity         TEXT        NOT NULL CHECK (severity IN ('info','warning','critical')),
    status           TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open','acknowledged','resolved')),
    title            TEXT        NOT NULL,
    description      TEXT        NOT NULL,
    country          TEXT        NOT NULL,
    region           TEXT,
    metadata_enc     TEXT,
    raised_by        UUID        NOT NULL,
    acknowledged_by  UUID,
    resolved_by      UUID,
    resolved_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_country    ON governance_alerts(country);
CREATE INDEX idx_alerts_severity   ON governance_alerts(severity);
CREATE INDEX idx_alerts_status     ON governance_alerts(status);
CREATE INDEX idx_alerts_rule_id    ON governance_alerts(rule_id);
CREATE INDEX idx_alerts_created_at ON governance_alerts(created_at DESC);

-- ─── Governance Audit Log ─────────────────────────────────────────────────────

CREATE TABLE governance_audit_log (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type  TEXT         NOT NULL,
    resource_id    UUID         NOT NULL,
    action         TEXT         NOT NULL CHECK (action IN ('create','read','update','delete')),
    accessor_id    UUID         NOT NULL,
    accessor_role  TEXT         NOT NULL,
    ip_address     TEXT         NOT NULL,
    request_id     TEXT         NOT NULL,
    changes        JSONB,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_gov_audit_resource_id ON governance_audit_log(resource_id);
CREATE INDEX idx_gov_audit_accessor_id ON governance_audit_log(accessor_id);
CREATE INDEX idx_gov_audit_created_at  ON governance_audit_log(created_at);

CREATE RULE gov_audit_no_update AS ON UPDATE TO governance_audit_log DO INSTEAD NOTHING;
CREATE RULE gov_audit_no_delete AS ON DELETE TO governance_audit_log DO INSTEAD NOTHING;
