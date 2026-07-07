

interface MetricProps {
  label: string;
  value: string | number;
  sub: string;
  percent: number;
  color: string;
}

function Metric({ label, value, sub, percent, color }: MetricProps) {
  return (
    <div className="panel metric">
      <div className="metric-label">{label}</div>
      <div className="metric-val" style={{ color: `var(--${color})` }}>{value}</div>
      <div className="metric-sub">{sub}</div>
      <div className="metric-bar"><div style={{ width: `${percent}%`, background: `var(--${color})` }}></div></div>
    </div>
  );
}

export function Overview({ data, setView }: { data: any, setView: (v: string) => void }) {
  if (!data) return null;
  const score = data.healthScore || 78;
  const bandClass = score > 90 ? 'band-healthy' : score > 50 ? 'band-degraded' : 'band-critical';
  const bandText = score > 90 ? 'healthy' : score > 50 ? 'degraded' : 'critical';

  return (
    <section className="view active">
      <div className="panel hero">
        <div className="hero-score">
          <div className="score-num" id="score-num">{score}</div>
          <span className={`score-band ${bandClass}`} id="score-band">{bandText}</span>
          <div className="score-delta">&#8599; +4 vs last week</div>
          <div className="score-caption">composite &#183; org rollup</div>
        </div>
        <div className="hero-chart">
          <div className="scanline"></div>
          <svg viewBox="0 0 640 190" style={{ width: '100%', height: '190px', display: 'block' }}>
            <line x1="0" y1="42" x2="640" y2="42" stroke="#5CE1A5" strokeWidth="1" strokeDasharray="4 4" opacity="0.45"/>
            <text x="6" y="36" className="trace-text dim">80 &#183; healthy</text>
            <line x1="0" y1="96" x2="640" y2="96" stroke="#E5484D" strokeWidth="1" strokeDasharray="4 4" opacity="0.45"/>
            <text x="6" y="90" className="trace-text dim">50 &#183; critical</text>
            <path d="M0,34 L91,38 L182,50 L273,60 L365,67 L456,58 L548,52 L640,43" fill="none" stroke="#5CE1A5" strokeWidth="2"/>
            <circle cx="640" cy="43" r="4" fill="#F5A623"/>
            <text x="0" y="182" className="trace-text dim">7d ago</text>
            <text x="602" y="182" className="trace-text dim">now</text>
          </svg>
        </div>
      </div>

      <div className="grid4">
        <Metric label="Cardinality alerts" value={data.metrics?.cardinality?.value || 3} sub="1 key-space anomaly" percent={70} color="red" />
        <Metric label="Orphan rate" value={data.metrics?.orphans?.value || '6.2%'} sub="above 5% threshold" percent={62} color="amber" />
        <Metric label="Coverage gaps" value={data.metrics?.coverage?.value || 1} sub="service silent 14m" percent={20} color="amber" />
        <Metric label="Est. cost impact" value="$4.1k/mo" sub="from cardinality alone" percent={44} color="phosphor" />
      </div>

      <h2 className="section-title">Active issues</h2>
      <div className="panel panel-tight">
        <div className="rack-row">
          <span className="rled r"></span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div className="rack-svc">payments-api</div>
            <div className="rack-desc">Broken trace chain &#183; 18% orphan rate &#183; §8.2</div>
          </div>
          <div className="rack-impact">&minus;18</div>
          <button className="btn" onClick={() => setView('remediation')}>remediate &#9656;</button>
        </div>
        <div className="rack-row">
          <span className="rled a"></span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div className="rack-svc">checkout-service</div>
            <div className="rack-desc">Cardinality spike &#183; user_id_raw &#183; §8.1</div>
          </div>
          <div className="rack-impact">&minus;12</div>
          <button className="btn" onClick={() => setView('remediation')}>remediate &#9656;</button>
        </div>
        <div className="rack-row">
          <span className="rled a"></span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div className="rack-svc">inventory-worker</div>
            <div className="rack-desc">Coverage gap &#183; silent 14m &#183; §8.3</div>
          </div>
          <div className="rack-impact">&minus;8</div>
          <button className="btn" onClick={() => setView('remediation')}>remediate &#9656;</button>
        </div>
      </div>
    </section>
  );
}
