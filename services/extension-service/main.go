package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/klinova/kinara-os/extension-service/auth"
	"github.com/klinova/kinara-os/extension-service/db"
	"github.com/klinova/kinara-os/extension-service/handlers"
	"github.com/klinova/kinara-os/extension-service/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	jwtKeyPath := os.Getenv("JWT_PUBLIC_KEY")
	if jwtKeyPath == "" {
		jwtKeyPath = "/run/secrets/jwt_public.pem"
	}
	validator, err := auth.NewValidator(jwtKeyPath)
	if err != nil {
		logger.Error("jwt validator init failed", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	h := handlers.New(db.New(pool))
	r := mux.NewRouter()
	r.Use(middleware.Logging(logger))
	r.Use(middleware.JWT(validator))
	h.Register(r)

	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"extension-service"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8126"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	logger.Info("extension-service starting", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
