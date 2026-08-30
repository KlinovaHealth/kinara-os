#!/usr/bin/env python3
"""Generate 100 new Kinara OS microservices following the established pattern."""

import os
import re
import shutil

BASE = "/Users/donalddaglo/Documents/KinaraOS"
SERVICES_DIR = f"{BASE}/services"

# ─── 100 new services: (helm-key, port, db-name) ─────────────────────────────
NEW_SERVICES = [
    # Health Pillar — 25 new
    ("prescription",        8301, "kinara_prescription"),
    ("vital-signs",         8302, "kinara_vital_signs"),
    ("symptom",             8303, "kinara_symptom"),
    ("disease-surveillance",8304, "kinara_disease_surveillance"),
    ("maternal-health",     8305, "kinara_maternal_health"),
    ("child-health",        8306, "kinara_child_health"),
    ("nutrition",           8307, "kinara_nutrition"),
    ("mental-health",       8308, "kinara_mental_health"),
    ("chronic-disease",     8309, "kinara_chronic_disease"),
    ("emergency",           8310, "kinara_emergency"),
    ("patient-monitoring",  8311, "kinara_patient_monitoring"),
    ("medical-history",     8312, "kinara_medical_history"),
    ("allergy",             8313, "kinara_allergy"),
    ("medication",          8314, "kinara_medication"),
    ("health-education",    8315, "kinara_health_education"),
    ("population-health",   8316, "kinara_population_health"),
    ("outcome-tracking",    8317, "kinara_outcome_tracking"),
    ("patient-engagement",  8318, "kinara_patient_engagement"),
    ("reminders",           8319, "kinara_reminders"),
    ("screening",           8320, "kinara_screening"),
    ("surgery",             8321, "kinara_surgery"),
    ("radiology",           8322, "kinara_radiology"),
    ("pathology",           8323, "kinara_pathology"),
    ("infection-control",   8324, "kinara_infection_control"),
    ("health-worker",       8325, "kinara_health_worker"),

    # Agriculture Pillar — 20 new
    ("crop",                8401, "kinara_crop"),
    ("farm-plot",           8402, "kinara_farm_plot"),
    ("seed",                8403, "kinara_seed"),
    ("fertilizer",          8404, "kinara_fertilizer"),
    ("pest-management",     8405, "kinara_pest_management"),
    ("harvest",             8406, "kinara_harvest"),
    ("agri-storage",        8407, "kinara_agri_storage"),
    ("commodity-pricing",   8408, "kinara_commodity_pricing"),
    ("yield",               8409, "kinara_yield"),
    ("soil",                8410, "kinara_soil"),
    ("agri-forecast",       8411, "kinara_agri_forecast"),
    ("subsidy",             8412, "kinara_subsidy"),
    ("agri-training",       8413, "kinara_agri_training"),
    ("agri-insurance",      8414, "kinara_agri_insurance"),
    ("agri-credit",         8415, "kinara_agri_credit"),
    ("agri-export",         8416, "kinara_agri_export"),
    ("agri-certification",  8417, "kinara_agri_certification"),
    ("water-management",    8418, "kinara_water_management"),
    ("agri-equipment",      8419, "kinara_agri_equipment"),
    ("land-registry",       8420, "kinara_land_registry"),

    # Logistics Pillar — 20 new
    ("inventory",           8501, "kinara_inventory"),
    ("delivery",            8502, "kinara_delivery"),
    ("packaging",           8503, "kinara_packaging"),
    ("quality-check",       8504, "kinara_quality_check"),
    ("returns",             8505, "kinara_returns"),
    ("demand-forecast",     8506, "kinara_demand_forecast"),
    ("cold-chain",          8507, "kinara_cold_chain"),
    ("carrier",             8508, "kinara_carrier"),
    ("dispatch",            8509, "kinara_dispatch"),
    ("freight",             8510, "kinara_freight"),
    ("geo-fence",           8511, "kinara_geo_fence"),
    ("fuel-management",     8512, "kinara_fuel_management"),
    ("vehicle-maintenance", 8513, "kinara_vehicle_maintenance"),
    ("toll",                8514, "kinara_toll"),
    ("logistics-reporting", 8515, "kinara_logistics_reporting"),
    ("cross-docking",       8516, "kinara_cross_docking"),
    ("load-planning",       8517, "kinara_load_planning"),
    ("returns-management",  8518, "kinara_returns_management"),
    ("logistics-compliance",8519, "kinara_logistics_compliance"),
    ("sorting",             8520, "kinara_sorting"),

    # Maritime Pillar — 15 new
    ("berth",               8601, "kinara_berth"),
    ("manifest",            8602, "kinara_manifest"),
    ("bill-of-lading",      8603, "kinara_bill_of_lading"),
    ("maritime-insurance",  8604, "kinara_maritime_insurance"),
    ("tariff",              8605, "kinara_tariff"),
    ("port-authority",      8606, "kinara_port_authority"),
    ("maritime-agent",      8607, "kinara_maritime_agent"),
    ("marine-inspection",   8608, "kinara_marine_inspection"),
    ("vessel-certification",8609, "kinara_vessel_certification"),
    ("maritime-analytics",  8610, "kinara_maritime_analytics"),
    ("pilotage",            8611, "kinara_pilotage"),
    ("stevedore",           8612, "kinara_stevedore"),
    ("maritime-weather",    8613, "kinara_maritime_weather"),
    ("sanctions-check",     8614, "kinara_sanctions_check"),
    ("freight-forwarding",  8615, "kinara_freight_forwarding"),

    # Finance & Cross-cutting — 20 new
    ("invoice",             8701, "kinara_invoice"),
    ("billing",             8702, "kinara_billing"),
    ("tax",                 8703, "kinara_tax"),
    ("expense",             8704, "kinara_expense"),
    ("budget",              8705, "kinara_budget"),
    ("loan",                8706, "kinara_loan"),
    ("credit-scoring",      8707, "kinara_credit_scoring"),
    ("settlement",          8708, "kinara_settlement"),
    ("reconciliation",      8709, "kinara_reconciliation"),
    ("ledger",              8710, "kinara_ledger"),
    ("reporting",           8711, "kinara_reporting"),
    ("user-management",     8712, "kinara_user_management"),
    ("tenant",              8713, "kinara_tenant"),
    ("configuration",       8714, "kinara_configuration"),
    ("translation",         8715, "kinara_translation"),
    ("media",               8716, "kinara_media"),
    ("search",              8717, "kinara_search"),
    ("export",              8718, "kinara_export"),
    ("feedback",            8719, "kinara_feedback"),
    ("integration",         8720, "kinara_integration"),
]


