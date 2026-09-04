FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/port-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/port-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o port-service .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/port-service /port-service
EXPOSE 8098
ENTRYPOINT ["/port-service"]
