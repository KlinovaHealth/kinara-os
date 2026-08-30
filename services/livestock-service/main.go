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

	"github.com/klinova/kinara-os/livestock-service/auth"
	"github.com/klinova/kinara-os/livestock-service/db"
	"github.com/klinova/kinara-os/livestock-service/handlers"
	"github.com/klinova/kinara-os/livestock-service/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	jwtKeyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
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
	r.Use(middleware.JWT(validator))
	r.Use(middleware.Logging(logger))
	h.Register(r)
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"livestock-service"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}
	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}
	logger.Info("livestock-service starting", "port", port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
