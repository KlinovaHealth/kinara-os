FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/audit-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/audit-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o audit-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/audit-service .
EXPOSE 8115
USER nobody:nobody
ENTRYPOINT ["./audit-service"]
