# ─── Stage 1: builder ─────────────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Download dependencies first (layer cache optimisation)
COPY go.mod ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /patient-service ./main.go

# ─── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM alpine:3.19

# Security: run as non-root
RUN addgroup -S kinara && adduser -S patient -G kinara

WORKDIR /app

# Copy compiled binary and CA certs
COPY --from=builder /patient-service /app/patient-service
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Runtime secrets/certs are mounted at /app/certs via Kubernetes Secrets
RUN mkdir -p /app/certs && chown -R patient:kinara /app

USER patient

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/patient-service"]
