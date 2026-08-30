-- =============================================================================
-- Kinara OS — Auth Service
-- Migration : V202609010045__Auth_Service__Init.sql
-- Database  : kinara_auth
-- Description: Initialises the Auth Service schema: users, JWT refresh tokens,
--              login attempt tracking, API keys, and an immutable audit log.
-- =============================================================================

\c kinara_auth;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- users
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email               TEXT        UNIQUE NOT NULL,
    phone_enc           TEXT,       -- stored encrypted at application layer
    full_name           TEXT        NOT NULL,
    role                TEXT        NOT NULL DEFAULT 'patient'
                        CHECK (role IN (
                            'patient','farmer','doctor','nurse','frontdesk',
                            'admin','epidemiologist','vet','driver','dispatcher',
                            'logistics_manager','port_officer','customs_officer',
                            'government_official','kinara_team'
                        )),
    status              TEXT        NOT NULL DEFAULT 'active'
                        CHECK (status IN (
                            'active','suspended','pending_verification','deactivated'
                        )),
    last_login_at       TIMESTAMPTZ,
    mfa_enabled         BOOLEAN     NOT NULL DEFAULT false,
    mfa_secret_enc      TEXT,       -- TOTP secret stored encrypted at application layer
    country             TEXT,
    language            TEXT        NOT NULL DEFAULT 'en',
    tenant_id           TEXT        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE users IS
    'Central user directory for all Kinara OS roles. '
    'phone_enc and mfa_secret_enc are stored ciphertext — encryption/decryption '
    'is handled exclusively at the application layer using a KMS-managed key. '
    'Supports 15 role types spanning all four pillars (health, agri, logistics, maritime) '
    'plus platform administration. tenant_id enforces data isolation.';

-- ---------------------------------------------------------------------------
-- refresh_tokens
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   TEXT        UNIQUE NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    ip_address   INET,
    user_agent   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE refresh_tokens IS
    'Hashed JWT refresh tokens issued during authentication. '
    'Only the SHA-256 hash of the raw token is persisted, never the raw token itself. '
    'Revocation sets revoked_at; the auth service rejects any token where '
    'revoked_at IS NOT NULL or expires_at < NOW().';

-- ---------------------------------------------------------------------------
-- login_attempts
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS login_attempts (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        REFERENCES users(id) ON DELETE CASCADE,
    ip_address      INET,
    success         BOOLEAN     NOT NULL,
    failure_reason  TEXT,
    attempted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE login_attempts IS
    'Login attempt log used for brute-force detection and security alerting. '
    'Consecutive failures from a single IP or for a single user_id trigger '
    'progressive backoff and account lockout at the application layer. '
    'user_id may be NULL when an unknown email address is submitted.';

-- ---------------------------------------------------------------------------
-- api_keys
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS api_keys (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash      TEXT        UNIQUE NOT NULL,
    owner_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description   TEXT,
    scopes        TEXT[]      NOT NULL DEFAULT '{}',
    last_used_at  TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ,
    status        TEXT        NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','revoked')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE api_keys IS
    'Hashed API keys for service-to-service authentication and third-party integrations. '
    'scopes is an array of permission strings (e.g. ARRAY[''health:read'',''agri:write'']). '
    'Only the hash of the raw key is stored; the raw key is shown to the user exactly once '
    'at creation time and never persisted.';

-- ---------------------------------------------------------------------------
-- auth_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth_audit_log (
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

COMMENT ON TABLE auth_audit_log IS
    'Immutable append-only audit trail for all Auth Service write operations. '
    'Critical for security investigations, SOC 2 compliance, and forensic analysis '
    'of unauthorised access events. UPDATE and DELETE are blocked by rules.';

CREATE RULE no_update_auth_audit AS ON UPDATE TO auth_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_auth_audit AS ON DELETE TO auth_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_auth_audit_entity
    ON auth_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_auth_audit_actor
    ON auth_audit_log(actor_id, occurred_at);

-- Tracks who logged in / performed privileged actions
CREATE INDEX IF NOT EXISTS idx_auth_audit_log_actor
    ON auth_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_users_email
    ON users(email);

CREATE INDEX IF NOT EXISTS idx_users_role_status
    ON users(role, status);

CREATE INDEX IF NOT EXISTS idx_users_tenant
    ON users(tenant_id);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user
    ON refresh_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expiry
    ON refresh_tokens(expires_at);

CREATE INDEX IF NOT EXISTS idx_login_attempts_user_time
    ON login_attempts(user_id, attempted_at DESC);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS auth_audit_log CASCADE;
-- DROP TABLE IF EXISTS api_keys CASCADE;
-- DROP TABLE IF EXISTS login_attempts CASCADE;
-- DROP TABLE IF EXISTS refresh_tokens CASCADE;
-- DROP TABLE IF EXISTS users CASCADE;
