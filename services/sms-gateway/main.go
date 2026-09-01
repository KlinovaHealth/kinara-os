package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/sms-gateway/db"
	"github.com/klinova/kinara-os/sms-gateway/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	h := handlers.NewHandler(queries)

	r := mux.NewRouter()
	h.RegisterRoutes(r)
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok","service":"sms-gateway"}`)
	}).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8200"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Printf("sms-gateway listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
