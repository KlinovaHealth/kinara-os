# Kinara OS — Internal Security Review

**Date:** 2026-08-30  
**Scope:** Production cluster `do-fra1-production-kinara`, namespace `kinara-production`, repo `/KinaraOS`  
**Method:** Automated scanning only — gitleaks 8.30.1 (secret detection), trivy (CVE scan, 4 representative images sampled), kubectl manifest inspection  
**Performed by:** Internal automated review (Claude Code)

---

> **THIS SCAN DOES NOT REPLACE A PENETRATION TEST.**
>
> This document is an internal pre-check. Before the Gates Foundation review and before handling real patient data, a qualified third-party penetration test is required. That test must include authenticated and unauthenticated attack paths, API fuzzing, business-logic abuse, and multi-tenant isolation testing. Automated scanners find known CVEs and configuration drift; they do not find logic flaws, chain attacks, or novel vulnerabilities.

---

## Summary Table

| Severity | Count | Status |
|---|---|---|
| CRITICAL | 5 | 2 resolved (C1 TLS, C5 DB firewall); 3 remaining require dev work (C2, C3 CVEs; C4 FALSE POSITIVE) |
| HIGH | 5 | Remediation required before Gates review |
| MEDIUM | 5 | Address within 30 days |
| INFO / PASS | 8 | No action required |

---

## CRITICAL Findings

### C1 — Ingress is HTTP-only; no TLS termination ✅ RESOLVED (2026-08-31)

**Resolution:**
- cert-manager v1.16.3 installed; Let's Encrypt production cert issued for `api.kinaraos.com` + `admin.kinaraos.com`
- Ingress migrated from `67.207.76.53.nip.io` (nip.io) to `api.kinaraos.com` (Cloudflare DNS, A record)
- HTTP→HTTPS redirect live (308) on port 80
- mTLS to backend services: nginx now presents client cert (`kinara-nginx-client-cert`) via `proxy_ssl_certificate` directive injected via server-snippet ConfigMap
- TLS 1.3 confirmed end-to-end

**Original evidence:**
```
kubectl get ingress kinara-ingress -n kinara-production
NAME             CLASS   HOSTS                 ADDRESS        PORTS   AGE
kinara-ingress   nginx   67.207.76.53.nip.io   67.207.76.53   80      5h43m
```
The ingress spec had no `tls:` block.

**Risk (before fix):** Patient data in transit was fully readable to any on-path observer.

**Fix:**
1. Obtain a real domain (not nip.io).
2. Add a `tls:` stanza to the Ingress with a valid cert-manager-issued certificate.
3. Set `nginx.ingress.kubernetes.io/ssl-redirect: "true"` to enforce HTTPS.
4. Confirm port 80 redirects to 443.

---

### C2 — pgx v5 memory-safety CVEs (CVE-2026-33815, CVE-2026-33816)

**Evidence (trivy, auth-service and patient-service):**
```
github.com/jackc/pgx/v5  CVE-2026-33815  CRITICAL  v5.5.0  fix: 5.9.0
                          CVE-2026-33816  CRITICAL  v5.5.0  fix: 5.9.0
```
Auth-service uses `pgx/v5 v5.5.0`. These CVEs are memory-safety issues in the pgx driver itself. The 152-service fleet uses pgx versions ranging from v5.5.0 to v5.6.0 — all below the fixed version (5.9.0).

**Risk:** Memory-safety vulnerabilities in the database driver processing server-originated data. Potential for memory corruption or information disclosure on malformed Postgres responses.

**Fix:** Bump `github.com/jackc/pgx/v5` to `v5.9.0` across all 152 services. This requires a `go get` pass and rebuild of all images.

---

### C3 — Go 1.21 stdlib crypto/tls CVE (CVE-2025-68121)

**Evidence (trivy):**
```
stdlib  CVE-2025-68121  CRITICAL  v1.21.13  fix: 1.24.13, 1.25.7
        description: crypto/tls: Incorrect certificate validation
```
All 152 service images are built with Go 1.21.13 (confirmed in `.github/workflows/ci.yml`). CVE-2025-68121 is a certificate validation bypass in `crypto/tls`. Any service that validates TLS certificates (e.g., the Postgres connection, inter-service mTLS) may be vulnerable.

**Risk:** Attacker with a position in the network could present an invalid certificate that the service accepts, enabling MITM of encrypted channels.

**Fix:** Bump `GO_VERSION` in `ci.yml` from `1.21` to `1.25` (or `1.24`). Rebuild all 152 images.

---

### C5 — Production database firewall had zero trusted-source rules ✅ RESOLVED (2026-09-03)

