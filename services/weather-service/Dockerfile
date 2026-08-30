FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o weather-service .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/weather-service /weather-service
EXPOSE 8088
ENTRYPOINT ["/weather-service"]
