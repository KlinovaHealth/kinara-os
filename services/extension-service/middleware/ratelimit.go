package middleware

import (
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) func(http.Handler) http.Handler {
	rl := &rateLimiter{requests: make(map[string][]time.Time), limit: limit, window: window}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			rl.mu.Lock()
			now := time.Now()
			var recent []time.Time
			for _, t := range rl.requests[ip] {
				if now.Sub(t) < rl.window {
					recent = append(recent, t)
				}
			}
			rl.requests[ip] = append(recent, now)
			rl.mu.Unlock()
			if len(recent) >= rl.limit {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
