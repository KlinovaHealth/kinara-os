FROM golang:1.25-alpine AS builder

WORKDIR /build
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*


COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o clinical-service \
    ./main.go

# ─── Runtime stage ────────────────────────────────────────────────────────────

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S kinara && \
    adduser -S -G kinara clinical

WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*


COPY --from=builder /build/clinical-service .

USER clinical:kinara

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- --no-check-certificate https://localhost:8080/health || exit 1

ENTRYPOINT ["./clinical-service"]
