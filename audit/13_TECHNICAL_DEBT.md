# 13 — Technical Debt Register

| Priority | Debt Item | Impact | Effort | Category |
|----------|-----------|--------|--------|----------|
| **P0** | Remove mock/fallback data from production code paths | Data integrity | 1 day | Architecture |
| **P0** | Complete OIDC auth middleware implementation | Security | 2 days | Security |
| **P0** | Remove `.exe` binaries from git history | Repository health | 1 hour | DevOps |
| **P0** | Fix CORS wildcard default | Security | 1 hour | Security |
| **P0** | Add CMD to control-plane Dockerfile | Deployability | 30 min | DevOps |
| **P1** | Extract `server.go` god-file into focused packages | Maintainability | 2 days | Architecture |
| **P1** | Create centralized config struct from scattered `os.Getenv` | Reliability | 1 day | Architecture |
| **P1** | Add multi-stage Docker builds | DevOps | 4 hours | DevOps |
| **P1** | Fix rate limiter port-stripping bug | Security | 30 min | Security |
| **P1** | Close rawspan Kafka writer in Producer.Close() | Resource leak | 15 min | Backend |
| **P1** | Batch Kafka publishes in ingest gateway | Performance | 4 hours | Performance |
| **P1** | Fix Prometheus label cardinality explosion | Performance | 1 hour | Performance |
| **P2** | Wire streaming jobs into worker cmd | Feature completeness | 1 day | Architecture |
| **P2** | Add database migration tooling | Schema evolution | 1 day | Database |
| **P2** | Add request body size limits | Security | 1 hour | Security |
| **P2** | Standardize tenant_id type across CH schema | Consistency | 4 hours | Database |
| **P2** | Add bloom filter index on attributes column | Performance | 1 hour | Database |
| **P2** | Add security headers middleware | Security | 2 hours | Security |
| **P2** | Validate CH db/table names against regex | Security | 1 hour | Security |
| **P3** | Add frontend testing (Vitest + RTL) | Quality | 3 days | Testing |
| **P3** | Add integration test suite (Kafka→CH) | Quality | 3 days | Testing |
| **P3** | Add `-race` flag to CI test runs | Concurrency safety | 30 min | Testing |
| **P3** | Add fuzz tests for YAML parser and JWT parser | Security testing | 1 day | Testing |
| **P3** | Remove unused swagger dependencies | Cleanup | 30 min | Dependencies |
| **P3** | Add `.env.example` documentation | DX | 1 hour | Documentation |
| **P3** | Add Makefile with standard targets | DX | 2 hours | DevOps |

## Total Estimated Effort

| Priority | Items | Estimated Days |
|----------|-------|----------------|
| P0 (Critical) | 5 | ~4 days |
| P1 (High) | 7 | ~5 days |
| P2 (Medium) | 7 | ~4 days |
| P3 (Low) | 7 | ~9 days |
| **Total** | **26** | **~22 days (1 engineer)** |
