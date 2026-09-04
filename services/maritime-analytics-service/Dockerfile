FROM golang:1.25-alpine AS builder

WORKDIR /build
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

RUN apk --no-cache add git

COPY services/maritime-analytics-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download

COPY services/maritime-analytics-service/. .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o maritime-analytics-service \
    ./main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S kinara && \
    adduser -S -G kinara svc

WORKDIR /app
COPY --from=builder /build/maritime-analytics-service .

USER svc:kinara

EXPOSE 8080

ENTRYPOINT ["./maritime-analytics-service"]
