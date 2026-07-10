import { useState, useEffect } from 'react';
import { AlertCircle } from 'lucide-react';
import type { DashboardData } from '../../App';

interface TraceOrphanData {
  orphanRate: string;
  topOrphanedService: string;
  missingParents: number;
}

interface TraceChainsProps {
  data: DashboardData;
  tenantId: string;
}

export function TraceChains({ data, tenantId }: TraceChainsProps) {
  const [traceData, setTraceData] = useState<TraceOrphanData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<boolean>(false);
  const orphanRate = data?.metrics?.orphans?.value || '6.2%';

  useEffect(() => {
    setLoading(true);
    setError(false);

    fetch(`http://localhost:8080/api/v1/tenant/${tenantId}/traces/orphans`)
      .then((r) => {
        if (!r.ok) throw new Error('Failed to load trace orphans');
        return r.json();
      })
      .then(setTraceData)
      .catch((err) => {
        console.error(err);
        setError(true);
        // Fallback mock data
        setTraceData({
          orphanRate: '6.2%',
          topOrphanedService: 'payments-api',
          missingParents: 142
        });
      })
      .finally(() => {
        setLoading(false);
      });
  }, [tenantId]);

  return (
    <section className="view active">
      <div className="eyebrow">03 • broken trace-chain detector • §8.2</div>

      <div className="tag-row">
        <span className="tag">
          orphan rate <b style={{ color: 'var(--amber)' }}>{traceData?.orphanRate || orphanRate}</b>
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
        <div style={{ background: 'rgba(239, 68, 68, 0.08)', padding: '10px 16px', borderRadius: '4px', border: '1px solid var(--red)', color: 'var(--red)', marginBottom: '14px', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <AlertCircle size={14} />
          <span>Error loading trace analytics. Showing local fallback.</span>
        </div>
      )}

      {loading && !traceData ? (
        <div className="animate-pulse" style={{ display: 'grid', gridTemplateColumns: '1.5fr 1fr', gap: '14px' }}>
          <div style={{ height: '240px', background: 'var(--panel)', border: '1px solid var(--bezel)', borderRadius: '6px' }}></div>
          <div style={{ height: '240px', background: 'var(--panel)', border: '1px solid var(--bezel)', borderRadius: '6px' }}></div>
        </div>
      ) : (
        <div className="grid2">
          <div className="panel">
            <div className="metric-label" style={{ marginBottom: '14px' }}>
              trace 9f3a2c • {traceData?.topOrphanedService || 'payments-api'}
            </div>
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
              <text x="290" y="153" className="trace-text" style={{ fill: '#E5484D' }}>
                payment-capture — orphan
              </text>
              <text x="280" y="122" className="trace-text dim">
                missing parent_span_id: 7bd1
              </text>
            </svg>
          </div>

          <div className="panel panel-tight">
            <div className="metric-label" style={{ padding: '12px 6px 4px' }}>
              recent orphan events ({traceData?.missingParents || 0} total)
            </div>
            <div className="rack-row">
              <span className="rled r"></span>
              <div style={{ flex: 1 }}>
                <div className="rack-svc" style={{ fontSize: '12px' }}>
                  span 4a91 • collector-07
                </div>
                <div className="rack-desc">parent 7bd1 not found • correlated after 31s</div>
              </div>
            </div>
            <div className="rack-row">
              <span className="rled r"></span>
              <div style={{ flex: 1 }}>
                <div className="rack-svc" style={{ fontSize: '12px' }}>
                  span 2e6f • collector-03
                </div>
                <div className="rack-desc">parent c910 not found • correlated after 12s</div>
              </div>
            </div>
            <div className="rack-row">
              <span className="rled a"></span>
              <div style={{ flex: 1 }}>
                <div className="rack-svc" style={{ fontSize: '12px' }}>
                  span 88bd • collector-07
                </div>
                <div className="rack-desc">late arrival • resolved within window</div>
              </div>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
