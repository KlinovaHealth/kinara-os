FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/supply-chain-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/supply-chain-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o supply-chain-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/supply-chain-service .
EXPOSE 8113
USER nobody:nobody
ENTRYPOINT ["./supply-chain-service"]
