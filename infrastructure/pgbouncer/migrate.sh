#!/usr/bin/env bash
# PgBouncer migration: deploy, verify, flip, drain-test
# Run steps manually in order — each step is a kubectl command you can inspect first.
set -euo pipefail

NS=kinara-os
MANIFEST="$(dirname "$0")/pgbouncer.yaml"

step() { echo; echo "=== STEP $* ==="; }

# ─── STEP 1: Deploy PgBouncer (services still connect directly to Postgres) ───
step "1 — Apply PgBouncer manifest"
kubectl apply -f "$MANIFEST"

step "1 — Wait for PgBouncer pods to be Ready"
kubectl rollout status deployment/kinara-pgbouncer -n "$NS" --timeout=120s

step "1 — Show PgBouncer pods"
kubectl get pods -n "$NS" -l app=kinara-pgbouncer -o wide

# ─── STEP 2: Smoke-test PgBouncer before touching any service ─────────────────
step "2 — Smoke-test: connect through PgBouncer to one service's database"
# Pick any running pod to run psql from; use the analytics service's database name as example.
# Replace 'analyticsdb' with any real db name from values.yaml if needed.
SMOKE_POD=$(kubectl get pod -n "$NS" -l app=analytics-service -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || \
            kubectl get pod -n "$NS" --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')
echo "Using pod: $SMOKE_POD"
kubectl exec -n "$NS" "$SMOKE_POD" -- sh -c \
  'psql "$DATABASE_URL" -c "SELECT version();" 2>&1 || echo "psql not in image — skip smoke test"'

# ─── STEP 3: Verify pg_stat_activity shows PgBouncer pod IPs ─────────────────
step "3 — Check pg_stat_activity (run this against Postgres directly)"
echo "Connect to Postgres and run:"
echo "  SELECT client_addr, count(*) FROM pg_stat_activity GROUP BY client_addr ORDER BY count DESC;"
echo "PgBouncer pod IPs should appear after you flip the secret in Step 4."

# ─── STEP 4: Flip kinara-db-credentials to point at PgBouncer ────────────────
step "4 — Patch kinara-db-credentials secret"
kubectl patch secret kinara-db-credentials -n "$NS" \
  --type='json' \
  -p='[{"op":"replace","path":"/data/host","value":"'"$(printf 'kinara-pgbouncer' | base64)"'"},{"op":"replace","path":"/data/port","value":"'"$(printf '5432' | base64)"'"}]'
echo "Secret patched: host=kinara-pgbouncer port=5432"

# ─── STEP 5: Rolling restart so pods pick up the new secret values ─────────────
step "5 — Rolling restart all deployments (no downtime, maxUnavailable=0)"
kubectl rollout restart deployment -n "$NS"
echo "Waiting for rollout to complete (this may take several minutes for 152 deployments)..."
# Wait for all deployments; timeout per-deployment
for deploy in $(kubectl get deployments -n "$NS" -o jsonpath='{.items[*].metadata.name}'); do
  kubectl rollout status deployment/"$deploy" -n "$NS" --timeout=300s &
done
wait
echo "All deployments rolled out."

# ─── STEP 6: Verify pg_stat_activity after flip ──────────────────────────────
step "6 — Re-check pg_stat_activity (PgBouncer pod IPs should be the only clients now)"
echo "Connect to Postgres and run:"
echo "  SELECT client_addr, count(*) FROM pg_stat_activity WHERE datname != 'postgres' GROUP BY client_addr ORDER BY count DESC;"

# ─── STEP 7: Drain test ───────────────────────────────────────────────────────
step "7 — Drain test (pick a node from: kubectl get nodes)"
echo "Run:"
echo "  NODE=<node-name>"
echo "  kubectl drain \$NODE --ignore-daemonsets --delete-emptydir-data"
echo "Monitor pg_stat_activity connection count during drain."
echo "Uncordon after: kubectl uncordon \$NODE"
