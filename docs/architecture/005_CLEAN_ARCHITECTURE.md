# TelemetryHealth Architecture Documentation

**Document ID:** TH-ARCH-007
**Title:** Dependency Rules
**Status:** Approved
**Version:** 3.0
**Owner:** TelemetryHealth Core Team
**Related Documents:**
- TH-ARCH-005 Clean Architecture
- TH-ARCH-006 Repository Structure
- RFC-002 Dependency Injection

---

# 1. Purpose

This document defines the dependency rules governing every package, module, and component within TelemetryHealth.

Its purpose is to ensure that:

- Business logic remains independent.
- Infrastructure is replaceable.
- Circular dependencies never occur.
- Architectural boundaries remain enforceable.

These rules are normative.

---

# 2. Fundamental Rule

Dependencies SHALL always point toward the Domain.

```mermaid
graph TD;
    "Presentation" --> "Interfaces";
    "Interfaces" --> "Application";
    "Application" --> "Domain";
    "Domain" --> "↑";
    "↑" --> "Infrastructure";
```

The Domain Layer is the center of the architecture.

Nothing inside the Domain may depend on any outer layer.

---

# 3. Layer Dependency Matrix

| Layer | Allowed Dependencies | Forbidden Dependencies |
|--------|----------------------|--------------------------|
| Domain | Standard Library | Application, Interfaces, Infrastructure |
| Application | Domain | Interfaces, Infrastructure Implementations |
| Interfaces | Application | Infrastructure Implementations (except composition root) |
| Infrastructure | Domain Interfaces | Interface Layer |

---

# 4. Domain Layer Rules

The Domain Layer SHALL contain only business concepts.

Allowed imports:

- context
- time
- errors
- fmt (sparingly)

Forbidden imports:

- net/http
- database/sql
- clickhouse-go
- kafka clients
- grpc
- OpenTelemetry SDK
- yaml
- json encoding for transport

The Domain must remain deterministic and side-effect free.

---

# 5. Application Layer Rules

Application Services orchestrate use cases.

Responsibilities:

- Coordinate repositories
- Execute business workflows
- Publish domain events
- Enforce application policies

Application Services MUST NOT:

- Execute SQL
- Build HTTP responses
- Parse YAML
- Perform authentication directly
- Depend on concrete infrastructure

---

# 6. Interface Layer Rules

The Interface Layer adapts external requests into application commands.

Responsibilities:

- Request validation
- Authentication
- Authorization
- DTO mapping
- Error translation

Interfaces MUST NOT:

- Calculate business metrics
- Execute business policies
- Query databases directly

---

# 7. Infrastructure Layer Rules

Infrastructure provides implementations of ports defined by the Domain or Application layers.

Examples:

- ClickHouseRepository
- KafkaEventBus
- SigNozExporter
- SlackNotifier

Infrastructure MUST NOT:

- Define business entities
- Contain domain rules
- Change aggregate state without going through the Application Layer

---

# 8. Composition Root

Dependency injection SHALL occur only at the Composition Root.

Typical locations:

```
cmd/api-server/main.go

cmd/worker/main.go

cmd/mcp-server/main.go
```

Responsibilities:

- Construct concrete implementations
- Wire interfaces to implementations
- Load configuration
- Start services

No business logic belongs here.

---

# 9. Ports and Adapters

Every external dependency SHALL be accessed through a port.

Example:

```
Domain
    │
HealthRepository
    │
ClickHouseHealthRepository
```

The Domain knows only the port.

Infrastructure provides the adapter.

---

# 10. Dependency Injection

Preferred style:

Constructor Injection.

Example:

```go
type HealthService struct {
    repo HealthRepository
    bus  EventBus
}
```

Avoid:

- Global singletons
- Service locators
- Hidden dependencies

---

# 11. Event Dependencies

Cross-context communication SHOULD use domain events.

Example:

```mermaid
graph TD;
    "ReplayCreated" --> "BehaviorGenerated";
    "BehaviorGenerated" --> "HealthCalculated";
    "HealthCalculated" --> "DecisionGenerated";
```

Services should communicate through events when synchronous coupling is unnecessary.

---

# 12. Repository Dependencies

Repositories belong to the Domain.

Example:

```
HealthRepository

ReplayRepository

BehaviorRepository
```

Implementations belong to Infrastructure.

Repositories must expose business-oriented methods, not storage-specific operations.

Good:

```
SaveReplay()

FindHealthSnapshot()
```

Bad:

```
ExecuteSQL()

RunQuery()

InsertRow()
```

---

# 13. Import Rules

Allowed:

```mermaid
graph TD;
    "domain" --> "application";
    "application" --> "interfaces";
```

Infrastructure may import Domain interfaces.

Forbidden:

```
domain → infrastructure

application → clickhouse

interfaces → kafka

dashboard → database
```

---

# 14. Cross-Context Rules

Bounded contexts communicate only through:

- Application Services
- Domain Events
- Public Interfaces

They SHALL NOT:

- Import each other's internal packages
- Access each other's repositories
- Modify each other's aggregates

---

# 15. Plugin Dependencies

Plugins are optional adapters.

They depend on public interfaces only.

Example:

```mermaid
graph TD;
    "Slack Plugin" --> "NotificationPort";
    "NotificationPort" --> "Application";
```

The Application layer is unaware of plugin implementations.

---

# 16. Testing Dependencies

Testing hierarchy:

| Layer | Mock |
|--------|------|
| Domain | None |
| Application | Repository Interfaces |
| Interfaces | Application Services |
| Infrastructure | External Systems |

Tests should verify contracts, not implementation details.

---

# 17. Dependency Violations

The following are considered architectural defects:

- Circular imports
- Domain importing infrastructure
- SQL inside handlers
- HTTP inside Domain
- Kafka inside Entities
- Business rules inside adapters
- Shared mutable global state

These SHOULD block code review.

---

# 18. Automated Enforcement

Continuous Integration SHOULD verify:

- Import graph
- Circular dependencies
- Layer boundaries
- Test coverage
- Static analysis
- Lint rules

Architecture should be enforced automatically wherever possible.

---

# 19. Future Considerations

Potential future additions:

- Architecture linter
- Dependency graph visualization
- Build-time architectural validation
- Package ownership metadata
- Automated RFC compliance checks

These tools should reinforce, not replace, architectural discipline.

---

# 20. Summary

Dependency management is the foundation of maintainable software.

By ensuring that all dependencies flow inward toward the Domain, TelemetryHealth remains:

- Modular
- Testable
- Replaceable
- Vendor-neutral
- Evolvable

Every dependency should be intentional, explicit, and justified.

---

## Related Documents

- TH-ARCH-005 Clean Architecture
- TH-ARCH-006 Repository Structure
- TH-ARCH-008 Plugin Architecture
- RFC-002 Dependency Injection

---

**End of Document**