# 11 — Dead Code Report

| Type | Location | Description |
|------|----------|-------------|
| Unused struct | `health_repository.go` L22-25 | `PricingConfig` exported but only used internally |
| Unused function | `health_repository.go` L318-320 | `Named()` wraps `ch.Named()` — callers use `ch.Named()` directly |
| Unused metric | `telemetry/metrics.go` L34-39 | `ClickHouseWriteDuration` declared but never observed |
| Unused variable | `server_test.go` L129-133 | `expectedSubstring` assigned but never used in assertion |
| Unused import guard | `workers.go` L188-192 | `var _ = kafkago.Message{}`, `var _ = strconv.Itoa`, `var _ = time.Now` |
| Unused import guard | `oidc_auth.go` L189 | `var _ = oauth2.NoContext` |
| Unused variable | `health_repository.go` L560 | `var _ driver.Conn` — blank identifier import guard |
| Unused packages | `internal/streaming/` | 6 job files (ai_health, cardinality, coverage, healthscore, tracechain) — none imported by any cmd |
| Unused files | `casting.yaml`, `casting.yaml.lock` | Unknown purpose, not referenced anywhere |
| Unused directory | `FOCUS/` | Unknown purpose |
| Dead binary artifacts | `*.exe` files | Should not be in repo — 131 MB total |
| Unused flag | `mcp-server/main.go` L65 | `sseFlag` is parsed but never read (only `stdioFlag` is checked) |
| Unused bridges | `alerting/pagerduty_bridge.go`, `alerting/slack_bridge.go` | Defined but never instantiated or wired |
| Dead comment | `server.go` L121-122 | States `*` CORS is "explicitly rejected" but code defaults to `*` |
| Unreachable code | `health_repository.go` L670-675 | Empty lines at end of function after all return paths |
