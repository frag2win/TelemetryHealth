# Runbook — Ingest Gateway Consumer Lag

## Symptom
Kafka consumer lag increases on the ingest gateway topic (`telemetryhealth.cardinality`, `telemetryhealth.orphan`, `telemetryhealth.coverage`). This leads to a latency breach in composite health score calculation (SLO: staleness ≤ 5s, G1 p95 detection latency ≤ 60s).

## Troubleshooting Steps

1. **Verify Kafka Broker Status**
   Check if the Kafka cluster is responding to metadata requests.

2. **Verify Ingest Gateway Logs**
   Search gateway logs for rate-limiting, authentication rejections (e.g. invalid mTLS tenant certs), or connection timeouts to ClickHouse.

3. **Check Consumer Group State**
   Run the partition offset query:
   ```bash
   kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group cardinality-worker
   ```

## Mitigation

1. **Scale Consumers**
   If lag is due to high CPU load or trace volume spike, scale out the ingest gateway and stream worker deployments:
   ```bash
   kubectl scale deployment telemetryhealth-controlplane-worker --replicas=10
   ```

2. **Enable Fallback Options**
   If the downstream storage (ClickHouse) is backing up, enable backpressure drop rules or increase batch size/concurrency configurations.
