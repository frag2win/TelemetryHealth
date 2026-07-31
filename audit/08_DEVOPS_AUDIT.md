# 08 — DevOps Audit

### DEV-01: Compiled Binaries in Git Repository
- **Severity**: 🔴 Critical
- **Files**: `control-plane/api-server.exe` (~48MB), `control-plane/ingest-gateway.exe` (~36MB), `control-plane/worker.exe` (~28MB), `control-plane/e2e-test.exe` (~19MB)
- **Explanation**: ~131 MB of Windows executables committed to the repository. `.gitignore` has `*.exe` but files were likely committed before the rule was added.
- **How to fix**: `git rm --cached *.exe`, ensure `.gitignore` is effective, use CI artifacts or GitHub Releases instead.

### DEV-02: Dockerfile Has No Build Stage
- **Severity**: 🟠 High
- **File**: `control-plane/Dockerfile`
- **Explanation**: The Dockerfile copies pre-built binaries (`api-server-linux`) — there's no multi-stage build. The binaries must be built externally, and there's no Makefile or build script to do so.
- **How to fix**: Use multi-stage build: `FROM golang:1.22 AS builder` → `go build` → `FROM alpine`.

### DEV-03: No CMD/ENTRYPOINT in Control Plane Dockerfile
- **Severity**: 🟠 High
- **File**: `control-plane/Dockerfile` L10-11
- **Explanation**: The Dockerfile EXPOSEs ports but has no CMD or ENTRYPOINT. The container won't start any process by default.

### DEV-04: Dashboard Dockerfile Copies Pre-built `dist/`
- **Severity**: 🟡 Medium
- **File**: `dashboard/Dockerfile`
- **Explanation**: No build stage — requires `npm run build` to run externally first.

### DEV-05: docker-compose Uses Outdated `version: '3.8'` Key
- **Severity**: 🟢 Low
- **File**: `docker-compose.db.yaml` L1
- **Explanation**: The `version` key is deprecated in modern Docker Compose.

### DEV-06: No Health Checks in docker-compose
- **Severity**: 🟡 Medium
- **Explanation**: Neither ClickHouse nor Redpanda services have health checks defined. Services that depend on them may start before they're ready.

### DEV-07: CI Go Version Mismatch
- **Severity**: 🟡 Medium
- **File**: `.github/workflows/verify-build.yaml` L20
- **Explanation**: `verify-build.yaml` pins Go `1.22`, but `go.mod` requires `go 1.26.3`, and `ci.yml` uses `stable`. These should be consistent.

### DEV-08: No Frontend CI
- **Severity**: 🟡 Medium
- **Explanation**: No CI workflow builds, lints, or tests the React dashboard. A broken frontend can be merged without detection.

### DEV-09: No Makefile or Task Runner
- **Severity**: 🟡 Medium
- **Explanation**: No `Makefile`, `Taskfile`, or `justfile` exists. Contributors must read README and guess build commands.

### DEV-10: No `.env.example` File
- **Severity**: 🟡 Medium
- **Explanation**: With 30+ environment variables scattered across the codebase, there's no `.env.example` documenting required/optional vars with defaults.

### DEV-11: Redpanda Image Version Old
- **Severity**: 🟢 Low
- **File**: `docker-compose.db.yaml` L11
- **Explanation**: Uses `redpandadata/redpanda:v23.2.19` which is over 3 years old. Should update to a recent stable release.
