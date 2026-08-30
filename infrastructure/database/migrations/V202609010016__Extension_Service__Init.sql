-- Extension Service — Agricultural extension resources, farmer consultations, feedback, and best practices
-- Database: kinara_extension
-- Supports farmer-to-officer consultation booking, training materials, and agronomic best practices
-- Note: run 'CREATE DATABASE kinara_extension;' as superuser if not exists

\c kinara_extension;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS extension_resources (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title            TEXT        NOT NULL,
    content_summary  TEXT,
    crop_type        TEXT,
    language         TEXT        NOT NULL DEFAULT 'en'
                     CHECK (language IN ('en','fr','ha','sw')),
    resource_type    TEXT        NOT NULL DEFAULT 'guide'
                     CHECK (resource_type IN ('guide','video','checklist','infographic')),
    viewed_count     INT         NOT NULL DEFAULT 0 CHECK (viewed_count >= 0),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  extension_resources               IS 'Farmer-facing training materials: guides, videos, checklists, infographics';
COMMENT ON COLUMN extension_resources.crop_type     IS 'Target crop for this resource, e.g. maize, cocoa, millet — null means general';
COMMENT ON COLUMN extension_resources.language      IS 'Content language: en (English), fr (French), ha (Hausa), sw (Swahili)';
COMMENT ON COLUMN extension_resources.resource_type IS 'Format: guide (PDF/article), video (URL), checklist (interactive), infographic (image)';
COMMENT ON COLUMN extension_resources.viewed_count  IS 'Cumulative view count for popularity ranking — incremented at API layer';

CREATE INDEX IF NOT EXISTS idx_er_crop_type ON extension_resources(crop_type);
CREATE INDEX IF NOT EXISTS idx_er_language  ON extension_resources(language);
CREATE INDEX IF NOT EXISTS idx_er_popular   ON extension_resources(viewed_count DESC);

CREATE TABLE IF NOT EXISTS consultations (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    consult_ref    TEXT        NOT NULL UNIQUE,
    farmer_id      UUID        NOT NULL,
    officer_id     UUID        NOT NULL,
    topic          TEXT        NOT NULL,
    crop_type      TEXT,
    preferred_date TIMESTAMPTZ,
    status         TEXT        NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','scheduled','completed','cancelled')),
    notes          TEXT,
    tenant_id      TEXT        NOT NULL,
    booked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  consultations               IS 'Farmer-officer consultation bookings — agronomic advisory sessions';
COMMENT ON COLUMN consultations.consult_ref   IS 'Human-readable unique reference, e.g. EXT-20260901-0001';
COMMENT ON COLUMN consultations.topic         IS 'Subject of consultation, e.g. pest management, soil health, market prices';
COMMENT ON COLUMN consultations.preferred_date IS 'Farmer-requested date; officer confirms and sets final scheduled time';
COMMENT ON COLUMN consultations.status         IS 'Booking lifecycle: pending → scheduled → completed | cancelled';

CREATE INDEX IF NOT EXISTS idx_ext_consult_farmer  ON consultations(farmer_id, booked_at DESC);
CREATE INDEX IF NOT EXISTS idx_ext_consult_officer ON consultations(officer_id, booked_at DESC);
CREATE INDEX IF NOT EXISTS idx_ext_consult_status  ON consultations(status);
CREATE INDEX IF NOT EXISTS idx_ext_consult_tenant  ON consultations(tenant_id);

CREATE TABLE IF NOT EXISTS extension_feedback (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id   UUID        NOT NULL REFERENCES consultations(id),
    farmer_id         UUID        NOT NULL,
    rating            INT         NOT NULL CHECK (rating BETWEEN 1 AND 5),
    notes             TEXT,
    result            TEXT,
    submitted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  extension_feedback                IS 'Farmer satisfaction scores after completed consultations';
COMMENT ON COLUMN extension_feedback.rating         IS 'Farmer satisfaction 1 (very poor) to 5 (excellent)';
COMMENT ON COLUMN extension_feedback.result         IS 'Self-reported outcome: improved_yield, resolved_pest, no_change, other';

CREATE INDEX IF NOT EXISTS idx_ef_consultation ON extension_feedback(consultation_id);
CREATE INDEX IF NOT EXISTS idx_ef_farmer       ON extension_feedback(farmer_id, submitted_at DESC);

CREATE TABLE IF NOT EXISTS best_practices (
    id                              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    crop_type                       TEXT          NOT NULL,
    technique                       TEXT          NOT NULL,
    description                     TEXT,
    expected_yield_improvement_pct  NUMERIC(5,2),
    climate                         TEXT          NOT NULL
                                    CHECK (climate IN ('arid','semi-arid','tropical','temperate'))
);

COMMENT ON TABLE  best_practices                                IS 'Curated agronomic best practices by crop type and climate zone';
COMMENT ON COLUMN best_practices.technique                      IS 'Practice name, e.g. conservation tillage, intercropping, drip irrigation';
COMMENT ON COLUMN best_practices.expected_yield_improvement_pct IS 'Evidence-based yield improvement percentage vs. baseline practice';
COMMENT ON COLUMN best_practices.climate                        IS 'Applicable climate zone for this practice';

CREATE INDEX IF NOT EXISTS idx_bp_crop    ON best_practices(crop_type);
CREATE INDEX IF NOT EXISTS idx_bp_climate ON best_practices(climate);

INSERT INTO best_practices (crop_type, technique, description, expected_yield_improvement_pct, climate) VALUES
    ('maize',  'Microdosing Fertilizer',     'Apply 6 g DAP per planting hole at sowing — reduces input cost by 40% while maintaining yields', 25.0, 'semi-arid'),
    ('maize',  'Tied Ridges',                'Form ridges with cross-ties every 2 m to retain rainfall in furrows',                            18.0, 'arid'),
    ('cocoa',  'Shade Tree Integration',     'Intercrop with Gliricidia for nitrogen fixation and thermal buffering',                           22.0, 'tropical'),
    ('sorghum','Early Maturing Varieties',   'Use drought-tolerant varieties (ICSV111) to avoid terminal drought stress',                       30.0, 'semi-arid'),
    ('millet', 'Zaï Planting Pits',          'Hand-dig pits 20–30 cm diameter, 10–15 cm deep, fill with compost before sowing',                35.0, 'arid')
ON CONFLICT DO NOTHING;

-- Immutable audit log
CREATE TABLE IF NOT EXISTS extension_audit_log (
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

COMMENT ON TABLE extension_audit_log IS 'Append-only audit trail for extension consultations, resources, and feedback';

CREATE RULE no_update_extension_audit AS ON UPDATE TO extension_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_extension_audit AS ON DELETE TO extension_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_extension_audit_entity ON extension_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_extension_audit_actor  ON extension_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS extension_audit_log CASCADE;
-- DROP TABLE IF EXISTS best_practices CASCADE;
-- DROP TABLE IF EXISTS extension_feedback CASCADE;
-- DROP TABLE IF EXISTS consultations CASCADE;
-- DROP TABLE IF EXISTS extension_resources CASCADE;
