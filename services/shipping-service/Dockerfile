FROM golang:1.21-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o shipping-service .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/shipping-service /shipping-service
EXPOSE 8103
ENTRYPOINT ["/shipping-service"]
