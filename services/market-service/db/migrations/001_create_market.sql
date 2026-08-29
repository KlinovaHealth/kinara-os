-- Market Service Schema
-- Agriculture Pillar: price discovery, buyer-seller matching, commodity listings

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── market_listings ──────────────────────────────────────────────────────────
CREATE TABLE market_listings (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id       UUID        NOT NULL,
    crop_type       TEXT        NOT NULL,
    variety         TEXT        NOT NULL DEFAULT '',
    quantity_kg     DOUBLE PRECISION NOT NULL CHECK (quantity_kg > 0),
    quantity_available DOUBLE PRECISION NOT NULL CHECK (quantity_available >= 0),
    price_per_unit  NUMERIC(12,4) NOT NULL CHECK (price_per_unit > 0),
    currency        TEXT        NOT NULL DEFAULT 'USD',
    price_unit      TEXT        NOT NULL DEFAULT 'kg'
                        CHECK (price_unit IN ('kg','tonne','bag','crate','bushel','litre')),
    quality_grade   TEXT        NOT NULL DEFAULT 'B'
                        CHECK (quality_grade IN ('A','B','C')),
    country         TEXT        NOT NULL,
    region          TEXT        NOT NULL DEFAULT '',
    market          TEXT        NOT NULL DEFAULT '',
    harvested_at    TIMESTAMPTZ,
    available_from  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    available_until TIMESTAMPTZ,
    status          TEXT        NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','reserved','sold','expired','cancelled')),
    description     TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_listings_farmer     ON market_listings(farmer_id);
CREATE INDEX idx_listings_crop       ON market_listings(crop_type);
CREATE INDEX idx_listings_country    ON market_listings(country);
CREATE INDEX idx_listings_region     ON market_listings(region);
CREATE INDEX idx_listings_status     ON market_listings(status);
CREATE INDEX idx_listings_price      ON market_listings(price_per_unit);
CREATE INDEX idx_listings_available  ON market_listings(available_from, available_until);

-- ─── price_records ────────────────────────────────────────────────────────────
-- Historical price data for analytics and alerts
CREATE TABLE price_records (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    crop_type   TEXT        NOT NULL,
    market      TEXT        NOT NULL,
    country     TEXT        NOT NULL,
    region      TEXT        NOT NULL DEFAULT '',
    price_per_kg NUMERIC(12,4) NOT NULL,
    currency    TEXT        NOT NULL DEFAULT 'USD',
    source      TEXT        NOT NULL DEFAULT 'listing'
                    CHECK (source IN ('listing','reported','official')),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_by UUID        NOT NULL
);

CREATE INDEX idx_prices_crop     ON price_records(crop_type);
CREATE INDEX idx_prices_market   ON price_records(market);
CREATE INDEX idx_prices_country  ON price_records(country);
CREATE INDEX idx_prices_date     ON price_records(recorded_at DESC);

-- ─── market_bids ─────────────────────────────────────────────────────────────
-- Buyer bids on active listings
CREATE TABLE market_bids (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID        NOT NULL REFERENCES market_listings(id),
    buyer_id    UUID        NOT NULL,
    quantity_kg DOUBLE PRECISION NOT NULL CHECK (quantity_kg > 0),
    bid_price   NUMERIC(12,4) NOT NULL CHECK (bid_price > 0),
    currency    TEXT        NOT NULL DEFAULT 'USD',
    status      TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','accepted','rejected','withdrawn','expired')),
    message     TEXT        NOT NULL DEFAULT '',
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bids_listing ON market_bids(listing_id);
CREATE INDEX idx_bids_buyer   ON market_bids(buyer_id);
CREATE INDEX idx_bids_status  ON market_bids(status);

-- ─── market_audit_log ─────────────────────────────────────────────────────────
CREATE TABLE market_audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   UUID,
    user_id     UUID        NOT NULL,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    ip_address  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE market_audit_log_no_update AS
    ON UPDATE TO market_audit_log DO INSTEAD NOTHING;

CREATE RULE market_audit_log_no_delete AS
    ON DELETE TO market_audit_log DO INSTEAD NOTHING;
