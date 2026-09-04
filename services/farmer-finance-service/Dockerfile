FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/farmer-finance-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/farmer-finance-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o farmer-finance-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/farmer-finance-service .
EXPOSE 8112
USER nobody:nobody
ENTRYPOINT ["./farmer-finance-service"]
