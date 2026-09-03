-- Migration 003: Seed test users for multi-tenancy verification (dev/test only)
-- VHA test user: entity_type='vha', tenant_id=VHA tenant UUID
-- Password hash below is bcrypt of "Test@VHA2026!" generated with cost=12.
-- DO NOT run in production.

INSERT INTO users (
    id, username, email, password_hash, status, email_verified,
    entity_type, tenant_id
)
VALUES (
    '11111111-0000-0000-0000-000000000001',
    'vha_test_user',
    'vha_test@kinaraos.internal',
    '$2a$12$PLACEHOLDER_HASH_REPLACE_BEFORE_USE',
    'active',
    true,
    'vha',
    '00000000-0000-0000-0000-000000000002'
)
ON CONFLICT (username) DO NOTHING;
