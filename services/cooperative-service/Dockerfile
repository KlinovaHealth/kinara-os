FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

RUN apk add --no-cache git ca-certificates
COPY services/cooperative-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/cooperative-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cooperative-service .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/cooperative-service /cooperative-service
EXPOSE 8087
ENTRYPOINT ["/cooperative-service"]
