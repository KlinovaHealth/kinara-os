package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimit enforces a sliding-window rate limit per certificate CN or IP.
func RateLimit(rdb *redis.Client, requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "rl:general:" + clientKey(r)
			ctx := context.Background()

			pipe := rdb.Pipeline()
			incr := pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, time.Minute)
			if _, err := pipe.Exec(ctx); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			count := int(incr.Val())
			remaining := requestsPerMinute - count
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			if count > requestsPerMinute {
				http.Error(w, `{"success":false,"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CheckLoginRateLimit returns (allowed bool, attemptsLeft int).
// Enforces max 5 login attempts per minute per IP.
func CheckLoginRateLimit(rdb *redis.Client, r *http.Request) (bool, int) {
	const maxAttempts = 5
	key := "rl:login:" + remoteIP(r)
	ctx := context.Background()

	pipe := rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, maxAttempts // allow on Redis error, don't block
	}
	count := int(incr.Val())
	if count > maxAttempts {
		return false, 0
	}
	return true, maxAttempts - count
}

// RecordFailedLogin increments the failed-login counter for the given IP.
// Separate from CheckLoginRateLimit so the check and record are explicit.
func RecordFailedLogin(rdb *redis.Client, r *http.Request) {
	key := "rl:login:" + remoteIP(r)
	ctx := context.Background()
	pipe := rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	pipe.Exec(ctx)
}

func clientKey(r *http.Request) string {
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		if cn := r.TLS.PeerCertificates[0].Subject.CommonName; cn != "" {
			return "cert:" + cn
		}
	}
	return "ip:" + remoteIP(r)
}

func remoteIP(r *http.Request) string {
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}
