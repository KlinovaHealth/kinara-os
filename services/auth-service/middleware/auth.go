package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/klinova/kinara-os/auth-service/auth"
)

type contextKey string

const ContextKeyClaims contextKey = "claims"

// JWT returns middleware that validates Bearer tokens for protected endpoints
// within the auth service itself (e.g., profile, MFA, admin routes).
func JWT(issuer *auth.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"missing authorization header"}}`, http.StatusUnauthorized)
				return
			}
			claims, err := issuer.Validate(strings.TrimPrefix(authHeader, "Bearer "))
			if err != nil {
				http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"invalid or expired token"}}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves JWT claims from a request context.
func ClaimsFromContext(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(ContextKeyClaims).(*auth.Claims)
	return c
}

// RequireRole returns a middleware that enforces one of the allowed roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil || !allowed[claims.Role] {
				http.Error(w, `{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient role"}}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
