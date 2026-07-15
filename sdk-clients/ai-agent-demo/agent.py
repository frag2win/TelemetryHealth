import time
import random
import uuid
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor, ConsoleSpanExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource

# Setup OpenTelemetry
resource = Resource(attributes={"service.name": "ai-agent"})
provider = TracerProvider(resource=resource)
# Export to standard OTLP endpoint (e.g., SigNoz or local collector)
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4317", insecure=True)
provider.add_span_processor(BatchSpanProcessor(otlp_exporter))
# Also export to console for visibility during demo
provider.add_span_processor(BatchSpanProcessor(ConsoleSpanExporter()))
trace.set_tracer_provider(provider)

tracer = trace.get_tracer(__name__)

def perform_research(topic: str):
    with tracer.start_as_current_span("agent.research") as span:
        span.set_attribute("llm.tool_name", "web_search")
        span.set_attribute("llm.token_usage", random.randint(50, 200))
        
        # Simulate tool failure (Intentional issue for demo)
        if random.random() < 0.3:
            span.set_attribute("llm.tool_call.error", "TimeoutError: connection refused")
            span.set_status(trace.StatusCode.ERROR, "Tool execution failed")
            print(f"[x] Research tool failed on topic: {topic}")
            return None
            
        print(f"[*] Researched topic: {topic}")
        time.sleep(random.uniform(0.5, 1.5))
        return f"Information about {topic}"

def summarize(content: str):
    with tracer.start_as_current_span("agent.summarize") as span:
        span.set_attribute("llm.model", "gpt-4o")
        span.set_attribute("llm.token_usage", random.randint(500, 1500))
        
        # Intentional issue: High cardinality attribute
        # Using a raw query or timestamp as an attribute key/value creates cardinality explosion
        span.set_attribute(f"llm.prompt.raw_{uuid.uuid4().hex[:8]}", content)
        
        print("[*] Summarized content")
        time.sleep(random.uniform(1.0, 2.0))
        return "Summary complete."

def run_agent_workflow(topic: str):
    print(f"--- Starting Agent Workflow for: {topic} ---")
    with tracer.start_as_current_span("agent.workflow") as span:
        span.set_attribute("workflow.topic", topic)
        
        research_result = perform_research(topic)
        
        if research_result:
            summarize(research_result)
            
        # Simulate broken trace chain (Intentional issue for demo)
        if random.random() < 0.2:
            print("[!] Simulating broken trace chain: generating orphan span")
            orphan_span = tracer.start_span("agent.orphan_process")
            orphan_span.set_attribute("llm.token_usage", 5000) # High token burn
            orphan_span.end()
            
    print("--- Workflow complete ---\n")

if __name__ == "__main__":
    topics = [
        "Observability best practices",
        "OpenTelemetry auto-instrumentation",
        "AI Agent prompt engineering",
        "Vector databases",
        "LLM cost optimization"
    ]
    
    print("Starting AI Agent Demo...")
    try:
        for i in range(10):
            run_agent_workflow(random.choice(topics))
            time.sleep(2)
    except KeyboardInterrupt:
        print("Stopping...")
    finally:
        # Flush traces before exit
        provider.shutdown()
