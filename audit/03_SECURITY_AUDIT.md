# 03 — Security Audit

## Critical Severity

### SEC-01: OIDC Auth Middleware Is a No-Op in Production Path
- **Severity**: 🔴 Critical
- **File**: `control-plane/internal/api/rest/server.go` L265-290
- **Line**: 287-288
- **Explanation**: When `OIDC_ISSUER` is set (production), the middleware calls `next.ServeHTTP(w, r)` WITHOUT verifying the token. The `verifyOIDCToken()` function in `oidc_auth.go` is never called from the middleware. The production code path at L287 simply passes the request through.
- **Why it matters**: Any request with any `Authorization` header (or none) will pass through in production, completely bypassing authentication.
- **How to fix**: Call `verifyOIDCToken()` and extract claims before calling `next.ServeHTTP`. Set actor context values from verified claims.
- **Risk**: Complete authentication bypass
- **Estimated effort**: 2 hours
- **Breaking change**: No

### SEC-02: Hardcoded Demo API Key in Source Code
- **Severity**: 🔴 Critical
- **File**: `control-plane/internal/api/rest/server.go` L275
- **File**: `dashboard/src/App.tsx` L179
- **Explanation**: The string `Bearer health-demo-key-2026` is hardcoded in both backend middleware and the React frontend. This acts as a static password with no rotation or per-user scoping.
- **Why it matters**: Anyone reading the source code (or the frontend JS bundle) gains full API access in dev mode.
- **How to fix**: Remove hardcoded keys. Use proper OIDC token flow even in dev mode (e.g., with a local Keycloak/Dex instance) or use a short-lived JWT signed with a local dev key.
- **Risk**: Credential exposure via open source publication
- **Estimated effort**: 4 hours
- **Breaking change**: Yes (frontend auth flow changes)

### SEC-03: CORS Defaults to Wildcard `*`
- **Severity**: 🔴 Critical
- **File**: `control-plane/internal/api/rest/server.go` L128-131
- **Explanation**: Despite a comment (L121-122) stating "Wildcard `*` is explicitly rejected", when both `ALLOWED_ORIGINS` and `CORS_ORIGIN` are unset, the code defaults to `"*"` (L129). This is the common case in development and will likely leak into production.
- **Why it matters**: Allows any website to make authenticated API requests on behalf of a user (CSRF vector).
- **How to fix**: Default to an empty string and reject requests with no matching origin, or require the env var to be explicitly set.
- **Risk**: CSRF / Cross-origin data exfiltration
- **Estimated effort**: 1 hour
- **Breaking change**: No

---

## High Severity

### SEC-04: No Request Body Size Limits
- **Severity**: 🟠 High
- **File**: `control-plane/internal/api/rest/server.go`
- **Explanation**: The HTTP server does not set `http.MaxBytesReader` on any request body. While `ApplyRemediation` checks YAML size after decoding (L482-486), the JSON decoder will fully read an arbitrarily large body into memory first.
- **Why it matters**: A single request with a multi-GB body will OOM the API server.
- **How to fix**: Wrap `r.Body` with `http.MaxBytesReader(w, r.Body, maxBodySize)` before decoding, or use a global middleware.
- **Risk**: Denial of Service
- **Estimated effort**: 1 hour
- **Breaking change**: No

### SEC-05: SQL Injection via String Interpolation in ClickHouse Queries
- **Severity**: 🟠 High
- **File**: `control-plane/internal/storage/clickhouse/health_repository.go` L390-401
- **Line**: 398, 626, 637-639
- **Explanation**: The SigNoz query uses `fmt.Sprintf` to interpolate `r.dbName` and `r.tableName` into SQL strings. While these come from env vars (not user input), the pattern is dangerous and sets a bad precedent for contributors. The `traceID` in the parameterized query is safe, but the table/db names are not.
- **Why it matters**: If `CLICKHOUSE_DB` or `CLICKHOUSE_TABLE` env vars are ever influenced by user input or misconfigured, this becomes a SQL injection vulnerability.
- **How to fix**: Validate `dbName` and `tableName` against a strict regex `^[a-zA-Z_][a-zA-Z0-9_]*$` at initialization time.
- **Risk**: SQL Injection (indirect)
- **Estimated effort**: 1 hour
- **Breaking change**: No

