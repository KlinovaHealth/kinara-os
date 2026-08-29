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
	"github.com/klinova/kinara-os/analytics-service/auth"
	"github.com/klinova/kinara-os/analytics-service/db"
	"github.com/klinova/kinara-os/analytics-service/handlers"
	"github.com/klinova/kinara-os/analytics-service/middleware"
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
	r := mux.NewRouter()
	r.Use(middleware.Logging(slog.Default()))
	r.Use(middleware.RateLimit(os.Getenv("REDIS_URL"), "analytics-service", 300, time.Minute))
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.JWTAuth(jwtValidator))
	h.RegisterRoutes(api)

	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok","service":"analytics-service"}`)
	}).Methods("GET")

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
	srv := &http.Server{
		Addr:         ":8108",
		Handler:      r,
		TLSConfig:    tlsCfg,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	if certFile != "" && keyFile != "" {
		log.Println("analytics-service listening on :8108 (TLS)")
		log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
	} else {
		log.Println("analytics-service listening on :8108")
		srv.TLSConfig = nil
		log.Fatal(srv.ListenAndServe())
	}
}
