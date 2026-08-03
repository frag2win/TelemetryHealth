# 🛠 TelemetryHealth — Post-Audit Implementation Plan

**Date:** August 3, 2026  
**Source:** Implementation Verification Audit  
**Methodology:** Execution-path analysis, not keyword grep — every finding traces a caller → callee → consumer chain.

---

## How This Plan Was Built

1. An independent audit verified every claim in `CHANGES_LOG.md` against actual code.
2. Each finding was classified as ✅ Complete, 🟡 Partial, ❌ Missing, or ⚠ Dead Code.
3. Findings were reviewed by the engineering lead and re-classified by confidence and severity.
4. This plan reflects the **agreed-upon prioritization** — not every audit finding, only actionable ones.

---

## P0 — Critical (Fix Before Any Feature Work)

These issues affect authentication, data integrity, or credential exposure in production.

---

### P0.1 — Wire OIDC Token Verification Into Auth Middleware

**Problem:**  
`verifyOIDCToken()` in `oidc_auth.go` (L73) was built — JWKS verification, claim extraction, RBAC mapping — but **never called**. The `oidcAuthMiddleware` in `middleware.go` (L163–183) skips directly to `next.ServeHTTP(w, r)` when `OIDC_ISSUER` is set, without verifying the Bearer token.

**Execution path today:**
```
Request → oidcAuthMiddleware → issuer != "" → next.ServeHTTP(w, r) ← NO VERIFICATION
```

**Expected execution path:**
```
Request → oidcAuthMiddleware → issuer != "" → extract Bearer token → verifyOIDCToken() → inject claims → next.ServeHTTP()
```

**Files:**
- `control-plane/internal/api/rest/middleware.go` — L163–183
- `control-plane/internal/api/rest/oidc_auth.go` — L73 (`verifyOIDCToken`)

**Fix:**  
In `oidcAuthMiddleware`, when `issuer != ""`:
1. Extract `Bearer <token>` from `Authorization` header
2. Call `verifyOIDCToken(r.Context(), issuer, rawToken)`
3. Inject returned `actorID` and `actorRole` into context
4. On failure, return 401

**Verification:**  
- Unit test: mock OIDC provider, verify token verification is called
- Manual test: set `OIDC_ISSUER` to an invalid URL, confirm requests are rejected

---

### P0.2 — Eliminate Silent Mock Fallback in Production

**Problem:**  
`cmd/api-server/main.go` (L69–73) silently falls back to `mock.NewRepository()` when ClickHouse is unavailable. The production API then serves fabricated health metrics, agent traces, and span data through real endpoints. No response header or field indicates the data is fake.

**Execution path today:**
```
main() → ch.NewClient() fails → mock.NewRepository() → rest.NewServer(mockRepo) → API serves fake data
```

**Files:**
- `control-plane/cmd/api-server/main.go` — L69–73

**Fix:**  
Two options (choose one):
- **Option A (Recommended for production):** In production mode (`ENV=production`), refuse to start if ClickHouse is unavailable. Exit with a clear error.
- **Option B (Dev-friendly):** Keep mock fallback but add a `X-TelemetryHealth-DataSource: mock` response header to every response when mock is active. Log a prominent warning at startup.

**Verification:**  
- Set `ENV=production`, stop ClickHouse, start API server — confirm it refuses to start (Option A) or returns the header (Option B)

---

### P0.3 — Remove Frontend Demo Key

**Problem:**  
`health-demo-key-2026` is hardcoded in 8 locations across the React dashboard. The backend no longer validates this key, but it's visible in compiled JS bundles and browser DevTools.

**Files (8 occurrences):**
- `dashboard/src/App.tsx` — L221, L324
- `dashboard/src/components/views/Remediation.tsx` — L354
- `dashboard/src/components/views/SigNozIntegration.tsx` — L45
- `dashboard/src/components/SigNozComponents.tsx` — L26, L198, L410
- `dashboard/src/components/Shared.tsx` — L89

**Fix:**  
Replace all hardcoded `'Bearer health-demo-key-2026'` with a centralized auth helper that:
1. In dev mode: omits the header or sends a dev marker
2. In production: acquires a real OIDC token (post P0.1 completion)

**Verification:**  
- `grep -r "health-demo-key" dashboard/` returns zero results
- Dashboard still loads and fetches data in dev mode

---