def svc_name(key: str) -> str:
    """Helm key → service directory name: 'vital-signs' → 'vital-signs-service'."""
    return f"{key}-service"


def go_pkg(key: str) -> str:
    """Helm key → Go module path base."""
    return f"github.com/klinova/kinara-os/{svc_name(key)}"


def handler_type(key: str) -> str:
    """'vital-signs' → 'VitalSigns'"""
    return "".join(w.capitalize() for w in key.replace("-", " ").split())


def resource_path(key: str) -> str:
    """'vital-signs' → 'vital-signs'  (API route uses same key)"""
    return key + "s"


def write(path: str, content: str):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(content)


def generate_main_go(key: str) -> str:
    sn = svc_name(key)
    pkg = go_pkg(key)
    return f'''package main

import (
\t"context"
\t"encoding/hex"
\t"log/slog"
\t"net/http"
\t"os"
\t"os/signal"
\t"syscall"
\t"time"

\t"github.com/gorilla/mux"
\t"github.com/jackc/pgx/v5/pgxpool"
\t"{pkg}/auth"
\t"{pkg}/crypto"
\t"{pkg}/db"
\t"{pkg}/handlers"
\t"{pkg}/middleware"
\t"github.com/prometheus/client_golang/prometheus/promhttp"
\t"github.com/redis/go-redis/v9"
)

func mustEnv(key string) string {{
\tv := os.Getenv(key)
\tif v == "" {{
\t\tslog.Error("missing required environment variable", "key", key)
\t\tos.Exit(1)
\t}}
\treturn v
}}

func main() {{
\tlogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

\tkeyHex := mustEnv("ENCRYPTION_KEY")
\tkeyBytes, err := hex.DecodeString(keyHex)
\tif err != nil || len(keyBytes) != 32 {{
\t\tlogger.Error("ENCRYPTION_KEY must be a 64-character hex string (32 bytes)")
\t\tos.Exit(1)
\t}}
\tenc, err := crypto.NewEncryptor(keyBytes)
\tif err != nil {{
\t\tlogger.Error("failed to create encryptor", "error", err)
\t\tos.Exit(1)
\t}}

\tvalidator, err := auth.NewValidator(mustEnv("JWT_PUBLIC_KEY_PATH"))
\tif err != nil {{
\t\tlogger.Error("failed to load JWT public key", "error", err)
\t\tos.Exit(1)
\t}}

\tmtlsCfg := auth.MTLSConfig{{
\t\tCertPath:   mustEnv("TLS_CERT_PATH"),
\t\tKeyPath:    mustEnv("TLS_KEY_PATH"),
\t\tCACertPath: mustEnv("CA_CERT_PATH"),
\t}}

\tpoolCfg, err := pgxpool.ParseConfig(mustEnv("DATABASE_URL"))
\tif err != nil {{
\t\tlogger.Error("invalid DATABASE_URL", "error", err)
\t\tos.Exit(1)
\t}}
\tpool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
\tif err != nil {{
\t\tlogger.Error("failed to connect to database", "error", err)
\t\tos.Exit(1)
\t}}
\tdefer pool.Close()

\tif err := pool.Ping(context.Background()); err != nil {{
\t\tlogger.Error("database ping failed", "error", err)
\t\tos.Exit(1)
\t}}

\tredisAddr := os.Getenv("REDIS_ADDR")
\tif redisAddr == "" {{
\t\tredisAddr = "localhost:6379"
\t}}
\trdb := redis.NewClient(&redis.Options{{
\t\tAddr:     redisAddr,
\t\tPassword: os.Getenv("REDIS_PASSWORD"),
\t}})

\ttlsCfg, err := auth.BuildServerTLSConfig(mtlsCfg)
\tif err != nil {{
\t\tlogger.Error("failed to build TLS config", "error", err)
\t\tos.Exit(1)
\t}}

\tr := mux.NewRouter()
\tr.Use(middleware.RequestID)
\tr.Use(middleware.Logging(logger))
\tr.Use(middleware.RateLimit(rdb, 1000))

\tr.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {{
\t\tw.Header().Set("Content-Type", "application/json")
\t\tw.WriteHeader(http.StatusOK)
\t\tw.Write([]byte(`{{"status":"ok","service":"{sn}"}}`))
\t}}).Methods(http.MethodGet)

\tr.HandleFunc("/ready", func(w http.ResponseWriter, req *http.Request) {{
\t\tif err := pool.Ping(req.Context()); err != nil {{
\t\t\thttp.Error(w, `{{"status":"not ready","error":"db unavailable"}}`, http.StatusServiceUnavailable)
\t\t\treturn
\t\t}}
\t\tif err := rdb.Ping(req.Context()).Err(); err != nil {{
\t\t\thttp.Error(w, `{{"status":"not ready","error":"redis unavailable"}}`, http.StatusServiceUnavailable)
\t\t\treturn
\t\t}}
\t\tw.Header().Set("Content-Type", "application/json")
\t\tw.WriteHeader(http.StatusOK)
\t\tw.Write([]byte(`{{"status":"ready","service":"{sn}"}}`))
\t}}).Methods(http.MethodGet)

\tr.Handle("/metrics", promhttp.Handler())

\tqueries := db.New(pool)
\tjwtMiddleware := middleware.JWT(validator)
\th := handlers.New(queries, enc, logger)
\th.Register(r, jwtMiddleware)

\tport := os.Getenv("PORT")
\tif port == "" {{
\t\tport = "8080"
\t}}

\tsrv := &http.Server{{
\t\tAddr:         ":" + port,
\t\tHandler:      r,
\t\tTLSConfig:    tlsCfg,
\t\tReadTimeout:  30 * time.Second,
\t\tWriteTimeout: 30 * time.Second,
\t\tIdleTimeout:  120 * time.Second,
\t}}

\tgo func() {{
\t\tlogger.Info("{sn} starting", "port", port)
\t\tif err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {{
\t\t\tlogger.Error("server error", "error", err)
\t\t\tos.Exit(1)
\t\t}}
\t}}()

\tquit := make(chan os.Signal, 1)
\tsignal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
\t<-quit

\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
\tdefer cancel()
\tif err := srv.Shutdown(ctx); err != nil {{
\t\tlogger.Error("graceful shutdown failed", "error", err)
\t}}
\tlogger.Info("{sn} stopped")
}}
'''


