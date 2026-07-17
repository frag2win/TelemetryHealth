# TelemetryHealth Architecture Documentation

**Document ID:** TH-ARCH-003  
**Title:** Bounded Contexts  
**Status:** Approved  
**Version:** 3.0  
**Owner:** TelemetryHealth Core Team  
**Authors:** Shubham Pawar & Contributors  
**Last Updated:** July 2026

---

# 1. Purpose

This document defines the Domain-Driven Design (DDD) bounded contexts of TelemetryHealth.

A bounded context represents a logical business capability with its own:

- Responsibilities
- Domain model
- APIs
- Events
- Dependencies
- Ownership

The objective is to reduce coupling while increasing maintainability and independent evolution.

---

# 2. Architectural Philosophy

TelemetryHealth is **not** a collection of Go packages.

It is a collection of independent business capabilities.

Each capability owns its own domain model.

Communication between contexts should occur through:

- Application Services
- Domain Events
- Published Interfaces

Contexts MUST NOT access each other's internal implementation.

---

# 3. Context Map

```text
                   +----------------------+
                   |   Edge Processing    |
                   +----------+-----------+
                              |
                              v
                   +----------------------+
                   |      Ingestion       |
                   +----------+-----------+
                              |
                              v
                   +----------------------+
                   |      Streaming       |
                   +----------+-----------+
                              |
          +-------------------+-------------------+
          |                   |                   |
          v                   v                   v
+----------------+   +----------------+   +----------------+
| Health Engine  |   | Behavior Engine|   | Replay Engine  |
+-------+--------+   +-------+--------+   +-------+--------+
        |                    |                    |
        +----------+---------+---------+----------+
                   |                   |
                   v                   v
          +----------------+   +----------------+
          | Decision Engine|   | Root Cause     |
          +-------+--------+   +-------+--------+
                  |                    |
                  +---------+----------+
                            |
                            v
                  +----------------------+
                  | Remediation Engine   |
                  +----------+-----------+
                             |
         +-------------------+-------------------+
         |                                       |
         v                                       v
 +------------------+                  +------------------+
 | Alerting Context |                  | Dashboard Context|
 +------------------+                  +------------------+
```

---

# 4. Context Principles

Every bounded context SHALL:

- Own its own business logic
- Own its own domain model
- Expose public interfaces only
- Publish domain events
- Hide implementation details

Every bounded context SHALL NOT:

- Access another context's database
- Import another context's internal package
- Modify another context's entities directly

---

# 5. Edge Processing Context

## Purpose

Performs lightweight, real-time inspection of telemetry inside the OpenTelemetry Collector.

### Responsibilities

- Cardinality estimation
- Missing span detection
- Instrumentation validation
- Sampling analysis
- Fail-open protection

### Inputs

OTLP telemetry.

### Outputs

Telemetry Health Events.

### Dependencies

None.

This context should remain lightweight and deterministic.

---

# 6. Ingestion Context

## Purpose

Receives telemetry health events from collectors.

### Responsibilities

- Authentication
- Authorization
- Validation
- Rate limiting
- Event acceptance

### Outputs

Validated domain events.

The ingestion context should never contain business intelligence.

---

# 7. Streaming Context

## Purpose

Acts as the transport layer between ingestion and analysis.

### Responsibilities

- Event buffering
- Ordering
- Retry
- Backpressure isolation
- Stream partitioning

This context owns message delivery—not business rules.

---

# 8. Replay Context

## Purpose

Stores and reconstructs telemetry execution history.

### Responsibilities

- Replay persistence
- Timeline generation
- Session reconstruction
- Historical lookup

### Domain Objects

- Replay
- ReplayFrame
- ReplaySession

---

# 9. Health Context

## Purpose

Calculates telemetry quality.

### Responsibilities

- Health Score
- Coverage Score
- Cardinality Risk
- Sampling Health
- Pipeline Health

### Domain Objects

- HealthScore
- HealthPolicy
- HealthSnapshot

