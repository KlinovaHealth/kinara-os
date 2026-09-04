FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

RUN apk add --no-cache git ca-certificates
COPY services/compliance-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/compliance-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o compliance-service .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/compliance-service /compliance-service
EXPOSE 8086
ENTRYPOINT ["/compliance-service"]