### P0.4 — Remove `QueryAgentTraces` Hardcoded Fallback

**Problem:**  
`health_repository.go` (L410–439) returns fabricated agent traces (`trace-991`, `trace-992`) when ClickHouse returns no data. This mock data is indistinguishable from real telemetry.

**File:**
- `control-plane/internal/storage/clickhouse/health_repository.go` — L410–439

**Fix:**  
Replace the fallback block with `return traces, nil` (return empty slice). Let the API layer handle the empty-data case with appropriate UI messaging.

**Verification:**  
- Start with empty ClickHouse — `/agents` endpoint returns `[]` instead of fake traces
- Dashboard gracefully shows "No agent traces found" instead of fake data

---

## P1 — High (Fix Before Next Release)

These issues affect code quality, maintainability, and architectural consistency.

---

### P1.1 — Wire Centralized Config Loader

**Problem:**  
`config.LoadConfig()` in `internal/config/config.go` was built and tested but never called from any runtime code. There are 40+ direct `os.Getenv()` calls across the codebase.

**Files:**
- `control-plane/internal/config/config.go` — `LoadConfig()` (L25)
- `control-plane/cmd/api-server/main.go` — primary consumer

**Fix:**
1. Call `cfg := config.LoadConfig()` at the top of `api-server/main.go`
2. Pass `cfg` fields to `NewServer()`, `NewHealthRepository()`, and middleware
3. Replace direct `os.Getenv()` calls in `middleware.go`, `helpers.go`, and `handlers_remediation.go` with injected config values
4. Note: `os.Getenv()` in `cmd/` entrypoints and `oidc_auth.go` (runtime OIDC config) is acceptable

**Verification:**  
- All tests pass
- Application starts with environment variables set via config
- `grep -rn "os.Getenv" internal/api/rest/` shows zero results outside `oidc_auth.go`

---

### P1.2 — Remove Dead Code

**Problem:**  
~280 lines of dead code identified across multiple files.

**Files and specific removals:**

| File | Dead Code | Lines |
|---|---|---|
| `oidc_auth.go` | `parseJWTStructural()` | L140–186 |
| `oidc_auth.go` | `oidcClaims` struct | L29–35 |
| `oidc_auth.go` | `_ = oauth2.NoContext` | L189 |
| `mcp/client.go` | `QueryAgentTraces()` function | L24–31 |
| `health_repository.go` | `Named()` wrapper | L278–280 |
| `health_repository.go` | `_ driver.Conn` | L520 |

**Note:** Do NOT remove `verifyOIDCToken()`, `getOrCreateVerifier()`, `extractClaim()`, or `mapToRBACRole()` — these become live code after P0.1.

**Verification:**  
- `go build ./...` succeeds
- All tests pass
- No new lint warnings

---

### P1.3 — Remove Fake Tenant IDs from Replay Repository

**Problem:**  
`replay_repository.go` (L40–49) has hardcoded tenant names (`tenant-alpha`, `tenant-beta`, `tenant-gamma`) in a switch-case that assigns arbitrary query offsets.

**File:**
- `control-plane/internal/storage/clickhouse/replay_repository.go` — L40–49

**Fix:**  
Remove the switch-case. Use `offset = 0` for all tenants, or derive offset from a hash of the tenant UUID if per-tenant variation is needed.

**Verification:**  
- `grep -rn "tenant-alpha\|tenant-beta\|tenant-gamma" control-plane/` returns zero results

---

### P1.4 — Remove Hardcoded Fallback Graphs from Handlers

**Problem:**  
`handlers_agent.go` (L269–351) contains ~85 lines of hardcoded fallback behavior graphs, decision graphs, and root cause data. These are effectively mock data embedded in handler code.

**File:**
- `control-plane/internal/api/rest/handlers_agent.go` — L269–351

**Fix:**  
Replace fallback functions with empty-state responses:
- `fallbackBehaviorGraph()` → return `engine.Graph{Nodes: []engine.GraphNode{}, Edges: []engine.GraphEdge{}}`
- `fallbackDecisionGraph()` → return `&models.DecisionGraph{TraceID: traceID}`
- `fallbackRootCause()` → return `&models.RootCause{TraceID: traceID, Status: "No data"}`

**Verification:**  
- Request behavior/decisions/root-cause for a non-existent trace — returns empty structure, not fabricated data

---

## P2 — Medium (Improve When Capacity Allows)

