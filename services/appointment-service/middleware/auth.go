package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/klinova/kinara-os/appointment-service/auth"
)

type contextKey string

const claimsKey contextKey = "claims"

func JWT(v *auth.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
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
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(claimsKey).(*auth.Claims)
	return c
}

func SetClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}
