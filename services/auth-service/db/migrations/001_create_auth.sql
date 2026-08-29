-- Kinara Governance OS — Auth Service schema
-- Handles identity, RBAC, API keys, sessions, MFA, and immutable access logs.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── Users ────────────────────────────────────────────────────────────────────

CREATE TABLE users (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username       TEXT         NOT NULL UNIQUE,
    email          TEXT         NOT NULL UNIQUE,
    password_hash  TEXT         NOT NULL,
    status         TEXT         NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active','inactive','suspended')),
    email_verified BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_login_at  TIMESTAMPTZ
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email    ON users(email);
CREATE INDEX idx_users_status   ON users(status);

-- ─── User Profiles ────────────────────────────────────────────────────────────

CREATE TABLE user_profiles (
    user_id          UUID        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    full_name_enc    TEXT        NOT NULL,
    department_enc   TEXT,
    phone_enc        TEXT,
    country          TEXT        NOT NULL DEFAULT '',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Roles ────────────────────────────────────────────────────────────────────

CREATE TABLE roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_roles_name ON roles(name);

-- ─── Permissions ──────────────────────────────────────────────────────────────

CREATE TABLE permissions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,
    description TEXT        NOT NULL DEFAULT '',
    resource    TEXT        NOT NULL,
    action      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_permissions_resource_action ON permissions(resource, action);

-- ─── Role ↔ Permission mapping ────────────────────────────────────────────────

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ─── User ↔ Role mapping ──────────────────────────────────────────────────────

CREATE TABLE user_roles (
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_by UUID,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

-- ─── API Keys ─────────────────────────────────────────────────────────────────
-- key_hash stores SHA-256(plaintext_key). The plaintext is never stored.

CREATE TABLE api_keys (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    key_hash     TEXT        NOT NULL UNIQUE,
    permissions  TEXT[]      NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_user_id  ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

-- ─── Sessions ─────────────────────────────────────────────────────────────────
-- refresh_token_hash stores SHA-256(plaintext_refresh_token).

CREATE TABLE sessions (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT        NOT NULL UNIQUE,
    mfa_verified        BOOLEAN     NOT NULL DEFAULT FALSE,
    ip_address          TEXT        NOT NULL,
    user_agent          TEXT        NOT NULL DEFAULT '',
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id            ON sessions(user_id);
CREATE INDEX idx_sessions_refresh_token_hash ON sessions(refresh_token_hash);
CREATE INDEX idx_sessions_expires_at         ON sessions(expires_at);

-- ─── MFA Devices ──────────────────────────────────────────────────────────────
-- secret_enc stores AES-256-GCM encrypted TOTP secret.

CREATE TABLE mfa_devices (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT        NOT NULL CHECK (type IN ('totp')),
    name       TEXT        NOT NULL DEFAULT 'Authenticator App',
    secret_enc TEXT        NOT NULL,
    verified   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mfa_devices_user_id ON mfa_devices(user_id);

-- ─── Access Log ───────────────────────────────────────────────────────────────
-- Immutable: UPDATE and DELETE are blocked by PostgreSQL rules.

CREATE TABLE access_log (
    id         UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID,
    action     TEXT            NOT NULL,
    resource   TEXT            NOT NULL DEFAULT '',
    status     TEXT            NOT NULL CHECK (status IN ('success','failure','denied')),
    ip_address TEXT            NOT NULL,
    user_agent TEXT            NOT NULL DEFAULT '',
    details    TEXT            NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_log_user_id    ON access_log(user_id);
CREATE INDEX idx_access_log_status     ON access_log(status);
CREATE INDEX idx_access_log_created_at ON access_log(created_at DESC);
CREATE INDEX idx_access_log_ip         ON access_log(ip_address);

CREATE RULE access_log_no_update AS ON UPDATE TO access_log DO INSTEAD NOTHING;
CREATE RULE access_log_no_delete AS ON DELETE TO access_log DO INSTEAD NOTHING;

-- ─── Seed system roles ────────────────────────────────────────────────────────

INSERT INTO roles (name, description) VALUES
    ('admin',              'Full system access across all pillars'),
    ('clinician',          'Read and write patient clinical data'),
    ('nurse',              'Create and update patient and clinical records'),
    ('doctor',             'Full clinical access including diagnoses and prescriptions'),
    ('patient',            'Read own patient record'),
    ('frontdesk',          'Register patients and schedule consultations'),
    ('analyst',            'Read-only access to aggregate data and audit logs'),
    ('government',         'Epidemiology, compliance, and governance access'),
    ('ministry_official',  'Compliance report submission and review'),
    ('facility_admin',     'Manage facility users and reports'),
    ('farmer',             'Access own farm and cooperative data'),
    ('cooperative_manager','Manage cooperative and member farm data'),
    ('logistics',          'Fleet, warehouse, and delivery operations'),
    ('fleet_operator',     'Manage vehicles and routes'),
    ('port_operator',      'Port operations and berth scheduling'),
    ('customs_officer',    'Customs clearance and documentation'),
    ('pharmacist',         'Dispense prescriptions and manage drug inventory'),
    ('system',             'Inter-service machine identity (mTLS only)');