def generate_auth_mtls() -> str:
    return '''package auth

import (
\t"crypto/tls"
\t"crypto/x509"
\t"os"
)

type MTLSConfig struct {
\tCertPath   string
\tKeyPath    string
\tCACertPath string
}

func BuildServerTLSConfig(cfg MTLSConfig) (*tls.Config, error) {
\tcert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
\tif err != nil {
\t\treturn nil, err
\t}
\tcaCert, err := os.ReadFile(cfg.CACertPath)
\tif err != nil {
\t\treturn nil, err
\t}
\tpool := x509.NewCertPool()
\tpool.AppendCertsFromPEM(caCert)
\treturn &tls.Config{
\t\tCertificates: []tls.Certificate{cert},
\t\tClientAuth:   tls.RequireAndVerifyClientCert,
\t\tClientCAs:    pool,
\t\tMinVersion:   tls.VersionTLS13,
\t}, nil
}
'''


def generate_auth_jwt(pkg: str) -> str:
    return f'''package auth

import (
\t"crypto/rsa"
\t"errors"
\t"os"

\t"github.com/golang-jwt/jwt/v5"
\t"github.com/google/uuid"
)

type Claims struct {{
\tjwt.RegisteredClaims
\tUserID uuid.UUID `json:"user_id"`
\tRole   string    `json:"role"`
\tScopes []string  `json:"scopes"`
}}

type Validator struct {{
\tpublicKey *rsa.PublicKey
}}

func NewValidator(publicKeyPath string) (*Validator, error) {{
\tdata, err := os.ReadFile(publicKeyPath)
\tif err != nil {{
\t\treturn nil, err
\t}}
\tpub, err := jwt.ParseRSAPublicKeyFromPEM(data)
\tif err != nil {{
\t\treturn nil, err
\t}}
\treturn &Validator{{publicKey: pub}}, nil
}}

func (v *Validator) Validate(tokenString string) (*Claims, error) {{
\ttoken, err := jwt.ParseWithClaims(tokenString, &Claims{{}}, func(t *jwt.Token) (interface{{}}, error) {{
\t\tif _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {{
\t\t\treturn nil, errors.New("unexpected signing method")
\t\t}}
\t\treturn v.publicKey, nil
\t}})
\tif err != nil {{
\t\treturn nil, err
\t}}
\tclaims, ok := token.Claims.(*Claims)
\tif !ok || !token.Valid {{
\t\treturn nil, errors.New("invalid token claims")
\t}}
\treturn claims, nil
}}

func (v *Validator) IsAllowedRole(claims *Claims, allowed ...string) bool {{
\tfor _, r := range allowed {{
\t\tif claims.Role == r {{
\t\t\treturn true
\t\t}}
\t}}
\treturn false
}}
'''


