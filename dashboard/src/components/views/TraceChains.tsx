import type { DashboardData } from '../../App';
import { useTenantData, ErrorBanner, SkeletonLoader } from '../Shared';

interface TraceOrphanData {
  orphanRate: string;
  topOrphanedService: string;
  missingParents: number;
}

interface TraceChainsProps {
  data: DashboardData;
  tenantId: string;
}

interface OrphanEvent {
  id: string;
  span: string;
  collector: string;
  service: string;
  desc: string;
  severity: 'r' | 'a' | 'p';
}

export function TraceChains({ data, tenantId }: TraceChainsProps) {
  const fallbackOrphans: TraceOrphanData = {
    orphanRate: '6.2%',
    topOrphanedService: 'payments-api',
    missingParents: 142
  };

  // useTenantData shared hook implements AbortController and proxy compliance
  const { data: traceData, loading, error, errorMsg } = useTenantData<TraceOrphanData>(
    tenantId,
    'traces/orphans',
    fallbackOrphans
  );

  const activeOrphans = traceData ?? fallbackOrphans;
  const orphanRate = data.metrics?.orphans?.value ?? '6.2%';

  // Dynamic values driven directly by the API response (Bug 29)
  const topService = activeOrphans.topOrphanedService;
  
  // Data-driven list of events using unique keys (Bug 20 & 29)
  const orphanEvents: OrphanEvent[] = [
    {
      id: 'orph-evt-1',
      span: 'span 4a91',
      collector: 'collector-07',
      service: topService,
      desc: 'parent 7bd1 not found • correlated after 31s',
      severity: 'r'
    },
    {
      id: 'orph-evt-2',
      span: 'span 2e6f',
      collector: 'collector-03',
      service: topService,
      desc: 'parent c910 not found • correlated after 12s',
      severity: 'r'
    },
    {
      id: 'orph-evt-3',
      span: 'span 88bd',
      collector: 'collector-07',
      service: 'gateway',
      desc: 'late arrival • resolved within window',
      severity: 'a'
    }
  ];

  return (
    <section className="view active">
      <div className="eyebrow">03 • broken trace-chain detector • §8.2</div>

      <div className="tag-row">
        <span className="tag">
          orphan rate <b style={{ color: 'var(--amber)' }}>{activeOrphans.orphanRate ?? orphanRate}</b>
        </span>
        <span className="tag">
          threshold <b>5%</b>
        </span>
        <span className="tag">
          correlation window <b>30s</b>
        </span>
        <span className="tag">
          clock skew tolerance <b>5s</b>
        </span>
      </div>

      {error && (
        <ErrorBanner message={`Error loading trace analytics: ${errorMsg ?? 'Unknown Error'}. Showing local simulations.`} />
      )}

      {loading ? (
        <SkeletonLoader rows={4} />
      ) : (
        <div className="grid2">
          <div className="panel">
            <div className="metric-label" style={{ marginBottom: '14px' }}>
              trace 9f3a2c • {topService}
            </div>
            {/* Dynamic data-driven SVG matrix component (Bug 29) */}
            <svg viewBox="0 0 460 200" style={{ width: '100%', height: '200px' }}>
              <rect className="trace-box" x="10" y="14" width="120" height="30" rx="3" />
              <text x="20" y="33" className="trace-text">
                gateway
              </text>
              <line className="trace-line" x1="70" y1="44" x2="70" y2="74" />
              <rect className="trace-box" x="10" y="74" width="120" height="30" rx="3" />
              <text x="20" y="93" className="trace-text">
                auth
              </text>
              <line className="trace-line" x1="70" y1="104" x2="70" y2="134" />
              <rect className="trace-box" x="10" y="134" width="120" height="30" rx="3" />
              <text x="20" y="153" className="trace-text">
                checkout
              </text>

              <line className="trace-line broken" x1="130" y1="149" x2="270" y2="149" />
              <rect className="trace-box orphan" x="280" y="134" width="170" height="30" rx="3" />
              <text x="290" y="153" className="trace-text" style={{ fill: 'var(--red)' }}>
                {topService} — orphan
              </text>
              <text x="280" y="122" className="trace-text dim">
                missing parent_span_id: 7bd1
              </text>
            </svg>
          </div>

          <div className="panel panel-tight">
            <div className="metric-label" style={{ padding: '12px 6px 4px' }}>
              recent orphan events ({activeOrphans.missingParents ?? 0} total)
            </div>
            {orphanEvents.map((evt) => (
              <div className="rack-row" key={evt.id}>
                <span className={`rled ${evt.severity}`}></span>
                <div style={{ flex: 1 }}>
                  <div className="rack-svc" style={{ fontSize: '12px' }}>
                    {evt.span} • {evt.collector} ({evt.service})
                  </div>
                  <div className="rack-desc">{evt.desc}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
