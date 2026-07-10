# Runbook — ClickHouse Disk Pressure

## Symptom
The ClickHouse cluster disk utilization exceeds 85%. ClickHouse blocks writes and query processing degrades.

## Troubleshooting Steps

1. **Verify Partition Sizes**
   Run the following query to check partition size distribution:
   ```sql
   SELECT partition, name, active, bytes_on_disk 
   FROM system.parts 
   WHERE table = 'cardinality_signal' AND active = 1
   ```

2. **Check TTL Expiry Behavior**
   Ensure ClickHouse is actively dropping expired parts (TTLs: 30 days for signals, 12 months for health scores, 90 days for remediation events).

## Mitigation

1. **Trigger Manual TTL Cleanup**
   Force ClickHouse to run TTL merges and free disk space:
   ```sql
   ALTER TABLE telemetry_health.cardinality_signal MATERIALIZE TTL
   ```

2. **Resize Disk Volume**
   Increase PVC sizes or assign more storage space to the ClickHouse stateful set.
