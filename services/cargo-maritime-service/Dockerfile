FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/cargo-maritime-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/cargo-maritime-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o cargo-maritime-service .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/cargo-maritime-service /cargo-maritime-service
EXPOSE 8101
ENTRYPOINT ["/cargo-maritime-service"]
