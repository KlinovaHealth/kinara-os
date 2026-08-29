package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/customs-service/auth"
	"github.com/klinova/kinara-os/customs-service/db"
	"github.com/klinova/kinara-os/customs-service/handlers"
	"github.com/klinova/kinara-os/customs-service/middleware"
)

func main() {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil { log.Fatalf("db connect: %v", err) }
	defer pool.Close()

	jwtValidator, err := auth.NewValidator(os.Getenv("JWT_PUBLIC_KEY_PATH"))
	if err != nil { log.Fatalf("jwt: %v", err) }

	tlsCfg, err := auth.BuildServerTLSConfig(auth.MTLSConfig{
		CertFile: os.Getenv("TLS_CERT_FILE"),
		KeyFile:  os.Getenv("TLS_KEY_FILE"),
		CAFile:   os.Getenv("TLS_CA_FILE"),
	})
	if err != nil { log.Fatalf("tls: %v", err) }

	queries := db.New(pool)
	h := handlers.NewHandler(queries)

	r := mux.NewRouter()
	r.Use(middleware.Logging)
	r.Use(middleware.RateLimit(os.Getenv("REDIS_URL"), "customs-service", 200))
	r.Use(middleware.Auth(jwtValidator))

	api := r.PathPrefix("/api/v1").Subrouter()
	h.RegisterRoutes(api)

	srv := &http.Server{
		Addr:      ":8102",
		Handler:   r,
		TLSConfig: tlsCfg,
	}
	log.Println("customs-service listening on :8102")
	if err := srv.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("server: %v", err)
	}
}
