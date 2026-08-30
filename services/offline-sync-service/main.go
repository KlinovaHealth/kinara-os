package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/offline-sync-service/auth"
	"github.com/klinova/kinara-os/offline-sync-service/db"
	"github.com/klinova/kinara-os/offline-sync-service/handlers"
	"github.com/klinova/kinara-os/offline-sync-service/middleware"
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

	validator, err := auth.NewValidator(mustEnv("JWT_PUBLIC_KEY_PATH"))
	if err != nil {
		logger.Error("failed to load JWT public key", "error", err)
		os.Exit(1)
	}

	mtlsCfg := auth.MTLSConfig{
		CertPath:   mustEnv("TLS_CERT_PATH"),
		KeyPath:    mustEnv("TLS_KEY_PATH"),
		CACertPath: mustEnv("CA_CERT_PATH"),
	}

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

	// Patient service URL for cross-service data pull
	patientServiceURL := os.Getenv("PATIENT_SERVICE_URL")
	if patientServiceURL == "" {
		patientServiceURL = "https://patient-service:8081"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	tlsCfg, err := auth.BuildServerTLSConfig(mtlsCfg)
	if err != nil {
		logger.Error("failed to build TLS config", "error", err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(logger))
	r.Use(middleware.RateLimit(rdb, 1000))

	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"offline-sync-service"}`))
	}).Methods(http.MethodGet)

	r.Handle("/metrics", promhttp.Handler())

	queries := db.New(pool)
	jwtMW := middleware.JWT(validator)
	h := handlers.New(queries, patientServiceURL, logger)
	h.Register(r, jwtMW)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		TLSConfig:    tlsCfg,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("offline-sync-service starting", "port", port)
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
	logger.Info("offline-sync-service stopped")
}
