# 19 — Production Roadmap

## Phase 1: Critical Production Blockers (Week 1-2)
- [ ] Complete OIDC authentication flow (SEC-01)
- [ ] Remove hardcoded credentials from source (SEC-02)
- [ ] Fix CORS wildcard default (SEC-03)
- [ ] Add request body size limits (SEC-04)
- [ ] Remove `.exe` binaries from repository (DEV-01)
- [ ] Fix rawspan Kafka writer leak (BUG-01)
- [ ] Fix rate limiter port-stripping (SEC-09)
- [ ] Remove mock data from production code paths
- [ ] Add CMD to control-plane Dockerfile (DEV-03)

## Phase 2: Security (Week 3)
- [ ] Validate ClickHouse DB/table names (SEC-05)
- [ ] Require CH_PASSWORD in production (SEC-06)
- [ ] Guard `parseJWTStructural` from production use (SEC-07)
- [ ] Add security headers middleware (SEC-12)
- [ ] Validate X-Forwarded-For against trusted proxies (SEC-13)
- [ ] Add RBAC enforcement (currently role is extracted but never checked)

## Phase 3: Performance (Week 4)
- [ ] Batch Kafka publishes in ingest gateway (PERF-04)
- [ ] Add bloom filter index on attributes column (PERF-02)
- [ ] Fix Prometheus label cardinality explosion (PERF-06)
- [ ] Add context deadlines to DB queries (PERF-03)
- [ ] Increase Kafka consumer fetch timeout (PERF-05)

## Phase 4: Architecture (Week 5-6)
- [ ] Create centralized configuration system
- [ ] Extract server.go god-file
- [ ] Add database migration tooling
- [ ] Wire streaming jobs into worker cmd
- [ ] Standardize tenant_id type across schema
- [ ] Add multi-stage Docker builds

## Phase 5: Developer Experience (Week 7-8)
- [ ] Add Makefile with standard targets
- [ ] Generate and serve Swagger/OpenAPI docs
- [ ] Add `docker-compose.yml` for full-stack local dev
- [ ] Add contributing guide with setup instructions
- [ ] Add pre-commit hooks (lint, format, vet)

## Phase 6: Open Source Release (Week 9-10)
- [ ] LICENSE file (currently claimed MIT but no LICENSE file exists)
- [ ] SECURITY.md with vulnerability reporting process
- [ ] CONTRIBUTING.md
- [ ] GitHub issue/PR templates
- [ ] SBOM generation in CI
- [ ] Signed releases
- [ ] Public documentation site
