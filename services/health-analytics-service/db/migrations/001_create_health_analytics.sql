-- Health analytics service schema
-- All tables are immutable (append-only) to protect audit integrity.

CREATE TABLE IF NOT EXISTS disease_reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id    UUID NOT NULL,
    country      CHAR(2) NOT NULL,
    region       TEXT NOT NULL DEFAULT '',
    icd10_code   TEXT NOT NULL,
    disease_name TEXT NOT NULL,
    case_count   INTEGER NOT NULL DEFAULT 1,
    period       TEXT NOT NULL CHECK (period IN ('daily','weekly','monthly')),
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    severity     TEXT NOT NULL DEFAULT 'mild'
                    CHECK (severity IN ('mild','moderate','severe','critical')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_disease_reports AS ON UPDATE TO disease_reports DO INSTEAD NOTHING;
CREATE RULE no_delete_disease_reports AS ON DELETE TO disease_reports DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS outbreak_alerts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_ref    TEXT NOT NULL UNIQUE,
    clinic_id    UUID NOT NULL,
    country      CHAR(2) NOT NULL,
    region       TEXT NOT NULL DEFAULT '',
    icd10_code   TEXT NOT NULL,
    disease_name TEXT NOT NULL,
    case_count   INTEGER NOT NULL,
    threshold    INTEGER NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','monitoring','resolved')),
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS clinic_metrics (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id                UUID NOT NULL,
    country                  CHAR(2) NOT NULL,
    period                   TEXT NOT NULL,
    period_start             TIMESTAMPTZ NOT NULL,
    period_end               TIMESTAMPTZ NOT NULL,
    total_patients           INTEGER NOT NULL DEFAULT 0,
    avg_visit_minutes        NUMERIC(6,2) NOT NULL DEFAULT 0,
    referral_count           INTEGER NOT NULL DEFAULT 0,
    referral_success_rate    NUMERIC(5,2) NOT NULL DEFAULT 0,
    patient_outcome_improved INTEGER NOT NULL DEFAULT 0,
    patient_outcome_stable   INTEGER NOT NULL DEFAULT 0,
    patient_outcome_worsened INTEGER NOT NULL DEFAULT 0,
    cost_per_visit_usd       NUMERIC(10,2) NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_clinic_metrics AS ON UPDATE TO clinic_metrics DO INSTEAD NOTHING;
CREATE RULE no_delete_clinic_metrics AS ON DELETE TO clinic_metrics DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS health_analytics_audit_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id   UUID NOT NULL,
    action     TEXT NOT NULL,
    resource   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE RULE no_update_ha_audit AS ON UPDATE TO health_analytics_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_ha_audit AS ON DELETE TO health_analytics_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_disease_reports_country   ON disease_reports(country);
CREATE INDEX IF NOT EXISTS idx_disease_reports_icd10     ON disease_reports(icd10_code);
CREATE INDEX IF NOT EXISTS idx_outbreak_alerts_status    ON outbreak_alerts(status);
CREATE INDEX IF NOT EXISTS idx_outbreak_alerts_country   ON outbreak_alerts(country);
CREATE INDEX IF NOT EXISTS idx_clinic_metrics_clinic     ON clinic_metrics(clinic_id);
CREATE INDEX IF NOT EXISTS idx_clinic_metrics_country    ON clinic_metrics(country);
