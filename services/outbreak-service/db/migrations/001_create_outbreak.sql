CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS outbreak_responses (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    response_ref     TEXT NOT NULL UNIQUE,
    alert_ref        TEXT NOT NULL,
    disease_name     TEXT NOT NULL,
    country          TEXT NOT NULL,
    region           TEXT,
    status           TEXT NOT NULL DEFAULT 'active',
    lead_coordinator UUID NOT NULL,
    team_size        INT NOT NULL DEFAULT 0,
    cases_targeted   INT NOT NULL DEFAULT 0,
    population       INT NOT NULL DEFAULT 0,
    tenant_id        TEXT NOT NULL,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    contained_at     TIMESTAMPTZ,
    resolved_at      TIMESTAMPTZ,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS response_actions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    response_id UUID NOT NULL REFERENCES outbreak_responses(id),
    action_type TEXT NOT NULL,
    description TEXT NOT NULL,
    assigned_to UUID NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_responses_country ON outbreak_responses(country);
CREATE INDEX IF NOT EXISTS idx_responses_tenant ON outbreak_responses(tenant_id);
CREATE INDEX IF NOT EXISTS idx_actions_response ON response_actions(response_id);

CREATE RULE no_delete_actions AS ON DELETE TO response_actions DO INSTEAD NOTHING;
