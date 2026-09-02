# Disaster Recovery Runbook — KinaraOS

**Last verified:** 2026-09-02  
**Tested by:** Production DR drill — backup: 2026-09-01 20:09:28 UTC, restore cluster: `kinara-prod-restore-test`

---

## Recovery Time Objective (RTO)

| Metric | Value |
|---|---|
| **Measured RTO** | **4 minutes 0 seconds (240 s)** |
| Backup timestamp used | 2026-09-01 20:09:28 UTC |
| Backup size | 1.263 GB |
| Restore target size | `db-s-2vcpu-4gb`, 60 GiB, 1 node, fra1 |
| Time to `online` status | 240 s from `doctl databases create` call |

> Note: RTO covers **database restore only**. Full service restoration (Kubernetes rolling update of 152 services, DNS cutover if applicable) adds additional time and must be measured separately.

---

## Architecture Overview

| Component | Details |
|---|---|
| Database cluster | `kinara-prod` — DigitalOcean managed PostgreSQL 15 |
| Cluster ID | `c3ece802-0929-4e9e-aa33-95906e9f4a4e` |
| Region | fra1 (Frankfurt) |
| Node size | `db-s-2vcpu-4gb`, 60 GiB disk |
| Database count | 145 `kinara_*` databases (one per microservice) |
| Backup schedule | Automated daily, retained 7 days |
| Connection proxy | PgBouncer (2 replicas in `kinara-production` namespace) |
| Port | **25060** (DO managed DBs do not use 5432) |

---

## Backup Schedule and Retention

DigitalOcean managed databases take automated daily backups. To view current backups:

```bash
doctl databases backups c3ece802-0929-4e9e-aa33-95906e9f4a4e
```

Example output (as of 2026-09-01):

```
Size in Gigabytes    Created At
0.033598             2026-08-30 07:51:38 +0000 UTC   # schema-only, pre-seed
1.263504             2026-08-30 20:10:14 +0000 UTC
1.263381             2026-08-31 20:11:08 +0000 UTC
1.263422             2026-09-01 20:09:28 +0000 UTC   # latest
```

---

## Restore Procedure

### Step 1 — Identify the target backup

```bash
doctl databases backups c3ece802-0929-4e9e-aa33-95906e9f4a4e
```

Select the most recent backup with size > 1 GB (smaller entries are schema-only pre-seed snapshots).

### Step 2 — Create a restore cluster

> **CRITICAL: Never restore over the production cluster. Always restore to a new cluster and validate before any cutover.**

```bash
# Record start time for RTO measurement
RESTORE_START=$(date +%s)
echo "Restore started: $(date -u)"

doctl databases create kinara-prod-YYYYMMDD-restore \
  --engine pg \
  --version 15 \
  --region fra1 \
  --size db-s-2vcpu-4gb \
  --num-nodes 1 \
  --restore-from-cluster-name kinara-prod \
  --restore-from-timestamp "YYYY-MM-DD HH:MM:SS +0000 UTC" \
  --output json | tee /tmp/restore_output.json
```

Replace `YYYY-MM-DD HH:MM:SS` with the backup timestamp from Step 1.  
Replace `YYYYMMDD` in the cluster name with today's date.

**Size constraint:** The restore cluster must be `db-s-2vcpu-4gb` or larger. `db-s-1vcpu-1gb` (max 50 GiB) cannot hold the 60 GiB production disk allocation.

### Step 3 — Wait for the cluster to go online

The cluster will initially show status `forking`. Poll until `online`:

```bash
RESTORE_ID=$(cat /tmp/restore_output.json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d[0]['id'])")

watch -n 30 "doctl databases get $RESTORE_ID --output json | \
  python3 -c \"import sys,json; d=json.load(sys.stdin); print(d[0]['status'])\""
```

Record elapsed time when `online` appears. **Expected: ~4 minutes.**

```bash
RESTORE_END=$(date +%s)
echo "RTO: $((RESTORE_END - RESTORE_START)) seconds"
```

### Step 4 — Retrieve connection credentials

```bash
doctl databases get $RESTORE_ID --output json | \
  python3 -c "import sys,json; d=json.load(sys.stdin); conn=d[0]['connection']; print(conn['uri'])"
```

### Step 5 — Validate restored data

Connect and verify key row counts:

