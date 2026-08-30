-- =============================================================================
-- Kinara OS — Dock Service
-- Migration : V202609010038__Dock_Service__Init.sql
-- Database  : kinara_dock
-- Description: Initialises the Dock Service schema: dock operations,
--              stevedore teams, equipment usage logs, and an immutable audit log.
-- =============================================================================

\c kinara_dock;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- dock_operations
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS dock_operations (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    operation_ref   TEXT        UNIQUE,
    port_id         UUID        NOT NULL,
    berth_id        UUID,
    vessel_id       UUID,
    operation_type  TEXT
                    CHECK (operation_type IN (
                        'loading','unloading','bunkering','water_supply','waste_collection'
                    )),
    cargo_type      TEXT,
    quantity_tonnes NUMERIC(12,3),
    shift           TEXT
                    CHECK (shift IN ('day','night','weekend')),
    equipment_used  JSONB,
    start_time      TIMESTAMPTZ,
    end_time        TIMESTAMPTZ,
    status          TEXT        NOT NULL DEFAULT 'planned'
                    CHECK (status IN (
                        'planned','in_progress','completed','suspended','cancelled'
                    )),
    supervisor_id   UUID,
    gang_size       INT         NOT NULL DEFAULT 0,
    notes           TEXT,
    tenant_id       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE dock_operations IS
    'Individual dock operation records covering cargo loading, unloading, bunkering, '
    'water supply, and waste collection. Tracks shift, equipment manifest, gang size, '
    'and real-time status for port throughput reporting.';

-- ---------------------------------------------------------------------------
-- stevedore_teams
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stevedore_teams (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_name        TEXT        NOT NULL,
    supervisor_id    UUID,
    port_id          UUID        NOT NULL,
    size             INT         NOT NULL DEFAULT 0,
    specialization   TEXT,
    status           TEXT        NOT NULL DEFAULT 'available'
                     CHECK (status IN ('available','deployed','resting','standby')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE stevedore_teams IS
    'Stevedore gangs available at each port. Tracks team size, specialization '
    '(e.g. bulk grain, containers, tanker operations), and current deployment status.';

-- ---------------------------------------------------------------------------
-- equipment_usage
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS equipment_usage (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    operation_id     UUID        NOT NULL REFERENCES dock_operations(id) ON DELETE CASCADE,
    equipment_type   TEXT
                     CHECK (equipment_type IN (
                         'crane','forklift','conveyor','reach_stacker','side_loader'
                     )),
    equipment_id     TEXT,
    start_time       TIMESTAMPTZ,
    end_time         TIMESTAMPTZ,
    operator_id      UUID
);

COMMENT ON TABLE equipment_usage IS
    'Individual equipment deployment records linked to a dock operation. '
    'Enables equipment utilisation analysis, maintenance scheduling, '
    'and operator accountability tracking.';

-- ---------------------------------------------------------------------------
-- dock_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS dock_audit_log (
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

COMMENT ON TABLE dock_audit_log IS
    'Immutable append-only audit trail for all Dock Service write operations. '
    'UPDATE and DELETE are blocked by rules to preserve port operations records.';

CREATE RULE no_update_dock_audit AS ON UPDATE TO dock_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_dock_audit AS ON DELETE TO dock_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_dock_audit_entity
    ON dock_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_dock_audit_actor
    ON dock_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_dock_ops_port_start
    ON dock_operations(port_id, start_time);

CREATE INDEX IF NOT EXISTS idx_dock_ops_vessel
    ON dock_operations(vessel_id);

CREATE INDEX IF NOT EXISTS idx_dock_ops_status
    ON dock_operations(status);

CREATE INDEX IF NOT EXISTS idx_stevedore_teams_port_status
    ON stevedore_teams(port_id, status);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS dock_audit_log CASCADE;
-- DROP TABLE IF EXISTS equipment_usage CASCADE;
-- DROP TABLE IF EXISTS stevedore_teams CASCADE;
-- DROP TABLE IF EXISTS dock_operations CASCADE;
