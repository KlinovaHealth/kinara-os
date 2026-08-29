# Kinara OS

**"Where Africa Coordinates"**

Kinara OS is a government-grade health coordination platform built for African public health systems. It provides real-time coordination across clinics, hospitals, pharmacies, CHWs, and government health ministries.

**Owner:** Klinova LLC  
**Target:** Gates Foundation pitch — 150 microservices by January 2027  
**Domain:** kinaraos.com

---

## Architecture

| Layer | Technology |
|-------|-----------|
| Language | Go 1.21+ |
| API | REST (gorilla/mux) + gRPC (inter-service) |
| Database | PostgreSQL 16 (pgx/v5) |
| Cache | Redis 7 |
| Auth | JWT RS256 (external) + mTLS (internal) |
| Encryption | AES-256-GCM per field |
| Audit | Immutable PostgreSQL rule-based log |
| Container | Docker (Alpine multi-stage) |
| Orchestration | Kubernetes (DigitalOcean DOKS) |
| Observability | Prometheus + Grafana |

---

## Services

### Phase 1 — Core (Month 1–3)

| Service | Status | Description |
|---------|--------|-------------|
| `patient-service` | ✅ Complete | Encrypted patient registry |
| `clinical-service` | ✅ Complete | Consultations, diagnoses, treatments, notes, prescriptions |
| `auth-service` | 🔜 Next | JWT issuance, RBAC, mTLS cert management |
| `referral-service` | 🔜 | Inter-facility patient referrals |
| `medicine-service` | 🔜 | Drug inventory and formulary |
| `telemedicine-service` | 🔜 | Video/audio consultation sessions |

### Phase 2 — Analytics (Month 4–6)

- `analytics-service` — disease burden dashboards, outbreak detection
- `reporting-service` — government ministry reports, WHO/DHIS2 export
- `alert-service` — epidemic threshold alerts, push notifications

### Phase 3 — Coordination (Month 7–12)

- `supply-chain-service` — medical supply logistics
- `laboratory-service` — lab test orders and results
- `immunization-service` — vaccine scheduling and coverage tracking
- `maternal-health-service` — antenatal care, birth outcomes

---

## Security Model

- **Zero trust:** Every request requires a valid JWT. Inter-service calls require mTLS.
- **PHI encryption:** Every sensitive field encrypted individually with AES-256-GCM (random nonce per write).
- **Immutable audit:** PostgreSQL rules block UPDATE/DELETE on audit tables. Every read and write is logged.
- **Soft delete:** No patient or clinical record is physically destroyed.
- **Rate limiting:** Redis sliding-window, 1,000 req/min per certificate CN.
- **No secrets in code:** All keys from environment variables.

---

## Quick Start

```bash
# Generate encryption key
openssl rand -hex 32

# Start infrastructure
docker compose up -d

# Apply migrations
psql $DATABASE_URL -f services/patient-service/db/migrations/001_create_patients.sql
psql $DATABASE_URL -f services/clinical-service/db/migrations/001_create_clinical.sql

# Run tests
cd services/patient-service && go test ./... -v -cover
cd services/clinical-service && go test ./... -v -cover
```

---

## Deployment

```bash
# Build and push all services
docker build -t registry.digitalocean.com/kinara-os/patient-service:latest services/patient-service/
docker build -t registry.digitalocean.com/kinara-os/clinical-service:latest services/clinical-service/
docker push registry.digitalocean.com/kinara-os/patient-service:latest
docker push registry.digitalocean.com/kinara-os/clinical-service:latest

# Deploy to Kubernetes
kubectl apply -f services/patient-service/k8s/deployment.yaml
kubectl apply -f services/clinical-service/k8s/deployment.yaml
```
