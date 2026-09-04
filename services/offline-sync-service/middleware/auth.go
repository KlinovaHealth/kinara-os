package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/offline-sync-service/auth"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
)

type contextKey string

const (
	claimsKey   contextKey = "claims"
	clinicIDKey contextKey = "clinic_id"
)

func JWT(v *auth.Validator) func(http.Handler) http.Handler {
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
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			ctx = pkgauth.InjectClaims(ctx, &pkgauth.Claims{
				UserID:     claims.UserID,
				Role:       claims.Role,
				Scopes:     claims.Scopes,
				EntityType: claims.EntityType,
				TenantID:   claims.TenantID,
				ClinicID:   claims.ClinicID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireClinicScope enforces "clinic:<id>" scope on all sync endpoints.
// Fail closed: no valid device session = 403.
func RequireClinicScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims == nil {
			forbidden(w, "clinic scope required")
			return
		}
		if !claims.IsDeviceSession() {
			forbidden(w, "device token with clinic scope required")
			return
		}
		if claims.ClinicID == nil {
			forbidden(w, "missing clinic_id in device token")
			return
		}
		ctx := context.WithValue(r.Context(), clinicIDKey, *claims.ClinicID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ClaimsFromContext(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(claimsKey).(*auth.Claims)
	return c
}

func ClinicIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(clinicIDKey).(uuid.UUID)
	return id, ok
}

func forbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `{"success":false,"error":{"code":"FORBIDDEN","message":"%s"}}`, msg)
}
