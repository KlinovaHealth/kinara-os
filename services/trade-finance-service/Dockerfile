FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o trade-finance-service .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/trade-finance-service /trade-finance-service
EXPOSE 8105
ENTRYPOINT ["/trade-finance-service"]
