FROM golang:1.21-alpine AS builder

WORKDIR /build

RUN apk --no-cache add git

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o notification-service \
    ./main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S kinara && \
    adduser -S -G kinara notifsvc

WORKDIR /app

COPY --from=builder /build/notification-service .

USER notifsvc:kinara

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- --no-check-certificate https://localhost:8080/health || exit 1

ENTRYPOINT ["./notification-service"]
