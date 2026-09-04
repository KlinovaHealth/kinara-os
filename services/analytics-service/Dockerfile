FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/analytics-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/analytics-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o analytics-service ./main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -u 1000 appuser
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/analytics-service .
USER appuser
EXPOSE 8108
CMD ["./analytics-service"]