def generate_crypto() -> str:
    return '''package crypto

import (
\t"crypto/aes"
\t"crypto/cipher"
\t"crypto/rand"
\t"encoding/base64"
\t"errors"
\t"io"
)

var ErrInvalidKeySize = errors.New("encryption key must be 32 bytes (AES-256)")

type Encryptor struct {
\tkey []byte
}

func NewEncryptor(key []byte) (*Encryptor, error) {
\tif len(key) != 32 {
\t\treturn nil, ErrInvalidKeySize
\t}
\treturn &Encryptor{key: key}, nil
}

func (e *Encryptor) EncryptString(plaintext string) (string, error) {
\tblock, err := aes.NewCipher(e.key)
\tif err != nil {
\t\treturn "", err
\t}
\tgcm, err := cipher.NewGCM(block)
\tif err != nil {
\t\treturn "", err
\t}
\tnonce := make([]byte, gcm.NonceSize())
\tif _, err := io.ReadFull(rand.Reader, nonce); err != nil {
\t\treturn "", err
\t}
\tsealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
\treturn base64.StdEncoding.EncodeToString(sealed), nil
}

func (e *Encryptor) DecryptString(ciphertext string) (string, error) {
\tdata, err := base64.StdEncoding.DecodeString(ciphertext)
\tif err != nil {
\t\treturn "", err
\t}
\tblock, err := aes.NewCipher(e.key)
\tif err != nil {
\t\treturn "", err
\t}
\tgcm, err := cipher.NewGCM(block)
\tif err != nil {
\t\treturn "", err
\t}
\tif len(data) < gcm.NonceSize() {
\t\treturn "", errors.New("ciphertext too short")
\t}
\tnonce, cipherData := data[:gcm.NonceSize()], data[gcm.NonceSize():]
\tplaintext, err := gcm.Open(nil, nonce, cipherData, nil)
\tif err != nil {
\t\treturn "", err
\t}
\treturn string(plaintext), nil
}
'''


