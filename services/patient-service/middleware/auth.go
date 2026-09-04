// Package middleware contains HTTP middleware for the patient service.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/klinova/kinara-os/patient-service/auth"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
	"github.com/klinova/kinara-os/patient-service/models"
)

type contextKey string

const (
	ContextKeyClaims   contextKey = "claims"
	ContextKeyRequestID contextKey = "request_id"
)

// JWT returns middleware that validates the Bearer token in every request.
// Unauthenticated requests receive 401; requests with a valid token have
// the parsed Claims injected into the request context.
func JWT(publicKeyPath string) func(http.Handler) http.Handler {
	validator, err := auth.NewValidator(publicKeyPath)
	if err != nil {
		// Fail fast at startup if the key cannot be loaded.
		panic("middleware: failed to initialise JWT validator: " + err.Error())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				respondUnauthorized(w, "authorization header must be 'Bearer <token>'")
				return
			}

			claims, err := validator.Validate(parts[1])
			if err != nil {
				respondUnauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
			ctx = pkgauth.InjectClaims(ctx, &pkgauth.Claims{
				UserID:     claims.UserID,
				Role:       claims.Role,
				Scopes:     claims.Scopes,
				EntityType: claims.EntityType,
				TenantID:   claims.TenantID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext extracts the JWT claims injected by the JWT middleware.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(ContextKeyClaims).(*auth.Claims)
	return c, ok
}

func respondUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	resp := models.APIResponse{
		Success: false,
		Error: &models.APIError{
			Code:    "UNAUTHORIZED",
			Message: msg,
		},
	}
	writeJSON(w, resp)
}
