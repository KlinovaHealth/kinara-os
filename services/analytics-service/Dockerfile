FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o analytics-service ./main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -u 1000 appuser
WORKDIR /app
COPY --from=builder /app/analytics-service .
USER appuser
EXPOSE 8108
CMD ["./analytics-service"]
