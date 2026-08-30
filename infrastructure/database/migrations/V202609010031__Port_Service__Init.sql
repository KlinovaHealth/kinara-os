-- =============================================================================
-- Kinara OS — Port Service
-- Migration : V202609010031__Port_Service__Init.sql
-- Database  : kinara_port
-- Description: Initialises the Port Service schema: ports, berths, port calls,
--              tariffs, and an immutable audit log.
-- =============================================================================

\c kinara_port;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- ports
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ports (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    code          TEXT        UNIQUE NOT NULL,
    country       TEXT        NOT NULL,
    city          TEXT,
    latitude      NUMERIC(9,6),
    longitude     NUMERIC(9,6),
    max_draft_m   NUMERIC(5,2)  NOT NULL DEFAULT 0,
    total_berths  INT           NOT NULL DEFAULT 0,
    alert_level   TEXT          NOT NULL DEFAULT 'normal'
                  CHECK (alert_level IN ('normal','elevated','high','critical')),
    status        TEXT          NOT NULL DEFAULT 'operational'
                  CHECK (status IN ('operational','restricted','closed')),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ports IS
    'Master registry of African ports managed within the Kinara OS maritime pillar. '
    'Each port record tracks location, physical capacity, and current operational status.';

-- ---------------------------------------------------------------------------
-- berths
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS berths (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id              UUID        NOT NULL REFERENCES ports(id) ON DELETE CASCADE,
    berth_number         TEXT        NOT NULL,
    status               TEXT        NOT NULL DEFAULT 'available'
                         CHECK (status IN ('available','occupied','maintenance','reserved')),
    max_length_m         NUMERIC(7,2)  NOT NULL DEFAULT 0,
    max_draft_m          NUMERIC(5,2)  NOT NULL DEFAULT 0,
    max_tonnage_t        NUMERIC(10,2) NOT NULL DEFAULT 0,
    berthed_vessel_id    UUID,
    berthed_at           TIMESTAMPTZ,
    estimated_departure  TIMESTAMPTZ,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE berths IS
    'Individual berth slots within a port. Tracks real-time occupancy, physical limits, '
    'and the vessel currently berthed (if any).';

-- ---------------------------------------------------------------------------
-- port_calls
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS port_calls (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id         UUID        NOT NULL REFERENCES ports(id),
    vessel_id       UUID        NOT NULL,
    berth_id        UUID,
    arrival_at      TIMESTAMPTZ,
    departure_at    TIMESTAMPTZ,
    purpose         TEXT
                    CHECK (purpose IN ('loading','unloading','bunkering','crew_change','repair','transit')),
    status          TEXT        NOT NULL DEFAULT 'expected'
                    CHECK (status IN ('expected','arrived','departed','cancelled')),
    cargo_tonnage   NUMERIC(12,2),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE port_calls IS
    'Records every scheduled or completed port call by a vessel. '
    'Links vessels, berths, cargo tonnage, and purpose to support port planning and billing.';

-- ---------------------------------------------------------------------------
-- port_tariffs
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS port_tariffs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    port_id         UUID        NOT NULL REFERENCES ports(id),
    service_type    TEXT
                    CHECK (service_type IN ('berth','pilotage','towage','stevedoring','storage')),
    rate            NUMERIC(12,4) NOT NULL,
    currency        TEXT          NOT NULL DEFAULT 'USD',
    unit            TEXT
                    CHECK (unit IN ('per_day','per_tonne','per_move','per_call')),
    effective_from  DATE,
    effective_until DATE
);

COMMENT ON TABLE port_tariffs IS
    'Tariff schedules for port services. Multiple rates per port are supported, '
    'each scoped to a service type, unit basis, and effective date range.';

-- ---------------------------------------------------------------------------
-- port_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS port_audit_log (
    id              BIGSERIAL   PRIMARY KEY,
    entity_id       UUID,
    action          TEXT        NOT NULL,
    actor_id        TEXT        NOT NULL,
    old_data        JSONB,
    new_data        JSONB,
    signature_hash  TEXT,
    ip_address      INET,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE port_audit_log IS
    'Immutable append-only audit trail for all Port Service write operations. '
    'UPDATE and DELETE are blocked by rules to preserve the integrity of the log.';

CREATE RULE no_update_port_audit AS ON UPDATE TO port_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_port_audit AS ON DELETE TO port_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_port_audit_entity
    ON port_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_port_audit_actor
    ON port_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_ports_country_status
    ON ports(country, status);

CREATE INDEX IF NOT EXISTS idx_berths_port_status
    ON berths(port_id, status);

CREATE INDEX IF NOT EXISTS idx_port_calls_port_arrival
    ON port_calls(port_id, arrival_at);

CREATE INDEX IF NOT EXISTS idx_port_calls_vessel
    ON port_calls(vessel_id);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS port_audit_log CASCADE;
-- DROP TABLE IF EXISTS port_tariffs CASCADE;
-- DROP TABLE IF EXISTS port_calls CASCADE;
-- DROP TABLE IF EXISTS berths CASCADE;
-- DROP TABLE IF EXISTS ports CASCADE;
