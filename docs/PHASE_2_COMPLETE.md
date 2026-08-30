# Kinara OS — Phase 2: Database Infrastructure Complete

**Completed:** 2026-08-30  
**Cluster:** DigitalOcean Managed PostgreSQL 15 (`kinara-prod`, fra1, `db-s-1vcpu-1gb`)  
**Cluster ID:** `c3ece802-0929-4e9e-aa33-95906e9f4a4e`  
**Host:** `kinara-prod-do-user-43233394-0.l.db.ondigitalocean.com:25060`  

---

## What Was Done

### 1. Managed PostgreSQL Cluster
- Created DigitalOcean Managed PostgreSQL 15 cluster in Frankfurt (fra1)
- Configured SSL-only connections (`sslmode=require`)
- Credentials stored in `.env.prod` (excluded from git via `.gitignore`)

### 2. Database Creation
All 44 service databases created:

| Pillar | Databases |
|--------|-----------|
| **Health** | kinara_patient, kinara_clinical, kinara_telemedicine, kinara_pharmacy, kinara_referral, kinara_analytics, kinara_compliance, kinara_governance, kinara_appointment, kinara_immunization, kinara_lab, kinara_outbreak |
| **Agriculture** | kinara_farmer, kinara_market, kinara_cooperative, kinara_weather, kinara_farmer_finance, kinara_input, kinara_irrigation, kinara_livestock, kinara_extension |
| **Logistics** | kinara_transport, kinara_warehouse, kinara_lastmile, kinara_shipment, kinara_fleet, kinara_supply_chain, kinara_logistics_analytics, kinara_vehicle_tracking |
| **Maritime** | kinara_port, kinara_cargo, kinara_customs, kinara_shipping, kinara_trade_finance, kinara_documentation, kinara_vessel, kinara_dock, kinara_crew, kinara_voyage |
| **Platform** | kinara_payment, kinara_audit, kinara_notification, kinara_auth, kinara_sms |

### 3. Migrations Applied — 49/49

All migrations from `V202609010001` through `V202609010049` applied successfully.  
Migration naming: `V<timestamp>__<Service>__<Description>.sql`

Notable migrations:
- **V001–V012**: Health pillar schema (patients, clinics, consultations, pharmacy, referrals, analytics, governance, appointments, immunization, lab, outbreak, compliance)
- **V018–V023**: Agriculture pillar (farmers, market, cooperative, weather, finance)
- **V031–V039**: Maritime pillar (ports, cargo, customs, shipping, vessel, dock, crew)
- **V040–V048**: Platform services (payment with FX rates, audit, notification, auth, SMS)
- **V049**: Togo Pilot Seed Data (rewritten to match actual schemas — see note below)

#### V049 Schema Alignment Note
The original seed migration used simplified/demo column names. V049 was rewritten to match the production schemas:
- `full_name_enc` / `phone_enc` (not `name` / `phone`) for PHI-encrypted fields
- `price_indices` (not `market_prices`) — V019 market service schema
- `currency_rates` (not `fx_rates`) — V040 payment service schema
- `code` / `total_berths` / `status` (not `port_code` / `berths_total` / `is_active`) — V031 port schema
- `registration_no` / `total_members` (not `registration_number` / `member_count`) — V020 cooperative schema
- `template_key` with language suffix / `body_template` (not `name` / `body`) — V044 notification schema
- `clinics` table created by V049 itself (no separate service migration existed)
- PHI columns (`national_id_enc`, `full_name_enc`, etc.) use SHA-256 placeholder ciphertext; real AES-256-GCM values must be written via the application layer

### 4. Togo Pilot Seed Data Loaded

| Table | Database | Count |
|-------|----------|-------|
| clinics | kinara_patient | 100 |
| patients | kinara_patient | 1,000 |
| farmers | kinara_farmer | 500 |
| price_indices | kinara_market | 20 |
| currency_rates | kinara_payment | 24 (9 from V040 + 15 from V049) |
| ports | kinara_port | 10 |
| berths | kinara_port | 12 (Lomé berths) |
| cooperatives | kinara_cooperative | 10 |
| notification_templates | kinara_notification | 6 |

### 5. Database Roles

Three limited-privilege roles created (passwords in `.env.prod` / secrets manager):

| Role | Permissions | Purpose |
|------|-------------|---------|
| `kinara_app` | CONNECT + SELECT/INSERT/UPDATE/DELETE | Application services |
| `kinara_readonly` | CONNECT + SELECT | Analytics, reporting, dashboards |
| `kinara_backup` | CONNECT + SELECT | `pg_dump` backups |

All roles granted on all 44 service databases with `ALTER DEFAULT PRIVILEGES` set for future tables.

### 6. Backup Script

`scripts/backup-db.sh` — dumps all 44 databases using `pg_dump --format=custom --compress=9`.

```bash
# Full backup
./scripts/backup-db.sh

# Single database
./scripts/backup-db.sh kinara_patient

# Custom output directory
BACKUP_DIR=/mnt/backups ./scripts/backup-db.sh
```

Outputs to `backups/YYYY-MM-DD/` with per-database `.pgdump` files and a `backup.log`.

---

## Security Invariants (Must Never Change)

- **PHI encryption**: `national_id_enc`, `full_name_enc`, `date_of_birth_enc`, `phone_number_enc` — AES-256-GCM at application layer only. Never plaintext in DB.
- **`national_id_enc` never in list views** — only returned on individual GET by ID.
- **Immutable audit logs**: All `*_audit_log` tables have PostgreSQL RULEs blocking UPDATE and DELETE. Do not drop these rules.
- **JWT RS256**: Services validate with public key only. Only `auth-service` holds the private key.
- **mTLS**: Inter-service communication requires mutual TLS.
- **`.env.prod` is gitignored** — never commit database passwords.

---

## Next Steps (Phase 3+)

- [ ] Kubernetes cluster provisioning (`kinara-togo-prod`) — enables `DEPLOY_ENABLED=true` in CI
- [ ] Helm chart values for Togo production (`infrastructure/helm/kinara-os/values-togo-prod.yaml`)
- [ ] Real AES-256-GCM ciphertext for seed patients/farmers (app-layer seeding tool)
- [ ] Set up automated daily backups (cron or DO Managed DB automated backups)
- [ ] V202609010017 vehicle_tracking GPS partitioning (gps_locations is high-volume)
- [ ] Gates Foundation milestone: 150 services by Jan 2027
