-- =============================================================================
-- Kinara OS — Vessel Service
-- Migration : V202609010037__Vessel_Service__Init.sql
-- Database  : kinara_vessel
-- Description: Initialises the Vessel Service schema: vessel registry,
--              statutory certificates, AIS/manual position tracking,
--              and an immutable audit log.
-- =============================================================================

\c kinara_vessel;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- vessels
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS vessels (
    id                      UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT           NOT NULL,
    imo_number              TEXT           UNIQUE,
    vessel_type             TEXT
                            CHECK (vessel_type IN (
                                'bulk_carrier','container','tanker','ro_ro',
                                'ferry','fishing','tug','general_cargo'
                            )),
    flag_state              TEXT,
    gross_tonnage           NUMERIC(12,3),
    deadweight_tonnage      NUMERIC(12,3),
    length_overall_m        NUMERIC(8,2),
    beam_m                  NUMERIC(7,2),
    draft_m                 NUMERIC(5,2),
    build_year              INT,
    classification_society  TEXT,
    status                  TEXT           NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','laid_up','scrapped','under_repair')),
    current_port_id         UUID,
    owner_id                UUID,
    operator_id             UUID,
    created_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE vessels IS
    'Master registry of vessels operating within the Kinara OS maritime pillar. '
    'Each vessel is identified by its IMO number and tracks physical dimensions, '
    'flag state, classification society, and current status. Supports all vessel '
    'types active in African coastal and inland waterway trade.';

-- ---------------------------------------------------------------------------
-- vessel_certificates
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS vessel_certificates (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    vessel_id          UUID        NOT NULL REFERENCES vessels(id) ON DELETE CASCADE,
    cert_type          TEXT
                       CHECK (cert_type IN (
                           'seaworthiness','safety_management','pollution_prevention',
                           'load_line','tonnage'
                       )),
    cert_number        TEXT,
    issuing_authority  TEXT,
    issued_at          DATE,
    expires_at         DATE,
    status             TEXT        NOT NULL DEFAULT 'valid'
                       CHECK (status IN ('valid','expired','suspended','cancelled'))
);

COMMENT ON TABLE vessel_certificates IS
    'Statutory and class certificates held by each vessel. '
    'Expiry tracking enables proactive renewal alerts and port state control compliance checks.';

-- ---------------------------------------------------------------------------
-- vessel_positions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS vessel_positions (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    vessel_id    UUID           NOT NULL REFERENCES vessels(id),
    latitude     NUMERIC(9,6),
    longitude    NUMERIC(9,6),
    speed_knots  NUMERIC(5,2),
    heading_deg  NUMERIC(5,1),
    recorded_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    source       TEXT           NOT NULL DEFAULT 'ais'
                 CHECK (source IN ('ais','manual','satellite'))
);

COMMENT ON TABLE vessel_positions IS
    'Time-series vessel position records ingested from AIS transponders, '
    'satellite tracking, or manual entry. Supports voyage reconstruction, '
    'ETA calculations, and real-time maritime traffic monitoring.';

-- ---------------------------------------------------------------------------
-- vessel_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS vessel_audit_log (
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

COMMENT ON TABLE vessel_audit_log IS
    'Immutable append-only audit trail for all Vessel Service write operations. '
    'UPDATE and DELETE are blocked by rules to meet maritime authority record-keeping standards.';

CREATE RULE no_update_vessel_audit AS ON UPDATE TO vessel_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_vessel_audit AS ON DELETE TO vessel_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_vessel_audit_entity
    ON vessel_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_vessel_audit_actor
    ON vessel_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_vessels_imo
    ON vessels(imo_number);

CREATE INDEX IF NOT EXISTS idx_vessels_status
    ON vessels(status);

CREATE INDEX IF NOT EXISTS idx_vessel_certs_vessel_expiry
    ON vessel_certificates(vessel_id, expires_at);

CREATE INDEX IF NOT EXISTS idx_vessel_positions_vessel_time
    ON vessel_positions(vessel_id, recorded_at DESC);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS vessel_audit_log CASCADE;
-- DROP TABLE IF EXISTS vessel_positions CASCADE;
-- DROP TABLE IF EXISTS vessel_certificates CASCADE;
-- DROP TABLE IF EXISTS vessels CASCADE;
