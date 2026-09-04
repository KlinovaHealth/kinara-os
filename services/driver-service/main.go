package main

import (
	"context"
	"encoding/hex"
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

	"github.com/klinova/kinara-os/driver-service/auth"
	"github.com/klinova/kinara-os/driver-service/crypto"
	"github.com/klinova/kinara-os/driver-service/db"
	"github.com/klinova/kinara-os/driver-service/handlers"
	"github.com/klinova/kinara-os/driver-service/middleware"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
)

func main() {
	port := env("PORT", "8090")
	dsn := env("DATABASE_URL", "")
	redisAddr := env("REDIS_ADDR", "localhost:6379")
	jwtPubKey := env("JWT_PUBLIC_KEY_PATH", "/etc/kinara/jwt/public.pem")
	tlsCert := env("TLS_CERT_PATH", "/etc/kinara/tls/server.crt")
	tlsKey := env("TLS_KEY_PATH", "/etc/kinara/tls/server.key")
	tlsCA := env("TLS_CA_PATH", "/etc/kinara/tls/ca.crt")
	encKeyHex := env("ENCRYPTION_KEY", "")

	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if encKeyHex == "" {
		log.Fatal("ENCRYPTION_KEY is required (32-byte hex)")
	}

	encKeyBytes, err := hex.DecodeString(encKeyHex)
	if err != nil || len(encKeyBytes) != 32 {
		log.Fatal("ENCRYPTION_KEY must be a 64-character hex string (32 bytes)")
	}

	enc, err := crypto.NewEncryptor(encKeyBytes)
	if err != nil {
		log.Fatalf("encryptor init: %v", err)
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
	driverHandler := handlers.NewDriverHandler(queries, enc)

	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"driver-service"}`))
	}).Methods(http.MethodGet)

	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.Logging(slog.Default()))
	api.Use(middleware.RateLimit(rdb, 300))
	api.Use(middleware.JWT(jwtValidator))
	api.Use(pkgauth.RequireTenantScope("driver-service", nil))
	driverHandler.RegisterRoutes(api)

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
		log.Printf("driver-service listening on :%s (TLS)", port)
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
	log.Println("driver-service shutdown complete")
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