These issues improve robustness, testing, and operational hygiene.

---

### P2.1 — Harden CORS Validation

**Problem:**  
Wildcard `*` is not explicitly rejected in production when set via `ALLOWED_ORIGINS="*"`. Production default is `http://localhost:5173`.

**File:**
- `control-plane/internal/api/rest/middleware.go` — L24–57

**Fix:**
1. After resolving allowed origins, if `ENV=production` and any origin is `*`, reject with a startup warning or replace with a safe default
2. Consider requiring `ALLOWED_ORIGINS` to be explicitly set in production (no default)

**Engineering decision required:** Is this a startup failure or a runtime warning?

---

### P2.2 — Add Missing Unit Tests

**Currently untested areas:**

| Area | Priority |
|---|---|
| CORS middleware (production vs dev behavior) | High |
| OIDC auth middleware (all three paths) | High (after P0.1) |
| `validateTenantID` (UUID vs slug vs invalid) | Medium |
| `GetTenantHealth` handler | Medium |
| `GetCoverage` / `GetTracesOrphans` error paths | Medium |
| Rate limiter IP extraction (X-Forwarded-For) | Low |

---

### P2.3 — Fix Goroutine Lifecycle Management

**Items:**
1. **Alerting poller** (`api-server/main.go` L86): Pass a cancellable context instead of `context.Background()` so the poller stops on shutdown
2. **Rate limiter cleanup** (`middleware.go` L105–117): Add `ctx.Done()` select if server lifecycle management is added later. Low priority — the current single-goroutine design is acceptable for now.

**Engineering decision:** The rate limiter goroutine is process-lifetime by design. Only change if hot-reload or dynamic lifecycle management is planned.

---

### P2.4 — Create Application Docker Compose

**Problem:**  
`docker-compose.db.yaml` defines only ClickHouse and Redpanda. No compose file exists for the application services (api-server, worker, ingest-gateway, dashboard).

**Fix:**  
Create `docker-compose.yaml` that includes all application services with proper environment variables (`CH_HOST`, `CH_PORT`, `ENV`, `ALLOWED_ORIGINS`, etc.) wired to the infrastructure services.

---

### P2.5 — Fix Swallowed Errors in Coverage/Orphan Handlers

**Problem:**  
`GetCoverage` (L83) and `GetTracesOrphans` (L103) in `handlers_health.go` silently drop database errors and return empty results.

**Fix:**  
Log the error and return a 503 response (matching `GetTenantHealth` pattern) instead of silently falling through to empty data.

---

### P2.6 — Remove Dashboard Simulated Terminal

**Problem:**  
`LiveTerminal.tsx` (L73–103) generates fake log lines from hardcoded service names and message templates.

**Fix:**  
Either connect to a real log stream (WebSocket/SSE) or clearly label the component as "Demo Mode" in the UI.

---

## Dependency Graph

```
P0.1 (OIDC auth) ──────────────────┐
                                    ├──→ P0.3 (Frontend key cleanup — needs auth strategy from P0.1)
P0.2 (Mock fallback) ──────────────┤
P0.4 (Agent trace fallback) ───────┘
                                    
P1.1 (Config wiring) ──── standalone
P1.2 (Dead code) ────────── after P0.1 (don't delete OIDC functions that become live)
P1.3 (Fake tenants) ────── standalone
P1.4 (Fallback graphs) ─── after P0.4

P2.* ──── all standalone, after P0/P1
```

---

## Execution Order

```
Week 1:  P0.1 → P0.2 → P0.4 → P0.3
Week 2:  P1.2 → P1.1 → P1.3 → P1.4
Week 3+: P2.1 → P2.2 → P2.5 → P2.3 → P2.4 → P2.6
```

---

## Success Criteria

- [ ] `verifyOIDCToken()` is called on every authenticated request when `OIDC_ISSUER` is set
- [ ] No mock data is served through production API endpoints without explicit indication
- [ ] Zero occurrences of `health-demo-key-2026` in the codebase
- [ ] `config.LoadConfig()` is the entry point for configuration in `api-server`
- [ ] `grep -rn "tenant-alpha\|tenant-beta\|tenant-gamma\|\[SIMULATED\]" control-plane/internal/` returns zero results
- [ ] All existing tests continue to pass
- [ ] No dead code remains in `oidc_auth.go` or `mcp/client.go`
