# Kinara Governance OS

**"Where Africa Coordinates"**

Kinara Governance OS is the coordination layer for African economies — a government-grade operating system that coordinates health, agriculture, logistics, and maritime across the continent in real time.

**Owner:** Klinova LLC (Virginia, USA) — EIN 42-4472128  
**Domain:** kinaraos.com  
**Target:** Gates Foundation pitch — 150 microservices by January 2027  

---

## Four Pillars

### Pillar 1: Health (100M+ patients)
Clinics, hospitals, CHWs, and telemedicine coordinated on one platform. Patient records, clinical data, prescriptions, epidemiology tracking, and national disease surveillance — all encrypted, audited, and real-time.

### Pillar 2: Agriculture (100M+ farmers)
Farms, cooperatives, buyers, and markets coordinated. Price discovery, weather alerts, supply chain visibility, and smallholder farmer access to finance — closing the gap between field and fork.

### Pillar 3: Logistics (Continental distribution)
Transportation networks, warehouse capacity, and last-mile delivery optimized end-to-end. Cross-border routing, cold chain management, and freight visibility across 54 countries.

### Pillar 4: Maritime ($500B+/year trade, 54 African ports)
Port operations, customs clearance, shipping coordination, and trade finance — digitized and coordinated. Reducing dwell times from days to hours.

### The Integration
```
Farmer → Cooperative → Logistics → Port → Maritime → Market → Customer
                    COORDINATED THROUGH ONE OS
```
Network effects are exponential: each pillar strengthens all others.

---

## Architecture

| Layer | Technology |
|-------|-----------|
| Language | Go 1.21+ |
| Router | gorilla/mux (REST) |
| Database | PostgreSQL 16 (pgx/v5, sqlc) |
| Cache / Rate Limiting | Redis 7 |
| Auth | JWT RS256 (external) + mTLS (inter-service) |
| PHI Encryption | AES-256-GCM, random nonce per field per write |
| Audit | Immutable PostgreSQL rule-based append-only log |
| Container | Docker (Alpine multi-stage) |
| Orchestration | Kubernetes (DigitalOcean DOKS) |
| Observability | Prometheus + Grafana |

---

## Services

### Health Pillar

| Service | Status | Description |
|---------|--------|-------------|
| `patient-service` | ✅ Complete | Encrypted patient registry, soft delete, audit |
| `clinical-service` | ✅ Complete | Consultations, diagnoses, treatments, notes, prescriptions |
| `governance-service` | ✅ Complete | Compliance reporting, epidemiology tracking, coordination rules, alerts |
| `auth-service` | 🔜 Next | JWT issuance, RBAC, mTLS certificate management |
| `referral-service` | 🔜 | Inter-facility patient referrals |
| `telemedicine-service` | 🔜 | Video/audio consultation sessions |
| `pharmacy-service` | 🔜 | Drug inventory, dispensing, cold chain |

### Agriculture Pillar

| Service | Status | Description |
|---------|--------|-------------|
| `farmer-service` | 🔜 | Farmer registry, cooperative membership |
| `market-service` | 🔜 | Price discovery, buyer-seller matching |
| `weather-service` | 🔜 | Weather alerts, planting calendars |
| `finance-service` | 🔜 | Input credit, harvest insurance, mobile money |

### Logistics Pillar

| Service | Status | Description |
|---------|--------|-------------|
| `fleet-service` | 🔜 | Vehicle registry, route optimization |
| `warehouse-service` | 🔜 | Inventory management, cold chain |
| `delivery-service` | 🔜 | Last-mile coordination, proof of delivery |

### Maritime Pillar

| Service | Status | Description |
|---------|--------|-------------|
| `port-service` | 🔜 | Port operations, berth scheduling |
| `customs-service` | 🔜 | Customs clearance automation |
| `shipping-service` | 🔜 | Vessel tracking, cargo coordination |
| `trade-finance-service` | 🔜 | Letters of credit, invoice financing |

---

## Security Model

- **Zero trust:** Every request requires a valid JWT. Inter-service calls require mTLS.
- **PHI encryption:** Every sensitive field encrypted individually with AES-256-GCM (random nonce per write). Non-PHI fields stored plaintext for indexing.
- **Immutable audit:** PostgreSQL rules block UPDATE/DELETE on audit tables. Every read and write is logged with accessor ID, role, IP, and request ID.
- **Soft delete:** No patient or clinical record is physically destroyed.
- **Rate limiting:** Redis sliding-window, 1,000 req/min per mTLS certificate CN.
- **No secrets in code:** All keys and credentials from environment variables.

---

## Quick Start

```bash
# Generate a 32-byte encryption key
openssl rand -hex 32

# Run PostgreSQL and Redis
docker compose up -d

# Apply migrations
psql $DATABASE_URL -f services/patient-service/db/migrations/001_create_patients.sql
psql $DATABASE_URL -f services/clinical-service/db/migrations/001_create_clinical.sql
psql $DATABASE_URL -f services/governance-service/db/migrations/001_create_governance.sql

# Run tests
cd services/patient-service && go test ./... -v -cover
cd services/clinical-service && go test ./... -v -cover
cd services/governance-service && go test ./... -v -cover
```

---

## Deployment

```bash
# Build and push images
for svc in patient-service clinical-service governance-service; do
  docker build -t registry.digitalocean.com/kinara-os/$svc:latest services/$svc/
  docker push registry.digitalocean.com/kinara-os/$svc:latest
done

# Deploy to Kubernetes
kubectl apply -f services/patient-service/k8s/deployment.yaml
kubectl apply -f services/clinical-service/k8s/deployment.yaml
kubectl apply -f services/governance-service/k8s/deployment.yaml
```

---

## Brand

| Element | Value |
|---------|-------|
| Name | Kinara Governance OS |
| Tagline | "Where Africa Coordinates" |
| Fire Orange | `#FF6B35` |
| Deep Gold | `#D4A574` |
| Charcoal | `#2C3E50` |
| Health Green | `#2ECC71` |
| Symbol | Fireplace + Compass Rose |
