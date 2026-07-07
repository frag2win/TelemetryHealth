FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all three binaries
RUN go build -o /api-server ./cmd/api-server
RUN go build -o /ingest-gateway ./cmd/ingest-gateway
RUN go build -o /worker ./cmd/worker

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /api-server /app/api-server
COPY --from=builder /ingest-gateway /app/ingest-gateway
COPY --from=builder /worker /app/worker
COPY docs /app/docs

EXPOSE 8080 4317 9090 9091
