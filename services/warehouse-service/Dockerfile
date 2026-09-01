FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

RUN apk add --no-cache git ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o warehouse-service .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/warehouse-service /warehouse-service
EXPOSE 8086
ENTRYPOINT ["/warehouse-service"]