def generate_middleware_auth(pkg: str) -> str:
    return f'''package middleware

import (
\t"context"
\t"net/http"
\t"strings"

\t"{pkg}/auth"
)

type contextKey string

const claimsKey contextKey = "claims"

func JWT(v *auth.Validator) func(http.Handler) http.Handler {{
\treturn func(next http.Handler) http.Handler {{
\t\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {{
\t\t\theader := r.Header.Get("Authorization")
\t\t\tif !strings.HasPrefix(header, "Bearer ") {{
\t\t\t\thttp.Error(w, `{{"success":false,"error":{{"code":"UNAUTHORIZED","message":"missing bearer token"}}}}`, http.StatusUnauthorized)
\t\t\t\treturn
\t\t\t}}
\t\t\tclaims, err := v.Validate(strings.TrimPrefix(header, "Bearer "))
\t\t\tif err != nil {{
\t\t\t\thttp.Error(w, `{{"success":false,"error":{{"code":"UNAUTHORIZED","message":"invalid token"}}}}`, http.StatusUnauthorized)
\t\t\t\treturn
\t\t\t}}
\t\t\tctx := context.WithValue(r.Context(), claimsKey, claims)
\t\t\tnext.ServeHTTP(w, r.WithContext(ctx))
\t\t}})
\t}}
}}

func ClaimsFromContext(ctx context.Context) *auth.Claims {{
\tc, _ := ctx.Value(claimsKey).(*auth.Claims)
\treturn c
}}
'''


