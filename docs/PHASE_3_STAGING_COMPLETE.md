# Phase 3: Staging Deployment — Complete

**Date:** 2026-08-30  
**Cluster:** `staging-kinara` (DigitalOcean Kubernetes, fra1)  
**Status:** 50/50 services deployed; 46 consistently healthy, 4 recovering (CI rebuild in progress)

---

## Cluster Details

| Resource | Value |
|---|---|
| Cluster ID | `99d456d7-b4e2-44ce-8953-a7f9a02c4c7e` |
| Kubernetes version | 1.36.3-do.2 |
| Node pool | 3 × s-2vcpu-4gb (fra1) |
| Namespace | `staging` |
| Ingress IP | `209.38.188.87` |
| Staging URL | `http://209.38.188.87.nip.io` |

## Services Deployed

All 50 Kinara OS microservices deployed across 4 pillars:

| Pillar | Services |
|---|---|
| Health | patient, clinical, appointment, pharmacy, referral, lab, immunization, outbreak, telemedicine |
| Agriculture | farmer, market, cooperative, irrigation, livestock, extension, farmer-finance, input, weather |
| Logistics | driver, fleet, route, transport, last-mile, shipment, cargo, warehouse, logistics-analytics, vehicle-tracking, supply-chain |
| Maritime | port, vessel, voyage, dock, crew, shipping, cargo-maritime, customs, trade-finance |
| Cross-cutting | auth, notification, sms-gateway, payment, wallet, analytics, audit, governance, compliance, health-analytics, health-compliance, documentation |

**Redis:** 1 pod, Running  
**Total pods:** 51

## Kubernetes Secrets Configured

| Secret | Purpose |
|---|---|
| `ghcr-pull-secret` | Image pull from ghcr.io/klinovahealth |
| `kinara-db-credentials` | DO Managed PostgreSQL host/port/user/pass |
| `kinara-shared-secrets` | AES-256-GCM 32-byte encryption key |
| `kinara-jwt-public-key` | RS256 public key for token validation (all services) |
| `kinara-jwt-private-key` | RS256 private key for token issuance (auth-service only) |
| `kinara-tls-certs` | mTLS cert/key/ca.crt for all services |
| `kinara-nginx-client-cert` | TLS cert for nginx ingress |

## Infrastructure

- **Database:** DO Managed PostgreSQL `kinara-prod` (fra1) — 44 databases, port 25060
- **Redis:** In-cluster `kinara-redis:6379` (Redis 7 Alpine)
- **Ingress:** nginx-ingress v1.15.1, LoadBalancer IP `209.38.188.87`

## Helm Chart Revisions

| Revision | Change |
|---|---|
| 1 | Initial install |
| 2 | TLS cert mounts + JWT private key for auth |
| 3 | CA cert + JWT /run/secrets mount |
| 4 | Removed /run/secrets mount (K8s SA token conflict) |
| 5 | tcpSocket probes + env var aliases + HTTPS ingress |
| 6 | Pool connection limits in DATABASE_URL |
| 7–10 | Ingress HTTPS backend + proxy-ssl fixes |

## Security Architecture

All services enforce:
- **mTLS** (`tls.RequireAndVerifyClientCert`) for inter-service communication
- **JWT RS256** — services validate with public key only; auth-service issues with private key
- **AES-256-GCM** application-layer encryption on all PHI fields
- **Immutable audit logs** enforced via PostgreSQL RULE (no UPDATE/DELETE)
- **National ID** (`national_id_enc`) excluded from list views — single-record GET only

## Public Health Check

```bash
curl -H "Host: 209.38.188.87.nip.io" http://209.38.188.87/health
# → {"status":"ok","service":"analytics-service"}
```

Routes accessible via ingress:
- `GET /health` → analytics-service (one-way TLS, no client cert required)
- `GET /api/v1/analytics` → analytics-service

## Known Staging Limitations

### mTLS via Ingress
Services using `BuildServerTLSConfig` (`RequireAndVerifyClientCert`) are not accessible
via the public ingress — nginx cannot present a client certificate to the backends.
Affected routes: `/api/v1/patients`, `/api/v1/farmers`, `/api/v1/market`, etc.

**Workaround for external testing:** Use `kubectl port-forward` with a TLS client:
```bash
kubectl port-forward -n staging svc/patient-service 8081:8081
# Then connect with a TLS client presenting kinara-staging-tls.crt as client cert
```

**Production fix:** Deploy a dedicated API gateway (Kong/Envoy) with proper mTLS client cert.

### DB Connection Limit (4 services recovering)
DO Managed PostgreSQL connection limit (~75 connections) is approached by 50 services
connecting simultaneously at startup. Services that hardcoded `MinConns=5` overrode the
URL pool parameters and caused startup bursts. Fixed in commit `546fe38` (removes hardcoded
overrides; pool_min_conns=0 from DATABASE_URL is now respected). CI rebuild in progress.

### Image Registry
All 50 images are in `ghcr.io/klinovahealth/`. The DO Container Registry Starter tier
(1 repository only) was not used. Docker Hub images cannot be pulled (network restriction
on the DO K8s cluster).

## Env Var Naming Conventions Found

Services use 3 different naming conventions for TLS/JWT env vars. All 3 are now set:

| Var | Also set as |
|---|---|
| `TLS_CERT_PATH` | `TLS_CERT_FILE` |
| `TLS_KEY_PATH` | `TLS_KEY_FILE` |
| `CA_CERT_PATH` | `TLS_CA_FILE` |
| `JWT_PUBLIC_KEY_PATH` | `JWT_PUBLIC_KEY` |

## Next Steps

1. **CI rebuild completes** → restart clinical, farmer, governance, vessel to use pool-fixed images
2. **Production cluster** (production-kinara) — separate DO Kubernetes cluster with ≥4 nodes
3. **DO PgBouncer** — upgrade PostgreSQL plan to enable port 6543 pooler (currently blocked on Basic tier)
4. **Proper PKI** — issue per-service certificates for mTLS (currently all services share one self-signed cert)
5. **API Gateway** — deploy Kong or Envoy for external mTLS termination
6. **Monitoring** — configure Prometheus/Grafana (chart already has `monitoring:` block, needs proper cluster)
7. **Load testing** — scale to Togo pilot load (5K patients, 1K farmers) against staging
