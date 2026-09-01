FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY go.mod ./
RUN go mod download
COPY . .
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
