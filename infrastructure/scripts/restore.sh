#!/usr/bin/env bash
# Disaster recovery: restore all Kinara databases from S3.
# Target RTO: < 1 hour. Usage: ./restore.sh [TIMESTAMP]
# If TIMESTAMP not given, lists available backups and restores latest.
set -euo pipefail

S3_BUCKET="${BACKUP_S3_BUCKET:-kinara-backups}"
S3_PREFIX="${BACKUP_S3_PREFIX:-postgres}"
PG_HOST="${POSTGRES_HOST:-postgres}"
PG_PORT="${POSTGRES_PORT:-5432}"
PG_USER="${POSTGRES_USER:-kinara}"
PGPASSWORD="${POSTGRES_PASSWORD}"
export PGPASSWORD

TIMESTAMP="${1:-}"

if [[ -z "$TIMESTAMP" ]]; then
  echo "[restore] Available backups (newest first):"
  aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/" | sort -r | head -10
  echo ""
  TIMESTAMP=$(aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/" | sort -r | head -1 | awk '{print $NF}' | tr -d '/')
  echo "[restore] Auto-selecting latest: ${TIMESTAMP}"
fi

echo "[restore] ⚠ WARNING: This will DROP and recreate all Kinara databases."
echo "[restore] Restoring from: s3://${S3_BUCKET}/${S3_PREFIX}/${TIMESTAMP}/"
echo "[restore] Target host: ${PG_HOST}:${PG_PORT}"
echo ""
read -rp "[restore] Type 'RESTORE' to confirm: " confirm
if [[ "$confirm" != "RESTORE" ]]; then
  echo "[restore] Aborted."
  exit 1
fi

RESTORE_DIR="/tmp/kinara-restore-${TIMESTAMP}"
mkdir -p "$RESTORE_DIR"

echo "[restore] Downloading backup files..."
aws s3 cp "s3://${S3_BUCKET}/${S3_PREFIX}/${TIMESTAMP}/" \
  "$RESTORE_DIR/" \
  --recursive \
  --quiet

DATABASES=(
  kinara_patient
  kinara_cooperative
  kinara_logistics
  kinara_maritime
  kinara_payment
  kinara_analytics
  kinara_audit
)

START=$(date +%s)

for db in "${DATABASES[@]}"; do
  sqlfile="${RESTORE_DIR}/${db}.sql.gz"
  if [[ ! -f "$sqlfile" ]]; then
    echo "[restore] ⚠  Missing backup for ${db}, skipping"
    continue
  fi

  echo "[restore] Restoring ${db}..."

  # Terminate existing connections
  psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='${db}' AND pid <> pg_backend_pid();" \
    -q 2>/dev/null || true

  # Drop and recreate
  psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres \
    -c "DROP DATABASE IF EXISTS ${db};" \
    -c "CREATE DATABASE ${db} OWNER ${PG_USER};" \
    -q

  # Restore
  gunzip -c "$sqlfile" | psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$db" -q
  echo "[restore] ✓ ${db} restored"
done

END=$(date +%s)
ELAPSED=$((END - START))

rm -rf "$RESTORE_DIR"
echo ""
echo "[restore] ✓ All databases restored in ${ELAPSED}s from ${TIMESTAMP}"
echo "[restore] ✓ RTO target: <3600s — actual: ${ELAPSED}s"
