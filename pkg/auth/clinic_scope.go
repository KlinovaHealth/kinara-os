package auth

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type contextKey string

const (
	claimsCtxKey   contextKey = "kinara_claims"
	clinicIDCtxKey contextKey = "kinara_clinic_id"
)

// ClaimsFromContext retrieves the validated Claims from the request context.
// Returns nil if no claims are present (unauthenticated request).
func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsCtxKey).(*Claims)
	return c
}

// ClinicIDFromContext retrieves the clinic_id injected by RequireClinicScope.
// Returns uuid.Nil if not present.
func ClinicIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(clinicIDCtxKey).(uuid.UUID)
	return id
}

// InjectClaims injects validated JWT claims into the request context.
// Called by the JWT authentication middleware after token validation.
func InjectClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsCtxKey, c)
}

// RequireClinicScope is an HTTP middleware that enforces the "clinic:<id>" scope on
// any handler that accesses patient or clinic-scoped PHI.
//
// Behaviour:
//   - If the token has a device scope ("clinic:<uuid>"), it extracts the clinic_id
//     and injects it into the context. Handlers MUST use ClinicIDFromContext to
//     scope their DB queries.
//   - If the token has no scope (human admin session), the clinic_id is uuid.Nil
//     and the handler is responsible for validating access itself.
//   - If claims are missing entirely, it fails closed with 403.
//
// Fail closed: no claim = 403. This protects PHI even if upstream JWT middleware
// is accidentally bypassed.
func RequireClinicScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, `{"success":false,"error":{"code":"FORBIDDEN","message":"clinic scope required"}}`, http.StatusForbidden)
			return
		}

		ctx := r.Context()

		if claims.IsDeviceSession() {
			// Device session: enforce clinic scope
			if claims.ClinicID == nil {
				http.Error(w, `{"success":false,"error":{"code":"FORBIDDEN","message":"missing clinic_id in device token"}}`, http.StatusForbidden)
				return
			}
			ctx = context.WithValue(ctx, clinicIDCtxKey, *claims.ClinicID)
		}
		// Human admin session: clinic_id stays uuid.Nil; handler validates access.

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// JWTMiddleware is a factory that builds a Bearer-token extraction + validation
// middleware using the shared Validator, injecting Claims into the context.
func JWTMiddleware(v *Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"missing bearer token"}}`, http.StatusUnauthorized)
				return
			}
			claims, err := v.Validate(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"invalid token"}}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(InjectClaims(r.Context(), claims)))
		})
	}
}

// TenantScopeMode controls whether RequireTenantScope blocks on violations or only reports them.
// Read once at middleware construction from TENANT_SCOPE_MODE env var.
type TenantScopeMode string

const (
	TenantScopeModeReport  TenantScopeMode = "report"
	TenantScopeModeEnforce TenantScopeMode = "enforce"
)

func tenantScopeModeFromEnv() TenantScopeMode {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TENANT_SCOPE_MODE")), "enforce") {
		return TenantScopeModeEnforce
	}
	return TenantScopeModeReport
}

// tenantScopeDecisions counts every RequireTenantScope evaluation.
// Labels: mode ("report"|"enforce"), would_block ("true"|"false"), service (service name).
var tenantScopeDecisions = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "tenant_scope_decisions_total",
		Help: "Number of tenant scope enforcement decisions evaluated.",
	},
	[]string{"mode", "would_block", "service"},
)

// RequireTenantScope evaluates tenant isolation on every authenticated request.
//
// Mode is read once at construction from TENANT_SCOPE_MODE:
//   - "report" (default): evaluate, log, emit metric — always allow through
//   - "enforce": 403 on any violation
//
// serviceName is the label value on the tenant_scope_decisions_total counter.
// tenantClinics, if non-nil, verifies that a device-session clinic_id belongs to the
// token's tenant. Pass nil to report on entity_type/tenant_id presence only.
func RequireTenantScope(serviceName string, tenantClinics func(tenantID uuid.UUID) (map[uuid.UUID]struct{}, error)) func(http.Handler) http.Handler {
	mode := tenantScopeModeFromEnv()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())

			wouldBlock := false
			reason := "ok"

			switch {
			case claims == nil:
				wouldBlock = true
				reason = "no_claims"

			case claims.EntityType == "":
				// Token predates migration 002 — entity not stamped at login.
				wouldBlock = true
				reason = "missing_entity_type"

			case claims.ClinicID != nil && *claims.ClinicID != uuid.Nil && tenantClinics != nil:
				allowed, err := tenantClinics(claims.TenantID)
				if err != nil {
					wouldBlock = true
					reason = "tenant_lookup_error"
				} else if _, ok := allowed[*claims.ClinicID]; !ok {
					wouldBlock = true
					reason = "clinic_not_in_tenant"
				}
			}

			// Safe field extraction — handles nil claims.
			entityType := ""
			tenantID := uuid.Nil
			clinicID := uuid.Nil
			userID := uuid.Nil
			if claims != nil {
				entityType = claims.EntityType
				tenantID = claims.TenantID
				userID = claims.UserID
				if claims.ClinicID != nil {
					clinicID = *claims.ClinicID
				}
			}

			wouldBlockStr := "false"
			if wouldBlock {
				wouldBlockStr = "true"
			}

			slog.Info("tenant_scope_decision",
				"mode", string(mode),
				"would_block", wouldBlock,
				"reason", reason,
				"service", serviceName,
				"path", r.URL.Path,
				"entity_type", entityType,
				"tenant_id", tenantID.String(),
				"clinic_id", clinicID.String(),
				"user_id", userID.String(),
			)

			tenantScopeDecisions.WithLabelValues(string(mode), wouldBlockStr, serviceName).Inc()

			if wouldBlock && mode == TenantScopeModeEnforce {
				http.Error(w,
					`{"success":false,"error":{"code":"FORBIDDEN","message":"`+reason+`"}}`,
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IsClinicScoped returns true when the scope string matches "clinic:<uuid>".
func IsClinicScoped(scope string) bool {
	return strings.HasPrefix(scope, "clinic:") && len(scope) > len("clinic:")
}

// ClinicIDFromScope parses "clinic:<uuid>" and returns the UUID.
func ClinicIDFromScope(scope string) (uuid.UUID, error) {
	if !IsClinicScoped(scope) {
		return uuid.Nil, nil
	}
	return uuid.Parse(strings.TrimPrefix(scope, "clinic:"))
}
