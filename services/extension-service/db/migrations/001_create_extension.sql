CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS extension_resources (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title         TEXT NOT NULL,
    content_summary TEXT,
    crop_type     TEXT,
    language      TEXT NOT NULL DEFAULT 'en',
    resource_type TEXT NOT NULL DEFAULT 'guide',
    viewed_count  INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO extension_resources (title, content_summary, crop_type, language, resource_type) VALUES
('Guide du maïs — semis et fertilisation', 'Techniques de semis du maïs et calendrier de fertilisation pour la région Maritime', 'maize', 'fr', 'guide'),
('Maize Planting Guide', 'Best practices for maize planting in tropical climates with rainfall guidance', 'maize', 'en', 'guide'),
('Cacao: taille et traitements', 'Méthodes de taille du cacao et calendrier phytosanitaire pour les Plateaux', 'cocoa', 'fr', 'guide'),
('Gestion de leau pour le sorgho', 'Irrigation optimale et conservation de leau pour le sorgho en zone semi-aride', 'sorghum', 'fr', 'guide'),
('Mil: techniques culturales Savanes', 'Pratiques agricoles durables pour le mil dans la région des Savanes', 'millet', 'fr', 'guide')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS consultations (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    consult_ref    TEXT NOT NULL UNIQUE,
    farmer_id      UUID NOT NULL,
    officer_id     UUID,
    topic          TEXT NOT NULL,
    crop_type      TEXT,
    preferred_date TIMESTAMPTZ,
    status         TEXT NOT NULL DEFAULT 'pending',
    notes          TEXT,
    tenant_id      TEXT NOT NULL,
    booked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_consultations_farmer ON consultations(farmer_id);

CREATE TABLE IF NOT EXISTS extension_feedback (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    consultation_id UUID NOT NULL REFERENCES consultations(id),
    farmer_id       UUID NOT NULL,
    rating          INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    notes           TEXT,
    result          TEXT,
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS best_practices (
    id                             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    crop_type                      TEXT NOT NULL,
    technique                      TEXT NOT NULL,
    description                    TEXT,
    expected_yield_improvement_pct NUMERIC(5,2),
    climate                        TEXT
);

INSERT INTO best_practices (crop_type, technique, description, expected_yield_improvement_pct, climate) VALUES
('maize', 'Micro-dosing fertilizer', 'Apply 6g of fertilizer per planting hole instead of broadcast application', 25.0, 'semi-arid'),
('maize', 'Tied ridges', 'Create ridges perpendicular to slope to capture rainwater', 30.0, 'arid'),
('cocoa', 'Shade tree management', 'Maintain 30-40% shade cover with nitrogen-fixing trees', 20.0, 'tropical'),
('sorghum', 'Zai pits', 'Dig 20cm planting pits to concentrate water and organic matter', 40.0, 'arid'),
('millet', 'Mulching', 'Use crop residue mulch to reduce evaporation and suppress weeds', 15.0, 'semi-arid')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS extension_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    consult_id  UUID,
    action      TEXT NOT NULL,
    actor_id    TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_ext_audit AS ON UPDATE TO extension_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_ext_audit AS ON DELETE TO extension_audit_log DO INSTEAD NOTHING;
