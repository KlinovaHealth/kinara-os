package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/offline-sync-service/auth"
)

// InjectClaimsForTest injects claims into a context for unit tests.
// Only use in _test.go files.
func InjectClaimsForTest(ctx context.Context, c *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

// InjectClinicIDForTest injects a clinic ID into a context for unit tests.
func InjectClinicIDForTest(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, clinicIDKey, id)
}
