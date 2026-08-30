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
  # Core / identity
  kinara_auth
  kinara_audit
  kinara_notification

  # Health
  kinara_patient
  kinara_clinical
  kinara_appointment
  kinara_immunization
  kinara_lab
  kinara_outbreak
  kinara_pharmacy
  kinara_referral
  kinara_telemedicine
  kinara_health_analytics
  kinara_health_compliance

  # Agriculture
  kinara_farmer
  kinara_market
  kinara_cooperative
  kinara_weather
  kinara_input
  kinara_extension
  kinara_irrigation
  kinara_livestock
  kinara_farmer_finance

  # Logistics
  kinara_fleet
  kinara_driver
  kinara_cargo
  kinara_route
  kinara_transport
  kinara_lastmile
  kinara_shipment
  kinara_logistics_analytics
  kinara_vehicle_tracking
  kinara_supply_chain
  kinara_warehouse

  # Maritime
  kinara_port
  kinara_vessel
  kinara_dock
  kinara_cargo_maritime
  kinara_customs
  kinara_shipping
  kinara_crew
  kinara_voyage

  # Trade / finance
  kinara_payment
  kinara_trade_finance
  kinara_documentation
  kinara_compliance

  # Analytics / governance
  kinara_analytics
  kinara_governance

  # SMS
  kinara_sms
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
