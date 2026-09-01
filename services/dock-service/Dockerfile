FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOFLAGS=-mod=mod
ENV GONOSUMDB=*

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o dock-service .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/dock-service /dock-service
EXPOSE 8100
ENTRYPOINT ["/dock-service"]
