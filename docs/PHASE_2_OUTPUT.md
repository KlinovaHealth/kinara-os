# Phase 2 Database Setup — Output Record

**Date**: 2026-08-29
**Status**: Infrastructure-as-Code complete; ready for provisioning

## What Was Built

### Provisioning
| Artifact | Path | Purpose |
|---|---|---|
| Terraform | `infrastructure/terraform/digitalocean.tf` | Declarative cluster + replica + firewall |
| Python script | `infrastructure/scripts/provision.py` | Imperative DO API provisioning (alternative) |
| Flyway config | `infrastructure/flyway/flyway-prod.conf` | Points at managed PG on port 25060 |

### Migrations
| Artifact | Path | |
|---|---|---|
| V001–V048 | `infrastructure/database/migrations/` | Full schema for all 49 service databases |
| V049 seed | `infrastructure/database/migrations/V202609010049__Togo_Pilot__Seed_Data.sql` | 100 clinics, 1000 patients, 500 farmers, 10 ports, FX rates, cooperatives |
| psql runner | `infrastructure/scripts/run-migrations.sh` | Handles `\c kinara_xxx` DB switching (Flyway JDBC cannot) |

### Database Operations
| Artifact | Path | |
|---|---|---|
| Users SQL | `infrastructure/database/scripts/create_db_users.sql` | app_user / read_only / backup_user |
| Backup | `infrastructure/scripts/backup.sh` | pg_dump all 49 DBs → S3 |
| Restore | `infrastructure/scripts/restore.sh` | Download from S3 → psql |
| Verify | `infrastructure/scripts/verify-db.sh` | Row-count checks post-migration |
| K8s secret | `infrastructure/scripts/k8s-secrets.sh` | Applies kinara-db-credentials secret |

## Target Infrastructure (Togo Pilot)

```
DigitalOcean Managed PostgreSQL 15
  Region:       Frankfurt (fra1)
  Size:         db-s-4vcpu-8gb (2 nodes HA)
  Port:         25060 (SSL required)
  Read replica: fra1 (analytics workloads)
  VPC:          kinara-prod (tag-based firewall)
  Databases:    49 (one per microservice)
```

## Seed Data (V049)

| Table | Database | Count |
|---|---|---|
| clinics | kinara_patient | 100 |
| patients | kinara_patient | 1,000 |
| farmers | kinara_farmer | 500 |
| ports | kinara_port | 10 |
| berths (Lomé) | kinara_port | 12 |
| cooperatives | kinara_cooperative | 10 |
| fx_rates | kinara_payment | 20 |
| market_prices | kinara_market | 20 |
| notification_templates | kinara_notification | 6 |

## Users Created

| User | Permissions | Used by |
|---|---|---|
| admin | Superuser (DO-managed) | migrations only |
| app_user | SELECT/INSERT/UPDATE/DELETE on all tables | all microservices |
| read_only | SELECT only | analytics, Gates Foundation reporting |
| backup_user | SELECT only | backup.sh / pg_dump |

## Provisioning Runbook

```bash
# 1. Provision cluster (choose one):
#    a) Terraform
cd infrastructure/terraform && terraform apply

#    b) Python script
export DO_API_TOKEN=<token>
python3 infrastructure/scripts/provision.py

# 2. Source credentials
source .env.prod

# 3. Create limited users (run per database as needed)
psql $DATABASE_URL \
  -v app_pw="$APP_USER_PASSWORD" \
  -v ro_pw="$READ_ONLY_PASSWORD" \
  -v bk_pw="$BACKUP_USER_PASSWORD" \
  -f infrastructure/database/scripts/create_db_users.sql

# 4. Run migrations (psql runner — handles \c db switching)
export POSTGRES_HOST=<do-host>
export POSTGRES_PORT=25060
export POSTGRES_USER=doadmin
export POSTGRES_PASSWORD=<do-password>
./infrastructure/scripts/run-migrations.sh

# 5. Verify
./infrastructure/scripts/verify-db.sh

# 6. Apply Kubernetes secret
source .env.prod
./infrastructure/scripts/k8s-secrets.sh
```

## Backup Configuration

- **Script**: `infrastructure/scripts/backup.sh`
- **Schedule**: Daily at 02:00 UTC via cron
- **Destination**: `s3://kinara-backups-2026/`
- **Scope**: All 49 databases (incremental via pg_dump)
- **Retention**: S3 lifecycle policy → 90 days; local → 0 (streaming to S3)
- **Restore**: `infrastructure/scripts/restore.sh`

## Security Controls

- AES-256-GCM at application layer for all PHI fields
- mTLS between microservices (governance, notification services)
- JWT RS256 — services validate only (public key); auth-service issues (private key)
- Immutable audit logs via PostgreSQL RULE (DO INSTEAD NOTHING on UPDATE/DELETE)
- National ID stored encrypted; never returned in list views
- Database port 25060 — SSL required, VPC firewall restricts to k8s node tag
- No direct internet access to database cluster

## Gates Foundation Target

- **Togo pilot launch**: October 2026
- **150-service milestone**: January 2027
- **Current status**: 47 services with tests; CI green target = this week
