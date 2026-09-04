package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/customs-service/auth"
	"github.com/klinova/kinara-os/customs-service/db"
	"github.com/klinova/kinara-os/customs-service/handlers"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
	"github.com/klinova/kinara-os/customs-service/middleware"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/redis/go-redis/v9"
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

	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL")})
	r := mux.NewRouter()
	r.Use(middleware.Logging(slog.Default()))
	r.Use(middleware.RateLimit(rdb, 200))
	r.Use(middleware.JWT(jwtValidator))
	r.Use(pkgauth.RequireTenantScope("customs-service", nil))

	api := r.PathPrefix("/api/v1").Subrouter()
	h.RegisterRoutes(api)

	r.Handle("/metrics", promhttp.Handler())

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
