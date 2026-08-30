package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
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
