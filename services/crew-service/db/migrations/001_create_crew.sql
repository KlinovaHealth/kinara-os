CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS crew_members (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    crew_ref        TEXT NOT NULL UNIQUE,
    full_name       TEXT NOT NULL,
    nationality     TEXT NOT NULL,
    passport_number TEXT,
    rank            TEXT NOT NULL,
    vessel_id       UUID,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    tenant_id       TEXT NOT NULL,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crew_certifications (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    crew_id     UUID NOT NULL REFERENCES crew_members(id),
    cert_type   TEXT NOT NULL,
    cert_number TEXT NOT NULL,
    issued_by   TEXT NOT NULL,
    issued_at   DATE NOT NULL,
    expires_at  DATE NOT NULL,
    status      TEXT NOT NULL DEFAULT 'valid',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crew_vessel ON crew_members(vessel_id);
CREATE INDEX IF NOT EXISTS idx_crew_tenant ON crew_members(tenant_id);
CREATE INDEX IF NOT EXISTS idx_certs_crew ON crew_certifications(crew_id);
CREATE INDEX IF NOT EXISTS idx_certs_expiry ON crew_certifications(expires_at);
