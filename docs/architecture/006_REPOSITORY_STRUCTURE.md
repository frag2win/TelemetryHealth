# TelemetryHealth Architecture Documentation

**Document ID:** TH-ARCH-006
**Title:** Repository Structure
**Status:** Draft v1.0
**Version:** 3.0
**Owner:** TelemetryHealth Core Team
**Related RFCs:**
- RFC-001 Repository Reconstruction
- RFC-002 Dependency Injection
- RFC-003 Plugin System

---

# 1. Purpose

This document defines the canonical repository structure for TelemetryHealth.

It establishes:

- Directory ownership
- Package responsibilities
- Dependency boundaries
- Naming conventions
- Testing conventions
- Repository evolution rules

Every new package SHALL conform to this document.

---

# 2. Repository Philosophy

The repository SHALL be organized around **business capabilities**, not technologies.

Good:

```
health/
behavior/
replay/
remediation/
```

Bad:

```
sql/
helpers/
utils/
common/
misc/
services/
```

Technology changes.

Business capabilities remain.

---

# 3. High-Level Layout

```
TelemetryHealth/

├── cmd/
├── internal/
├── pkg/
├── plugins/
├── api/
├── dashboard/
├── processor/
├── sdk/
├── docs/
├── deployments/
├── scripts/
├── tools/
├── test/
├── examples/
└── .github/
```

---

# 4. cmd/

Purpose:

Executable entrypoints.

Examples:

```
cmd/

api-server/

mcp-server/

worker/

ingest-gateway/

benchmark/

simulator/

cli/
```

Rules:

- One executable per directory.
- No business logic.
- Bootstrap only.

Maximum target size:

300 LOC.

---

# 5. internal/

Contains implementation details.

Structure:

```
internal/

application/

domain/

interfaces/

infrastructure/

events/
```

Nothing outside the repository should import packages under `internal/`.

---

# 6. internal/domain/

The heart of TelemetryHealth.

```
domain/

health/

behavior/

decision/

replay/

rootcause/

remediation/

alert/
```

Contains:

- Entities
- Aggregates
- Value Objects
- Policies
- Specifications
- Events
- Repository Interfaces

Forbidden:

- SQL
- HTTP
- Kafka
- ClickHouse
- YAML
- JSON serialization

---

# 7. internal/application/

Contains use cases.

```
application/

health/

behavior/

decision/

replay/

rootcause/

remediation/
```

Contains:

- Services
- Commands
- Queries
- Orchestrators

No infrastructure code.

---

# 8. internal/interfaces/

Responsible for exposing functionality.

```
interfaces/

rest/

mcp/

grpc/

cli/

websocket/
```

Contains:

- Handlers
- DTOs
- Validation
- Authentication
- Response Mapping

Never contains business rules.

---

# 9. internal/infrastructure/

Implements external systems.

```
infrastructure/

clickhouse/

kafka/

storage/

signoz/

tempo/

jaeger/

yaml/

auth/

metrics/
```

Responsibilities:

- Database access
- Network clients
- External APIs
- Storage
- Plugin implementations

---

# 10. internal/events/

Contains the event system.

```
events/

bus/

handlers/

publishers/

subscribers/
```

Responsible for:

- Event dispatching
- Event registration
- Async processing

---

# 11. plugins/

Every optional integration belongs here.

Examples:

```
plugins/

slack/

pagerduty/

webhook/

tempo/

jaeger/

signoz/

prometheus/
```

Plugin requirements:

- Implements interface
- Independent lifecycle
- No domain dependencies

---

# 12. pkg/

Reusable public packages.

Examples:

```
pkg/

telemetry/

graph/

yaml/

otlp/

client/
```

Packages here MAY be imported by external projects.

---

# 13. dashboard/

React application.

Responsibilities:

- Visualization
- User interaction
- API consumption

Forbidden:

- SQL
- ClickHouse
- Business rules

---

# 14. processor/

OpenTelemetry Collector components.

Contains:

- Processors
- Exporters
- Extensions

Must remain lightweight.

Target latency:

<5 ms per batch.

---

# 15. sdk/

Example SDK integrations.

Contains:

- Python
- Go
- Java
- JavaScript

These are reference implementations.

---

# 16. deployments/

Deployment manifests.

Examples:

```
kubernetes/

docker/

helm/

compose/
```

Contains:

- Helm charts
- Docker Compose
- Kubernetes YAML
- Production manifests

---

# 17. docs/

Architecture.

RFCs.

Guides.

Diagrams.

Specifications.

Documentation is version controlled.

---

# 18. test/

Contains:

```
integration/

performance/

fixtures/

datasets/

e2e/
```

Production code should not depend on test assets.

---

# 19. scripts/

Automation only.

Examples:

- Release
- Benchmark
- Migration
- Build
- CI

Scripts should remain stateless.

---

# 20. tools/

Developer tooling.

Examples:

- Linters
- Architecture validators
- Code generators

Not part of production runtime.

---

# 21. Package Naming Rules

Package names should be nouns.

Good:

```
health

behavior

decision

graph

policy
```

Bad:

```
helpers

utils

manager

handler2

common
```

---

# 22. File Naming Rules

Prefer:

```
entity.go

service.go

repository.go

policy.go

event.go
```

Avoid:

```
helpers.go

misc.go

temp.go

new.go

old.go
```

---

# 23. Import Rules

Allowed:

```
Interface

↓

Application

↓

Domain
```

Infrastructure implements interfaces.

Forbidden:

```
Domain

↓

Infrastructure
```

Circular imports are prohibited.

---

# 24. Repository Evolution Rules

Every new package MUST answer:

- Which bounded context owns it?
- Which layer does it belong to?
- Which interfaces does it expose?
- Which dependencies does it require?
- Why can't this functionality live elsewhere?

If these questions cannot be answered, the package should not be created.

---

# 25. Repository Governance

Major structural changes require:

- RFC
- Architecture review
- Migration plan
- Backward compatibility assessment
- Documentation updates

Repository structure is considered part of the public architecture.

---

# 26. Future Expansion

Expected future directories include:

```
ml/

security/

cost/

fleet/

policy/

analytics/
```

These should be added without restructuring existing contexts.

---

# 27. Summary

The repository structure exists to communicate architecture.

A contributor should be able to locate the correct package for any feature without prior project knowledge.

The structure should evolve slowly, intentionally, and through documented architectural decisions.

---

## Related Documents

- TH-ARCH-005 Clean Architecture
- TH-ARCH-007 Dependency Rules
- RFC-001 Repository Reconstruction

---

**End of Document**