-- =============================================================================
-- V202609010019__Market_Service__Init.sql
-- Kinara OS — Market Service
-- =============================================================================
\c kinara_market;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- market_listings
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS market_listings (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id          UUID        NOT NULL,
    crop_type          TEXT        NOT NULL,
    variety            TEXT        DEFAULT '',
    quantity_kg        DOUBLE PRECISION        CHECK (quantity_kg > 0),
    quantity_available DOUBLE PRECISION        CHECK (quantity_available >= 0),
    price_per_unit     NUMERIC(12,4)           CHECK (price_per_unit > 0),
    currency           TEXT        DEFAULT 'USD',
    price_unit         TEXT        DEFAULT 'kg'
                       CHECK (price_unit IN ('kg', 'tonne', 'bag', 'crate', 'bushel')),
    quality_grade      TEXT        DEFAULT 'B'
                       CHECK (quality_grade IN ('A', 'B', 'C')),
    country            TEXT        NOT NULL,
    region             TEXT        DEFAULT '',
    market             TEXT        DEFAULT '',
    harvested_at       TIMESTAMPTZ,
    available_from     TIMESTAMPTZ DEFAULT NOW(),
    available_until    TIMESTAMPTZ,
    status             TEXT        DEFAULT 'active'
                       CHECK (status IN ('active', 'reserved', 'sold', 'expired', 'cancelled')),
    description        TEXT        DEFAULT '',
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE market_listings IS 'Agricultural commodity listings published by farmers for sale';
COMMENT ON COLUMN market_listings.farmer_id         IS 'Cross-DB reference to farmers.id in kinara_farmer';
COMMENT ON COLUMN market_listings.quantity_available IS 'Remaining quantity not yet reserved or sold';
COMMENT ON COLUMN market_listings.price_per_unit    IS 'Listed price in the specified currency per price_unit';
COMMENT ON COLUMN market_listings.quality_grade     IS 'A = premium, B = standard, C = sub-standard';

-- ---------------------------------------------------------------------------
-- market_orders
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS market_orders (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id    UUID        NOT NULL REFERENCES market_listings(id),
    buyer_id      UUID        NOT NULL,
    quantity_kg   DOUBLE PRECISION        CHECK (quantity_kg > 0),
    agreed_price  NUMERIC(12,4),
    currency      TEXT        DEFAULT 'USD',
    total_amount  NUMERIC(14,4),
    status        TEXT        DEFAULT 'pending'
                  CHECK (status IN ('pending', 'confirmed', 'paid', 'delivered', 'cancelled', 'disputed')),
    payment_ref   TEXT,
    notes         TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE market_orders IS 'Buyer purchase orders against active market listings';
COMMENT ON COLUMN market_orders.agreed_price IS 'Price locked at order creation — may differ from listing price after negotiation';
COMMENT ON COLUMN market_orders.total_amount IS 'Derived: quantity_kg * agreed_price; stored for audit stability';
COMMENT ON COLUMN market_orders.payment_ref  IS 'Mobile-money or bank transaction reference';

-- ---------------------------------------------------------------------------
-- price_indices
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS price_indices (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    crop_type    TEXT        NOT NULL,
    market       TEXT,
    country      TEXT,
    price_per_kg NUMERIC(10,4),
    currency     TEXT        DEFAULT 'USD',
    recorded_at  DATE        DEFAULT CURRENT_DATE,
    source       TEXT        DEFAULT 'kinara'
);

COMMENT ON TABLE price_indices IS 'Reference spot prices per commodity, market, and country for benchmarking';
COMMENT ON COLUMN price_indices.source      IS 'Origin of the price data (kinara, FAO, WFP, local-survey, etc.)';
COMMENT ON COLUMN price_indices.recorded_at IS 'Date the price observation was captured';

-- ---------------------------------------------------------------------------
-- market_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS market_audit_log (
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

COMMENT ON TABLE market_audit_log IS 'Immutable audit trail for all market-service mutations';

CREATE RULE no_update_market_audit AS ON UPDATE TO market_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_market_audit AS ON DELETE TO market_audit_log DO INSTEAD NOTHING;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_market_listings_farmer        ON market_listings(farmer_id);
CREATE INDEX IF NOT EXISTS idx_market_listings_type_status   ON market_listings(crop_type, status);
CREATE INDEX IF NOT EXISTS idx_market_listings_country_region ON market_listings(country, region);
CREATE INDEX IF NOT EXISTS idx_market_orders_listing         ON market_orders(listing_id);
CREATE INDEX IF NOT EXISTS idx_market_orders_buyer           ON market_orders(buyer_id);
CREATE INDEX IF NOT EXISTS idx_price_indices_type_date       ON price_indices(crop_type, recorded_at);
CREATE INDEX IF NOT EXISTS idx_market_audit_entity           ON market_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_market_audit_actor            ON market_audit_log(actor_id, occurred_at);

-- =============================================================================
-- DOWN (rollback)
-- DROP INDEX IF EXISTS idx_market_audit_actor;
-- DROP INDEX IF EXISTS idx_market_audit_entity;
-- DROP INDEX IF EXISTS idx_price_indices_type_date;
-- DROP INDEX IF EXISTS idx_market_orders_buyer;
-- DROP INDEX IF EXISTS idx_market_orders_listing;
-- DROP INDEX IF EXISTS idx_market_listings_country_region;
-- DROP INDEX IF EXISTS idx_market_listings_type_status;
-- DROP INDEX IF EXISTS idx_market_listings_farmer;
-- DROP RULE IF EXISTS no_delete_market_audit ON market_audit_log;
-- DROP RULE IF EXISTS no_update_market_audit ON market_audit_log;
-- DROP TABLE IF EXISTS market_audit_log;
-- DROP TABLE IF EXISTS price_indices;
-- DROP TABLE IF EXISTS market_orders;
-- DROP TABLE IF EXISTS market_listings;
-- =============================================================================
