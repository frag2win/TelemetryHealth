<div align="center">

# 🩺 TelemetryHealth

### Production-Grade Telemetry Health Monitoring Platform

**Detect, Score, and Auto-Remediate broken observability pipelines — before your users notice.**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react)](https://react.dev)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Collector-orange?style=flat-square&logo=opentelemetry)](https://opentelemetry.io)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

</div>

---

## 🤖 AI Agent Observability (Hackathon Theme)

TelemetryHealth was built for the **SigNoz Agents of Observability Hackathon**. It monitors AI agent workflows instrumented with OpenTelemetry, detecting:

- **Token cost explosions** - Agents burning through LLM credits
- **Broken decision chains** - Missing spans in agentic reasoning loops  
- **Tool call failures** - Silent failures in agent tool execution
- **Cardinality explosions** - High-cardinality attributes in agent prompts

See [`sdk-clients/ai-agent-demo/`](./sdk-clients/ai-agent-demo/) for a sample instrumented agent.

### 🎥 Demo Video
[Link to Hackathon Demo Video](#) (Replace with actual link before submission)

---

## 📌 What Is TelemetryHealth?

TelemetryHealth is a production-ready observability reliability platform. It sits **inline with your OpenTelemetry Collector pipeline** and continuously monitors the health of your telemetry signals — catching cardinality explosions, broken trace chains, and coverage gaps **before** they corrupt your dashboards or exceed your billing.

Every tenant gets a real-time **Composite Health Score (0–100)**, and when an issue is detected, the platform generates a **one-click OTel YAML remediation snippet** to fix it.

---

## ✨ Features

| Feature | Description |
|---|---|
| 🔢 **Cardinality Detection** | HyperLogLog sketches detect attribute cardinality explosions per service/key across the entire collector fleet |
| 🔗 **Broken Trace-Chain Detection** | Identifies orphaned spans with no parent across collectors using bounded out-of-order event correlation |
| 📡 **Coverage Gap Monitoring** | Detects services that silently stop emitting telemetry |
| 📊 **Composite Health Score** | A weighted (0-100) score combining all signal sources, configurable per tenant |
| 🛠 **Auto-Remediation** | Generates OTel Collector YAML config patches validated via YAML structural checks and an OTel component allowlist |
| 🔔 **Alerting Bridges** | Integrates with SigNoz Alertmanager with deduplication and 15-minute cooldown suppression |
| 🏢 **Multi-Tenancy** | Zero-trust mTLS authentication validates tenant claims against SPIFFE/X.509 certificate SANs |
| 🛡 **Fail-Open Design** | All processor logic is wrapped in a circuit breaker — a processor crash **never** blocks the primary OTel pipeline |
| 🤖 **SigNoz MCP Server** | Exposes the TelemetryHealth insights and autonomous remediation tools to SigNoz's AI agents via the Model Context Protocol |

---

## 🏗 Architecture

```
┌───────────────────────────────────────────────────────┐
│                Your OTel Collector Fleet               │
│  ┌──────────────────────────────────────────────────┐ │
│  │  TelemetryHealth Processor (Go, OTel Collector)   │ │
│  │  ┌─────────────┐  ┌────────────┐  ┌──────────┐  │ │
│  │  │ Cardinality │  │ TraceChain │  │ Coverage │  │ │
│  │  │  Tracker    │  │  Detector  │  │  Monitor │  │ │
│  │  └──────┬──────┘  └─────┬──────┘  └────┬─────┘  │ │
│  │         └───────────────┴───────────────┘        │ │
│  │                  Circuit Breaker (Fail-Open)       │ │
│  └─────────────────────────┬────────────────────────┘ │
└────────────────────────────┼──────────────────────────┘
                             │ gRPC / OTLP (mTLS)
                             ▼
┌───────────────────────────────────────────────────────┐
│                  Control Plane (Go)                    │
│  ┌────────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │ Ingest Gateway │  │ Stream Jobs  │  │ REST API  │ │
│  │   (gRPC/OTLP) │  │ (HLL Merge)  │  │  (HTTP)   │ │
│  └────────────────┘  └──────┬───────┘  └─────┬─────┘ │
│                             ▼                 │       │
│                    ┌────────────────┐         │       │
│                    │   ClickHouse   │◄────────┘       │
│                    │  (TTL / AggMT) │                 │
│                    └────────────────┘                 │
└───────────────────────────────────────────────────────┘
                             │
                             ▼
┌───────────────────────────────────────────────────────┐
│           React Dashboard (Vite + TypeScript)          │
│    Health Score Gauge ▪ Metric Cards ▪ YAML Viewer    │
└───────────────────────────────────────────────────────┘
```

---

## 🗂 Repository Structure

```
TelemetryHealth/
├── processor/                    # OTel Collector Processor (Go module)
│   ├── failopen/                 #   Fail-open circuit breaker
│   ├── cardinality/              #   Local HyperLogLog tracker
│   ├── tracechain/               #   Local orphan span detector
│   ├── traces_consumer.go        #   OTLP Traces hook
│   ├── metrics_consumer.go       #   OTLP Metrics hook
│   └── logs_consumer.go          #   OTLP Logs hook
│
├── control-plane/                # Control Plane (Go module)
│   ├── cmd/ingest-gateway/       #   gRPC OTLP receiver entrypoint
│   ├── internal/
│   │   ├── authz/                #   mTLS / SPIFFE tenant verification
│   │   ├── ingest/               #   gRPC server (Traces, Metrics, Logs)
│   │   ├── streaming/            #   Stream processing jobs
│   │   ├── storage/clickhouse/   #   ClickHouse DDL & query layer
│   │   ├── api/rest/             #   HTTP REST API server
│   │   ├── remediation/          #   YAML config generator & validator
│   │   └── alerting/             #   SigNoz alerting bridge
│   └── deployments/helm/         #   Kubernetes Helm chart
│
├── dashboard/                    # Frontend React App (Vite + TypeScript)
│   └── src/
│       ├── components/
│       │   ├── Layout.tsx        #   Sidebar + header navigation
│       │   ├── HealthGauge.tsx   #   Animated circular SVG gauge
│       │   ├── MetricCard.tsx    #   KPI metric cards
│       │   └── RemediationPanel.tsx  # YAML copy-paste viewer
│       └── App.tsx               #   Main app with API fetch hook
│
└── app/DOCS/                     # Documentation & Status Tracking
    ├── Implementation_Status.md
    ├── CHANGELOG.md
    └── Build_Issue_Report.md
```

---

## 🚀 Getting Started

### Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | ≥ 1.26.3 | [go.dev](https://go.dev/dl/) |
| Node.js | ≥ 20 | [nodejs.org](https://nodejs.org) |
| Git | any | [git-scm.com](https://git-scm.com) |

---

### 1. Clone the Repository

```bash
git clone https://github.com/frag2win/TelemetryHealth.git
cd TelemetryHealth
```

---

### 2. Start the Go Control Plane API

```bash
cd control-plane
go mod tidy
go run ./cmd/api-server
```

> The REST API will start on **`http://localhost:8080`**

---

### 2.5 Start the MCP Server (SigNoz AI Agent Integration)

Open a **new terminal**:

```bash
cd control-plane
go run ./cmd/mcp-server
```

> The MCP server will start on port **`8081`** in SSE mode by default. You can also run it in stdio mode using `go run ./cmd/mcp-server --stdio`.

---

### 3. Start the React Dashboard

Open a **new terminal**:

```bash
cd dashboard
npm install
npm run dev
```

> The Dashboard will open on **`http://localhost:5173`**

---

### 4. (Optional) Run the OTel Processor Tests

```bash
cd processor
go test ./... -v -cover
```

---

## 🔌 API Reference

### `GET /api/v1/tenant/{tenant_id}/health`

Returns the current composite health state for a tenant.

**Response Example:**
```json
{
  "healthScore": 84,
  "metrics": {
    "cardinality": { "value": "1.2M", "change": 14.5 },
    "orphans":     { "value": "432",  "change": -5.2 },
    "coverage":    { "value": "14",   "change": 0 }
  },
  "remediation": {
    "issueType": "High Cardinality (user_id on checkout_service)",
    "yaml": "processors:\n  attributes/remediation:\n    actions:\n      - key: user_id\n        action: delete"
  }
}
```

---

## 🤖 SigNoz MCP Server Integration

TelemetryHealth implements a **Model Context Protocol (MCP)** server to natively integrate with SigNoz's AI workflows. The MCP server exposes our deep telemetry insights directly to SigNoz as autonomous tools:

1. **`GetTelemetryHealth`**: SigNoz AI agents can query the real-time composite health score, cardinality metrics, and orphan span rates for any tenant.
2. **`GenerateRemediation`**: When an issue is detected, SigNoz agents can use this tool to autonomously request a verified, ready-to-deploy OTel YAML configuration patch (e.g., dropping high-cardinality attributes).

This integration is located in `control-plane/internal/mcp/tools.go` and transforms TelemetryHealth from a passive monitoring system into an **Autonomous Telemetry Intelligence Platform**.

---

## 🔐 Security Model

- **mTLS Everywhere**: The Ingest Gateway requires mutual TLS for all incoming OTLP connections.
- **Zero-Trust Tenant Verification**: Every gRPC call is intercepted and the `x-tenant-id` header is cryptographically verified against the client certificate's SAN/SPIFFE URI — ensuring tenants cannot spoof each other's identity.
- **Fail-Open Circuit Breaker**: If the TelemetryHealth processor crashes or panics, the circuit breaker trips and allows telemetry to flow through **unprocessed** rather than dropping data.

---

## 🧪 Running Tests

```bash
# Processor unit tests
cd processor && go test ./... -cover

# Control plane unit tests
cd control-plane && go test ./...
```

**Current Coverage:**
- `processor/cardinality`: **93.3%** 
- `processor/failopen`: **93.9%**
- `control-plane/authz`: **100%** (4/4 tests)

---

## 📚 Documentation

Full technical documentation, implementation status, and build reports are tracked in [`app/DOCS/`](./app/DOCS/).

| Document | Description |
|---|---|
| [Implementation Status](./app/DOCS/Implementation_Status.md) | PRD section completion tracker |
| [Changelog](./app/DOCS/CHANGELOG.md) | Release history |
| [Build Issues](./app/DOCS/Build_Issue_Report.md) | Known issues & resolutions |

---

## 🗺 Roadmap

| Milestone | Status | Description |
|---|---|---|
| M1 — Core Detection (Alpha) | ✅ Complete | Processor, Circuit Breaker, HLL Cardinality, Orphan Detector |
| M2 — Control Plane (Beta) | ✅ Complete | Ingest Gateway, mTLS AuthZ, Stream Jobs, ClickHouse Schema |
| M3 — Remediation & Hardening (GA) | ✅ Complete | Remediation Generator, SigNoz Bridge, Helm Charts |
| M4 — Dashboard | ✅ Complete | React UI with Health Gauge, Metric Cards, YAML Viewer |
| M5 — Kafka Integration | ✅ Complete | Stream jobs wired to Kafka producer/worker sets in `cmd/ingest-gateway` and `cmd/worker` |
| M6 — ClickHouse Seeder | 🔜 Planned | Inject realistic historical telemetry data for demo |

---

## 🤝 Contributing

Contributions are welcome! Please follow the commit convention defined in [`AGENT_RULES.md`](./AGENT_RULES.md):

- `FEATURE:` — New functionality
- `BUG:` — Bug fixes
- `REFACTOR:` — Code improvements without behavior change
- `DOCS:` — Documentation updates
- `TEST:` — Test additions or changes

---

## 📄 License

This project is licensed under the **MIT License**.

---

<div align="center">

Built with ❤️ using Go, React, and OpenTelemetry

</div>

---

## 🤖 AI Assistant Usage

This project was built and debugged with assistance from:
- GitHub Copilot (code completion)
- ChatGPT (architecture review)
- Gemini (implementation assistance)
- Antigravity (Google DeepMind's agentic AI coding assistant for automated testing, debugging, and MCP server integration)
