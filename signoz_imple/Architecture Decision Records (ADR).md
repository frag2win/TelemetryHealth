# Section 18
# Architecture Decision Records (ADR)

Version: 1.0

---

## ADR-001: Use ClickHouse as the Primary Storage Engine

### Status
Accepted

### Context
TelemetryHealth requires high-ingestion, analytical storage for replay events and behavioral intelligence.

### Decision
Use ClickHouse as the primary analytical database.

### Alternatives Considered
- PostgreSQL
- Elasticsearch
- Apache Druid

### Rationale
- High write throughput
- Excellent compression
- Columnar analytics
- Fast aggregations
- Native support for time-series workloads

### Trade-offs
Pros:
- Excellent analytical performance
- Cost-efficient storage

Cons:
- More operational complexity than PostgreSQL
- Less suited for transactional workloads

---

## ADR-002: Deterministic Inference Instead of LLM-Based Reasoning

### Status
Accepted

### Context
The Behavior Inference Engine reconstructs decisions from telemetry.

### Decision
Use deterministic rules and evidence correlation instead of LLM reasoning.

### Rationale
- Explainable
- Reproducible
- Auditable
- Low operational cost

### Trade-offs
Pros:
- Consistent outputs
- Easy to debug

Cons:
- Requires rule maintenance
- Less flexible for unknown patterns

---

## ADR-003: Store Replay Events, Not Derived Graphs

...