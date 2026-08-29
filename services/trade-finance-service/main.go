package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/trade-finance-service/auth"
	"github.com/klinova/kinara-os/trade-finance-service/db"
	"github.com/klinova/kinara-os/trade-finance-service/handlers"
	"github.com/klinova/kinara-os/trade-finance-service/middleware"
)

func main() {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil { log.Fatalf("db connect: %v", err) }
	defer pool.Close()

	jwtValidator, err := auth.NewValidator(os.Getenv("JWT_PUBLIC_KEY_PATH"))
	if err != nil { log.Fatalf("jwt: %v", err) }

	tlsCfg, err := auth.BuildServerTLSConfig(auth.MTLSConfig{
		CertPath: os.Getenv("TLS_CERT_FILE"),
		KeyPath:  os.Getenv("TLS_KEY_FILE"),
		CACertPath:   os.Getenv("TLS_CA_FILE"),
	})
	if err != nil { log.Fatalf("tls: %v", err) }

	queries := db.New(pool)
	h := handlers.NewHandler(queries)

	r := mux.NewRouter()
	r.Use(middleware.Logging(slog.Default()))
	r.Use(middleware.RateLimit(os.Getenv("REDIS_URL"), "trade-finance-service", 200))
	r.Use(middleware.Auth(jwtValidator))

	api := r.PathPrefix("/api/v1").Subrouter()
	h.RegisterRoutes(api)

	srv := &http.Server{
		Addr:      ":8105",
		Handler:   r,
		TLSConfig: tlsCfg,
	}
	log.Println("trade-finance-service listening on :8105")
	if err := srv.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("server: %v", err)
	}
}