**Discovery:** During staging isolation work on 2026-09-03, the production managed PostgreSQL cluster (`kinara-prod`, `c3ece802`) was found to have an empty trusted-sources list. DigitalOcean treats an empty list as **accept connections from any source on the internet**. TLS encryption and credential authentication were the only controls in place — there was no network-layer restriction.

**Evidence:**
```bash
doctl databases firewalls list c3ece802-0929-4e9e-aa33-95906e9f4a4e
UUID    ClusterUUID    Type    Value
# (empty — zero rows)
```

Connectivity confirmed from the staging Kubernetes cluster (a separate account-internal resource that should have had no access):
```
psql "host=kinara-prod-do-user-43233394-0.l.db.ondigitalocean.com port=25060 ..."
SELECT 1 → 1 row returned
```

**Risk:** Any host on the internet with valid credentials could connect directly to the production Postgres cluster, bypassing PgBouncer, connection pooling, and all application-layer controls. The 144 `kinara_*` databases — including patient, payment, and clinical records — were network-reachable from arbitrary sources.

**Root cause:** The original scan (2026-08-30) checked transport encryption (`server_tls_sslmode`, mTLS to backends) but did not check DigitalOcean managed-database firewall state. An empty trusted-sources list is the DO default on cluster creation; it requires an explicit action to restrict, and no such action was taken at provisioning time.

**Resolution (2026-09-03):**
```bash
doctl databases firewalls append c3ece802-0929-4e9e-aa33-95906e9f4a4e \
  --rule "k8s:82f21e09-b4d1-4059-9a87-9a2ec9c07d57"
```
Trusted sources restricted to the production Kubernetes cluster (`82f21e09`, `production-kinara`) only. No developer machine IPs or staging cluster included.

**Verification:**
- Production health endpoint `GET https://api.kinaraos.com/health` → `{"status":"ok"}` — production healthy post-change.
- `SELECT 1` via PgBouncer from a production pod → succeeded.
- Zero pods in CrashLoopBackOff.
- Staging connection attempt post-change → `timeout expired` — refused at network layer.

**Standing check added:** Database firewall state must be verified on every new managed-database provisioning event and included in future security scans. `doctl databases firewalls list <id>` returning zero rows is a critical misconfiguration, not a clean state.

---

### C4 — SQL injection via pgx protocol message size overflow (CVE-2024-27304)

**Evidence (trivy):**
```
github.com/jackc/pgx/v5  CVE-2024-27304  HIGH  v5.5.0  fix: 5.5.4
description: pgx: SQL Injection via Protocol Message Size Overflow
```
This CVE is rated HIGH by NVD but is functionally a SQL injection vector. Services using `default_query_exec_mode=simple_protocol` (which all 152 services now use via `DATABASE_URL`) are particularly exposed if they construct any queries with concatenated user input rather than parameterized queries.

**Note:** The shift to `simple_protocol` mode (for PgBouncer compatibility) increases the attack surface for this CVE because the driver falls back to sending query text rather than binary protocol messages. Review is warranted.

**Fix:** Upgrade pgx to v5.5.4 or higher (v5.9.0 resolves this along with C2). Audit all service query construction for non-parameterized inputs.

---

## HIGH Findings

### H1 — No Network Policies; zero pod-to-pod segmentation

**Evidence:**
```
kubectl get networkpolicies -n kinara-production
No resources found in kinara-production namespace.
```
Any pod in `kinara-production` can make TCP connections to any other pod on any port, including `auth-service`, `payment-service`, `patient-service`, and `kinara-pgbouncer`. A compromised low-privilege pod (e.g., `weather-service`) has full network access to the payment database pooler.

**Risk:** Lateral movement from a compromised service to high-value targets. Compromise of one service provides full network access to the entire namespace.

**Fix:** Define `NetworkPolicy` resources that:
- Default-deny all ingress/egress in `kinara-production`
- Allow only necessary service-to-service paths
- Allow only the services that need Postgres access to reach `kinara-pgbouncer:5432`
- Allow ingress from the ingress-nginx namespace only

---

### H2 — JWT library memory allocation CVE (CVE-2025-30204)

**Evidence (trivy):**
```
github.com/golang-jwt/jwt/v5  CVE-2025-30204  HIGH  v5.2.0  fix: 5.2.2
description: golang-jwt/jwt: jwt-go allows excessive memory allocation
```
Auth-service and any other service that validates JWTs uses `golang-jwt/jwt/v5 v5.2.0`. A crafted JWT with a malicious header field can cause unbounded memory allocation, enabling DoS.

**Fix:** Bump `golang-jwt/jwt/v5` to `v5.2.2` or later in all services that import it.

---

### H3 — golang.org/x/crypto multiple CVEs

