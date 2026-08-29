-- Cooperative Service Schema
-- Agriculture Pillar: farmer groups, collective selling, revenue distribution

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── cooperatives ─────────────────────────────────────────────────────────────
CREATE TABLE cooperatives (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    registration_no TEXT        NOT NULL UNIQUE,
    coop_type       TEXT        NOT NULL DEFAULT 'multi_purpose'
                        CHECK (coop_type IN ('production','marketing','credit','multi_purpose')),
    status          TEXT        NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','suspended','dissolved')),
    country         TEXT        NOT NULL,
    region          TEXT        NOT NULL DEFAULT '',
    district        TEXT        NOT NULL DEFAULT '',
    total_members   INT         NOT NULL DEFAULT 0 CHECK (total_members >= 0),
    total_farm_ha   DOUBLE PRECISION NOT NULL DEFAULT 0,
    description     TEXT        NOT NULL DEFAULT '',
    contact_phone   TEXT        NOT NULL DEFAULT '',
    contact_email   TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_coops_country ON cooperatives(country);
CREATE INDEX idx_coops_region  ON cooperatives(region);
CREATE INDEX idx_coops_status  ON cooperatives(status);

-- ─── coop_members ─────────────────────────────────────────────────────────────
CREATE TABLE coop_members (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    coop_id     UUID        NOT NULL REFERENCES cooperatives(id),
    farmer_id   UUID        NOT NULL,
    role        TEXT        NOT NULL DEFAULT 'member'
                    CHECK (role IN ('member','secretary','treasurer','chairman')),
    status      TEXT        NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','suspended','exited')),
    shares_held INT         NOT NULL DEFAULT 1 CHECK (shares_held >= 0),
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    exited_at   TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (coop_id, farmer_id)
);

CREATE INDEX idx_members_coop   ON coop_members(coop_id);
CREATE INDEX idx_members_farmer ON coop_members(farmer_id);
CREATE INDEX idx_members_status ON coop_members(status);

-- ─── selling_pools ────────────────────────────────────────────────────────────
CREATE TABLE selling_pools (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    coop_id             UUID            NOT NULL REFERENCES cooperatives(id),
    crop_type           TEXT            NOT NULL,
    target_qty_kg       DOUBLE PRECISION NOT NULL CHECK (target_qty_kg > 0),
    collected_qty_kg    DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (collected_qty_kg >= 0),
    price_per_kg        NUMERIC(12,4)   NOT NULL DEFAULT 0,
    currency            TEXT            NOT NULL DEFAULT 'USD',
    status              TEXT            NOT NULL DEFAULT 'open'
                            CHECK (status IN ('open','closed','sold','cancelled')),
    open_until          TIMESTAMPTZ,
    sold_at             TIMESTAMPTZ,
    total_revenue       NUMERIC(14,4)   NOT NULL DEFAULT 0,
    description         TEXT            NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pools_coop   ON selling_pools(coop_id);
CREATE INDEX idx_pools_crop   ON selling_pools(crop_type);
CREATE INDEX idx_pools_status ON selling_pools(status);

-- ─── pool_contributions ───────────────────────────────────────────────────────
CREATE TABLE pool_contributions (
    id            UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id       UUID            NOT NULL REFERENCES selling_pools(id),
    farmer_id     UUID            NOT NULL,
    quantity_kg   DOUBLE PRECISION NOT NULL CHECK (quantity_kg > 0),
    payout_amount NUMERIC(14,4)   NOT NULL DEFAULT 0,
    payout_paid   BOOLEAN         NOT NULL DEFAULT false,
    paid_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (pool_id, farmer_id)
);

CREATE INDEX idx_contributions_pool   ON pool_contributions(pool_id);
CREATE INDEX idx_contributions_farmer ON pool_contributions(farmer_id);

-- ─── coop_audit_log ───────────────────────────────────────────────────────────
CREATE TABLE coop_audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   UUID,
    user_id     UUID        NOT NULL,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    ip_address  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE coop_audit_log_no_update AS
    ON UPDATE TO coop_audit_log DO INSTEAD NOTHING;

CREATE RULE coop_audit_log_no_delete AS
    ON DELETE TO coop_audit_log DO INSTEAD NOTHING;
