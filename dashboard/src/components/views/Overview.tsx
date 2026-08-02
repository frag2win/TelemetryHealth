import { useEffect, useState, useRef } from 'react';
import { ArrowUpRight, ArrowDownRight, AlertTriangle, Database, Activity, Clock, CheckCircle } from 'lucide-react';
import { Metric, useTenantData, ErrorBanner, SkeletonLoader } from '../Shared';
import { RootCauseGraph } from './RootCauseGraph';
import type { DashboardData } from '../../App';

interface AnimatedHealthGaugeProps {
  score: number;
}

const AnimatedHealthGauge = ({ score }: AnimatedHealthGaugeProps) => {
  const [displayScore, setDisplayScore] = useState<number>(score);
  const displayScoreRef = useRef(displayScore);
  const color = score >= 70 ? 'var(--phosphor)' : score >= 40 ? 'var(--amber)' : 'var(--red)';

  // Sync displayScoreRef with current state
  useEffect(() => {
    displayScoreRef.current = displayScore;
  }, [displayScore]);

  // Safe state-driven interval counter without displayScore dependency (prevents infinite loop)
  // Utilizes displayScoreRef to avoid capturing stale closures (Bug 2 fix)
  useEffect(() => {
    const targetScore = score;
    const currentScore = displayScoreRef.current;
    if (currentScore === targetScore) return;
    const stepCount = Math.abs(targetScore - currentScore);
    if (stepCount === 0) return;

    const totalDuration = 300; // 300ms transition
    const intervalTime = Math.max(15, totalDuration / stepCount);
    const stepValue = targetScore > currentScore ? 1 : -1;

    const interval = setInterval(() => {
      setDisplayScore((prev) => {
        const next = prev + stepValue;
        if ((stepValue > 0 && next >= targetScore) || (stepValue < 0 && next <= targetScore)) {
          clearInterval(interval);
          return targetScore;
        }
        return next;
      });
    }, intervalTime);

    return () => clearInterval(interval);
  }, [score]);

  return (
    <div className="health-gauge" style={{ display: 'flex', justifyContent: 'center' }}>
      <svg width="150" height="150" viewBox="0 0 200 200">
        {/* Background Dial Track */}
        <circle
          cx="100" cy="100" r="75"
          stroke="var(--bezel)" strokeWidth="10" fill="none"
          strokeDasharray="353.43 471.24"
          strokeLinecap="round"
          transform="rotate(135 100 100)"
        />
        {/* Active Progress Value */}
        <circle
          cx="100" cy="100" r="75"
          stroke={color} strokeWidth="12" fill="none"
          strokeDasharray="353.43 471.24"
          strokeDashoffset={353.43 - (353.43 * displayScore) / 100}
          strokeLinecap="round"
          transform="rotate(135 100 100)"
          className="transition-colors"
          style={{
            transition: 'stroke-dashoffset 0.3s ease-in-out, stroke 0.3s ease-in-out',
            filter: 'drop-shadow(0 0 6px ' + color + ')'
          }}
        />
        {/* Score Text */}
        <text x="100" y="110" textAnchor="middle" fill="var(--paper)" className="gauge-value" style={{ fontSize: '52px', fontWeight: 'bold', fontFamily: 'var(--mono)' }}>
          {typeof displayScore === 'number' && !isNaN(displayScore) ? Math.round(displayScore) : 0}
        </text>
        {/* Label inside gauge */}
        <text x="100" y="132" textAnchor="middle" fill="var(--muted)" style={{ fontSize: '10px', fontWeight: '500', fontFamily: 'var(--mono)', letterSpacing: '2px', textTransform: 'uppercase' }}>
          score
        </text>
      </svg>
    </div>
  );
};

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
  const [activeDrilldown, setActiveDrilldown] = useState<string | null>(null);
  const [expandedIssue, setExpandedIssue] = useState<string | null>(null);

  const fallbackIssues: IssueItem[] = [
    { id: 'iss-1', service: 'payments-api', description: 'Broken trace chain · 18% orphan rate · §8.2', impact: -18 },
    { id: 'iss-2', service: 'checkout-service', description: 'Cardinality spike · user_id_raw · §8.1', impact: -12 },
    { id: 'iss-3', service: 'inventory-worker', description: 'Coverage gap · silent 14m · §8.3', impact: -8 }
  ];

  // useTenantData custom hook safely retrieves dynamic issues utilizing AbortController
  const { data: issues, loading: issuesLoading, error: hasIssuesError, errorMsg: issuesErrorMsg } = useTenantData<IssueItem[]>(
    tenantId,
    'issues',
    fallbackIssues
  );

  const score = data.healthScore ?? 78;
  const bandClass = score >= 70 ? 'band-healthy' : score >= 40 ? 'band-degraded' : 'band-critical';
  const bandText = score >= 70 ? 'healthy' : score >= 40 ? 'degraded' : 'critical';
  const statusColor = score >= 70 ? 'var(--phosphor)' : score >= 40 ? 'var(--amber)' : 'var(--red)';

  // Math.max protection to prevent vertical SVG rendering overflows
  const calcY = (p: number) => Math.max(0, Math.round(150 - (p / 100) * 140));

  const createPath = (points: number[]) => {
    // Array length guards to prevent division-by-zero Infinity steps
    if (points.length <= 1) return '';
    const step = 640 / (points.length - 1);
    return points.map((p, i) => `${i === 0 ? 'M' : 'L'}${Math.round(i * step)},${calcY(p)}`).join(' ');
  };

  const createAreaPath = (points: number[]) => {
    if (points.length <= 1) return '';
    const pathStr = createPath(points);
    const step = 640 / (points.length - 1);
    const lastX = Math.round((points.length - 1) * step);
    return `${pathStr} L${lastX},150 L0,150 Z`;
  };
  
  const history = data.history ?? [
    Math.max(0, score - 12),
    Math.max(0, score - 5),
    Math.max(0, score - 8),
    Math.max(0, score - 2),
    Math.max(0, score - 10),
    Math.max(0, score - 3),
    score,
    score
  ];

  const firstScore = history[0];
  const lastScore = history[history.length - 1];
  const delta = lastScore - firstScore;

  const toggleDrilldown = (type: string) => {
    setActiveDrilldown((prev) => (prev === type ? null : type));
  };

  const activeIssuesList = issues ?? [];

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
              composite • ai agents • org rollup
            </div>
          </div>
        </div>
        <div className="hero-chart" style={{ flex: 2 }}>
          <div className="scanline"></div>
          <svg viewBox="0 0 640 190" style={{ width: '100%', height: '190px', display: 'block' }}>
            <defs>
              <linearGradient id="chart-glow" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={statusColor} stopOpacity="0.15" />
                <stop offset="100%" stopColor={statusColor} stopOpacity="0.0" />
              </linearGradient>
            </defs>
            <line x1="0" y1={calcY(80)} x2="640" y2={calcY(80)} stroke="var(--phosphor)" strokeWidth="1" strokeDasharray="4 4" opacity="0.25" />
            <text x="6" y={calcY(80) - 6} className="trace-text dim" style={{ fill: 'var(--phosphor)', opacity: 0.8 }}>
              80 • healthy
            </text>
            <line x1="0" y1={calcY(50)} x2="640" y2={calcY(50)} stroke="var(--red)" strokeWidth="1" strokeDasharray="4 4" opacity="0.25" />
            <text x="6" y={calcY(50) - 6} className="trace-text dim" style={{ fill: 'var(--red)', opacity: 0.8 }}>
              50 • critical
            </text>
            <path d={createAreaPath(history)} fill="url(#chart-glow)" />
            <path d={createPath(history)} fill="none" stroke={statusColor} strokeWidth="2.5" />
            <circle cx="640" cy={calcY(history[history.length - 1])} r="5" fill={statusColor} style={{ filter: 'drop-shadow(0 0 4px ' + statusColor + ')' }} />
            <text x="0" y="182" className="trace-text dim">
              7d ago
            </text>
            <text x="602" y="182" className="trace-text dim">
              now
            </text>
          </svg>
        </div>
      </div>

      {/* Grid containing metric cards in 2 balanced rows of 3 columns */}
      <div className="grid-metrics">
        <Metric
          label="Cardinality alerts"
          value={data.metrics?.cardinality?.value ?? '3'}
          sub="key-space anomalies"
          percent={70}
          color="red"
          tooltip="Max cardinality across all service/attribute pairs in a rolling 15m window"
          change={data.metrics?.cardinality?.change ?? 14.5}
          icon={Database}
          isInteractive
          isActive={activeDrilldown === 'cardinality'}
          onClick={() => toggleDrilldown('cardinality')}
        />
        <Metric
          label="Orphan rate"
          value={data.metrics?.orphans?.value ?? '6.2%'}
          sub="above 5% threshold"
          percent={62}
          color="amber"
          tooltip="Percentage of spans missing a parent trace context within a 30s arrival window"
          change={data.metrics?.orphans?.change ?? 1.2}
          icon={Activity}
          isInteractive
          isActive={activeDrilldown === 'orphans'}
          onClick={() => toggleDrilldown('orphans')}
        />
        <Metric
          label="Coverage gaps"
          value={data.metrics?.coverage?.value ?? '1'}
          sub="services silent"
          percent={20}
          color="amber"
          tooltip="Active services reporting telemetry compared to baseline expectations"
          change={data.metrics?.coverage?.change ?? 0.0}
          icon={CheckCircle}
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
          icon={Database}
          isInteractive
          isActive={activeDrilldown === 'cost'}
          onClick={() => toggleDrilldown('cost')}
        />
        <Metric
          label="Token Burn Rate"
          value={data.metrics?.tokenBurnRate?.value ?? '1,204'}
          sub="tokens / sec"
          percent={85}
          color="phosphor"
          tooltip="Average LLM token burn rate across all AI agents"
          change={data.metrics?.tokenBurnRate?.change ?? 12.5}
          icon={Clock}
        />
        <Metric
          label="Tool Call Success"
          value={data.metrics?.toolCallSuccess?.value ?? '98.5%'}
          sub="agent reliability"
          percent={98}
          color="phosphor"
          tooltip="Success rate of agent tool executions (no errors)"
          change={data.metrics?.toolCallSuccess?.change ?? 0.2}
          icon={CheckCircle}
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

      {hasIssuesError && (
        <ErrorBanner message={`Error loading live issues: ${issuesErrorMsg ?? 'Unknown Error'}. Showing fallback records.`} />
      )}

      <div className="panel panel-tight">
        {issuesLoading ? (
          <SkeletonLoader rows={3} />
        ) : activeIssuesList.length > 0 ? (
          activeIssuesList.map((iss) => (
            <div key={iss.id} style={{ display: 'flex', flexDirection: 'column' }}>
              <div className="rack-row">
                <span className={`rled ${iss.impact < -15 ? 'r' : 'a'}`}></span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div className="rack-svc">{iss.service}</div>
                  <div className="rack-desc">{iss.description}</div>
                </div>
                <div className="rack-impact">
                  {iss.impact > 0 ? `+${iss.impact}` : `\u2212${Math.abs(iss.impact)}`}
                </div>
                <button className="btn" onClick={() => setExpandedIssue(expandedIssue === iss.id ? null : iss.id)} style={{ marginRight: '8px' }}>
                  {expandedIssue === iss.id ? 'hide graph' : 'analyze root cause'}
                </button>
                <button className="btn" onClick={() => setView('remediation')}>
                  remediate <ArrowUpRight size={12} />
                </button>
              </div>
              {expandedIssue === iss.id && (
                <RootCauseGraph tenantId={tenantId} issueId={iss.id} />
              )}
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
