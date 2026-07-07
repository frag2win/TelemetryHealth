
const mockTraces = [
  {
    id: "trace-991",
    model: "gpt-4o",
    tokens: 4120,
    cost: 0.041,
    latency: "3.2s",
    hallucinationRisk: "Low",
    decisions: [
      { step: "Retrieved 15 similar spans from ClickHouse", tool: "query_clickhouse", status: "success" },
      { step: "Analyzed cardinality distribution for user_id", tool: "python_eval", status: "success" },
      { step: "Generated remediation YAML", tool: "generate_yaml", status: "success" }
    ]
  },
  {
    id: "trace-992",
    model: "claude-3-5-sonnet",
    tokens: 8450,
    cost: 0.025,
    latency: "6.1s",
    hallucinationRisk: "High",
    decisions: [
      { step: "Attempted to query missing index", tool: "query_clickhouse", status: "error" },
      { step: "Retried with full table scan (token limit warning)", tool: "query_clickhouse", status: "warning" },
      { step: "Formulated remediation with unverified field names", tool: "generate_yaml", status: "warning" }
    ]
  }
];

export function AgentTraces() {
  return (
    <section className="view active">
      <div className="grid4">
        <div className="panel metric">
          <div className="metric-label">Total LLM Calls</div>
          <div className="metric-val" style={{ color: 'var(--phosphor)' }}>1,432</div>
          <div className="metric-sub">Last 24 hours</div>
          <div className="metric-bar"><div style={{ width: '85%', background: 'var(--phosphor)' }}></div></div>
        </div>
        <div className="panel metric">
          <div className="metric-label">Token Cost</div>
          <div className="metric-val" style={{ color: 'var(--amber)' }}>$14.21</div>
          <div className="metric-sub">Daily run rate</div>
          <div className="metric-bar"><div style={{ width: '45%', background: 'var(--amber)' }}></div></div>
        </div>
        <div className="panel metric">
          <div className="metric-label">Avg Latency</div>
          <div className="metric-val" style={{ color: 'var(--paper)' }}>4.1s</div>
          <div className="metric-sub">Time to first token</div>
          <div className="metric-bar"><div style={{ width: '30%', background: 'var(--paper)' }}></div></div>
        </div>
        <div className="panel metric">
          <div className="metric-label">Hallucinations</div>
          <div className="metric-val" style={{ color: 'var(--red)' }}>12</div>
          <div className="metric-sub">Flagged by guardrails</div>
          <div className="metric-bar"><div style={{ width: '12%', background: 'var(--red)' }}></div></div>
        </div>
      </div>

      <h2 className="section-title">Agent Execution Traces</h2>
      <div className="panel panel-tight">
        {mockTraces.map(trace => (
          <div key={trace.id} className="rack-row" style={{ padding: '16px', display: 'block' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '12px' }}>
              <div>
                <span className="rack-svc">{trace.id}</span>
                <span className="dim" style={{ marginLeft: '8px', fontSize: '12px' }}>{trace.model}</span>
              </div>
              <div style={{ fontSize: '13px', color: 'var(--muted)' }}>
                {trace.tokens} tokens • ${trace.cost.toFixed(3)} • {trace.latency}
              </div>
            </div>
            
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', paddingLeft: '16px', borderLeft: '2px solid var(--bezel)' }}>
              {trace.decisions.map((d, i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <span className={`rled ${d.status === 'error' ? 'r' : d.status === 'warning' ? 'a' : 'p'}`}></span>
                  <span style={{ fontFamily: 'var(--mono)', fontSize: '11px', color: 'var(--muted-2)' }}>[{d.tool}]</span>
                  <span style={{ fontSize: '13px' }}>{d.step}</span>
                </div>
              ))}
            </div>
            
            {trace.hallucinationRisk === 'High' && (
              <div style={{ marginTop: '12px', padding: '8px', background: 'var(--red-dim)', color: 'var(--red)', fontSize: '12px', borderRadius: '4px', display: 'inline-block' }}>
                ⚠️ High Hallucination Risk Detected
              </div>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}
