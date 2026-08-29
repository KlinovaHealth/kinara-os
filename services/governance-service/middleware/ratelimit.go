package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func RateLimit(rdb *redis.Client, requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(r)
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
				http.Error(w, `{"success":false,"error":{"code":"RATE_LIMITED","message":"rate limit exceeded"}}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitKey(r *http.Request) string {
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		cn := r.TLS.PeerCertificates[0].Subject.CommonName
		if cn != "" {
			return "rl:cert:" + cn
		}
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return "rl:ip:" + addr
}
