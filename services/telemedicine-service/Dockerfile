FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY services/telemedicine-service/go.mod ./
COPY pkg/auth /pkg/auth
RUN go mod download
COPY services/telemedicine-service/. .
RUN CGO_ENABLED=0 GOOS=linux go build -o telemedicine-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY --from=builder /app/telemedicine-service .
EXPOSE 8109
USER nobody:nobody
ENTRYPOINT ["./telemedicine-service"]
