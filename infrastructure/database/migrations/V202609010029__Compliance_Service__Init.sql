-- =============================================================================
-- V202609010029__Compliance_Service__Init.sql
-- Kinara OS — Compliance Service
-- =============================================================================
\c kinara_compliance;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- compliance_frameworks
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS compliance_frameworks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    framework_code  TEXT        UNIQUE,
    name            TEXT        NOT NULL,
    version         TEXT,
    jurisdiction    TEXT,
    category        TEXT        CHECK (category IN ('health', 'financial', 'logistics', 'maritime', 'agricultural')),
    description     TEXT,
    effective_date  DATE,
    status          TEXT        DEFAULT 'active',
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE compliance_frameworks IS 'Registry of regulatory and industry compliance frameworks applicable to Kinara services';
COMMENT ON COLUMN compliance_frameworks.framework_code IS 'Short machine-readable code for the framework (e.g. WHO-GMP, ECOWAS-FS)';
COMMENT ON COLUMN compliance_frameworks.jurisdiction   IS 'Geographic or regulatory jurisdiction (e.g. Nigeria, ECOWAS, AU)';
COMMENT ON COLUMN compliance_frameworks.category       IS 'Domain classification of the framework';
COMMENT ON COLUMN compliance_frameworks.effective_date IS 'Date from which the framework version is enforceable';
COMMENT ON COLUMN compliance_frameworks.version        IS 'Framework revision or edition (e.g. 2026-v1)';

-- ---------------------------------------------------------------------------
-- compliance_requirements
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS compliance_requirements (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    framework_id      UUID        NOT NULL REFERENCES compliance_frameworks(id) ON DELETE CASCADE,
    requirement_code  TEXT        UNIQUE,
    title             TEXT        NOT NULL,
    description       TEXT,
    mandatory         BOOLEAN     DEFAULT true,
    check_frequency   TEXT        DEFAULT 'annual'
                      CHECK (check_frequency IN ('monthly', 'quarterly', 'annual', 'as_needed')),
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE compliance_requirements IS 'Individual compliance controls and obligations within each framework';
COMMENT ON COLUMN compliance_requirements.requirement_code IS 'Short unique code for the requirement (e.g. WHO-GMP-R01)';
COMMENT ON COLUMN compliance_requirements.mandatory        IS 'True = must be met; false = recommended best practice';
COMMENT ON COLUMN compliance_requirements.check_frequency  IS 'How often the requirement must be assessed';

-- ---------------------------------------------------------------------------
-- compliance_assessments
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS compliance_assessments (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id        UUID        NOT NULL,
    entity_type      TEXT        NOT NULL,
    requirement_id   UUID        NOT NULL REFERENCES compliance_requirements(id),
    status           TEXT        DEFAULT 'pending'
                     CHECK (status IN ('pending', 'compliant', 'non_compliant', 'waived')),
    score            NUMERIC(5,2),
    assessed_by      UUID,
    assessed_at      TIMESTAMPTZ,
    next_due         TIMESTAMPTZ,
    notes            TEXT,
    evidence_url     TEXT
);

COMMENT ON TABLE compliance_assessments IS 'Compliance assessment results linking entities to their requirement statuses';
COMMENT ON COLUMN compliance_assessments.entity_id    IS 'UUID of the entity being assessed (warehouse, cooperative, fleet, etc.)';
COMMENT ON COLUMN compliance_assessments.entity_type  IS 'Type discriminator for entity_id (warehouse, cooperative, vehicle, etc.)';
COMMENT ON COLUMN compliance_assessments.score        IS 'Numeric compliance score (0–100) where applicable';
COMMENT ON COLUMN compliance_assessments.assessed_by  IS 'UUID of the auditor or system that performed the assessment';
COMMENT ON COLUMN compliance_assessments.next_due     IS 'Timestamp when the next assessment must be completed';
COMMENT ON COLUMN compliance_assessments.evidence_url IS 'Cloud URL to supporting compliance documentation or certificate';

-- ---------------------------------------------------------------------------
-- compliance_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS compliance_audit_log (
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

COMMENT ON TABLE compliance_audit_log IS 'Immutable audit trail for all compliance-service mutations';

CREATE RULE no_update_compliance_audit AS ON UPDATE TO compliance_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_compliance_audit AS ON DELETE TO compliance_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_compliance_frameworks_category ON compliance_frameworks(category);
CREATE INDEX IF NOT EXISTS idx_compliance_requirements_framework ON compliance_requirements(framework_id);
CREATE INDEX IF NOT EXISTS idx_compliance_assessments_entity  ON compliance_assessments(entity_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_compliance_assessments_req     ON compliance_assessments(requirement_id);
CREATE INDEX IF NOT EXISTS idx_compliance_assessments_status  ON compliance_assessments(status);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_entity        ON compliance_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_actor         ON compliance_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_compliance_audit_actor;
-- DROP INDEX IF EXISTS idx_compliance_audit_entity;
-- DROP INDEX IF EXISTS idx_compliance_assessments_status;
-- DROP INDEX IF EXISTS idx_compliance_assessments_req;
-- DROP INDEX IF EXISTS idx_compliance_assessments_entity;
-- DROP INDEX IF EXISTS idx_compliance_requirements_framework;
-- DROP INDEX IF EXISTS idx_compliance_frameworks_category;
-- DROP RULE IF EXISTS no_delete_compliance_audit ON compliance_audit_log;
-- DROP RULE IF EXISTS no_update_compliance_audit ON compliance_audit_log;
-- DROP TABLE IF EXISTS compliance_audit_log;
-- DROP TABLE IF EXISTS compliance_assessments;
-- DROP TABLE IF EXISTS compliance_requirements;
-- DROP TABLE IF EXISTS compliance_frameworks;
-- =============================================================================
