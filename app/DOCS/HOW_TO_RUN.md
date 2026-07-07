# 🚀 TelemetryHealth — How to Run the Project

This guide walks you through setting up and running the complete TelemetryHealth stack on your local machine from scratch.

---

## 📦 Required Dependencies

Install all of the following before proceeding.

### 1. Go (≥ 1.22)
Used for all backend services (Control Plane, Processor, Stream Workers).

| OS | Download |
|---|---|
| Windows | https://go.dev/dl/ → download `.msi` installer |
| macOS | `brew install go` |
| Linux | `sudo apt install golang-go` or https://go.dev/dl/ |

Verify:
```bash
go version
# Expected: go version go1.22.x ...
```

---

### 2. Node.js (≥ 20 LTS)
Used to run the React Dashboard frontend.

| OS | Download |
|---|---|
| All | https://nodejs.org/en/download → LTS version |
| Windows | Download `.msi` installer |
| macOS | `brew install node` |

Verify:
```bash
node --version   # Expected: v20.x.x or higher
npm --version    # Expected: 10.x.x or higher
```

---

### 3. Docker Desktop
Used to run **ClickHouse** (database) and **Redpanda** (Kafka-compatible message broker) locally without manual installation.

| OS | Download |
|---|---|
| Windows | https://www.docker.com/products/docker-desktop/ |
| macOS | https://www.docker.com/products/docker-desktop/ |
| Linux | https://docs.docker.com/engine/install/ |

> ⚠️ Make sure Docker Desktop is **started and running** before proceeding to the next steps.

Verify:
```bash
docker --version   # Expected: Docker version 25.x.x or higher
docker ps          # Should return an empty table (no error)
```

---

### 4. Git
Used to clone the repository.

| OS | Download |
|---|---|
| Windows | https://git-scm.com/download/win |
| macOS | `brew install git` |
| Linux | `sudo apt install git` |

Verify:
```bash
git --version   # Expected: git version 2.x.x
```

---

## 🛠 One-Time Setup

Run these steps **only the first time** you set up the project.

### Step 1 — Clone the Repository
```bash
git clone https://github.com/frag2win/TelemetryHealth.git
cd TelemetryHealth
```

---

### Step 2 — Start ClickHouse (Database)
```bash
docker run -d --name clickhouse \
  -p 9000:9000 -p 8123:8123 \
  clickhouse/clickhouse-server:latest
```

> **Windows PowerShell** — use backtick instead of backslash:
> ```powershell
> docker run -d --name clickhouse `
>   -p 9000:9000 -p 8123:8123 `
>   clickhouse/clickhouse-server:latest
> ```

Wait ~5 seconds for it to boot, then create the database and user:

```bash
# Create the database
docker exec clickhouse clickhouse-client --query "CREATE DATABASE IF NOT EXISTS telemetry_health"

# Create a dedicated app user (no password, accessible from host)
docker exec clickhouse clickhouse-client -q "CREATE USER IF NOT EXISTS telemetry IDENTIFIED WITH no_password HOST ANY"
docker exec clickhouse clickhouse-client -q "GRANT ALL ON telemetry_health.* TO telemetry"
```

Create the three telemetry tables:

```bash
docker exec clickhouse clickhouse-client --query "
CREATE TABLE IF NOT EXISTS telemetry_health.cardinality_signal (
    tenant_id UUID,
    service LowCardinality(String),
    attribute_key LowCardinality(String),
    window_start DateTime64(3),
    unique_estimate UInt64
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(window_start)
ORDER BY (tenant_id, service, attribute_key, window_start)
TTL window_start + INTERVAL 30 DAY"

docker exec clickhouse clickhouse-client --query "
CREATE TABLE IF NOT EXISTS telemetry_health.orphan_signal (
    tenant_id UUID,
    trace_id String,
    span_id String,
    parent_span_id String,
    collector_id LowCardinality(String),
    detected_at DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(detected_at)
ORDER BY (tenant_id, detected_at)
TTL detected_at + INTERVAL 30 DAY"

docker exec clickhouse clickhouse-client --query "
CREATE TABLE IF NOT EXISTS telemetry_health.coverage_signal (
    tenant_id UUID,
    service LowCardinality(String),
    last_seen_at DateTime64(3),
    baseline_expected UInt8
) ENGINE = ReplacingMergeTree(last_seen_at)
ORDER BY (tenant_id, service)"
```

---

### Step 3 — Start Redpanda (Kafka Message Broker)
```bash
docker run -d --name redpanda \
  -p 9092:9092 -p 19092:19092 \
  redpandadata/redpanda:latest \
  redpanda start \
  --overprovisioned \
  --smp 1 --memory 512M \
  --reserve-memory 0M \
  --node-id 0 \
  --check=false \
  --kafka-addr PLAINTEXT://0.0.0.0:9092 \
  --advertise-kafka-addr PLAINTEXT://localhost:9092
```