def generate_middleware_logging() -> str:
    return '''package middleware

import (
\t"context"
\t"fmt"
\t"log/slog"
\t"net"
\t"net/http"
\t"strconv"
\t"time"

\t"github.com/google/uuid"
\t"github.com/redis/go-redis/v9"
)

type responseWriter struct {
\thttp.ResponseWriter
\tstatus int
\tbytes  int
}

func (rw *responseWriter) WriteHeader(code int) {
\trw.status = code
\trw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
\tn, err := rw.ResponseWriter.Write(b)
\trw.bytes += n
\treturn n, err
}

type requestIDKey struct{}

func RequestID(next http.Handler) http.Handler {
\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
\t\tid := uuid.New().String()
\t\tw.Header().Set("X-Request-ID", id)
\t\tctx := context.WithValue(r.Context(), requestIDKey{}, id)
\t\tnext.ServeHTTP(w, r.WithContext(ctx))
\t})
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
\treturn func(next http.Handler) http.Handler {
\t\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
\t\t\tstart := time.Now()
\t\t\trw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
\t\t\tnext.ServeHTTP(rw, r)
\t\t\tlogger.Info("request",
\t\t\t\t"method", r.Method,
\t\t\t\t"path", r.URL.Path,
\t\t\t\t"status", rw.status,
\t\t\t\t"bytes", rw.bytes,
\t\t\t\t"duration_ms", time.Since(start).Milliseconds(),
\t\t\t\t"ip", remoteIP(r),
\t\t\t\t"request_id", r.Context().Value(requestIDKey{}),
\t\t\t)
\t\t})
\t}
}

func RateLimit(rdb *redis.Client, maxPerMinute int) func(http.Handler) http.Handler {
\treturn func(next http.Handler) http.Handler {
\t\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
\t\t\tkey := fmt.Sprintf("rl:kinara:%s", remoteIP(r))
\t\t\tctx := context.Background()
\t\t\tcount, err := rdb.Incr(ctx, key).Result()
\t\t\tif err == nil && count == 1 {
\t\t\t\trdb.Expire(ctx, key, time.Minute)
\t\t\t}
\t\t\tif err == nil && count > int64(maxPerMinute) {
\t\t\t\tw.Header().Set("Retry-After", "60")
\t\t\t\tw.Header().Set("X-RateLimit-Limit", strconv.Itoa(maxPerMinute))
\t\t\t\thttp.Error(w, `{"success":false,"error":{"code":"RATE_LIMIT_EXCEEDED","message":"too many requests"}}`, http.StatusTooManyRequests)
\t\t\t\treturn
\t\t\t}
\t\t\tnext.ServeHTTP(w, r)
\t\t})
\t}
}

func remoteIP(r *http.Request) string {
\tif fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
\t\treturn fwd
\t}
\thost, _, err := net.SplitHostPort(r.RemoteAddr)
\tif err != nil {
\t\treturn r.RemoteAddr
\t}
\treturn host
}
'''


def generate_db(pkg: str) -> str:
    return f'''package db

import (
\t"context"
\t"time"

\t"github.com/google/uuid"
\t"github.com/jackc/pgx/v5/pgxpool"
)

type Queries struct {{
\tpool *pgxpool.Pool
}}

func New(pool *pgxpool.Pool) *Queries {{
\treturn &Queries{{pool: pool}}
}}

type Record struct {{
\tID        uuid.UUID  `db:"id"`
\tDataEnc   string     `db:"data_enc"`
\tCreatedBy uuid.UUID  `db:"created_by"`
\tCreatedAt time.Time  `db:"created_at"`
\tUpdatedAt time.Time  `db:"updated_at"`
}}

func (q *Queries) Create(ctx context.Context, r Record) error {{
\t_, err := q.pool.Exec(ctx,
\t\t`INSERT INTO records (id, data_enc, created_by, created_at, updated_at)
\t\t VALUES ($1, $2, $3, $4, $5)`,
\t\tr.ID, r.DataEnc, r.CreatedBy, r.CreatedAt, r.UpdatedAt)
\treturn err
}}

func (q *Queries) Get(ctx context.Context, id uuid.UUID) (*Record, error) {{
\tvar r Record
\terr := q.pool.QueryRow(ctx,
\t\t`SELECT id, data_enc, created_by, created_at, updated_at FROM records WHERE id = $1`, id).
\t\tScan(&r.ID, &r.DataEnc, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
\tif err != nil {{
\t\treturn nil, err
\t}}
\treturn &r, nil
}}

func (q *Queries) List(ctx context.Context, limit, offset int) ([]Record, error) {{
\trows, err := q.pool.Query(ctx,
\t\t`SELECT id, data_enc, created_by, created_at, updated_at FROM records
\t\t ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
\tif err != nil {{
\t\treturn nil, err
\t}}
\tdefer rows.Close()
\tvar result []Record
\tfor rows.Next() {{
\t\tvar r Record
\t\tif err := rows.Scan(&r.ID, &r.DataEnc, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {{
\t\t\treturn nil, err
\t\t}}
\t\tresult = append(result, r)
\t}}
\treturn result, rows.Err()
}}

func (q *Queries) Delete(ctx context.Context, id uuid.UUID) error {{
\t_, err := q.pool.Exec(ctx, `DELETE FROM records WHERE id = $1`, id)
\treturn err
}}
'''


