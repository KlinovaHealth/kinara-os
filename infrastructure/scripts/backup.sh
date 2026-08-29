#!/usr/bin/env bash
# Daily backup: dumps all Kinara databases to S3.
# Runs via: cron 0 2 * * * /opt/kinara/scripts/backup.sh
set -euo pipefail

S3_BUCKET="${BACKUP_S3_BUCKET:-kinara-backups}"
S3_PREFIX="${BACKUP_S3_PREFIX:-postgres}"
PG_HOST="${POSTGRES_HOST:-postgres}"
PG_PORT="${POSTGRES_PORT:-5432}"
PG_USER="${POSTGRES_USER:-kinara}"
PGPASSWORD="${POSTGRES_PASSWORD}"
export PGPASSWORD

TIMESTAMP=$(date -u +"%Y-%m-%dT%H-%M-%SZ")
BACKUP_DIR="/tmp/kinara-backup-${TIMESTAMP}"
mkdir -p "$BACKUP_DIR"

DATABASES=(
  kinara_patient
  kinara_cooperative
  kinara_logistics
  kinara_maritime
  kinara_payment
  kinara_analytics
  kinara_audit
)

echo "[backup] Starting Kinara database backup at ${TIMESTAMP}"

for db in "${DATABASES[@]}"; do
  outfile="${BACKUP_DIR}/${db}.sql.gz"
  echo "[backup] Dumping ${db}..."
  pg_dump -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" \
    --format=plain \
    --no-owner \
    --no-acl \
    "$db" | gzip -9 > "$outfile"
  size=$(du -sh "$outfile" | cut -f1)
  echo "[backup] ${db}: ${size} compressed"
done

# Upload manifest
manifest="${BACKUP_DIR}/manifest.json"
cat > "$manifest" <<EOF
{
  "timestamp": "${TIMESTAMP}",
  "host": "${PG_HOST}",
  "databases": $(printf '"%s",' "${DATABASES[@]}" | sed 's/,$//' | sed 's/^/[/' | sed 's/$/]/'),
  "retention_days": 30
}
EOF

echo "[backup] Uploading to s3://${S3_BUCKET}/${S3_PREFIX}/${TIMESTAMP}/"
aws s3 cp "$BACKUP_DIR/" \
  "s3://${S3_BUCKET}/${S3_PREFIX}/${TIMESTAMP}/" \
  --recursive \
  --storage-class STANDARD_IA \
  --quiet

echo "[backup] Pruning backups older than 30 days..."
CUTOFF=$(date -u -d "30 days ago" +"%Y-%m-%dT%H-%M-%SZ" 2>/dev/null || \
         date -u -v-30d +"%Y-%m-%dT%H-%M-%SZ")
aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/" | while read -r _ _ _ prefix; do
  if [[ "$prefix" < "$CUTOFF" ]]; then
    aws s3 rm "s3://${S3_BUCKET}/${S3_PREFIX}/${prefix}" --recursive --quiet
    echo "[backup] Deleted old backup: ${prefix}"
  fi
done

rm -rf "$BACKUP_DIR"
echo "[backup] Complete. Backup stored at s3://${S3_BUCKET}/${S3_PREFIX}/${TIMESTAMP}/"
