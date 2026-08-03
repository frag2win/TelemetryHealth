# 📋 TelemetryHealth Developer Change Log — Hardening & Cleanup

**Date:** August 3, 2026  
**Scope:** Mock cleanup, Phase 2 security hardening, rate limiter leak fix, DoS payload protection, and CORS policy enforcement.

---

## 🎯 Executive Summary for Team Developers

TelemetryHealth has completed **Phase 1 (Mock Cleanup)** and **Phase 2 (Security Hardening & Resource Protection)**:

1. **Production Realism:** All hardcoded fallback data, static JSON returns, fake tenant metrics, and hardcoded authentication bypass keys have been eliminated.
2. **Security & Resource Protection:** CORS policies enforce strict origins in production, rate limiter memory/goroutine leaks have been eliminated, IP extraction is port-sanitized, and request body size limits (1MB) prevent payload DoS.

---

## 🛠 Summary of Code Changes

### Phase 1 — Removal of Hardcoded Mocks
* **Storage Layer ([`health_repository.go`](file:///c:/Users/admin/Desktop/tele/TelemetryHealth/control-plane/internal/storage/clickhouse/health_repository.go#L266-L306)):** Removed hardcoded switch-case block injecting fake stats (`CardinalityMax = 120450`, `OrphanCount = 12`, `CompositeScore = 92`) for tenant IDs `acme-prod`, `acme-staging`, `tenant-alpha`, `tenant-beta`, and `tenant-gamma`.
* **REST Server ([`server.go`](file:///c:/Users/admin/Desktop/tele/TelemetryHealth/control-plane/internal/api/rest/server.go)):** Removed `"Bearer health-demo-key-2026"` hardcoded bearer key bypass; updated `GetCoverage` and `GetTracesOrphans` to query dynamic repository stats.
* **MCP Engine ([`client.go`](file:///c:/Users/admin/Desktop/tele/TelemetryHealth/control-plane/internal/mcp/client.go)):** Removed hardcoded `[SIMULATED] t1` and `[SIMULATED] t2` trace array elements from `QueryAgentTraces`.

### Phase 2 — Security Hardening & Leak Fixes
* **CORS Policy ([`server.go`](file:///c:/Users/admin/Desktop/tele/TelemetryHealth/control-plane/internal/api/rest/server.go#L122-L150)):** Rejects wildcard `*` origins in `ENV=production`, defaulting strictly to trusted origin `http://localhost:5173`.
* **Rate Limiter Memory & Goroutine Leak Fix ([`server.go`](file:///c:/Users/admin/Desktop/tele/TelemetryHealth/control-plane/internal/api/rest/server.go#L188-L225)):** Replaced `time.Tick` with `time.NewTicker` (with defer stop) and extracted clean IP addresses using `net.SplitHostPort(r.RemoteAddr)` to prevent visitor map explosion from ephemeral ports.
* **Payload DoS Protection ([`server.go`](file:///c:/Users/admin/Desktop/tele/TelemetryHealth/control-plane/internal/api/rest/server.go)):** Enforced **1MB maximum payload size limit** via `http.MaxBytesReader` across POST/PUT handlers (`/config`, `/simulate`, `/remediation/apply`).

---

## 🧪 Unit & Integration Test Results

All Go unit test suites across the repository were executed and passed cleanly:

| Subsystem / Package | Status | Test Time |
|---|---|---|
| `control-plane/internal/api/rest` | **PASS** | `0.045s` |
| `control-plane/internal/storage/clickhouse` | **PASS** | `0.447s` |
| `control-plane/internal/mcp` | **PASS** | `0.048s` |
| `processor/cardinality` | **PASS** | `0.074s` |
| `processor/failopen` | **PASS** | `0.163s` |
| `processor/tracechain` | **PASS** | `0.082s` |

---

## 📝 Developer Impact & Notes for Local Testing

1. **CORS:** When deploying in production, set `ALLOWED_ORIGINS="https://your-domain.com"` in environment variables.
2. **Payload Size:** Request payloads exceeding 1MB on API endpoints will be rejected with HTTP 413 (Payload Too Large).
3. **Rate Limiting:** Ephemeral client ports no longer spawn separate rate limit entries; rate limits operate on host IP addresses.
