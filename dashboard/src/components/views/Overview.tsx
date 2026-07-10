import { useEffect, useState } from 'react';
import { ArrowUpRight, ArrowDownRight, Info, AlertTriangle } from 'lucide-react';
import type { DashboardData } from '../../App';

const AnimatedHealthGauge = ({ score }: { score: number }) => {
  const [displayScore, setDisplayScore] = useState<number>(score);
  const color = score > 80 ? 'var(--phosphor)' : score > 50 ? 'var(--amber)' : 'var(--red)';

  // Safe React-state interval timer animation complying with instructions
  useEffect(() => {
    if (displayScore === score) return;
    const stepCount = Math.abs(score - displayScore);
    if (stepCount === 0) return;

    const totalDuration = 300; // 300ms transition
    const intervalTime = Math.max(15, totalDuration / stepCount);
    const stepValue = score > displayScore ? 1 : -1;

    const interval = setInterval(() => {
      setDisplayScore((prev) => {
        const next = prev + stepValue;
        if ((stepValue > 0 && next >= score) || (stepValue < 0 && next <= score)) {
          clearInterval(interval);
          return score;
        }
        return next;
      });
    }, intervalTime);

    return () => clearInterval(interval);
  }, [score, displayScore]);

  return (
    <div className="health-gauge" style={{ display: 'flex', justifyContent: 'center' }}>
      <svg width="140" height="140" viewBox="0 0 200 200">
        <circle cx="100" cy="100" r="80" stroke="var(--panel-2)" strokeWidth="15" fill="none" />
        <circle
          cx="100" cy="100" r="80"
          stroke={color} strokeWidth="15" fill="none"
          strokeDasharray="502"
          strokeDashoffset={502 - (502 * displayScore) / 100}
          strokeLinecap="round"
          transform="rotate(-90 100 100)"
          className="transition-colors"
          style={{ transition: 'stroke-dashoffset 0.3s ease-in-out, stroke 0.3s ease-in-out' }}
        />
        <text x="100" y="115" textAnchor="middle" fontSize="48" fontWeight="600" fill={color} className="transition-colors">
          {displayScore}
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
  change: number;
  isInteractive?: boolean;
  isActive?: boolean;
  onClick?: () => void;
}

function Metric({ label, value, sub, percent, color, tooltip, change, isInteractive, isActive, onClick }: MetricProps) {
  // Arrow and color styling logic based on target metric types
  const isGood = label.toLowerCase().includes('coverage') ? change >= 0 : change <= 0;
  const isNeutral = change === 0;

  return (
    <div
      className={`panel metric ${isInteractive ? 'metric-interactive' : ''}`}
      onClick={isInteractive ? onClick : undefined}
      style={{
        cursor: isInteractive ? 'pointer' : 'default',
        borderColor: isActive ? 'var(--phosphor)' : undefined,
        background: isActive ? 'var(--panel-2)' : undefined
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div className="metric-label">{label}</div>
        <div title={tooltip} style={{ cursor: 'help', color: 'var(--muted)' }}>
          <Info size={12} />
        </div>
      </div>
      <div className="metric-val" style={{ color: `var(--${color})` }}>
        <span>{value}</span>
        {!isNeutral && (
          <span style={{ display: 'inline-flex', alignItems: 'center', fontSize: '12px', fontWeight: '500', color: isGood ? 'var(--phosphor)' : 'var(--red)', marginLeft: '4px' }}>
            {change > 0 ? <ArrowUpRight size={14} /> : <ArrowDownRight size={14} />}
            {Math.abs(change)}%
          </span>
        )}
      </div>
      <div className="metric-sub">{sub}</div>
      <div className="metric-bar">
        <div style={{ width: `${percent}%`, background: `var(--${color})` }}></div>
      </div>
    </div>
  );
}

interface IssueItem {
  id: string;
  service: string;
  description: string;
  impact: number;
}

interface OverviewProps {
  data: DashboardData;
  setView: (v: string) => void;
  tenantId: string;
}

export function Overview({ data, setView, tenantId }: OverviewProps) {
  const [issues, setIssues] = useState<IssueItem[]>([]);
  const [activeDrilldown, setActiveDrilldown] = useState<string | null>(null);
  const [issuesLoading, setIssuesLoading] = useState<boolean>(true);

  useEffect(() => {
    setIssuesLoading(true);
    const fallbackIssues: IssueItem[] = [
      { id: 'iss-1', service: 'payments-api', description: 'Broken trace chain · 18% orphan rate · §8.2', impact: -18 },
      { id: 'iss-2', service: 'checkout-service', description: 'Cardinality spike · user_id_raw · §8.1', impact: -12 },
      { id: 'iss-3', service: 'inventory-worker', description: 'Coverage gap · silent 14m · §8.3', impact: -8 }
    ];

    fetch(`http://localhost:8080/api/v1/tenant/${tenantId}/issues`)
      .then((r) => {
        if (!r.ok) throw new Error('API status not OK');
        return r.json();
      })
      .then(setIssues)
      .catch(() => {
        setIssues(fallbackIssues);
      })
      .finally(() => {
        setIssuesLoading(false);
      });
  }, [tenantId]);

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
  
  const history = data.history || [
    Math.max(0, score - 12),
    Math.max(0, score - 5),
    score - 8,
    score - 2,
    score - 10,
    score - 3,
    score,
    score
  ];

  // Dynamic score history delta calculations
  const firstScore = history[0];
  const lastScore = history[history.length - 1];
  const delta = lastScore - firstScore;

  const toggleDrilldown = (type: string) => {
    setActiveDrilldown((prev) => (prev === type ? null : type));
  };

  return (
    <section className="view active">
      <div className="panel hero" style={{ display: 'flex', alignItems: 'center' }}>
        <div className="hero-score" style={{ flex: 1 }}>
          <AnimatedHealthGauge score={score} />
          <div style={{ textAlign: 'center', marginTop: '16px' }}>
            <span className={`score-band ${bandClass}`} id="score-band">
              {bandText}
            </span>
            <div className="score-delta" style={{ justifyContent: 'center', color: delta >= 0 ? 'var(--phosphor)' : 'var(--red)', marginTop: '6px' }}>
              {delta >= 0 ? <ArrowUpRight size={14} /> : <ArrowDownRight size={14} />}
              {delta >= 0 ? `+${delta}` : delta} vs last week
            </div>
            <div className="score-caption" style={{ marginTop: '4px' }}>
              composite • org rollup
            </div>
          </div>
        </div>
        <div className="hero-chart" style={{ flex: 2 }}>
          <div className="scanline"></div>
          <svg viewBox="0 0 640 190" style={{ width: '100%', height: '190px', display: 'block' }}>
            <line x1="0" y1={calcY(80)} x2="640" y2={calcY(80)} stroke="#5CE1A5" strokeWidth="1" strokeDasharray="4 4" opacity="0.45" />
            <text x="6" y={calcY(80) - 6} className="trace-text dim">
              80 • healthy
            </text>
            <line x1="0" y1={calcY(50)} x2="640" y2={calcY(50)} stroke="#E5484D" strokeWidth="1" strokeDasharray="4 4" opacity="0.45" />
            <text x="6" y={calcY(50) - 6} className="trace-text dim">
              50 • critical
            </text>
            <path d={createPath(history)} fill="none" stroke="#5CE1A5" strokeWidth="2" />
            <circle cx="640" cy={calcY(history[history.length - 1])} r="4" fill="#F5A623" />
            <text x="0" y="182" className="trace-text dim">
              7d ago
            </text>
            <text x="602" y="182" className="trace-text dim">
              now
            </text>
          </svg>
        </div>
      </div>

      {/* Grid containing metric cards */}
      <div className="grid4">
        <Metric
          label="Cardinality alerts"
          value={data.metrics?.cardinality?.value || '3'}
          sub="key-space anomalies"
          percent={70}
          color="red"
          tooltip="Max cardinality across all service/attribute pairs in a rolling 15m window"
          change={data.metrics?.cardinality?.change || 14.5}
          isInteractive
          isActive={activeDrilldown === 'cardinality'}
          onClick={() => toggleDrilldown('cardinality')}
        />
        <Metric
          label="Orphan rate"
          value={data.metrics?.orphans?.value || '6.2%'}
          sub="above 5% threshold"
          percent={62}
          color="amber"
          tooltip="Percentage of spans missing a parent trace context within a 30s arrival window"
          change={data.metrics?.orphans?.change || 1.2}
          isInteractive
          isActive={activeDrilldown === 'orphans'}
          onClick={() => toggleDrilldown('orphans')}
        />
        <Metric
          label="Coverage gaps"
          value={data.metrics?.coverage?.value || '1'}
          sub="services silent"
          percent={20}
          color="amber"
          tooltip="Active services reporting telemetry compared to baseline expectations"
          change={data.metrics?.coverage?.change || 0.0}
          isInteractive
          isActive={activeDrilldown === 'coverage'}
          onClick={() => toggleDrilldown('coverage')}
        />
        <Metric
          label="Est. cost impact"
          value="$4.1k/mo"
          sub="from cardinality bloat"
          percent={44}
          color="phosphor"
          tooltip="Estimated wasted infra spend due to bloated metric dimensions or duplicate traces"
          change={-5.4}
          isInteractive
          isActive={activeDrilldown === 'cost'}
          onClick={() => toggleDrilldown('cost')}
        />
      </div>

      {/* Expandable drill-down accordions */}
      {activeDrilldown && (
        <div className="panel" style={{ marginTop: '12px', borderLeft: '3px solid var(--phosphor)', animation: 'slideDown 0.15s ease-out' }}>
          {activeDrilldown === 'cardinality' && (
            <div>
              <h3 style={{ margin: '0 0 10px 0', fontSize: '13px', textTransform: 'uppercase', fontFamily: 'var(--mono)', color: 'var(--paper)' }}>
                Cardinality Drill-down Details
              </h3>
              <div style={{ fontSize: '13px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <div><strong>checkout-service:</strong> <code>user_id_raw</code> is currently emitting <strong>~14,382</strong> unique HLL hashes (exceeds threshold).</div>
                <div><strong>payments-api:</strong> <code>raw_url</code> is currently emitting <strong>~8,120</strong> unique hashes (high cardinality alert).</div>
                <div style={{ color: 'var(--muted)', fontSize: '11px', marginTop: '4px' }}>Recommendation: Use OTel attributes processor to hash or redact unneeded raw ID params.</div>
              </div>
            </div>
          )}
          {activeDrilldown === 'orphans' && (
            <div>
              <h3 style={{ margin: '0 0 10px 0', fontSize: '13px', textTransform: 'uppercase', fontFamily: 'var(--mono)', color: 'var(--paper)' }}>
                Orphan Spans Drill-down Details
              </h3>
              <div style={{ fontSize: '13px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <div><strong>payments-api:</strong> Orphan rate is <strong>18%</strong> (exceeds 5% safe warning limit).</div>
                <div><strong>checkout-service:</strong> Clock skew of <strong>+6.1s</strong> detected, causing matching latency delays.</div>
                <div style={{ color: 'var(--muted)', fontSize: '11px', marginTop: '4px' }}>Recommendation: Update context propagation injection header checks in payments client.</div>
              </div>
            </div>
          )}
          {activeDrilldown === 'coverage' && (
            <div>
              <h3 style={{ margin: '0 0 10px 0', fontSize: '13px', textTransform: 'uppercase', fontFamily: 'var(--mono)', color: 'var(--paper)' }}>
                Service Coverage Gaps
              </h3>
              <div style={{ fontSize: '13px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <AlertTriangle size={14} style={{ color: 'var(--amber)' }} />
                  <span><strong>inventory-worker:</strong> Silent for <strong>14 minutes</strong>. Expected emit interval is &lt; 30 seconds.</span>
                </div>
                <div><strong>auth-service:</strong> Active, last trace reported 1s ago.</div>
                <div style={{ color: 'var(--muted)', fontSize: '11px', marginTop: '4px' }}>Recommendation: Check if inventory-worker pod is running or verify sidecar exporter configs.</div>
              </div>
            </div>
          )}
          {activeDrilldown === 'cost' && (
            <div>
              <h3 style={{ margin: '0 0 10px 0', fontSize: '13px', textTransform: 'uppercase', fontFamily: 'var(--mono)', color: 'var(--paper)' }}>
                Estimated Observability Infrastructure Cost Impact
              </h3>
              <div style={{ fontSize: '13px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <div><strong>Estimated waste:</strong> <strong>$4.1k/month</strong>.</div>
                <div><strong>Bloated metrics size:</strong> Cardinality explosion in <code>user_id_raw</code> is adding ~4.8GB/day of index metrics storage.</div>
                <div><strong>Potential Saving:</strong> Applying proposed OTel yaml remediation will reduce overall logs volume by ~22%.</div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Active issues list */}
      <h2 className="section-title">Active issues</h2>
      <div className="panel panel-tight">
        {issuesLoading ? (
          <div className="animate-pulse" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <div style={{ height: '14px', background: 'var(--bezel-soft)', borderRadius: '4px', width: '60%' }}></div>
            <div style={{ height: '14px', background: 'var(--bezel-soft)', borderRadius: '4px', width: '80%' }}></div>
            <div style={{ height: '14px', background: 'var(--bezel-soft)', borderRadius: '4px', width: '40%' }}></div>
          </div>
        ) : issues.length > 0 ? (
          issues.map((iss) => (
            <div className="rack-row" key={iss.id}>
              <span className={`rled ${iss.impact < -15 ? 'r' : 'a'}`}></span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="rack-svc">{iss.service}</div>
                <div className="rack-desc">{iss.description}</div>
              </div>
              <div className="rack-impact">
                {iss.impact > 0 ? `+${iss.impact}` : `\u2212${Math.abs(iss.impact)}`}
              </div>
              <button className="btn" onClick={() => setView('remediation')}>
                remediate <ArrowUpRight size={12} />
              </button>
            </div>
          ))
        ) : (
          <div className="rack-row">
            <div className="rack-desc" style={{ padding: '12px' }}>
              No active issues detected.
            </div>
          </div>
        )}
      </div>
    </section>
  );
}
