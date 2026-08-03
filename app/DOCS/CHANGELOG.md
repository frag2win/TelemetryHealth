# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- Add centralized configuration loader `internal/config/config.go` with environment variable validation ([413db49](https://github.com/frag2win/TelemetryHealth/commit/413db49892a7e3d0c8e29d571088387df6eabe40))
- Add architectural decision record `docs/adr/ADR-0005-rest-api-modularization.md` ([413db49](https://github.com/frag2win/TelemetryHealth/commit/413db49892a7e3d0c8e29d571088387df6eabe40))
- Pivot to AI Agent Observability for SigNoz hackathon ([d198055](https://github.com/frag2win/TelemetryHealth/commit/d1980555c1b2d32be1b522c7e2a243b52e5e1d02))

### Changed

- Deconstruct 1,140-line `server.go` God-file into modular domain handlers (`handlers_health.go`, `handlers_agent.go`, `handlers_config.go`, `handlers_remediation.go`, `middleware.go`, `helpers.go`) ([413db49](https://github.com/frag2win/TelemetryHealth/commit/413db49892a7e3d0c8e29d571088387df6eabe40))
- Normalize Prometheus HTTP metric route pattern labels using Chi route templates ([413db49](https://github.com/frag2win/TelemetryHealth/commit/413db49892a7e3d0c8e29d571088387df6eabe40))
- Chain AI agent reconstruction engines (`behavior`, `decision`, `rootcause`) directly into REST API handlers ([413db49](https://github.com/frag2win/TelemetryHealth/commit/413db49892a7e3d0c8e29d571088387df6eabe40))

### Removed

- Remove hardcoded fallback mock data block from ClickHouse repository (`health_repository.go`) ([614c6d4](https://github.com/frag2win/TelemetryHealth/commit/614c6d415907e19e1416f98230adfe6d6b6294f3))
- Remove hardcoded `"Bearer health-demo-key-2026"` bearer authentication bypass key ([614c6d4](https://github.com/frag2win/TelemetryHealth/commit/614c6d415907e19e1416f98230adfe6d6b6294f3))
- Remove hardcoded `[SIMULATED]` trace returns from MCP client (`client.go`) ([614c6d4](https://github.com/frag2win/TelemetryHealth/commit/614c6d415907e19e1416f98230adfe6d6b6294f3))

### Security

- Reject `*` wildcard CORS origins in production, defaulting strictly to trusted origin `http://localhost:5173` ([614c6d4](https://github.com/frag2win/TelemetryHealth/commit/614c6d415907e19e1416f98230adfe6d6b6294f3))
- Enforce 1MB maximum payload size limit (`http.MaxBytesReader`) across POST/PUT REST endpoints ([614c6d4](https://github.com/frag2win/TelemetryHealth/commit/614c6d415907e19e1416f98230adfe6d6b6294f3))

### Fixed

- Fix rate limiter memory and goroutine leak by replacing `time.Tick` with `time.NewTicker` and sanitizing IP extraction with `net.SplitHostPort` ([614c6d4](https://github.com/frag2win/TelemetryHealth/commit/614c6d415907e19e1416f98230adfe6d6b6294f3))
