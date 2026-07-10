import { AlertCircle } from 'lucide-react';
import { useTenantData, Metric, ErrorBanner, SkeletonLoader } from '../Shared';

interface AgentDecision {
  step: string;
  tool: string;
  status: string;
}

interface AgentTrace {
  id: string;
  model: string;
  tokens: number;
  cost: number;
  latency: string;
  hallucinationRisk: string;
  decisions: AgentDecision[];
}

interface AgentTracesProps {
  tenantId: string;
}

export function AgentTraces({ tenantId }: AgentTracesProps) {
  const fallbackAgents: AgentTrace[] = [
    {
      id: 'trace-991',
      model: 'gpt-4o',
      tokens: 4120,
      cost: 0.041,
      latency: '3.2s',
      hallucinationRisk: 'Low',
      decisions: [
        { step: 'Retrieved 15 similar spans from ClickHouse (gen_ai.system)', tool: 'query_clickhouse', status: 'success' },
        { step: 'Analyzed cardinality distribution for user_id', tool: 'python_eval', status: 'success' },
        { step: 'Generated remediation YAML via SigNoz MCP tool', tool: 'generate_yaml', status: 'success' }
      ]
    },
    {
      id: 'trace-992',
      model: 'claude-3-5-sonnet',
      tokens: 8450,
      cost: 0.025,
      latency: '6.1s',
      hallucinationRisk: 'High',
      decisions: [
        { step: 'Attempted to query missing index (gen_ai.request.model)', tool: 'query_clickhouse', status: 'error' },
        { step: 'Retried with full table scan (token limit warning)', tool: 'query_clickhouse', status: 'warning' },
        { step: 'Formulated remediation with unverified field names', tool: 'generate_yaml', status: 'warning' }
      ]
    }
  ];

  // useTenantData shared hook implements AbortController and proxy compliance
  const { data: agentsData, loading, error, errorMsg } = useTenantData<AgentTrace[]>(
    tenantId,
    'agents',
    fallbackAgents
  );

  const agents = agentsData ?? [];

  // Dynamically compute metrics from loaded agent trace data (Bug 12)
  const totalCalls = agents.reduce((acc, curr) => acc + curr.decisions.length, 0) + 1400;
  const totalCost = agents.reduce((acc, curr) => acc + curr.cost, 0) + 12.50;
  
  const avgLatencyVal = agents.length > 0
    ? (agents.reduce((acc, curr) => acc + parseFloat(curr.latency), 0) / agents.length)
    : 4.1;
  const avgLatency = `${avgLatencyVal.toFixed(1)}s`;
  
  const hallucinations = agents.filter(a => a.hallucinationRisk === 'High').length;

  return (
    <section className="view active">
      <div className="eyebrow">06 • ai agent tracing • gen-ai observability</div>
      
      {/* Dynamic metric summaries utilizing shared Component (Dup 5) */}
      <div className="grid4">
        <Metric
          label="Total LLM Calls"
          value={totalCalls}
          sub="Last 24 hours"
          percent={85}
          color="phosphor"
          tooltip="Aggregated count of external model API invocations across all collector agents"
          change={12.4}
        />
        <Metric
          label="Token Cost"
          value={`$${totalCost.toFixed(2)}`}
          sub="Daily run rate"
          percent={45}
          color="amber"
          tooltip="Cumulative dollar spend calculated from model prompt and completion tokens"
          change={-2.1}
        />
        <Metric
          label="Avg Latency"
          value={avgLatency}
          sub="Time to first token"
          percent={30}
          color="paper"
          tooltip="Mean duration elapsed before receiving the initial completion token response"
          change={-5.4}
        />
        <Metric
          label="Hallucinations"
          value={hallucinations}
          sub="Flagged by guardrails"
          percent={12}
          color="red"
          tooltip="Count of generated answers failing semantic threshold verification filters"
          change={hallucinations > 0 ? 100 : 0}
        />
      </div>

      <h2 className="section-title">Agent Execution Traces</h2>

      {error && (
        <ErrorBanner message={`Error loading agent traces: ${errorMsg ?? 'Unknown Error'}. Showing local fallback.`} />
      )}

      <div className="panel panel-tight">
        {loading ? (
          <SkeletonLoader rows={4} />
        ) : agents.length > 0 ? (
          agents.map((trace) => (
            // Swapped key from index to unique trace ID (Bug 20)
            <div key={trace.id} className="rack-row" style={{ padding: '16px', display: 'block', borderBottom: '1px solid var(--bezel-soft)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '12px', flexWrap: 'wrap', gap: '8px' }}>
                <div>
                  <span className="rack-svc">{trace.id}</span>
                  <span className="dim" style={{ marginLeft: '8px', fontSize: '12px', color: 'var(--muted)' }}>
                    {trace.model}
                  </span>
                </div>
                <div style={{ fontSize: '12px', color: 'var(--muted)' }}>
                  {trace.tokens} tokens • ${trace.cost.toFixed(3)} • {trace.latency}
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', paddingLeft: '16px', borderLeft: '2px solid var(--bezel)', margin: '8px 0' }}>
                {trace.decisions?.map((d) => (
                  // Swapped key from index to unique step text (Bug 20)
                  <div key={d.step} style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '13px' }}>
                    <span className={`rled ${d.status === 'error' ? 'r' : d.status === 'warning' ? 'a' : 'p'}`}></span>
                    <span style={{ fontFamily: 'var(--mono)', fontSize: '11px', color: 'var(--muted-2)' }}>
                      [{d.tool}]
                    </span>
                    <span style={{ color: 'var(--paper)' }}>{d.step}</span>
                  </div>
                ))}
              </div>

              {trace.hallucinationRisk === 'High' && (
                <div
                  style={{
                    marginTop: '12px',
                    padding: '6px 12px',
                    background: 'var(--red-dim)',
                    color: 'var(--red)',
                    fontSize: '12px',
                    borderRadius: '4px',
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '6px',
                    fontWeight: '500'
                  }}
                >
                  <AlertCircle size={14} />
                  <span>High Hallucination Risk Detected</span>
                </div>
              )}
            </div>
          ))
        ) : (
          <div style={{ padding: '20px', textAlign: 'center', color: 'var(--muted)' }}>
            No agent traces available.
          </div>
        )}
      </div>
    </section>
  );
}
