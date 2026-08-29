package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/klinova/kinara-os/last-mile-service/auth"
	"github.com/klinova/kinara-os/last-mile-service/db"
	"github.com/klinova/kinara-os/last-mile-service/handlers"
	"github.com/klinova/kinara-os/last-mile-service/middleware"
)

func main() {
	port := env("PORT", "8094")
	dsn := env("DATABASE_URL", "")
	redisAddr := env("REDIS_ADDR", "localhost:6379")
	jwtPubKey := env("JWT_PUBLIC_KEY_PATH", "/etc/kinara/jwt/public.pem")
	tlsCert := env("TLS_CERT_PATH", "/etc/kinara/tls/server.crt")
	tlsKey := env("TLS_KEY_PATH", "/etc/kinara/tls/server.key")
	tlsCA := env("TLS_CA_PATH", "/etc/kinara/tls/ca.crt")

	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	jwtValidator, err := auth.NewValidator(jwtPubKey)
	if err != nil {
		log.Fatalf("jwt validator: %v", err)
	}

	queries := db.New(pool)
	lastMileHandler := handlers.NewLastMileHandler(queries)

	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"last-mile-service"}`))
	}).Methods(http.MethodGet)

	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.Logging(slog.Default()))
	api.Use(middleware.RateLimit(rdb, 300))
	api.Use(middleware.JWT(jwtValidator))
	lastMileHandler.RegisterRoutes(api)

	tlsCfg, err := auth.BuildServerTLSConfig(auth.MTLSConfig{
		CertPath: tlsCert,
		KeyPath:  tlsKey,
		CACertPath:   tlsCA,
	})
	if err != nil {
		log.Fatalf("tls config: %v", err)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		TLSConfig:    tlsCfg,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("last-mile-service listening on :%s (TLS)", port)
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("last-mile-service shutdown complete")
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
