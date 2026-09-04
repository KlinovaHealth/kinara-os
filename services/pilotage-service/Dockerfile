FROM golang:1.25-alpine AS builder

WORKDIR /build
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

RUN apk --no-cache add git

COPY services/pilotage-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download

COPY services/pilotage-service/. .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o pilotage-service \
    ./main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S kinara && \
    adduser -S -G kinara svc

WORKDIR /app
COPY --from=builder /build/pilotage-service .

USER svc:kinara

EXPOSE 8080

ENTRYPOINT ["./pilotage-service"]
