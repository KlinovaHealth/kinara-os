-- =============================================================================
-- Kinara OS — Crew Service
-- Migration : V202609010039__Crew_Service__Init.sql
-- Database  : kinara_crew
-- Description: Initialises the Crew Service schema: crew member profiles,
--              STCW/medical certifications, vessel assignments,
--              and an immutable audit log.
-- =============================================================================

\c kinara_crew;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- crew_members
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS crew_members (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_ref           TEXT        UNIQUE,
    full_name          TEXT        NOT NULL,
    nationality        TEXT,
    rank               TEXT
                       CHECK (rank IN (
                           'captain','chief_officer','second_officer',
                           'chief_engineer','engineer','boatswain',
                           'able_seaman','cook','cadet'
                       )),
    certification_no   TEXT,
    passport_no_enc    TEXT,      -- stored encrypted at application layer
    seaman_book_no     TEXT,
    years_experience   INT         NOT NULL DEFAULT 0,
    vessel_id          UUID,
    status             TEXT        NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active','on_leave','off_duty','retired','suspended')),
    tenant_id          TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE crew_members IS
    'Profiles for seafarers operating on vessels within the Kinara OS maritime pillar. '
    'Passport numbers are stored encrypted (passport_no_enc) at the application layer '
    'to protect PII. Rank and certification tracking supports STCW compliance workflows.';

-- ---------------------------------------------------------------------------
-- crew_certifications
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS crew_certifications (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id    UUID        NOT NULL REFERENCES crew_members(id) ON DELETE CASCADE,
    cert_type         TEXT
                      CHECK (cert_type IN (
                          'stcw','medical','survival_craft','fire_fighting',
                          'oil_tanker','passenger','basic_safety'
                      )),
    cert_number       TEXT,
    issued_by         TEXT,
    issued_at         DATE,
    expires_at        DATE,
    status            TEXT        NOT NULL DEFAULT 'valid'
                      CHECK (status IN ('valid','expired','suspended'))
);

COMMENT ON TABLE crew_certifications IS
    'STCW and flag-state certifications held by individual crew members. '
    'Expiry dates are indexed to drive automated renewal reminders and '
    'flag any crew whose certificates have lapsed before a vessel departs.';

-- ---------------------------------------------------------------------------
-- crew_assignments
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS crew_assignments (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID        NOT NULL REFERENCES crew_members(id),
    vessel_id      UUID        NOT NULL,
    rank           TEXT,
    signed_on      TIMESTAMPTZ,
    signed_off     TIMESTAMPTZ,
    voyage_ref     TEXT
);

COMMENT ON TABLE crew_assignments IS
    'Historical and current vessel assignments for each crew member. '
    'Captures sign-on/sign-off timestamps and the voyage reference '
    'to reconstruct a seafarer''s service record.';

-- ---------------------------------------------------------------------------
-- crew_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS crew_audit_log (
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

COMMENT ON TABLE crew_audit_log IS
    'Immutable append-only audit trail for all Crew Service write operations. '
    'UPDATE and DELETE are blocked by rules to preserve seafarer records for '
    'maritime authority inspection.';

CREATE RULE no_update_crew_audit AS ON UPDATE TO crew_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_crew_audit AS ON DELETE TO crew_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_crew_audit_entity
    ON crew_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_crew_audit_actor
    ON crew_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_crew_members_vessel_status
    ON crew_members(vessel_id, status);

CREATE INDEX IF NOT EXISTS idx_crew_certs_member_expiry
    ON crew_certifications(crew_member_id, expires_at);

CREATE INDEX IF NOT EXISTS idx_crew_assignments_vessel
    ON crew_assignments(vessel_id);

CREATE INDEX IF NOT EXISTS idx_crew_assignments_member
    ON crew_assignments(crew_member_id);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS crew_audit_log CASCADE;
-- DROP TABLE IF EXISTS crew_assignments CASCADE;
-- DROP TABLE IF EXISTS crew_certifications CASCADE;
-- DROP TABLE IF EXISTS crew_members CASCADE;
