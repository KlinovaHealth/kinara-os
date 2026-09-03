package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// tenantScopeDecisions mirrors the counter defined in pkg/auth so the gateway
// and per-service measurements use the same metric name and can be aggregated
// in Grafana by the "service" label.
var tenantScopeDecisions = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "tenant_scope_decisions_total",
		Help: "Number of tenant scope enforcement decisions evaluated.",
	},
	[]string{"mode", "would_block", "service"},
)

func tenantScopeModeFromEnv() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("TENANT_SCOPE_MODE")))
	if v == "enforce" {
		return "enforce"
	}
	return "report"
}

// forwardAuth is the Traefik forwardAuth endpoint: GET /api/v1/validate.
//
// Traefik forwards every inbound request here before routing it upstream.
// On 200 the response headers X-User-ID, X-User-Role, X-Tenant-ID, X-Entity-Type,
// X-Scope-Would-Block, and X-Scope-Reason are forwarded to the upstream service.
// On 401/403 Traefik blocks the request.
//
// Chain position: AFTER JWT signature verification (happens in this handler),
// BEFORE the request reaches any upstream service. This is the gateway-level
// tenant scope check. Per-service enforcement (Phase 3) adds a second layer.
//
// TENANT_SCOPE_MODE controls blocking:
//   - "report" (default): log + metric, always return 200
//   - "enforce": 403 on any violation
func (h *Handler) forwardAuth(w http.ResponseWriter, r *http.Request) {
	mode := tenantScopeModeFromEnv()

	// ── 1. Extract and validate the Bearer token ──────────────────────────────
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		http.Error(w, `{"code":"UNAUTHORIZED","message":"missing bearer token"}`, http.StatusUnauthorized)
		return
	}
	tokenStr := strings.TrimPrefix(header, "Bearer ")

	claims, err := h.issuer.Validate(tokenStr)
	if err != nil {
		http.Error(w, `{"code":"UNAUTHORIZED","message":"invalid or expired token"}`, http.StatusUnauthorized)
		return
	}

	// ── 2. Tenant scope check (same logic as pkg/auth.RequireTenantScope) ─────
	wouldBlock := false
	reason := "ok"

	switch {
	case claims.EntityType == "":
		wouldBlock = true
		reason = "missing_entity_type"
	}
	// clinic_id cross-tenant check requires a DB lookup; at the gateway we only
	// have the token. Clinic enforcement is deferred to per-service middleware
	// (Phase 3) which has access to the tenant→clinic mapping.

	// ── 3. Log the decision ───────────────────────────────────────────────────
	clinicID := uuid.Nil
	if claims.ClinicID != nil {
		clinicID = *claims.ClinicID
	}

	wouldBlockStr := "false"
	if wouldBlock {
		wouldBlockStr = "true"
	}

	slog.Info("tenant_scope_decision",
		"mode", mode,
		"would_block", wouldBlock,
		"reason", reason,
		"service", "gateway",
		"path", r.Header.Get("X-Forwarded-Uri"),
		"entity_type", claims.EntityType,
		"tenant_id", claims.TenantID.String(),
		"clinic_id", clinicID.String(),
		"user_id", claims.UserID.String(),
	)

	tenantScopeDecisions.WithLabelValues(mode, wouldBlockStr, "gateway").Inc()

	// ── 4. Block or pass ──────────────────────────────────────────────────────
	if wouldBlock && mode == "enforce" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w,
			`{"code":"FORBIDDEN","message":"`+reason+`"}`,
			http.StatusForbidden,
		)
		return
	}

	// ── 5. Set response headers forwarded to upstream by Traefik ─────────────
	scopeStr := ""
	if len(claims.Scopes) > 0 {
		scopeStr = strings.Join(claims.Scopes, ",")
	}

	w.Header().Set("X-User-ID", claims.UserID.String())
	w.Header().Set("X-User-Role", claims.Role)
	w.Header().Set("X-Tenant-ID", claims.TenantID.String())
	w.Header().Set("X-Entity-Type", claims.EntityType)
	w.Header().Set("X-Scope-Would-Block", wouldBlockStr)
	w.Header().Set("X-Scope-Reason", reason)
	w.Header().Set("X-User-Scopes", scopeStr)
	w.WriteHeader(http.StatusOK)
}
