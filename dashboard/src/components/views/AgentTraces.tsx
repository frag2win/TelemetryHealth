import { useState, useEffect } from 'react';
import { AlertCircle } from 'lucide-react';

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
  const [agents, setAgents] = useState<AgentTrace[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<boolean>(false);

  useEffect(() => {
    setLoading(true);
    setError(false);

    fetch(`http://localhost:8080/api/v1/tenant/${tenantId}/agents`)
      .then((r) => {
        if (!r.ok) throw new Error('Failed to load agent traces');
        return r.json();
      })
      .then(setAgents)
      .catch((err) => {
        console.error(err);
        setError(true);
        // Fallback mock traces
        setAgents([
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
        ]);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [tenantId]);

  return (
    <section className="view active">
      <div className="eyebrow">06 • ai agent tracing • gen-ai observability</div>
      
      <div className="grid4">
        <div className="panel metric">
          <div className="metric-label">Total LLM Calls</div>
          <div className="metric-val" style={{ color: 'var(--phosphor)' }}>1,432</div>
          <div className="metric-sub">Last 24 hours</div>
          <div className="metric-bar">
            <div style={{ width: '85%', background: 'var(--phosphor)' }}></div>
          </div>
        </div>
        <div className="panel metric">
          <div className="metric-label">Token Cost</div>
          <div className="metric-val" style={{ color: 'var(--amber)' }}>$14.21</div>
          <div className="metric-sub">Daily run rate</div>
          <div className="metric-bar">
            <div style={{ width: '45%', background: 'var(--amber)' }}></div>
          </div>
        </div>
        <div className="panel metric">
          <div className="metric-label">Avg Latency</div>
          <div className="metric-val" style={{ color: 'var(--paper)' }}>4.1s</div>
          <div className="metric-sub">Time to first token</div>
          <div className="metric-bar">
            <div style={{ width: '30%', background: 'var(--paper)' }}></div>
          </div>
        </div>
        <div className="panel metric">
          <div className="metric-label">Hallucinations</div>
          <div className="metric-val" style={{ color: 'var(--red)' }}>12</div>
          <div className="metric-sub">Flagged by guardrails</div>
          <div className="metric-bar">
            <div style={{ width: '12%', background: 'var(--red)' }}></div>
          </div>
        </div>
      </div>

      <h2 className="section-title">Agent Execution Traces</h2>

      {error && (
        <div style={{ background: 'rgba(239, 68, 68, 0.08)', padding: '10px 16px', borderRadius: '4px', border: '1px solid var(--red)', color: 'var(--red)', marginBottom: '14px', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <AlertCircle size={14} />
          <span>Error loading live agent traces. Showing local simulations.</span>
        </div>
      )}

      <div className="panel panel-tight">
        {loading ? (
          <div className="animate-pulse" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div style={{ height: '40px', background: 'var(--bezel-soft)', borderRadius: '4px' }}></div>
            <div style={{ height: '40px', background: 'var(--bezel-soft)', borderRadius: '4px' }}></div>
          </div>
        ) : agents.length > 0 ? (
          agents.map((trace) => (
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
                {trace.decisions?.map((d, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '13px' }}>
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
