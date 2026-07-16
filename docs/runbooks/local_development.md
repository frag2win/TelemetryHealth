# Local Development Runbook

This guide covers bootstrapping the TelemetryHealth local environment, running test harnesses, and verifying transport layers.

## 1. Bootstrapping Dependencies via Foundry

We utilize Foundry deployment targets for containerized bootstrapping.

1. Navigate to the infrastructure root where `casting.yaml` and `casting.yaml.lock` reside.
2. Spin up the dependencies (ClickHouse, SigNoz, OpenTelemetry Collector):
   ```bash
   foundry cast deploy local
   ```
3. Ensure all containers transition to a healthy state:
   ```bash
   foundry cast status
   ```

## 2. Executing Local Test Harnesses

To generate synthetic AI agent telemetry, run the trace generator:

1. Navigate to the generator directory:
   ```bash
   cd testing/harness
   ```
2. Execute the trace payload generator:
   ```bash
   go run main.go --mode=synthetic-ai --throughput=100
   ```
3. This will emit simulated traces containing `llm.request.tokens` and `behavior_node` events into the local OTLP ingestion port.

## 3. Verifying mTLS Variables

Secure gRPC transport requires explicit certificate validation.

1. Confirm your local shell has the appropriate TLS paths exported:
   ```bash
   export OTLP_CERT_FILE=/path/to/local/certs/ca.crt
   export OTLP_KEY_FILE=/path/to/local/certs/client.key
   ```
2. Validate the mTLS handshake locally using `grpcurl`:
   ```bash
   grpcurl -cacert $OTLP_CERT_FILE \
     -key $OTLP_KEY_FILE \
     -cert $OTLP_CERT_FILE \
     localhost:4317 list
   ```
3. If successful, you will see the standard OpenTelemetry export services listed.
