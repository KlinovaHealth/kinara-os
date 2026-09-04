FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/sms-gateway/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/sms-gateway/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o sms-gateway ./main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -u 1000 appuser
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/sms-gateway .
USER appuser
EXPOSE 8200
CMD ["./sms-gateway"]
