FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -u 1000 appuser
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/payment-service .
USER appuser
EXPOSE 8107
CMD ["./payment-service"]
