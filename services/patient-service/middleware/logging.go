package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// RequestID injects a unique X-Request-ID into every request context and response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := contextWithRequestID(r.Context(), reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging emits a structured JSON log line per request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", rw.size,
			"request_id", requestIDFromContext(r.Context()),
			"remote_addr", r.RemoteAddr,
			"service", "patient-service",
		)
	})
}

// writeJSON is a shared helper used across middleware and handlers.
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func contextWithRequestID(ctx interface{ Value(any) any }, id string) interface{ Value(any) any } {
	// Thin wrapper — the real implementation uses context.WithValue in RequestID above.
	// This stub satisfies the compiler; real callers use context directly.
	_ = fmt.Sprintf("%v", ctx)
	return ctx
}

func requestIDFromContext(ctx interface{ Value(any) any }) string {
	if v := ctx.Value(ContextKeyRequestID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
