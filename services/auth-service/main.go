package main

import (
	"context"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/auth-service/auth"
	"github.com/klinova/kinara-os/auth-service/crypto"
	"github.com/klinova/kinara-os/auth-service/db"
	"github.com/klinova/kinara-os/auth-service/handlers"
	"github.com/klinova/kinara-os/auth-service/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("missing required environment variable", "key", key)
		os.Exit(1)
	}
	return v
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// ── Encryption key ────────────────────────────────────────────────────────
	keyHex := mustEnv("ENCRYPTION_KEY")
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil || len(keyBytes) != 32 {
		logger.Error("ENCRYPTION_KEY must be a 64-character hex string (32 bytes)")
		os.Exit(1)
	}
	enc, err := crypto.NewEncryptor(keyBytes)
	if err != nil {
		logger.Error("failed to create encryptor", "error", err)
		os.Exit(1)
	}

	// ── JWT issuer (RS256: private key signs, public key validates) ───────────
	issuer, err := auth.NewIssuer(mustEnv("JWT_PRIVATE_KEY_PATH"), mustEnv("JWT_PUBLIC_KEY_PATH"))
	if err != nil {
		logger.Error("failed to load JWT keys", "error", err)
		os.Exit(1)
	}

	// ── mTLS config (CA key only needed for cert issuance) ────────────────────
	mtlsCfg := auth.MTLSConfig{
		CertPath:   mustEnv("TLS_CERT_PATH"),
		KeyPath:    mustEnv("TLS_KEY_PATH"),
		CACertPath: mustEnv("CA_CERT_PATH"),
		CAKeyPath:  os.Getenv("CA_KEY_PATH"), // optional
	}

	// ── PostgreSQL ────────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(mustEnv("DATABASE_URL"))
	if err != nil {
		logger.Error("invalid DATABASE_URL", "error", err)
		os.Exit(1)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Retry ping with backoff so services don't stampede the DB connection pool at startup.
	for attempt := 1; attempt <= 10; attempt++ {
		if err := pool.Ping(context.Background()); err == nil {
			break
		} else if attempt == 10 {
			logger.Error("database ping failed after retries", "error", err)
			os.Exit(1)
		} else {
			logger.Warn("database ping failed, retrying", "attempt", attempt, "error", err)
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}

	// ── Redis ─────────────────────────────────────────────────────────────────
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	// ── TLS server config ─────────────────────────────────────────────────────
	tlsCfg, err := auth.BuildServerTLSConfig(mtlsCfg)
	if err != nil {
		logger.Error("failed to build TLS config", "error", err)
		os.Exit(1)
	}

	// ── Router ────────────────────────────────────────────────────────────────
	r := mux.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(logger))
	r.Use(middleware.RateLimit(rdb, 1000))

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"auth-service"}`))
	}).Methods(http.MethodGet)

	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, `{"status":"not ready","error":"db unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			http.Error(w, `{"status":"not ready","error":"redis unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","service":"auth-service"}`))
	}).Methods(http.MethodGet)

	r.Handle("/metrics", promhttp.Handler())

	queries := db.New(pool)
	jwtMiddleware := middleware.JWT(issuer)

	h := handlers.New(queries, enc, issuer, mtlsCfg, rdb, logger)
	h.Register(r, jwtMiddleware)

	// ── Server ────────────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		TLSConfig:    tlsCfg,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("auth-service starting", "port", port)
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
	logger.Info("auth-service stopped")
}