> **Windows PowerShell** — use backtick line continuation:
> ```powershell
> docker run -d --name redpanda `
>   -p 9092:9092 -p 19092:19092 `
>   redpandadata/redpanda:latest `
>   redpanda start `
>   --overprovisioned `
>   --smp 1 --memory 512M `
>   --reserve-memory 0M `
>   --node-id 0 `
>   --check=false `
>   --kafka-addr PLAINTEXT://0.0.0.0:9092 `
>   --advertise-kafka-addr PLAINTEXT://localhost:9092
> ```

---

### Step 4 — Install Go Dependencies
```bash
cd control-plane
go mod tidy
cd ..
```

---

### Step 5 — Install Dashboard Dependencies
```bash
cd dashboard
npm install
cd ..
```

---

### Step 6 — Seed the Database with Demo Data
This injects realistic telemetry signals so the dashboard has something to display.

```bash
cd control-plane
go run ./cmd/seeder
cd ..
```

Expected output:
```
✓ cardinality_signal seeded
✓ orphan_signal seeded
✓ coverage_signal seeded
Seeding complete for tenant 00000000-0000-0000-0000-000000000001
```

> ⏰ **Note:** The seeded data expires after **30 minutes** (TTL window). Re-run the seeder anytime to refresh it.

---

## ▶️ Running the Project (Every Time)

Open **4 terminal windows** and run one command in each.

### Terminal 1 — Go REST API Server (Port 8080)
```bash
cd control-plane
go run ./cmd/api-server
```

Expected log:
```
{"msg":"ClickHouse connected — using real data"}
{"msg":"Starting API Server","addr":":8080"}
```

---

### Terminal 2 — Stream Worker (Kafka → ClickHouse)
```bash
cd control-plane
go run ./cmd/worker
```

Expected log:
```
{"msg":"Kafka topics ensured","topics":["telemetry.cardinality","telemetry.orphan","telemetry.coverage"]}
{"msg":"Stream worker started — consuming from Kafka, writing to ClickHouse"}
```

---

### Terminal 3 — Ingest Gateway (gRPC OTLP receiver, Port 4317)
```bash
cd control-plane
go run ./cmd/ingest-gateway
```

Expected log:
```
{"msg":"Kafka topics ensured"}
{"msg":"Ingest Gateway started on :4317"}
```

---

### Terminal 4 — React Dashboard (Port 5173)
```bash
cd dashboard
npm run dev
```

Expected output:
```
VITE v8.1.3  ready in 763 ms
  ➜  Local:   http://localhost:5173/
```

---

## 🌐 Open the Dashboard

Once all four terminals are running, open your browser:

**➜ [http://localhost:5173](http://localhost:5173)**

You should see:
- A **Health Score gauge** (should display ~42 with seeded data)
- **Metric cards** showing Cardinality, Orphaned Traces, and Active Services
- A **Remediation Panel** with an auto-generated OTel YAML fix

---

## 🔁 Restarting After a Machine Reboot

Docker containers stop when your machine restarts. Run this to bring everything back:

```bash
docker start clickhouse
docker start redpanda
```

Then open your 4 terminals and run the startup commands above.

---

## 🧪 Running Tests

### Processor (OTel Collector Plugin)
```bash
cd processor
go test ./... -v -cover
```

### Control Plane
```bash
cd control-plane
go test ./... -v
```

---

## 📊 Port Reference

| Port | Service |
|---|---|
| `5173` | React Dashboard (Vite) |
| `8080` | Go REST API |
| `4317` | OTLP gRPC Ingest Gateway |
| `9000` | ClickHouse (native protocol) |
| `8123` | ClickHouse (HTTP) |
| `9092` | Redpanda / Kafka |

---

## 🐛 Troubleshooting

### API shows mock data (health score: 84) instead of real score
> ClickHouse is either not running or the connection failed.

**Fix:**
```bash
docker start clickhouse
# Then restart the API server terminal
```

### Dashboard shows loading spinner forever
> The Go API server is not running or the seeded data expired.

**Fix:**
```bash
# Terminal 1: start API
cd control-plane && go run ./cmd/api-server

# Terminal (any): refresh data
cd control-plane && go run ./cmd/seeder
```

### `No such container: clickhouse` or `redpanda`
> Containers were never created. Run the full **One-Time Setup** steps above.

### Port already in use
```bash
# Windows PowerShell — find and kill process on a port (e.g. 8080)
Get-Process -Id (Get-NetTCPConnection -LocalPort 8080).OwningProcess | Stop-Process -Force
```

---

*Document maintained by the TelemetryHealth AI build agent. Last updated: 2026-07-07.*
