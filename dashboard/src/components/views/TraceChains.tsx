

export function TraceChains({ data }: { data?: any }) {
  const orphanRate = data?.metrics?.orphans?.value || '6.2%';
  return (
    <section className="view active">
      <div className="eyebrow">03 &#183; broken trace-chain detector &#183; §8.2</div>

      <div className="tag-row">
        <span className="tag">orphan rate <b style={{ color: 'var(--amber)' }}>{orphanRate}</b></span>
        <span className="tag">threshold <b>5%</b></span>
        <span className="tag">correlation window <b>30s</b></span>
        <span className="tag">clock skew tolerance <b>5s</b></span>
      </div>

      <div className="grid2">
        <div className="panel">
          <div className="metric-label" style={{ marginBottom: '14px' }}>trace 9f3a2c &#183; payments-api</div>
          <svg viewBox="0 0 460 200" style={{ width: '100%', height: '200px' }}>
            <rect className="trace-box" x="10" y="14" width="120" height="30" rx="3"/>
            <text x="20" y="33" className="trace-text">gateway</text>
            <line className="trace-line" x1="70" y1="44" x2="70" y2="74"/>
            <rect className="trace-box" x="10" y="74" width="120" height="30" rx="3"/>
            <text x="20" y="93" className="trace-text">auth</text>
            <line className="trace-line" x1="70" y1="104" x2="70" y2="134"/>
            <rect className="trace-box" x="10" y="134" width="120" height="30" rx="3"/>
            <text x="20" y="153" className="trace-text">checkout</text>

            <line className="trace-line broken" x1="130" y1="149" x2="270" y2="149"/>
            <rect className="trace-box orphan" x="280" y="134" width="170" height="30" rx="3"/>
            <text x="290" y="153" className="trace-text" style={{ fill: '#E5484D' }}>payment-capture &#8212; orphan</text>
            <text x="280" y="122" className="trace-text dim">missing parent_span_id: 7bd1</text>
          </svg>
        </div>

        <div className="panel panel-tight">
          <div className="metric-label" style={{ padding: '12px 6px 4px' }}>recent orphan events</div>
          <div className="rack-row">
            <span className="rled r"></span>
            <div style={{ flex: 1 }}>
              <div className="rack-svc" style={{ fontSize: '12px' }}>span 4a91 &#183; collector-07</div>
              <div className="rack-desc">parent 7bd1 not found &#183; correlated after 31s</div>
            </div>
          </div>
          <div className="rack-row">
            <span className="rled r"></span>
            <div style={{ flex: 1 }}>
              <div className="rack-svc" style={{ fontSize: '12px' }}>span 2e6f &#183; collector-03</div>
              <div className="rack-desc">parent c910 not found &#183; correlated after 12s</div>
            </div>
          </div>
          <div className="rack-row">
            <span className="rled a"></span>
            <div style={{ flex: 1 }}>
              <div className="rack-svc" style={{ fontSize: '12px' }}>span 88bd &#183; collector-07</div>
              <div className="rack-desc">late arrival &#183; resolved within window</div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
