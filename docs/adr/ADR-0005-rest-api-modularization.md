# ADR-0005: REST API Server Deconstruction & Centralized Configuration

**Date:** August 3, 2026  
**Status:** Accepted & Implemented  
**Context:** Control Plane REST API Server Architecture  

---

## 1. Context & Problem Statement

During early project development, the control plane REST API server grew into a monolithic 1,140-line `server.go` file. It combined HTTP routing, rate-limiting, CORS handling, authentication, error serialization, domain handler logic, and embedded mock fallbacks into a single file.

This caused several maintainability and operational challenges:
1. **High Cognitive Load:** Finding domain-specific logic required navigating a massive file.
2. **Coupled Dependencies:** Business handlers, middleware, and mock data fallbacks were tightly coupled.
3. **Configuration Scattering:** Environment variables (`os.Getenv`) were parsed on-the-fly deep inside handlers and middlewares.
4. **Testing Friction:** Unit-testing individual middlewares or handlers in isolation was difficult.

---

## 2. Decision Outcome

We decided to deconstruct `server.go` into domain-focused files within `package rest` and introduce a centralized configuration package `internal/config`.

### Architectural Breakdown

```
control-plane/internal/
├── config/
│   └── config.go             # Centralized environment configuration loader & validator
└── api/rest/
    ├── server.go             # Core Server struct, Start(), Shutdown()
    ├── routes.go             # Chi router & endpoint registration
    ├── helpers.go            # Response encoders, error handlers, UUID validators
    ├── middleware.go         # CORS, rate limiting, tracing, OIDC auth, Prometheus metrics
    ├── handlers_health.go    # Composite health score, coverage, and trace orphan endpoints
    ├── handlers_agent.go     # AI agent traces, behavior, decision, and root cause endpoints
    ├── handlers_config.go    # Tenant configuration endpoints
    └── handlers_remediation.go # YAML patch apply and simulation endpoints
```

### Key Decisions Made

1. **Centralized Configuration (`internal/config`):**
   - All 30+ environment variables (`ENV`, `PORT`, `CORS_ORIGIN`, `OIDC_ISSUER`, `CH_*`) are parsed once at startup into an immutable `Config` struct via `config.LoadConfig()`.
2. **Domain-Specific Handler Slices:**
   - Grouped endpoints logically into `handlers_health.go`, `handlers_agent.go`, `handlers_config.go`, and `handlers_remediation.go`.
3. **Route Metric Normalization:**
   - Updated Prometheus metrics middleware to use Chi route pattern templates (e.g. `/api/v1/tenant/{tenant_id}/health`) instead of raw URL paths to prevent label cardinality explosions.
4. **Strict Request Body Protection:**
   - Enforced a 1MB payload size limit (`http.MaxBytesReader`) on all POST/PUT endpoints.

---

## 3. Consequences

### Positive
* **Idiomatic Go Structure:** Clean separation of concerns makes the codebase easy to navigate for new contributors.
* **Testing:** Midleware and handlers can be tested in isolation; config defaults are covered by unit tests (`config_test.go`).
* **Security & Reliability:** Rate-limiting visitor maps no longer leak memory or goroutines, request payload DoS is prevented, and CORS defaults are strictly enforced.

### Negative / Trade-Offs
* **File Count:** Increased number of files in `internal/api/rest/` from 2 to 8 files.

---

## 4. References & PRD Alignment

* PRD §10: Non-Functional Requirements (Security, Performance, Maintainability)
* Engineering Audit: Finding #1 (God-file refactoring), Finding #12 (Payload limits)
