FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o last-mile-service .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/last-mile-service /last-mile-service
EXPOSE 8086
ENTRYPOINT ["/last-mile-service"]