**Evidence (trivy, multiple):**
```
golang.org/x/crypto  CVE-2024-45337  v0.17.0  fix: 0.31.0  (SSH misuse)
                     CVE-2025-22869  v0.17.0  fix: 0.35.0  (SSH DoS)
                     CVE-2025-47913  v0.17.0  fix: 0.43.0  (ssh/agent)
                     + 6 additional CVEs in 2026 range
```
Services import `golang.org/x/crypto` versions ranging from v0.9.0 to v0.17.0. None of the affected packages (`ssh`, `ssh/agent`, `ssh/knownhosts`) are used by these services directly — `x/crypto` is a transitive dependency. However, the package is linked into the binary.

**Risk:** Low if no code path exercises the `ssh` subpackage. Verify no service exposes an SSH interface. Upgrade is still required to close the vulnerability.

**Fix:** Upgrade `golang.org/x/crypto` to v0.52.0 or later. This is typically resolved by upgrading the Go version (C3 fix) and running `go mod tidy`.

---

### H4 — musl libc arbitrary code execution (CVE-2026-40200)

**Evidence (trivy, pgbouncer image):**
```
musl  CVE-2026-40200  HIGH  1.2.4_git20230717-r5  fix: 1.2.4_git20230717-r6
description: musl: musl libc: Arbitrary code execution and denial of service
```
The `kinara-pgbouncer` image is built on Alpine 3.20 and ships `musl` v1.2.4-r5. PgBouncer is the entry point for all database traffic (152 services post-cutover). A memory corruption in libc is particularly serious for a long-running network service.

**Fix:** Rebuild the `kinara-pgbouncer` Dockerfile `FROM alpine:3.21` (or ensure `alpine:3.20` package repo has the patched musl). Trigger the `build-pgbouncer.yml` workflow.

---

### H5 — PgBouncer client TLS set to `allow` instead of `require`

**Evidence:**
```
grep ssl /etc/pgbouncer/pgbouncer.ini
server_tls_sslmode = require   ← correct, PgBouncer→Postgres is TLS
client_tls_sslmode = allow     ← accepts plaintext client connections
```
Services currently specify `sslmode=require` in their `DATABASE_URL`, so in practice all connections are encrypted. However, PgBouncer will silently accept an unencrypted connection from any client that omits TLS negotiation. This provides no defense-in-depth if a service's sslmode is accidentally changed.

**Fix:** Change `client_tls_sslmode = allow` → `require` in `pgbouncer.yaml` and redeploy.

---

## MEDIUM Findings

### M1 — Database credentials injected as environment variables

**Evidence:**
```
kinara-db-credentials[host]  -> $_DB_HOST
kinara-db-credentials[pass]  -> $_DB_PASS
kinara-db-credentials[port]  -> $_DB_PORT
kinara-db-credentials[user]  -> $_DB_USER
kinara-shared-secrets[encryption-key] -> $ENCRYPTION_KEY
```
Kubernetes secrets mounted as environment variables appear in:
- `/proc/<pid>/environ` on the node
- Container crash dumps and OOM reports
- Any debug tooling that dumps process environment
- Kubernetes Event logs if a pod fails to start referencing the env var

The password is also URL-encoded in the secret value, which caused the PgBouncer authentication failure this session. URL-encoding of credentials in secrets is an antipattern.

**Recommendation:** For a future iteration, mount secrets as files (`secretKeyRef` → volume mount) rather than env vars, or use an external secrets manager (HashiCorp Vault, DO Secrets Manager, AWS Secrets Manager). Do not store URL-encoded credentials — store raw values and encode at the point of use.

---

### M2 — PgBouncer init container runs as root (uid=0)

**Evidence:**
```
ROOT(uid=0): kinara-pgbouncer-*/config-init
```
The `config-init` init container runs as root to write `pgbouncer.ini` and `userlist.txt` to the emptyDir volume. This is necessary given the current design (file ownership must be readable by the pgbouncer user uid=70). The main container drops to uid=70 correctly.

**Risk:** Low — the init container is short-lived and exits before the main container starts. However, any vulnerability in the Alpine shell or `sed` invoked during init runs with full root. The container has no privilege escalation beyond what root can do within the pod (no `privileged: true`, no `hostPID`).

**Recommendation:** Explore pre-baking a startup script into the image that can run as a non-root user with pre-set file permissions, eliminating the root init container.

---

### M3 — Helm release secrets contain historical values

**Evidence:** 9 Helm release secrets present (`sh.helm.release.v1.kinara-production.v1` through `.v9`). These are gzip+base64-encoded JSON blobs containing chart values at each revision, stored in Kubernetes etcd. If any Helm upgrade was run with `--set` flags containing credential values, those values persist in the cluster indefinitely.

**Check needed:** Confirm no previous `helm upgrade` commands passed credentials via `--set` flags. Review upgrade history with `helm history kinara-production -n kinara-production`.

