
const AnimatedHealthGauge = ({ score }: { score: number }) => {
  const color = score > 80 ? 'var(--phosphor)' : score > 50 ? 'var(--amber)' : 'var(--red)';
  return (
    <div className="health-gauge animate-pulse" style={{ display: 'flex', justifyContent: 'center' }}>
      <svg width="140" height="140" viewBox="0 0 200 200">
        <circle cx="100" cy="100" r="80" stroke="var(--panel-2)" strokeWidth="15" fill="none" />
        <circle 
          cx="100" cy="100" r="80" 
          stroke={color} strokeWidth="15" fill="none" 
          strokeDasharray="502" 
          strokeDashoffset={502 - (502 * score) / 100}
          strokeLinecap="round"
          transform="rotate(-90 100 100)"
          className="transition-colors"
        />
        <text x="100" y="115" textAnchor="middle" fontSize="48" fontWeight="600" fill={color} className="transition-colors">
          {score}
        </text>
      </svg>
    </div>
  );
};

interface MetricProps {
  label: string;
  value: string | number;
  sub: string;
  percent: number;
  color: string;
  tooltip: string;
}

function Metric({ label, value, sub, percent, color, tooltip }: MetricProps) {
  return (
    <div className="panel metric" title={tooltip} style={{ cursor: 'help' }}>
      <div className="metric-label">{label}</div>
      <div className="metric-val" style={{ color: `var(--${color})` }}>{value}</div>
      <div className="metric-sub">{sub}</div>
      <div className="metric-bar"><div style={{ width: `${percent}%`, background: `var(--${color})` }}></div></div>
    </div>
  );
}

import { useEffect, useState } from 'react';

export function Overview({ data, setView }: { data: any, setView: (v: string) => void }) {
  const [issues, setIssues] = useState<any[]>([]);

  useEffect(() => {
    fetch('/api/v1/tenant/acme-prod/issues')
      .then(r => r.json())
      .then(setIssues)
      .catch(console.error);
  }, []);
  if (!data) return null;
  const score = data.healthScore || 78;
  const bandClass = score > 90 ? 'band-healthy' : score > 50 ? 'band-degraded' : 'band-critical';
  const bandText = score > 90 ? 'healthy' : score > 50 ? 'degraded' : 'critical';

  const calcY = (p: number) => Math.round(150 - (p / 100) * 140);
  const createPath = (points: number[]) => {
    if (points.length === 0) return '';
    const step = 640 / (points.length - 1);
    return points.map((p, i) => `${i === 0 ? 'M' : 'L'}${Math.round(i * step)},${calcY(p)}`).join(' ');
  };
  const history = data.history || [Math.max(0, score-12), Math.max(0, score-5), score-8, score-2, score-10, score-3, score, score];

  return (
    <section className="view active">
      <div className="panel hero" style={{ display: 'flex', alignItems: 'center' }}>
        <div className="hero-score" style={{ flex: 1 }}>
          <AnimatedHealthGauge score={score} />
          <div style={{ textAlign: 'center', marginTop: '16px' }}>
            <span className={`score-band ${bandClass}`} id="score-band">{bandText}</span>
            <div className="score-delta">&#8599; +4 vs last week</div>
            <div className="score-caption">composite &#183; org rollup</div>
          </div>
        </div>
        <div className="hero-chart" style={{ flex: 2 }}>
          <div className="scanline"></div>
          <svg viewBox="0 0 640 190" style={{ width: '100%', height: '190px', display: 'block' }}>
            <line x1="0" y1={calcY(80)} x2="640" y2={calcY(80)} stroke="#5CE1A5" strokeWidth="1" strokeDasharray="4 4" opacity="0.45"/>
            <text x="6" y={calcY(80) - 6} className="trace-text dim">80 &#183; healthy</text>
            <line x1="0" y1={calcY(50)} x2="640" y2={calcY(50)} stroke="#E5484D" strokeWidth="1" strokeDasharray="4 4" opacity="0.45"/>
            <text x="6" y={calcY(50) - 6} className="trace-text dim">50 &#183; critical</text>
            <path d={createPath(history)} fill="none" stroke="#5CE1A5" strokeWidth="2"/>
            <circle cx="640" cy={calcY(history[history.length-1])} r="4" fill="#F5A623"/>
            <text x="0" y="182" className="trace-text dim">7d ago</text>
            <text x="602" y="182" className="trace-text dim">now</text>
          </svg>
        </div>
      </div>

      <div className="grid4">
        <Metric label="Cardinality alerts" value={data.metrics?.cardinality?.value || 3} sub="1 key-space anomaly" percent={70} color="red" tooltip="Max cardinality across all service/attribute pairs in a rolling 15m window" />
        <Metric label="Orphan rate" value={data.metrics?.orphans?.value || '6.2%'} sub="above 5% threshold" percent={62} color="amber" tooltip="Percentage of spans missing a parent trace context within a 30s arrival window" />
        <Metric label="Coverage gaps" value={data.metrics?.coverage?.value || 1} sub="service silent 14m" percent={20} color="amber" tooltip="Active services reporting telemetry compared to baseline expectations" />
        <Metric label="Est. cost impact" value="$4.1k/mo" sub="from cardinality alone" percent={44} color="phosphor" tooltip="Estimated wasted infra spend due to bloated metric dimensions or duplicate traces" />
      </div>

      <h2 className="section-title">Active issues</h2>
      <div className="panel panel-tight">
        {issues.length > 0 ? issues.map(iss => (
          <div className="rack-row" key={iss.id}>
            <span className={`rled ${iss.impact < -15 ? 'r' : 'a'}`}></span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div className="rack-svc">{iss.service}</div>
              <div className="rack-desc">{iss.description}</div>
            </div>
            <div className="rack-impact">{iss.impact > 0 ? `+${iss.impact}` : `\u2212${Math.abs(iss.impact)}`}</div>
            <button className="btn" onClick={() => setView('remediation')}>remediate &#9656;</button>
          </div>
        )) : (
          <div className="rack-row">
            <div className="rack-desc" style={{ padding: '12px' }}>No active issues detected.</div>
          </div>
        )}
      </div>
    </section>
  );
}
