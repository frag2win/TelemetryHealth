# 18 — Refactoring Plan

## Phase 1: Remove Hackathon Shortcuts (1 week)
**Goal:** Strip out all fake data and development crutches from production paths.

1. **Remove all hardcoded mock data** from `api/rest/server.go`
   - Rewrite `GetTenantIssues`, `GetCoverage`, `GetTracesOrphans` to query ClickHouse.
   - Remove `simulateTimeRangeMetrics` from frontend `App.tsx`.
2. **Remove `MockHealthRepository` from production**
   - Delete `cmd/mcp-server/main.go` lines 24-61 and import `storage/mock` if needed for testing, or better, wire the real ClickHouse repo.
3. **Clean up Repository**
   - Run `git rm --cached control-plane/*.exe`
   - Ensure `.gitignore` is comprehensive.
4. **Remove Env Mutation**
   - Delete `os.Setenv("INSECURE_DEV_MODE", "true")` from `api-server` and `ingest-gateway` main functions.
5. **Fix Resource Leaks**
   - Add `p.rawspan.Close()` to `producer.go` Close method.

## Phase 2: Security Hardening (1 week)
**Goal:** Ensure the system is safe to run on the public internet.

1. **Complete OIDC Authentication**
   - Call `verifyOIDCToken()` in the middleware when `OIDC_ISSUER` is set.
   - Set request context with verified claims.
2. **Remove Hardcoded API Key**
   - Delete `health-demo-key-2026` from `server.go` and `App.tsx`.
   - Implement proper dev-mode auth (e.g., short-lived signed JWTs).
3. **Fix CORS Configuration**
   - Change default from `*` to strict origin matching.
4. **Add Resource Limits**
   - Wrap `http.MaxBytesReader` around all request bodies before JSON decode.
5. **Fix Rate Limiter IP Extraction**
   - Use `net.SplitHostPort` on `r.RemoteAddr` to rate-limit by IP, not connection port.
6. **Add Security Headers**
   - Create and apply a middleware that adds HSTS, CSP, and X-Frame-Options headers.
7. **Database Security**
   - Require `CH_PASSWORD` in production; do not default to empty string.

## Phase 3: Architecture Cleanup (2 weeks)
**Goal:** Improve maintainability and reduce coupling.

1. **Extract `server.go` God-file**
   - Move middlewares to `internal/api/rest/middleware/`
   - Split handlers by domain: `handlers_health.go`, `handlers_config.go`, `handlers_agent.go`
2. **Centralize Configuration**
   - Create a `config.go` struct that loads and validates all env vars at startup.
   - Stop reading `os.Getenv` deep inside business logic.
3. **Batch Kafka Publishes**
   - Refactor `grpc_server.go` to collect spans and publish in a single batch per topic.
4. **Database Migrations**
   - Integrate `golang-migrate` to manage ClickHouse schema evolution.
5. **Prometheus Cardinality Fix**
   - Use chi route patterns for HTTP metrics instead of raw URL paths.

## Phase 4: Quality & Testing (2 weeks)
**Goal:** Establish confidence in the system through automated testing.

1. **Integration Test Suite**
   - Write tests using Testcontainers (ClickHouse, Redpanda) to verify the ingest → worker → db flow.
2. **API Contract Tests**
   - Add tests validating the exact JSON structure returned by the REST API.
3. **Frontend Testing**
   - Set up Vitest and React Testing Library.
   - Write unit tests for data formatting and simulation functions.
4. **CI Pipeline Hardening**
   - Add frontend build/lint/test job to GitHub Actions.
   - Run Go tests with the `-race` flag.
5. **API Documentation**
   - Add `swaggo` annotations to handlers and generate OpenAPI spec.
