FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/shipping-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/shipping-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o shipping-service .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/shipping-service /shipping-service
EXPOSE 8103
ENTRYPOINT ["/shipping-service"]
