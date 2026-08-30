#!/usr/bin/env bash
# Creates the kinara-db-credentials Kubernetes secret in the kinara-togo namespace.
# Reads from .env.prod — run provision.py and create_db_users.sql first.
#
# Usage:
#   source .env.prod  # loads DB_HOST, DB_PORT, DATABASE_URL, etc.
#   ./infrastructure/scripts/k8s-secrets.sh
set -euo pipefail

NS="${K8S_NAMESPACE:-kinara-togo}"

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "[error] DATABASE_URL not set. Run: source .env.prod" >&2
  exit 1
fi

echo "Creating/updating Kubernetes secret in namespace: $NS"

kubectl create secret generic kinara-db-credentials \
  --from-literal=DATABASE_URL="${DATABASE_URL}" \
  --from-literal=DB_HOST="${DB_HOST}" \
  --from-literal=DB_PORT="${DB_PORT:-25060}" \
  --from-literal=APP_USER_PASSWORD="${APP_USER_PASSWORD}" \
  --from-literal=READ_ONLY_PASSWORD="${READ_ONLY_PASSWORD}" \
  --from-literal=BACKUP_USER_PASSWORD="${BACKUP_USER_PASSWORD}" \
  --namespace="${NS}" \
  --save-config \
  --dry-run=client \
  -o yaml | kubectl apply -f -

echo "[done] secret 'kinara-db-credentials' applied to namespace '${NS}'"
kubectl get secret kinara-db-credentials -n "${NS}" --no-headers
