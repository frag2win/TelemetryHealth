# 🎬 TelemetryHealth Demo & Recording Guide

This guide is for the teammate recording the hackathon demo video. It provides the core context of what we built, why it's impressive, and a step-by-step script to guarantee a flawless recording.

---

## 🧠 Context: What Did We Build?
**Problem:** Companies waste millions of dollars storing useless telemetry data (high cardinality metrics, broken/orphaned traces, and silent services).
**Solution:** **TelemetryHealth** is a control plane that analyzes incoming OpenTelemetry traffic in real-time, scores your telemetry health, and **auto-generates the exact OTel Collector YAML configuration** needed to fix the waste at the edge.

**Why it's impressive for the hackathon:**
We aren't faking the backend. This is a **real distributed architecture** running locally:
1. **Ingest Gateway** (receives OTLP traces)
2. **Redpanda/Kafka** (message broker)
3. **Stream Worker** (processes signals)
4. **ClickHouse** (stores analytics)
5. **React Dashboard** (live visualizations)

---

## 🚀 Step 1: Pre-Recording Setup
Before hitting "Record", ensure the environment is fully running on your local machine.

1. Ensure Docker Desktop is open and running.
2. Open **4 separate terminal windows** at the project root (`TelemetryHealth_`):
   - **Terminal 1:** `cd control-plane && go run ./cmd/api-server`
   - **Terminal 2:** `cd control-plane && go run ./cmd/worker`
   - **Terminal 3:** `cd control-plane && go run ./cmd/ingest-gateway`
   - **Terminal 4:** `cd dashboard && npm run dev`
3. Open your browser and navigate to **http://localhost:5173/**. Ensure you see the `🟢 Live` indicator at the top right.

---

## 🎥 Step 2: The Demo Script (What to say and do)

### Scene 1: The Problem & The Dashboard
* **Action:** Start recording on the main Dashboard view.
* **Talk Track:** 
  > "Welcome to TelemetryHealth. Modern observability is expensive because we often ingest garbage data—exploding cardinality, broken traces, and silent services. Our platform acts as a health monitor for your telemetry pipeline. As you can see on our dashboard, we are currently connected to our live ClickHouse database tracking the `acme-prod` environment."
* **Action:** Hover over the Health Score and the three metric cards (Cardinality, Orphans, Coverage).
* **Talk Track:** 
  > "We calculate a live composite Health Score. Right now, it's detecting a massive cardinality spike on the `user_id_raw` attribute, which is going to cost us a fortune in Datadog or New Relic."

### Scene 2: Auto-Remediation (The "Aha!" Moment)
* **Action:** Scroll down or click to the **Remediation Panel** on the right side.
* **Talk Track:**
  > "Instead of just alerting us, TelemetryHealth automatically generates the exact OpenTelemetry Collector YAML configuration needed to drop this high-cardinality attribute at the edge, before it ever leaves our infrastructure."
* **Action:** Click the **"Apply Patch"** button.
* **Talk Track:**
  > "With one click, we apply this mutation directly to our collector fleet. The system logs this via a SOC-2 compliant audit trail."

### Scene 3: Proving it with LIVE Data
* **Action:** Bring up a terminal window on screen. 
* **Talk Track:**
  > "To prove this isn't just mock data, we built a fully distributed streaming pipeline. I'm going to run the official OpenTelemetry load generator to blast real OTLP traces into our Ingest Gateway at 50 traces per second."
* **Action:** Run the following command in the terminal:
  `docker compose -f test/load/docker-compose.yaml up`
* **Talk Track:** 
  > "As you can see, the traces are hitting our gRPC gateway, flowing through Kafka, being aggregated by our stream worker, and updating our ClickHouse database in real time. The dashboard instantly reflects the live infrastructure."

---

---

## 🔌 Offline Mock-Mode Verification (For Judges/Reviewers)

To make it as simple as possible for judges to verify and test the TelemetryHealth platform without setting up Docker or a ClickHouse database, you can run the entire workspace in **Offline Mock-Mode**. This uses our safe, fallback mock repository architecture.

### How to run the MCP Server in Mock-Mode:
1. Open a terminal and navigate to `control-plane`:
   ```bash
   cd control-plane
   ```
2. Start the MCP server. If ClickHouse is not running locally, the server will automatically detect it, log a warning, and fall back to the safe, in-memory mock repository:
   ```bash
   go run ./cmd/mcp-server
   ```
   *To run the MCP server in `stdio` mode for direct connection (e.g. Claude Desktop), add the `--stdio` flag:*
   ```bash
   go run ./cmd/mcp-server --stdio
   ```

### Mock-Mode Behavior:
- **`get_telemetry_health` Tool Call**: When ClickHouse is offline, invoking the `get_telemetry_health` tool through the MCP client returns pre-seeded, zero-panic fallback mock metrics safely (health score `100`, cardinality `0`, orphans `0`, active services `0`) instead of panicking.
- **`generate_remediation` Tool Call**: Generates Collector remediation snippets using the live rule validation engine.

---

## 💡 Pro-Tips for the Video
- **Keep it under 3 minutes.** Judges have a short attention span.
- **Hide your bookmarks bar** and put the browser in fullscreen mode for a cleaner look.
- **If data expires:** The seeded data in the database expires after 30 minutes. If the dashboard metrics drop to zero before you record, simply run `cd control-plane && go run ./cmd/seeder` to instantly replenish the database!
