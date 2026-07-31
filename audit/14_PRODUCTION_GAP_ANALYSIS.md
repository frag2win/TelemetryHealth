# 14 — Production Gap Analysis

| Requirement | Status | Gap Description |
|-------------|--------|-----------------|
| **Authentication** | ❌ Not functional | OIDC middleware is a no-op when issuer is set |
| **Authorization** | ❌ Not functional | No RBAC enforcement — role is extracted but never checked |
| **Rate Limiting** | ⚠️ Partially working | Port-stripping bug defeats per-IP limiting |
| **Health Checks** | ⚠️ Basic | `/healthz` returns 200; `/readyz` doesn't actually ping the DB |
| **Graceful Shutdown** | ✅ Implemented | All cmd entrypoints handle SIGTERM correctly |
| **Structured Logging** | ✅ Good | zap.Logger used consistently throughout |
| **Metrics** | ⚠️ Partial | Prometheus metrics defined but cardinality will explode with path labels |
| **Distributed Tracing** | ✅ Good | OTel SDK initialized, context propagated |
| **Connection Pooling** | ✅ Good | ClickHouse MaxOpenConns=25, MaxIdleConns=10 |
| **TLS (HTTP)** | ❌ Missing | HTTP servers have no TLS configuration |
| **TLS (gRPC)** | ⚠️ Partial | mTLS code exists in authz package but not wired into gRPC server options |
| **Secrets Management** | ❌ Missing | Passwords are empty strings, no secrets manager integration |
| **Configuration** | ❌ Fragile | 30+ scattered os.Getenv calls, no validation or defaults documentation |
| **Backpressure** | ⚠️ Partial | Kafka batching exists but no backpressure on ingest gateway |
| **Deployment** | ❌ Incomplete | Helm chart referenced but Dockerfiles are broken |
| **Self-Monitoring** | ⚠️ Partial | Prometheus metrics + SigNoz dashboards defined but may be stale |
| **Error Budget / SLOs** | ❌ Missing | No SLO definitions, no error budget tracking |
| **Alerting** | ⚠️ Partial | SigNoz bridge implemented, but PagerDuty/Slack bridges unwired |
| **Audit Logging** | ✅ Good | Remediation events logged to ClickHouse with full audit trail |
| **Multi-tenancy** | ⚠️ Partial | Tenant isolation via query WHERE clauses; no row-level security |
| **Data Retention** | ✅ Good | TTLs defined on all tables (30 days / 90 days / 12 months) |
| **Horizontal Scaling** | ⚠️ Partial | Kafka consumer groups support multiple instances; API server is stateless |
| **Disaster Recovery** | ❌ Missing | No backup/restore documentation or tooling |
| **Documentation** | ⚠️ Partial | Good README, missing API docs, no runbook |