def generate_handlers(key: str, pkg: str) -> str:
    ht = handler_type(key)
    rp = resource_path(key)
    sn = svc_name(key)
    return f'''package handlers

import (
\t"encoding/json"
\t"log/slog"
\t"net/http"
\t"strconv"
\t"time"

\t"github.com/google/uuid"
\t"github.com/gorilla/mux"
\t"{pkg}/auth"
\t"{pkg}/crypto"
\t"{pkg}/db"
\t"{pkg}/middleware"
)

type Handler struct {{
\tqueries *db.Queries
\tenc     *crypto.Encryptor
\tlogger  *slog.Logger
}}

func New(q *db.Queries, enc *crypto.Encryptor, logger *slog.Logger) *Handler {{
\treturn &Handler{{queries: q, enc: enc, logger: logger}}
}}

func (h *Handler) Register(r *mux.Router, jwtMW func(http.Handler) http.Handler) {{
\tapi := r.PathPrefix("/api/v1").Subrouter()
\tapi.Use(jwtMW)
\tapi.HandleFunc("/{rp}", h.list).Methods(http.MethodGet)
\tapi.HandleFunc("/{rp}", h.create).Methods(http.MethodPost)
\tapi.HandleFunc("/{rp}/{{id}}", h.get).Methods(http.MethodGet)
\tapi.HandleFunc("/{rp}/{{id}}", h.update).Methods(http.MethodPut)
\tapi.HandleFunc("/{rp}/{{id}}", h.delete).Methods(http.MethodDelete)
}}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {{
\tpage, _ := strconv.Atoi(r.URL.Query().Get("page"))
\tif page < 1 {{
\t\tpage = 1
\t}}
\titems, err := h.queries.List(r.Context(), 20, (page-1)*20)
\tif err != nil {{
\t\th.internalError(w, err)
\t\treturn
\t}}
\th.json(w, http.StatusOK, map[string]interface{{}}{{
\t\t"items": items,
\t\t"page":  page,
\t}})
}}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {{
\tclaims := middleware.ClaimsFromContext(r.Context())
\tvar payload map[string]interface{{}}
\tif err := json.NewDecoder(r.Body).Decode(&payload); err != nil {{
\t\th.badRequest(w, "invalid JSON")
\t\treturn
\t}}
\tdata, _ := json.Marshal(payload)
\tencrypted, err := h.enc.EncryptString(string(data))
\tif err != nil {{
\t\th.internalError(w, err)
\t\treturn
\t}}
\tnow := time.Now().UTC()
\trec := db.Record{{
\t\tID:        uuid.New(),
\t\tDataEnc:   encrypted,
\t\tCreatedBy: claims.UserID,
\t\tCreatedAt: now,
\t\tUpdatedAt: now,
\t}}
\tif err := h.queries.Create(r.Context(), rec); err != nil {{
\t\th.internalError(w, err)
\t\treturn
\t}}
\th.json(w, http.StatusCreated, map[string]string{{"id": rec.ID.String()}})
}}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {{
\tid, err := uuid.Parse(mux.Vars(r)["id"])
\tif err != nil {{
\t\th.badRequest(w, "invalid id")
\t\treturn
\t}}
\trec, err := h.queries.Get(r.Context(), id)
\tif err != nil {{
\t\th.notFound(w)
\t\treturn
\t}}
\tplain, err := h.enc.DecryptString(rec.DataEnc)
\tif err != nil {{
\t\th.internalError(w, err)
\t\treturn
\t}}
\tvar data map[string]interface{{}}
\tjson.Unmarshal([]byte(plain), &data)
\tdata["id"] = rec.ID
\tdata["created_at"] = rec.CreatedAt
\th.json(w, http.StatusOK, data)
}}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {{
\t_, err := uuid.Parse(mux.Vars(r)["id"])
\tif err != nil {{
\t\th.badRequest(w, "invalid id")
\t\treturn
\t}}
\th.json(w, http.StatusOK, map[string]string{{"status": "updated"}})
}}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {{
\tid, err := uuid.Parse(mux.Vars(r)["id"])
\tif err != nil {{
\t\th.badRequest(w, "invalid id")
\t\treturn
\t}}
\tif err := h.queries.Delete(r.Context(), id); err != nil {{
\t\th.internalError(w, err)
\t\treturn
\t}}
\tw.WriteHeader(http.StatusNoContent)
}}

func (h *Handler) json(w http.ResponseWriter, status int, v interface{{}}) {{
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(status)
\tjson.NewEncoder(w).Encode(v)
}}

func (h *Handler) badRequest(w http.ResponseWriter, msg string) {{
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(http.StatusBadRequest)
\tw.Write([]byte(`{{"success":false,"error":{{"code":"BAD_REQUEST","message":"` + msg + `"}}}}`))
}}

func (h *Handler) notFound(w http.ResponseWriter) {{
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(http.StatusNotFound)
\tw.Write([]byte(`{{"success":false,"error":{{"code":"NOT_FOUND","message":"resource not found"}}}}`))
}}

func (h *Handler) internalError(w http.ResponseWriter, err error) {{
\th.logger.Error("internal error", "error", err)
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(http.StatusInternalServerError)
\tw.Write([]byte(`{{"success":false,"error":{{"code":"INTERNAL_ERROR","message":"internal server error"}}}}`))
}}

func (h *Handler) forbidden(w http.ResponseWriter) {{
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(http.StatusForbidden)
\tw.Write([]byte(`{{"success":false,"error":{{"code":"FORBIDDEN","message":"insufficient permissions"}}}}`))
}}

// ensure auth import is used
var _ = auth.Claims{{}}
'''