**Recommendation:** Prune old Helm release secrets beyond the last 3 revisions using `helm plugin install https://github.com/helm/helm-mapkubeapis` or manual deletion.

---

### M4 — No ResourceQuota or LimitRange for the namespace

No `ResourceQuota` or `LimitRange` objects exist in `kinara-production`. All 209 pods have individual resource limits defined (this is correct), but there is no namespace-level ceiling. A runaway deployment could scale to maxReplicas across many services simultaneously, exhausting cluster resources.

**Recommendation:** Add a `ResourceQuota` capping total CPU/memory in `kinara-production` to prevent accidental resource exhaustion.

---

### M5 — nip.io domain in production ingress

The ingress hostname `67.207.76.53.nip.io` relies on the nip.io public DNS-to-IP service. This is a development convenience tool, not appropriate for production. It exposes the cluster's public IP via DNS lookup and creates dependency on an external third-party DNS service.

**Fix:** Register a real domain. Update the Ingress `host:` field. Add a cert-manager `ClusterIssuer` for automatic TLS (resolves C1 simultaneously).

---

## Pass / Info

| Check | Result |
|---|---|
| Secret scan — current files (gitleaks) | **CLEAN** — 0 findings across all files |
| Secret scan — git history (92 commits) | **CLEAN** — 0 findings; `staging-jwt-private.pem` was never committed |
| Privileged containers | **CLEAN** — no `privileged: true` in any pod |
| hostPID / hostNetwork / hostIPC | **CLEAN** — none set |
| PgBouncer external exposure | **PASS** — `kinara-pgbouncer` is `ClusterIP` only, not reachable outside cluster |
| PgBouncer → Postgres TLS | **PASS** — `server_tls_sslmode = require` confirmed in live pgbouncer.ini |
| Services with LoadBalancer/NodePort | **PASS** — none; all services are ClusterIP |
| Resource limits on all containers | **PASS** — all 209 running containers have CPU and memory limits defined |
| RBAC over-permissions | **PASS** — no custom ClusterRoleBindings for application workloads; all kinara services run as `default` ServiceAccount with no added permissions |

---

## Remediation Priority

```
Week 1 (before any real patient data touches the system):
  C1  Add HTTPS/TLS to ingress + real domain              ✅ RESOLVED 2026-08-31
  C5  Restrict production database firewall               ✅ RESOLVED 2026-09-03
  C2  Upgrade pgx to v5.9.0 across all services
  C3  Upgrade Go to 1.25 (CI_VERSION), rebuild all images
  C4  Audit parameterized queries; confirm no raw SQL concatenation
  H1  Define default-deny NetworkPolicies for kinara-production

Week 2–3 (before Gates Foundation technical review):
  H2  Upgrade golang-jwt/jwt to v5.2.2
  H3  Upgrade golang.org/x/crypto via go mod tidy after Go upgrade
  H4  Rebuild kinara-pgbouncer on Alpine 3.21
  H5  Set client_tls_sslmode = require in pgbouncer.yaml

Within 30 days:
  M1  Move to file-based secret mounts or external secrets manager
  M2  Refactor init container to non-root
  M3  Prune old Helm release secrets
  M4  Add ResourceQuota to kinara-production namespace
  M5  Register real domain; retire nip.io

Before penetration test:
  All of the above completed and verified.
```

---

## Trivy Sample Scope Note

Trivy was run against 4 images: `auth-service`, `patient-service`, `payment-service`, `kinara-pgbouncer`. The Go dependency CVEs (C2, C3, C4, H2, H3) are expected to affect all 152 service images since they share the same Go version and common dependency tree. A full fleet scan should be run in CI to confirm per-image totals.

**Recommended CI addition:** Add `trivy image --exit-code 1 --severity CRITICAL` as a blocking gate in `.github/workflows/ci.yml` before the `build-images` step.

---

## Third-Party Penetration Test Requirement

This automated scan covers:
- Known CVEs in declared dependencies (trivy)
- Secrets committed to version control (gitleaks)
- Kubernetes RBAC and pod security configuration
- Network exposure (service types, ingress config)

This scan **does not cover** and **cannot replace**:
- Authentication/authorization logic flaws
- API input validation and injection attacks
- Business-logic abuse (e.g., accessing another patient's records)
- Multi-tenant isolation between clinics
- JWT forgery and token replay
- Race conditions and TOCTOU vulnerabilities
- Social engineering and supply-chain attacks
- Physical security

**A third-party penetration test is required before:**
1. The Gates Foundation technical due-diligence review
2. Any real patient, provider, or farmer data entering the system
3. Any regulatory submission (GDPR, HIPAA, national health data regulations)

Recommended scope for the pentest: external API surface, authenticated service APIs (patient, pharmacy, payment), inter-service trust boundaries, and Kubernetes API server access from a compromised pod.
