#!/usr/bin/env bash
# Kinara OS — PostgreSQL backup script
# Dumps all 44 service databases from the DigitalOcean managed cluster.
# Backups are written to BACKUP_DIR (default: ./backups/YYYY-MM-DD/).
# Requires: pg_dump, psql, .env.prod (or explicit env vars)
#
# Usage:
#   ./scripts/backup-db.sh                    # backup all databases
#   ./scripts/backup-db.sh kinara_patient     # backup single database
#   BACKUP_DIR=/mnt/backups ./scripts/backup-db.sh

set -euo pipefail

# ─── Load credentials from .env.prod if not already set ─────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/../.env.prod"

if [[ -f "$ENV_FILE" ]]; then
  # shellcheck source=/dev/null
  set -a; source "$ENV_FILE"; set +a
fi

DB_HOST="${DB_HOST:?DB_HOST not set}"
DB_PORT="${DB_PORT:-25060}"
DB_USER="${DB_USER:?DB_USER not set}"
DB_PASS="${DB_PASS:?DB_PASS not set}"
export PGPASSWORD="$DB_PASS"

# ─── Config ──────────────────────────────────────────────────────────────────
TIMESTAMP=$(date +%Y-%m-%d)
BACKUP_DIR="${BACKUP_DIR:-$SCRIPT_DIR/../backups/$TIMESTAMP}"
LOG_FILE="$BACKUP_DIR/backup.log"
PG_DUMP_OPTS="--format=custom --compress=9 --no-password"

ALL_DATABASES=(
  kinara_patient kinara_clinical kinara_telemedicine kinara_pharmacy kinara_referral
  kinara_analytics kinara_compliance kinara_governance kinara_appointment kinara_immunization
  kinara_lab kinara_outbreak kinara_input kinara_irrigation kinara_livestock
  kinara_extension kinara_vehicle_tracking kinara_farmer kinara_market kinara_cooperative
  kinara_weather kinara_farmer_finance kinara_supply_chain kinara_transport kinara_warehouse
  kinara_lastmile kinara_shipment kinara_fleet kinara_logistics_analytics kinara_port
  kinara_cargo kinara_customs kinara_shipping kinara_trade_finance kinara_documentation
  kinara_vessel kinara_dock kinara_crew kinara_payment kinara_audit
  kinara_notification kinara_auth kinara_sms kinara_voyage
)

# ─── Helpers ─────────────────────────────────────────────────────────────────
log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a "$LOG_FILE"; }
fail() { log "ERROR: $*"; exit 1; }

# ─── Setup ───────────────────────────────────────────────────────────────────
mkdir -p "$BACKUP_DIR"
log "Kinara OS database backup — $TIMESTAMP"
log "Host: $DB_HOST:$DB_PORT  User: $DB_USER"
log "Backup dir: $BACKUP_DIR"

# connectivity check
psql "postgresql://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/defaultdb?sslmode=require" \
  -c "SELECT 1" -q --no-align -t &>/dev/null \
  || fail "Cannot connect to $DB_HOST:$DB_PORT — check credentials"

# ─── Determine which databases to back up ───────────────────────────────────
if [[ $# -gt 0 ]]; then
  TARGETS=("$@")
else
  TARGETS=("${ALL_DATABASES[@]}")
fi

# ─── Backup loop ─────────────────────────────────────────────────────────────
SUCCEEDED=0
FAILED=0
SKIPPED=0

for DB in "${TARGETS[@]}"; do
  OUT="$BACKUP_DIR/${DB}.pgdump"

  # check database exists
  EXISTS=$(psql "postgresql://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/defaultdb?sslmode=require" \
    -tAc "SELECT 1 FROM pg_database WHERE datname='$DB'" 2>/dev/null)

  if [[ "$EXISTS" != "1" ]]; then
    log "SKIP  $DB (database not found)"
    ((SKIPPED++))
    continue
  fi

  log "START $DB → ${OUT##*/}"
  if pg_dump $PG_DUMP_OPTS \
      --host="$DB_HOST" \
      --port="$DB_PORT" \
      --username="$DB_USER" \
      --dbname="$DB" \
      --file="$OUT" 2>>"$LOG_FILE"; then
    SIZE=$(du -sh "$OUT" 2>/dev/null | cut -f1)
    log "OK    $DB ($SIZE)"
    ((SUCCEEDED++))
  else
    log "FAIL  $DB"
    ((FAILED++))
  fi
done

# ─── Roles dump (cluster-level) ──────────────────────────────────────────────
ROLES_OUT="$BACKUP_DIR/kinara_roles.sql"
log "Dumping roles → ${ROLES_OUT##*/}"
pg_dumpall --globals-only --no-password \
  --host="$DB_HOST" --port="$DB_PORT" --username="$DB_USER" \
  > "$ROLES_OUT" 2>>"$LOG_FILE" \
  && log "OK    roles" \
  || log "WARN  roles dump failed (non-fatal)"

# ─── Summary ─────────────────────────────────────────────────────────────────
TOTAL_SIZE=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)
log "─────────────────────────────────────────"
log "Done: $SUCCEEDED OK  $FAILED FAILED  $SKIPPED SKIPPED  (total $TOTAL_SIZE)"
log "─────────────────────────────────────────"

[[ $FAILED -eq 0 ]]
