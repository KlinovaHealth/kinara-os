FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/wallet-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/wallet-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o wallet-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/wallet-service .
EXPOSE 8114
USER nobody:nobody
ENTRYPOINT ["./wallet-service"]
