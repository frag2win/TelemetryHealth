# AI Agent Demo

This is a simple demo of an AI agent instrumented with OpenTelemetry. It generates traces that simulate common AI observability issues which TelemetryHealth is designed to catch.

## Intentional Issues Generated

1. **Tool Call Failures:** Randomly injects `llm.tool_call.error` attributes into spans, marking them as errors.
2. **Cardinality Explosions:** Generates unique attribute keys per run (`llm.prompt.raw_{uuid}`), which can cause high cardinality issues in backend metrics.
3. **Broken Trace Chains:** Randomly generates orphan spans (spans without a parent) with high token usage.
4. **Token Burn:** Simulates varying `llm.token_usage` for health metrics.

## Running the Demo

1. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

2. Run the agent (ensure your OpenTelemetry collector is listening on `localhost:4317`):
   ```bash
   python agent.py
   ```