### SEC-06: ClickHouse Connection Uses Empty Password
- **Severity**: 🟠 High
- **File**: `control-plane/cmd/seeder/main.go` L29-31, `cmd/api-server/main.go` L65, `internal/storage/clickhouse/client.go` L23
- **Explanation**: All ClickHouse connections use `password: ""`. The `NewClient` function accepts a password parameter but every caller passes an empty string.
- **Why it matters**: ClickHouse in production must be authenticated. Setting empty password defaults normalizes insecure database access.
- **How to fix**: Read password from `CH_PASSWORD` env var (or secrets manager). Require non-empty password when `ENV=production`.
- **Risk**: Unauthorized database access
- **Estimated effort**: 1 hour
- **Breaking change**: No

### SEC-07: `parseJWTStructural` Performs No Signature Verification
- **Severity**: 🟠 High
- **File**: `control-plane/internal/api/rest/oidc_auth.go` L143-186
- **Explanation**: This function base64-decodes a JWT payload without verifying the signature. While it's documented as dev-only, it's an exported-accessible function with no guard.
- **Why it matters**: If accidentally called in production, any forged JWT would be accepted.
- **How to fix**: Add a build tag `//go:build !production` or a runtime guard that panics if `ENV=production`.
- **Risk**: Token forgery
- **Estimated effort**: 30 minutes
- **Breaking change**: No

### SEC-08: Unused `oauth2.NoContext` Reference
- **Severity**: 🟢 Low
- **File**: `control-plane/internal/api/rest/oidc_auth.go` L189
- **Explanation**: `var _ = oauth2.NoContext` is used solely to prevent the `oauth2` import from being removed. This is a code smell — if the package isn't needed, remove the import.
- **Risk**: Low — unnecessary dependency surface
- **Estimated effort**: 5 minutes
- **Breaking change**: No

---

## Medium Severity

### SEC-09: Rate Limiter Uses `RemoteAddr` Without Port Stripping
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/api/rest/server.go` L204
- **Explanation**: `r.RemoteAddr` includes the port number (e.g., `192.168.1.1:54321`). Each new TCP connection from the same IP gets a different port, which means each gets its own rate limiter — defeating the purpose entirely.
- **How to fix**: Use `net.SplitHostPort(r.RemoteAddr)` and use only the host portion as the key.

### SEC-10: Sensitive Error Details Leaked in API Responses
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/api/rest/server.go` L494, L696, L763, L881
- **Explanation**: Multiple handlers include raw Go error messages in HTTP response bodies (e.g., `"Validation failed: %v"`, `"failed to inject simulation data: "+err.Error()`). This leaks internal implementation details.
- **How to fix**: Log the full error server-side, return only a generic user-facing message.

### SEC-11: No CSRF Protection
- **Explanation**: The REST API uses cookie-less Bearer token auth, which provides natural CSRF protection. However, the dev mode hardcoded key effectively makes this a cookie-like static credential embedded in the frontend, re-enabling CSRF risk.

### SEC-12: Missing Security Headers
- **Explanation**: No `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`, `Content-Security-Policy` headers are set. The nginx.conf for the dashboard also lacks these.
- **How to fix**: Add a `securityHeaders` middleware.

### SEC-13: `X-Forwarded-For` Header Trusted Without Validation
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/api/rest/server.go` L205-207
- **Explanation**: The rate limiter and `ApplyRemediation` handler trust `X-Forwarded-For` directly. An attacker can spoof this header to bypass rate limits or forge audit trail source IPs.
- **How to fix**: Only trust `X-Forwarded-For` from known proxy CIDR ranges, or use the chi `middleware.RealIP` (which is already in the stack but the rate limiter re-implements its own logic).
