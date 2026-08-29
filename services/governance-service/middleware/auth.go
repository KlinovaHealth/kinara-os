package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/klinova/kinara-os/governance-service/auth"
)

type contextKey string

const ContextKeyClaims contextKey = "claims"

func JWT(publicKeyPath string) func(http.Handler) http.Handler {
	validator, err := auth.NewValidator(publicKeyPath)
	if err != nil {
		panic("governance-service: failed to load JWT public key: " + err.Error())
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"missing authorization header"}}`, http.StatusUnauthorized)
				return
			}
			claims, err := validator.Validate(strings.TrimPrefix(authHeader, "Bearer "))
			if err != nil {
				http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"invalid token"}}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(ContextKeyClaims).(*auth.Claims)
	return c
}
