FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o port-service .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/port-service /port-service
EXPOSE 8098
ENTRYPOINT ["/port-service"]
