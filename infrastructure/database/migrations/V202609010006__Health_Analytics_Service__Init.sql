-- Health Analytics Service — Aggregated health KPIs, disease surveillance, and generated reports
-- Database: kinara_analytics
-- Stores pre-computed metrics and surveillance data for dashboards and regulatory reporting

\c kinara_analytics;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS health_metrics (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_type   TEXT          NOT NULL,
    metric_name   TEXT          NOT NULL,
    value         NUMERIC(14,4) NOT NULL,
    unit          TEXT,
    period_start  TIMESTAMPTZ   NOT NULL,
    period_end    TIMESTAMPTZ   NOT NULL,
    clinic_id     TEXT,
    country       TEXT          NOT NULL,
    region        TEXT,
    calculated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  health_metrics              IS 'Aggregated health KPIs calculated at scheduled intervals for dashboard consumption';
COMMENT ON COLUMN health_metrics.metric_type  IS 'Category: mortality, morbidity, utilization, coverage, quality';
COMMENT ON COLUMN health_metrics.metric_name  IS 'Specific metric key, e.g. maternal_mortality_rate, anc4_coverage';
COMMENT ON COLUMN health_metrics.period_start IS 'Inclusive start of the aggregation window (UTC)';
COMMENT ON COLUMN health_metrics.period_end   IS 'Exclusive end of the aggregation window (UTC)';
COMMENT ON COLUMN health_metrics.clinic_id    IS 'Null means country/region aggregate; set for facility-level drill-down';

CREATE INDEX IF NOT EXISTS idx_hm_type_period   ON health_metrics(metric_type, period_start);
CREATE INDEX IF NOT EXISTS idx_hm_clinic_period ON health_metrics(clinic_id, period_start DESC);
CREATE INDEX IF NOT EXISTS idx_hm_country       ON health_metrics(country, period_start DESC);

CREATE TABLE IF NOT EXISTS disease_surveillance (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    disease_code  TEXT        NOT NULL,
    disease_name  TEXT        NOT NULL,
    case_count    INT         NOT NULL DEFAULT 0 CHECK (case_count >= 0),
    clinic_id     TEXT,
    country       TEXT        NOT NULL,
    week_start    DATE        NOT NULL,
    is_outbreak   BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  disease_surveillance            IS 'Weekly disease case counts per clinic for WHO/national surveillance reporting';
COMMENT ON COLUMN disease_surveillance.disease_code IS 'ICD-10 code or WHO surveillance disease identifier';
COMMENT ON COLUMN disease_surveillance.week_start  IS 'ISO week start (Monday) for the reporting period';
COMMENT ON COLUMN disease_surveillance.is_outbreak  IS 'True when case count exceeds configured epidemic threshold for that disease/region';

CREATE INDEX IF NOT EXISTS idx_surv_disease_week ON disease_surveillance(disease_code, week_start);
CREATE INDEX IF NOT EXISTS idx_surv_clinic_week  ON disease_surveillance(clinic_id, week_start DESC);
CREATE INDEX IF NOT EXISTS idx_surv_outbreak     ON disease_surveillance(is_outbreak) WHERE is_outbreak = true;

CREATE TABLE IF NOT EXISTS health_reports (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    report_type    TEXT        NOT NULL,
    report_name    TEXT        NOT NULL,
    parameters     JSONB,
    file_url       TEXT,
    generated_by   UUID        NOT NULL,
    generated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id      TEXT        NOT NULL
);

COMMENT ON TABLE  health_reports              IS 'Registry of generated PDF/CSV health reports for download and audit trail';
COMMENT ON COLUMN health_reports.report_type  IS 'Category: weekly_surveillance, monthly_kpi, compliance, outbreak, custom';
COMMENT ON COLUMN health_reports.parameters   IS 'JSONB snapshot of generation parameters — date range, filters, breakdown fields';
COMMENT ON COLUMN health_reports.file_url     IS 'Signed object-storage URL; may expire — re-generate if null or expired';

CREATE INDEX IF NOT EXISTS idx_hr_type_date  ON health_reports(report_type, generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_hr_tenant     ON health_reports(tenant_id, generated_at DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS health_analytics_audit_log (
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

COMMENT ON TABLE health_analytics_audit_log IS 'Append-only audit trail for analytics entities — including report generation events';

CREATE RULE no_update_health_analytics_audit AS ON UPDATE TO health_analytics_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_health_analytics_audit AS ON DELETE TO health_analytics_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_health_analytics_audit_entity ON health_analytics_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_health_analytics_audit_actor  ON health_analytics_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS health_analytics_audit_log CASCADE;
-- DROP TABLE IF EXISTS health_reports CASCADE;
-- DROP TABLE IF EXISTS disease_surveillance CASCADE;
-- DROP TABLE IF EXISTS health_metrics CASCADE;
