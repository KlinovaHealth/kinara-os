package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/klinova/kinara-os/patient-service/models"
)

// RateLimit enforces a sliding-window rate limit per mTLS client certificate CN
// (falling back to IP address). limit is the maximum requests per minute.
func RateLimit(rdb *redis.Client, limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(r)
			ctx := r.Context()

			count, err := increment(ctx, rdb, key)
			if err != nil {
				// Redis failure is non-fatal: allow the request through and log.
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, limit-count)))

			if count > limit {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				writeJSON(w, models.APIResponse{
					Success: false,
					Error: &models.APIError{
						Code:    "RATE_LIMITED",
						Message: "rate limit exceeded: 1000 requests per minute",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func increment(ctx context.Context, rdb *redis.Client, key string) (int, error) {
	pipe := rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int(incr.Val()), nil
}

func rateLimitKey(r *http.Request) string {
	// Prefer the mTLS client CN for per-service limiting.
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		return "rl:cert:" + r.TLS.PeerCertificates[0].Subject.CommonName
	}
	return "rl:ip:" + r.RemoteAddr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
