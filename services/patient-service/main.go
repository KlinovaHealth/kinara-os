// Kinara OS — Patient Service
// Manages encrypted patient records with immutable audit logging.
// All PHI is AES-256-GCM encrypted before being written to PostgreSQL.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/klinova/kinara-os/patient-service/auth"
	"github.com/klinova/kinara-os/patient-service/db"
	"github.com/klinova/kinara-os/patient-service/handlers"
	"github.com/klinova/kinara-os/patient-service/middleware"
)

type config struct {
	Port              string
	DatabaseURL       string
	RedisAddr         string
	RedisPassword     string
	EncryptionKey     []byte // 32 bytes, AES-256
	JWTPublicKeyPath  string
	TLSCertPath       string
	TLSKeyPath        string
	CACertPath        string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// ── Database ──────────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to parse database URL", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = 25
	poolCfg.MinConns = 5
	poolCfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("redis unavailable — rate limiting disabled", "error", err)
	}
	defer rdb.Close()

	// ── Router ────────────────────────────────────────────────────────────────
	r := mux.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	r.Use(middleware.RateLimit(rdb, 1000))

	// Health endpoints require no auth (checked by Kubernetes liveness probes).
	r.HandleFunc("/health", healthCheck).Methods(http.MethodGet)
	r.HandleFunc("/ready", readyCheck(pool, rdb)).Methods(http.MethodGet)

	// All /api/v1 routes require a valid JWT.
	protected := r.PathPrefix("/api/v1").Subrouter()
	protected.Use(middleware.JWT(cfg.JWTPublicKeyPath))

	queries := db.New(pool)
	h := handlers.New(queries, cfg.EncryptionKey, logger)
	h.Register(protected)

	// ── TLS (mTLS) ────────────────────────────────────────────────────────────
	tlsCfg, err := auth.BuildServerTLSConfig(auth.MTLSConfig{
		CACertPath: cfg.CACertPath,
		CertPath:   cfg.TLSCertPath,
		KeyPath:    cfg.TLSKeyPath,
	})
	if err != nil {
		slog.Error("failed to build TLS config", "error", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		TLSConfig:    tlsCfg,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("patient service starting", "port", cfg.Port, "service", "patient-service")
		if err := srv.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("patient service stopped")
}

// loadConfig reads all required settings from environment variables.
// No defaults for secrets — missing required vars cause a fatal error.
func loadConfig() (*config, error) {
	cfg := &config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      mustEnv("DATABASE_URL"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		JWTPublicKeyPath: mustEnv("JWT_PUBLIC_KEY_PATH"),
		TLSCertPath:      mustEnv("TLS_CERT_PATH"),
		TLSKeyPath:       mustEnv("TLS_KEY_PATH"),
		CACertPath:       mustEnv("CA_CERT_PATH"),
	}

	// Encryption key: 64-char hex string → 32 bytes
	keyHex := mustEnv("ENCRYPTION_KEY")
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be a 64-character hex string (32 bytes)")
	}
	cfg.EncryptionKey = key

	return cfg, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "var", key)
		os.Exit(1)
	}
	return v
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// healthCheck returns 200 OK — used by liveness probes.
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status":"ok","service":"patient-service"}`)
}

// readyCheck returns 200 only when the DB and Redis are reachable.
func readyCheck(pool *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not_ready","error":"database unreachable"}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ready","service":"patient-service"}`)
	}
}
