CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE impact_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pillar VARCHAR(20) NOT NULL,
    country VARCHAR(3) NOT NULL,
    metric_type VARCHAR(30) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC(18,4) NOT NULL,
    metric_unit VARCHAR(50) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    beneficiary_count BIGINT NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE cross_pillar_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country VARCHAR(3) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    health_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    agri_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    logistics_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    maritime_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    overall_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    total_beneficiaries BIGINT NOT NULL DEFAULT 0,
    total_services_delivered BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE government_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_ref VARCHAR(25) NOT NULL UNIQUE,
    country VARCHAR(3) NOT NULL,
    report_type VARCHAR(50) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    summary_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE bottlenecks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pillar VARCHAR(20) NOT NULL,
    country VARCHAR(3) NOT NULL,
    bottleneck_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'medium',
    affected_units INT NOT NULL DEFAULT 0,
    recommended_action TEXT,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE analytics_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- All analytics tables are immutable ledgers
CREATE RULE no_update_impact AS ON UPDATE TO impact_metrics DO INSTEAD NOTHING;
CREATE RULE no_delete_impact AS ON DELETE TO impact_metrics DO INSTEAD NOTHING;
CREATE RULE no_update_summary AS ON UPDATE TO cross_pillar_summaries DO INSTEAD NOTHING;
CREATE RULE no_delete_summary AS ON DELETE TO cross_pillar_summaries DO INSTEAD NOTHING;
CREATE RULE no_update_report AS ON UPDATE TO government_reports DO INSTEAD NOTHING;
CREATE RULE no_delete_report AS ON DELETE TO government_reports DO INSTEAD NOTHING;
CREATE RULE no_update_analytics_audit AS ON UPDATE TO analytics_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_analytics_audit AS ON DELETE TO analytics_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_impact_pillar_country ON impact_metrics(pillar, country);
CREATE INDEX idx_impact_period ON impact_metrics(period_start, period_end);
CREATE INDEX idx_summaries_country ON cross_pillar_summaries(country);
CREATE INDEX idx_bottlenecks_pillar ON bottlenecks(pillar, country);
CREATE INDEX idx_bottlenecks_unresolved ON bottlenecks(resolved_at) WHERE resolved_at IS NULL;
