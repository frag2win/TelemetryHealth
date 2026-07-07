FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY api-server-linux /app/api-server
COPY ingest-gateway-linux /app/ingest-gateway
COPY worker-linux /app/worker
COPY docs /app/docs

EXPOSE 8080 4317 9090 9091
