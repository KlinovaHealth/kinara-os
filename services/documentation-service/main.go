package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/documentation-service/auth"
	"github.com/klinova/kinara-os/documentation-service/db"
	"github.com/klinova/kinara-os/documentation-service/handlers"
	"github.com/klinova/kinara-os/documentation-service/middleware"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/redis/go-redis/v9"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	jwtValidator, err := auth.NewValidator(os.Getenv("JWT_PUBLIC_KEY_PATH"))
	if err != nil {
		log.Fatalf("jwt validator: %v", err)
	}

	queries := db.New(pool)
	h := handlers.NewHandler(queries)
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL")})
	r := mux.NewRouter()
	r.Use(middleware.Logging(slog.Default()))
	r.Use(middleware.RateLimit(rdb, 200))
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.JWT(jwtValidator))
	h.RegisterRoutes(api)

	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok","service":"documentation-service"}`)
	}).Methods("GET")

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
	srv := &http.Server{
		Addr:         ":8106",
		Handler:      r,
		TLSConfig:    tlsCfg,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	if certFile != "" && keyFile != "" {
		log.Println("documentation-service listening on :8106 (TLS)")
		log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
	} else {
		log.Println("documentation-service listening on :8106")
		srv.TLSConfig = nil
		log.Fatal(srv.ListenAndServe())
	}
}
