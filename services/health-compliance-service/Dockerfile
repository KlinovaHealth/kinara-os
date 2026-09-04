FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/health-compliance-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/health-compliance-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o health-compliance-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/health-compliance-service .
EXPOSE 8111
USER nobody:nobody
ENTRYPOINT ["./health-compliance-service"]
