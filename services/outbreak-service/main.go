package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/klinova/kinara-os/outbreak-service/auth"
	"github.com/klinova/kinara-os/outbreak-service/db"
	"github.com/klinova/kinara-os/outbreak-service/handlers"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
	"github.com/klinova/kinara-os/outbreak-service/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	jwtValidator, err := auth.NewValidator(os.Getenv("JWT_PUBLIC_KEY_PATH"))
	if err != nil {
		log.Fatalf("jwt validator: %v", err)
	}

	h := handlers.New(db.New(pool))
	r := mux.NewRouter()
	r.Use(middleware.JWT(jwtValidator))
	r.Use(pkgauth.RequireTenantScope("outbreak-service", nil))
	r.Use(middleware.Logging(logger))
	h.Register(r)
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"outbreak-service"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8123"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	logger.Info("outbreak-service starting", "port", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
