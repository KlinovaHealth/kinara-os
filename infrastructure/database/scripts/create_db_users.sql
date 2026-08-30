-- Kinara OS — Limited database users for production
-- Run as the admin user after migrations are applied:
--   psql $DATABASE_URL -f infrastructure/database/scripts/create_db_users.sql
--
-- Passwords are passed via psql variables so they never appear in this file:
--   psql $DATABASE_URL \
--        -v app_pw="$APP_USER_PASSWORD" \
--        -v ro_pw="$READ_ONLY_PASSWORD" \
--        -v bk_pw="$BACKUP_USER_PASSWORD" \
--        -f infrastructure/database/scripts/create_db_users.sql

-- ── app_user: read/write for all services ─────────────────────────────────────
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
    EXECUTE format('CREATE USER app_user WITH PASSWORD %L', :'app_pw');
  END IF;
END
$$;

GRANT CONNECT ON DATABASE kinara_auth         TO app_user;
GRANT CONNECT ON DATABASE kinara_audit        TO app_user;
GRANT CONNECT ON DATABASE kinara_notification TO app_user;
GRANT CONNECT ON DATABASE kinara_patient      TO app_user;
GRANT CONNECT ON DATABASE kinara_farmer       TO app_user;
GRANT CONNECT ON DATABASE kinara_market       TO app_user;
GRANT CONNECT ON DATABASE kinara_payment      TO app_user;
GRANT CONNECT ON DATABASE kinara_port         TO app_user;
GRANT CONNECT ON DATABASE kinara_cooperative  TO app_user;
GRANT CONNECT ON DATABASE kinara_sms          TO app_user;

-- Grant on current DB (run once per DB via \c switching — handled by run-migrations.sh)
GRANT USAGE ON SCHEMA public TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT USAGE ON SEQUENCES TO app_user;

-- ── read_only: analytics + reporting ─────────────────────────────────────────
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'read_only') THEN
    EXECUTE format('CREATE USER read_only WITH PASSWORD %L', :'ro_pw');
  END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO read_only;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO read_only;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO read_only;

-- ── backup_user: pg_dump only ─────────────────────────────────────────────────
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'backup_user') THEN
    EXECUTE format('CREATE USER backup_user WITH PASSWORD %L', :'bk_pw');
  END IF;
END
$$;

GRANT CONNECT ON DATABASE kinara_auth         TO backup_user;
GRANT CONNECT ON DATABASE kinara_patient      TO backup_user;
GRANT CONNECT ON DATABASE kinara_farmer       TO backup_user;
GRANT CONNECT ON DATABASE kinara_market       TO backup_user;
GRANT CONNECT ON DATABASE kinara_payment      TO backup_user;
GRANT CONNECT ON DATABASE kinara_port         TO backup_user;
GRANT CONNECT ON DATABASE kinara_cooperative  TO backup_user;
GRANT CONNECT ON DATABASE kinara_notification TO backup_user;
GRANT CONNECT ON DATABASE kinara_sms          TO backup_user;
GRANT CONNECT ON DATABASE kinara_audit        TO backup_user;
GRANT USAGE ON SCHEMA public TO backup_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO backup_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO backup_user;
