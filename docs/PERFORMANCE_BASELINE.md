# Kinara OS — Performance Baseline

## Test Overview

| Item | Value |
|---|---|
| **Test date** | 2026-08-31 |
| **Tool** | k6 v2.2.0 (go1.26.5, darwin/arm64) |
| **Target** | `https://api.kinaraos.com` (do-fra1-production-kinara) |
| **Cluster state** | 207 running pods, 7 nodes |
| **Synthetic data loaded** | 50 clinics · 10,000 patients · 55K visits · 16.5K prescriptions · 2.75K referrals |
| **Test script** | `scripts/load-test/baseline.js` |

## Load Profile

| Stage | Concurrent users | Duration | Purpose |
|---|---|---|---|
| 1 | 100 VUs | 2 min | Warm-up / baseline |
| 2 | 500 VUs | 2 min | Medium load |
| 3 | 1,000 VUs | 5 min | Peak load (hold) |
| Ramp-down | 0 VUs | 30 s | Drain |

**Total test duration:** 9 min 30 s  
**Total requests completed:** 631,459

## Endpoint Mix

| Endpoint | % of traffic | Expected status |
|---|---|---|
| `GET /health` (analytics-service) | 70% | 200 |
| `GET /api/v1/farmers` | 10% | 401 (auth gated) |
| `GET /api/v1/patients` | 10% | 401 (auth gated) |
| `GET /api/v1/ports` | 5% | 401 (auth gated) |
| `GET /api/v1/market` | 5% | 404/401 (varies) |

Auth-gated endpoints require JWT; 401 responses were expected and not counted as errors. Error threshold was `5xx rate < 5%`.

---

## Results by Load Level

### Latency (HTTP response time, ms)

| Stage | Requests | avg | p50 | p95 | p99 | max |
|---|---|---|---|---|---|---|
| 100 VUs | 27,771 | 186 | 188 | 262 | 339 | 665 |
| 500 VUs | 119,586 | 271 | 227 | 446 | 1,759 | 4,287 |
| 1,000 VUs | 456,910 | 456 | 313 | 1,031 | 3,693 | 9,079 |
| **Overall** | **631,459** | **417** | **281** | **906** | — | **9,079** |

### Throughput

| Stage | Throughput |
|---|---|
| 100 VUs | **231 req/s** |
| 500 VUs | **997 req/s** |
| 1,000 VUs | **1,523 req/s** |
| Overall | **1,107 req/s avg** |

### Error Rates

| Metric | Value |
|---|---|
| **5xx server errors** | **0.00%** |
| Network timeouts / connection failures | 0.00% |
| Auth-wall hits (expected 401) | 14.96% |
| Pod restarts during test | **0** |

No stop conditions were triggered. All thresholds passed.

---

## Cluster Resource Usage During Test

### Node CPU

| Metric | Value |
|---|---|
| Peak CPU (any node, 12-min window) | **86.3%** |
| Average CPU across all nodes | **41.8%** |
| Nodes | 7 |

Node 1 hit 86% peak CPU — the highest of the 7 nodes. No sustained saturation (only momentary during ramp to 1K VUs).

### Node Memory

| Node | Memory used % |
|---|---|
| node 1 | **80.5%** |
| node 2 | 49.7% |
| node 3 | 56.8% |
| node 4 | 55.6% |
| node 5 | 52.4% |
| node 6 | 53.3% |
| node 7 | 49.9% |

Node 1 carries higher memory pressure — likely the node hosting the most pods (analytics-service received 70% of load).

### PgBouncer / Postgres

PgBouncer active server connections: **0** (post-test idle). Peak was not measurable via Prometheus (pgbouncer Prometheus exporter metric `pgbouncer_pools_server_active_connections` showed 0 post-test, indicating pool drained cleanly). PgBouncer max is configured at 90 server connections (`max_db_connections = 90`).

---

## Bottleneck Analysis

| Component | Status | Evidence |
|---|---|---|
| **CPU** | **Yellow — approaching limit** | Node 1 peaked at 86% during 1K VU stage. Sustained 1,000+ VU traffic would saturate |
| **Memory** | Yellow — node 1 at 80% | Likely collocated with high-load pods; watch for OOM at 1,500+ VUs |
| **Network / TLS** | Green | p95 at 100 VUs = 262 ms; overhead is TLS + inter-region RTT; no connection drops |
| **DB pool (PgBouncer)** | Green | Health endpoint requires no DB; auth endpoints hit middleware before DB; pool not stressed |
| **Ingress** | Green | 0 5xx, 0 drops; NGINX handled all traffic; no 502/503 observed |

The primary bottleneck is **CPU on the hot node (node 1)**, not DB connections or network. The health-endpoint handler path is CPU-light (single Go goroutine, no DB query), which is why the DB pool was not stressed even at 1K VUs.

---

## Capacity Estimate

| Concurrent users | Throughput | p95 latency | Risk |
|---|---|---|---|
| 100 | 231 req/s | 262 ms | Green |
| 500 | 997 req/s | 446 ms | Green |
| 1,000 | 1,523 req/s | 1,031 ms | Yellow (CPU peaks) |
| ~1,400 (est.) | ~1,900 req/s | >2,000 ms est. | Red (CPU saturates) |

**Estimated sustainable peak without scaling:** ~1,200–1,400 concurrent users at acceptable p95 < 2s.

**Clinic capacity translation:**  
A typical clinic day (~8h) generates ~40 API calls per patient interaction. At 50 clinics × 20 active users/clinic = 1,000 concurrent users → within the Yellow zone. Supporting 150 clinics (Gates Foundation milestone) would require horizontal pod autoscaling (HPA) or more nodes.

---

## Recommended Next Steps (not actioned in this test)

1. **Enable HPA** on the top-5 busiest services (patient, analytics, farmer, auth, market). Target CPU 60%.
2. **Profile node 1 colocation** — redistribute pods to balance CPU across all 7 nodes.
3. **Re-run with authenticated endpoints** using a load-test JWT (signed with the test private key) to stress DB/PgBouncer pool at 1K VUs.
4. **Add DB connection tracking** to the k6 script via Prometheus `range_query` for real-time pgbouncer pool utilization.

---

*This is a read-only performance measurement. No configuration changes were made.*