This context owns the Health Score.

No other context may calculate it.

---

# 10. Behavior Context

## Purpose

Discovers runtime behavioral patterns.

### Responsibilities

- Behavior graph generation
- Flow analysis
- Dependency analysis
- Runtime relationship discovery

### Domain Objects

- BehaviorGraph
- ServiceNode
- BehaviorEdge

---

# 11. Root Cause Context

## Purpose

Explains failures.

### Responsibilities

- Correlation
- Causal inference
- Failure chain generation
- Confidence scoring

### Domain Objects

- RootCause
- RootCauseGraph
- Evidence
- Confidence

---

# 12. Decision Context

## Purpose

Converts analysis into operational decisions.

### Responsibilities

- Risk evaluation
- Policy evaluation
- Recommendation generation
- Prioritization

### Domain Objects

- Decision
- Recommendation
- DecisionPolicy

---

# 13. Remediation Context

## Purpose

Transforms recommendations into executable fixes.

### Responsibilities

- YAML generation
- Configuration patches
- Collector remediation
- Validation

### Domain Objects

- Remediation
- Patch
- PatchValidator

---

# 14. Alerting Context

## Purpose

Distributes operational notifications.

Supported channels may include:

- Slack
- PagerDuty
- Alertmanager
- Webhooks

Alerting owns delivery—not decision making.

---

# 15. Dashboard Context

## Purpose

Provides visualization.

Responsibilities include:

- Graph rendering
- Health dashboards
- Replay visualization
- Root cause display

Business logic MUST NOT exist inside the dashboard.

---

# 16. MCP Context

## Purpose

Expose platform capabilities to AI systems.

Supported operations include:

- Replay analysis
- Root cause requests
- Health summaries
- Behavior queries
- Remediation generation

The MCP context is an interface layer.

It does not own business logic.

---

# 17. Domain Events

The following events form the backbone of the platform.

```mermaid
graph TD;
    "TelemetryObserved" --> "HealthCalculated";
    "HealthCalculated" --> "BehaviorGenerated";
    "BehaviorGenerated" --> "ReplayStored";
    "ReplayStored" --> "RootCauseDiscovered";
    "RootCauseDiscovered" --> "DecisionCreated";
    "DecisionCreated" --> "RemediationGenerated";
    "RemediationGenerated" --> "AlertPublished";
```

Every context SHOULD publish domain events rather than directly invoking downstream services.

---

# 18. Context Dependency Rules

Allowed:

```mermaid
graph TD;
    "Edge" --> "Ingestion";
    "Ingestion" --> "Streaming";
    "Streaming" --> "Analysis";
    "Analysis" --> "Decision";
    "Decision" --> "Remediation";
    "Remediation" --> "Alerting";
```

Forbidden:

- Dashboard → Database
- Dashboard → Kafka
- Alerting → Health Engine
- Root Cause → ClickHouse
- Behavior → REST

All communication must occur through Application Services or published interfaces.

---

# 19. Repository Mapping

Future repository layout:

```
internal/

domain/

edge/

health/

behavior/

replay/

decision/

rootcause/

remediation/

alerting/

application/

interfaces/

infrastructure/
```

Each bounded context becomes a top-level package.

---

# 20. Future Evolution

Additional contexts may include:

- Cost Intelligence
- Security Intelligence
- AI Policy Engine
- Fleet Analytics
- Capacity Forecasting

These can be introduced without modifying existing contexts.

---

# 21. Summary

Bounded contexts provide the organizational foundation of TelemetryHealth.

By assigning ownership of business capabilities to independent contexts, the platform becomes easier to understand, extend, test, and evolve.

Every future architectural decision should preserve the independence and integrity of these contexts.

---

## Related Documents

- TH-ARCH-000 Project Vision
- TH-ARCH-001 Architecture Principles
- TH-ARCH-002 System Context
- TH-ARCH-004 Domain Model

---

**End of Document**