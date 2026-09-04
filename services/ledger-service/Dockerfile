FROM golang:1.25-alpine AS builder

WORKDIR /build
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

RUN apk --no-cache add git

COPY services/ledger-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download

COPY services/ledger-service/. .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o ledger-service \
    ./main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S kinara && \
    adduser -S -G kinara svc

WORKDIR /app
COPY --from=builder /build/ledger-service .

USER svc:kinara

EXPOSE 8080

ENTRYPOINT ["./ledger-service"]
