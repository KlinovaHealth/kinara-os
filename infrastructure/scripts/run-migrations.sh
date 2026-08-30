#!/usr/bin/env bash
# Applies all Kinara DB migrations in order using psql.
# Migrations use \c <db> switches so they cannot run through Flyway JDBC.
# This script respects the Flyway versioned naming convention and tracks
# which files have already been applied in a simple local state file.
#
# Usage:
#   export POSTGRES_HOST=<do-db-host>
#   export POSTGRES_PORT=25060
#   export POSTGRES_USER=kinara
#   export POSTGRES_PASSWORD=<password>
#   ./infrastructure/scripts/run-migrations.sh [--dry-run]
set -euo pipefail

PG_HOST="${POSTGRES_HOST:-localhost}"
PG_PORT="${POSTGRES_PORT:-5432}"
PG_USER="${POSTGRES_USER:-kinara}"
export PGPASSWORD="${POSTGRES_PASSWORD}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MIGRATIONS_DIR="${REPO_ROOT}/infrastructure/database/migrations"
SEEDS_DIR="${REPO_ROOT}/infrastructure/flyway/seeds"
STATE_FILE="${REPO_ROOT}/.migration-state"

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=true
  echo "[migrate] DRY RUN — no SQL will be executed"
fi

touch "$STATE_FILE"

run_sql() {
  local file="$1"
  if $DRY_RUN; then
    echo "[migrate]   (dry-run) would apply: $(basename "$file")"
    return 0
  fi
  psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" \
    --set=ON_ERROR_STOP=on \
    -f "$file" 2>&1 | sed "s/^/[psql] /"
}

apply_dir() {
  local dir="$1"
  local label="$2"
  echo ""
  echo "▸ ${label} (${dir})"

  if [[ ! -d "$dir" ]]; then
    echo "  [skip] directory not found"
    return
  fi

  local files
  mapfile -t files < <(find "$dir" -maxdepth 1 -name 'V*.sql' | sort)

  if [[ "${#files[@]}" -eq 0 ]]; then
    echo "  [skip] no migration files found"
    return
  fi

  for f in "${files[@]}"; do
    fname="$(basename "$f")"
    if grep -qxF "$fname" "$STATE_FILE" 2>/dev/null; then
      echo "  [skip] already applied: ${fname}"
      continue
    fi

    echo "  [apply] ${fname}"
    run_sql "$f"

    if ! $DRY_RUN; then
      echo "$fname" >> "$STATE_FILE"
    fi
    echo "  [done]  ${fname}"
  done
}

echo "╔══════════════════════════════════════════════════════════╗"
echo "║   Kinara OS — psql Migration Runner                     ║"
echo "║   Host: ${PG_HOST}:${PG_PORT}"
if $DRY_RUN; then
echo "║   Mode: DRY RUN"
fi
echo "╚══════════════════════════════════════════════════════════╝"

apply_dir "$MIGRATIONS_DIR" "Schema migrations (V001-V049)"
apply_dir "$SEEDS_DIR"      "Seed data (V100+)"

echo ""
echo "[migrate] All migrations complete."
if ! $DRY_RUN; then
  echo "[migrate] Applied migration log: ${STATE_FILE}"
fi
