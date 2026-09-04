FROM golang:1.25-alpine AS builder

WORKDIR /build
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*


RUN apk --no-cache add git

COPY services/auth-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download

COPY services/auth-service/. .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o auth-service \
    ./main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S kinara && \
    adduser -S -G kinara authsvc

WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*


COPY --from=builder /build/auth-service .

USER authsvc:kinara

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- --no-check-certificate https://localhost:8080/health || exit 1

ENTRYPOINT ["./auth-service"]
