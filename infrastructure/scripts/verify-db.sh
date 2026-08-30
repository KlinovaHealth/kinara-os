#!/usr/bin/env bash
# Post-migration verification: checks row counts and flyway_schema_history
# against expected minimums for each service database.
# Exit code 0 = all checks pass; non-zero = one or more failures.
set -euo pipefail

PG_HOST="${POSTGRES_HOST:-localhost}"
PG_PORT="${POSTGRES_PORT:-25060}"
PG_USER="${POSTGRES_USER:-kinara}"
export PGPASSWORD="${POSTGRES_PASSWORD}"

PASS=0
FAIL=0
WARNINGS=()
ERRORS=()

psql_q() {
  psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$1" -t -c "$2" 2>/dev/null | tr -d '[:space:]'
}

check() {
  local label="$1"
  local db="$2"
  local query="$3"
  local expected_min="$4"

  actual=$(psql_q "$db" "$query" 2>/dev/null) || actual="-1"
  if [[ "$actual" -ge "$expected_min" ]] 2>/dev/null; then
    echo "  [OK]  ${label}: ${actual} rows (min ${expected_min})"
    ((PASS++))
  else
    echo "  [FAIL] ${label}: got ${actual}, expected >= ${expected_min}"
    ERRORS+=("${label}: ${actual} < ${expected_min}")
    ((FAIL++))
  fi
}

flyway_check() {
  local db="$1"
  local expected_min="$2"
  local count
  count=$(psql_q "$db" "SELECT COUNT(*) FROM flyway_schema_history WHERE success = true" 2>/dev/null) || count="-1"
  if [[ "$count" -ge "$expected_min" ]] 2>/dev/null; then
    echo "  [OK]  ${db} flyway_schema_history: ${count} applied"
    ((PASS++))
  else
    echo "  [FAIL] ${db} flyway_schema_history: ${count} applied (expected >= ${expected_min})"
    ERRORS+=("flyway/${db}: ${count} applied < ${expected_min}")
    ((FAIL++))
  fi
}

# ── Banner ────────────────────────────────────────────────────────────────
echo "╔══════════════════════════════════════════════════════════╗"
echo "║   Kinara OS — Production Database Verification          ║"
echo "║   Host: ${PG_HOST}:${PG_PORT}"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

# ── Health: kinara_patient ────────────────────────────────────────────────
echo "▸ kinara_patient"
check "clinics"            kinara_patient "SELECT COUNT(*) FROM clinics"           100
check "patients"           kinara_patient "SELECT COUNT(*) FROM patients"          1000
check "active clinics"     kinara_patient "SELECT COUNT(*) FROM clinics WHERE is_active = true" 90
flyway_check kinara_patient 3

# ── Agriculture: kinara_farmer ────────────────────────────────────────────
echo ""
echo "▸ kinara_farmer"
check "farmers"            kinara_farmer  "SELECT COUNT(*) FROM farmers"            500
check "active farmers"     kinara_farmer  "SELECT COUNT(*) FROM farmers WHERE is_active = true" 480
flyway_check kinara_farmer 2

# ── Agriculture: kinara_market ────────────────────────────────────────────
echo ""
echo "▸ kinara_market"
check "market prices"      kinara_market  "SELECT COUNT(*) FROM market_prices"       15
flyway_check kinara_market 2

# ── Agriculture: kinara_cooperative ──────────────────────────────────────
echo ""
echo "▸ kinara_cooperative"
check "cooperatives"       kinara_cooperative "SELECT COUNT(*) FROM cooperatives"    5
flyway_check kinara_cooperative 2

# ── Maritime: kinara_port ─────────────────────────────────────────────────
echo ""
echo "▸ kinara_port"
check "ports"              kinara_port    "SELECT COUNT(*) FROM ports"               10
check "Lomé berths"        kinara_port    "SELECT COUNT(*) FROM berths WHERE port_id = (SELECT id FROM ports WHERE port_code = 'TGLFW')" 12
flyway_check kinara_port 3

# ── Payment: kinara_payment ───────────────────────────────────────────────
echo ""
echo "▸ kinara_payment"
check "FX rates"           kinara_payment "SELECT COUNT(*) FROM fx_rates"            15
check "XOF/USD rate"       kinara_payment "SELECT COUNT(*) FROM fx_rates WHERE from_currency = 'XOF' AND to_currency = 'USD'" 1
flyway_check kinara_payment 3

# ── Notifications: kinara_notification ───────────────────────────────────
echo ""
echo "▸ kinara_notification"
check "templates"          kinara_notification "SELECT COUNT(*) FROM notification_templates WHERE is_active = true" 5
flyway_check kinara_notification 2

# ── SMS: kinara_sms ───────────────────────────────────────────────────────
echo ""
echo "▸ kinara_sms"
# V048 creates the audit log — no seed data, just verify the table exists
check "audit log table"    kinara_sms     "SELECT COUNT(*) FROM sms_audit_logs"      0
flyway_check kinara_sms 1

# ── Auth: kinara_auth ─────────────────────────────────────────────────────
echo ""
echo "▸ kinara_auth"
flyway_check kinara_auth 2

# ── Clinical ──────────────────────────────────────────────────────────────
echo ""
echo "▸ kinara_clinical"
flyway_check kinara_clinical 2

# ── Logistics ─────────────────────────────────────────────────────────────
echo ""
for ldb in kinara_fleet kinara_route kinara_cargo kinara_transport kinara_shipment kinara_lastmile; do
  echo "▸ ${ldb}"
  flyway_check "$ldb" 1
done

# ── Maritime ─────────────────────────────────────────────────────────────
echo ""
for mdb in kinara_vessel kinara_customs kinara_voyage kinara_crew; do
  echo "▸ ${mdb}"
  flyway_check "$mdb" 1
done

# ── Additional health services ────────────────────────────────────────────
echo ""
for hdb in kinara_appointment kinara_immunization kinara_lab kinara_outbreak kinara_pharmacy kinara_referral; do
  echo "▸ ${hdb}"
  flyway_check "$hdb" 1
done

# ── Summary ───────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════"
echo "  Verification complete"
echo "  Passed : ${PASS}"
echo "  Failed : ${FAIL}"

if [[ "${#ERRORS[@]}" -gt 0 ]]; then
  echo ""
  echo "  Failures:"
  for e in "${ERRORS[@]}"; do
    echo "    • ${e}"
  done
  echo ""
  echo "  !! Production database NOT ready for traffic. Fix above errors."
  exit 1
else
  echo ""
  echo "  ✓ All checks passed. Database is ready for the Togo pilot."
  exit 0
fi