```bash
RESTORE_HOST="<host-from-step-4>"
RESTORE_PASS="<password-from-step-4>"

for DB_TABLE in \
  "kinara_patient:patients" \
  "kinara_patient:clinics" \
  "kinara_farmer:farmers" \
  "kinara_referral:referrals" \
  "kinara_notification:notification_templates"; do
  DB="${DB_TABLE%%:*}"
  TABLE="${DB_TABLE##*:}"
  COUNT=$(PGPASSWORD="$RESTORE_PASS" psql \
    "postgresql://doadmin@${RESTORE_HOST}:25060/${DB}?sslmode=require" \
    -t -c "SELECT count(*) FROM ${TABLE}" 2>/dev/null | tr -d ' ')
  echo "$DB.$TABLE: $COUNT"
done
```

Also verify total database count equals 145:

```bash
PGPASSWORD="$RESTORE_PASS" psql \
  "postgresql://doadmin@${RESTORE_HOST}:25060/defaultdb?sslmode=require" \
  -t -c "SELECT count(*) FROM pg_database WHERE datname LIKE 'kinara_%'"
```

### Step 6 — Point services at restored cluster (if performing full cutover)

Update the `kinara-db-credentials` Kubernetes secret to point to the restore cluster:

```bash
kubectl create secret generic kinara-db-credentials \
  -n kinara-production \
  --from-literal=host=<new-host> \
  --from-literal=port=25060 \
  --from-literal=user=doadmin \
  --from-literal=pass=<new-password> \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart PgBouncer to pick up new credentials
kubectl rollout restart deployment/kinara-pgbouncer -n kinara-production

# Rolling restart all 152 services to flush connection pools
helm upgrade kinara-production infrastructure/helm/kinara-os \
  --namespace kinara-production \
  --reuse-values \
  --set global.replicaOverride=1 \
  --disable-openapi-validation
```

### Step 7 — Cleanup (non-production restores)

After validation is complete, delete the throwaway restore cluster:

```bash
doctl databases delete $RESTORE_ID --force
```

---

## Validation Checklist

Run this checklist before declaring a restore successful:

- [ ] Restore cluster status is `online`
- [ ] 145 `kinara_*` databases present
- [ ] `kinara_patient.patients` row count matches expected
- [ ] `kinara_patient.clinics` row count matches expected
- [ ] `kinara_farmer.farmers` row count matches expected
- [ ] At least one table per pillar (Health, Agri, Logistics, Maritime) has data
- [ ] PgBouncer can connect to new cluster (`nc -zv <host> 25060` from pod)
- [ ] At least one microservice can execute a query through PgBouncer
- [ ] `kubectl get pods -n kinara-production` shows all pods `1/1 Ready`

---

## Known Constraints and Gotchas

| Issue | Detail |
|---|---|
| DO Postgres port | Always **25060**, never 5432. NetworkPolicy `allow-egress-pgbouncer` must allow port 25060. |
| Restore cluster minimum size | Must be `db-s-2vcpu-4gb` or larger — the 60 GiB disk allocation exceeds `db-s-1vcpu-1gb` max (50 GiB). |
| PgBouncer TLS | `client_tls_sslmode = require` and `server_tls_sslmode = require`. Both client and server TLS must be valid. |
| Helm SSA conflicts | If `kubectl set image` was used previously, `helm upgrade --disable-openapi-validation` bypasses field manager conflicts. |
| Rolling upgrade capacity | 7 nodes × ~3GB. Use `global.replicaOverride=1` during upgrade to halve peak pod count. `maxSurge: 0, maxUnavailable: 1` required. |

---

## Tested Backup Timestamps

| Date | Size | Status |
|---|---|---|
| 2026-08-30 07:51:38 UTC | 33 MB | Schema only, no data |
| 2026-08-30 20:10:14 UTC | 1.263 GB | Post-seed |
| 2026-08-31 20:11:08 UTC | 1.263 GB | Post-seed |
| **2026-09-01 20:09:28 UTC** | **1.263 GB** | **Validated — DR drill 2026-09-02** |

---

## Notification Contacts

| Role | Contact |
|---|---|
| Primary on-call | donalddaglo@gmail.com |
| DigitalOcean support | https://cloud.digitalocean.com/support |

---

## Related Documentation

- `docs/SECURITY_REVIEW.md` — security hardening, NetworkPolicy details
- `docs/PERFORMANCE_BASELINE.md` — cluster sizing and resource requests
- `infrastructure/pgbouncer/pgbouncer.yaml` — PgBouncer configuration
- `infrastructure/k8s/network-policies.yaml` — NetworkPolicy including PgBouncer egress (port 25060)