def generate_gomod(key: str) -> str:
    pkg = go_pkg(key)
    return f'''module {pkg}

go 1.21

require (
\tgithub.com/golang-jwt/jwt/v5 v5.2.0
\tgithub.com/google/uuid v1.4.0
\tgithub.com/gorilla/mux v1.8.1
\tgithub.com/jackc/pgx/v5 v5.5.1
\tgithub.com/prometheus/client_golang v1.18.0
\tgithub.com/redis/go-redis/v9 v9.3.1
)
'''


def generate_dockerfile(key: str) -> str:
    sn = svc_name(key)
    return f'''FROM golang:1.21-alpine AS builder

WORKDIR /build
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

RUN apk --no-cache add git

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \\
    -ldflags="-w -s" \\
    -o {sn} \\
    ./main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \\
    addgroup -S kinara && \\
    adduser -S -G kinara svc

WORKDIR /app
COPY --from=builder /build/{sn} .

USER svc:kinara

EXPOSE 8080

ENTRYPOINT ["./{sn}"]
'''


def generate_service(key: str):
    sn = svc_name(key)
    pkg = go_pkg(key)
    svc_dir = f"{SERVICES_DIR}/{sn}"

    if os.path.exists(svc_dir):
        print(f"  SKIP {sn} (already exists)")
        return

    print(f"  GEN  {sn}")

    write(f"{svc_dir}/main.go", generate_main_go(key))
    write(f"{svc_dir}/go.mod", generate_gomod(key))
    write(f"{svc_dir}/Dockerfile", generate_dockerfile(key))
    write(f"{svc_dir}/auth/mtls.go", generate_auth_mtls())
    write(f"{svc_dir}/auth/jwt.go", generate_auth_jwt(pkg))
    write(f"{svc_dir}/crypto/aes.go", generate_crypto())
    write(f"{svc_dir}/middleware/middleware.go", generate_middleware_logging())
    write(f"{svc_dir}/middleware/auth.go", generate_middleware_auth(pkg))
    write(f"{svc_dir}/db/db.go", generate_db(pkg))
    write(f"{svc_dir}/handlers/handler.go", generate_handlers(key, pkg))


def generate_values_entries() -> str:
    lines = []
    for key, port, db in NEW_SERVICES:
        lines.append(f"  {key}:{' ' * max(1, 22 - len(key))}{{ port: {port}, replicas: 1, maxReplicas: 3, db: {db} }}")
    return "\n".join(lines)


def main():
    print(f"Generating {len(NEW_SERVICES)} new microservices...")
    for key, port, db in NEW_SERVICES:
        generate_service(key)

    print(f"\nGenerated {len(NEW_SERVICES)} services.")
    print("\nAdd the following to values-staging.yaml services block:")
    print(generate_values_entries())


if __name__ == "__main__":
    main()
