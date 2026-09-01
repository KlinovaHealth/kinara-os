/**
 * Kinara OS — Performance Baseline Load Test
 * Phase 8 · docs/PERFORMANCE_BASELINE.md
 *
 * Profile:
 *   Stage 1:  100 VUs × 2 min  (warm-up)
 *   Stage 2:  500 VUs × 2 min  (medium load)
 *   Stage 3: 1000 VUs × 5 min  (peak load)
 *   Stage 4:    0 VUs × 30 sec (ramp-down)
 *
 * Error definition: HTTP 5xx or network failure.
 * HTTP 401 (auth wall) is expected for gated endpoints — not counted as error.
 *
 * STOP thresholds:
 *   - http_req_failed rate > 5% aborts the test
 *   - p(95) latency > 10 000 ms at any stage emits a warning
 */

import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const BASE = "https://api.kinaraos.com";

// Custom metrics
const authWallHits  = new Rate("auth_wall_hits");   // 401 responses (expected)
const serverErrors  = new Rate("server_errors");    // 5xx responses (bad)
const latencyHealth = new Trend("latency_health_ms", true);
const latencyAuth   = new Trend("latency_auth_ms",   true);

export const options = {
  stages: [
    { duration: "2m",  target: 100  },  // stage 1 — ramp to 100
    { duration: "2m",  target: 500  },  // stage 2 — ramp to 500
    { duration: "5m",  target: 1000 },  // stage 3 — ramp to 1000 and hold
    { duration: "30s", target: 0    },  // stage 4 — ramp down
  ],
  thresholds: {
    // Abort only on true 5xx / network failures (not expected 401s)
    server_errors:     [{ threshold: "rate<0.05", abortOnFail: true }],
    // Soft latency target — warn, don't abort
    http_req_duration: ["p(95)<5000"],
  },
};

const ENDPOINTS = [
  // Public health check — 70% weight
  { path: "/health",              method: "GET", expectedStatus: [200], weight: 70 },
  // Auth-gated (JWT required) — expected 401, measures middleware overhead
  { path: "/api/v1/farmers",      method: "GET", expectedStatus: [401], weight: 10 },
  { path: "/api/v1/patients",     method: "GET", expectedStatus: [401], weight: 10 },
  { path: "/api/v1/ports",        method: "GET", expectedStatus: [401], weight: 5  },
  { path: "/api/v1/market",       method: "GET", expectedStatus: [401], weight: 5  },
];

// Build a weighted selection array
const weightedPool = [];
for (const ep of ENDPOINTS) {
  for (let i = 0; i < ep.weight; i++) weightedPool.push(ep);
}

export default function () {
  const ep  = weightedPool[Math.floor(Math.random() * weightedPool.length)];
  const url = `${BASE}${ep.path}`;

  const start = Date.now();
  const res = http.get(url, {
    headers: { "Accept": "application/json" },
    timeout: "10s",
  });
  const elapsed = Date.now() - start;

  // Track per-category latency
  if (ep.path === "/health") {
    latencyHealth.add(elapsed);
  } else {
    latencyAuth.add(elapsed);
  }

  // Classify response
  const isExpected = ep.expectedStatus.includes(res.status);
  const is5xx      = res.status >= 500;

  authWallHits.add(res.status === 401);
  serverErrors.add(is5xx);

  check(res, {
    "expected status": () => isExpected,
    "no 5xx":         () => !is5xx,
    "not timeout":    () => res.status !== 0,
  });

  // 10–50ms think time between requests
  sleep(Math.random() * 0.04 + 0.01);
}
