FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o cargo-maritime-service .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/cargo-maritime-service /cargo-maritime-service
EXPOSE 8101
ENTRYPOINT ["/cargo-maritime-service"]
