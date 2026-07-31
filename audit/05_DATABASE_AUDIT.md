# 05 — Database Audit

## Schema Design

### DB-01: `cardinality_signal` Schema Mismatch
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/storage/clickhouse/schema.go` L30-40
- **Explanation**: The table has an `hll_sketch AggregateFunction(uniqCombined, String)` column, but the insert queries in `workers.go` and `seeder.go` never populate it — they only insert `unique_estimate`. The AggregatingMergeTree engine expects aggregate state columns to be populated via `-State` combinators.
- **How to fix**: Either remove the `hll_sketch` column (using MergeTree instead) or change the insert to use `uniqCombinedState(value)`.

### DB-02: `tenant_id` Type Inconsistency
- **Severity**: 🟠 High
- **Explanation**: `cardinality_signal`, `orphan_signal`, `coverage_signal`, `health_score`, `remediation_event`, `alert_event`, `tenant_config` use `UUID` type. But `telemetryhealth_trace_index_spans` uses `LowCardinality(String)`. The seeder uses `uuid.UUID` (Go type), the API validates UUIDs OR slugs like `tenant-alpha`.
- **How to fix**: Standardize on `String` throughout or enforce UUID-only in all layers.

### DB-03: No Migration Versioning
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/storage/clickhouse/schema.go`
- **Explanation**: Schema uses `CREATE TABLE IF NOT EXISTS` exclusively. There's no way to evolve the schema — adding a column, changing a type, or adding an index requires manual intervention. No migration tool (golang-migrate, goose, atlas) is integrated.
- **How to fix**: Adopt a migration tool with numbered migration files and a `schema_migrations` tracking table.

### DB-04: No Indexes Beyond Primary Key
- **Severity**: 🟡 Medium
- **Explanation**: No secondary indexes (skip indexes, bloom filters) are defined on any table. The `telemetryhealth_trace_index_spans.attributes` column, queried with `LIKE`, has no tokenbf index.

### DB-05: `coverage_signal` Has Extra Columns Not Populated
- **Severity**: 🟢 Low
- **Explanation**: `environment` and `grace_period_seconds` columns are defined in the schema but the CoverageEvent struct and Kafka publisher don't include these fields. The worker inserts `uint8(1)` for `baseline_expected` but skips `environment` and `grace_period_seconds`.

### DB-06: No Backup/Restore Strategy Documented
- **Severity**: 🟡 Medium
- **Explanation**: For production ClickHouse deployments, there's no documentation or tooling for backup/restore, point-in-time recovery, or disaster recovery procedures.
