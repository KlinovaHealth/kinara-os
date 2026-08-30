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

	"github.com/klinova/kinara-os/voyage-service/auth"
	"github.com/klinova/kinara-os/voyage-service/db"
	"github.com/klinova/kinara-os/voyage-service/handlers"
	"github.com/klinova/kinara-os/voyage-service/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	jwtValidator, err := auth.NewValidator(os.Getenv("JWT_PUBLIC_KEY"))
	if err != nil {
		logger.Error("failed to load JWT public key", "error", err)
		os.Exit(1)
	}

	h := handlers.New(db.New(pool))
	r := mux.NewRouter()
	r.Use(middleware.JWT(jwtValidator))
	r.Use(middleware.Logging(logger))
	h.Register(r)
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"voyage-service"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8119"
	}
	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}
	logger.Info("voyage-service starting", "port", port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
