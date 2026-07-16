import { useState, useEffect } from 'react';
import { Check } from 'lucide-react';
import type { DashboardData } from '../../App';

interface CardinalityRow {
  service: string;
  attributeKey: string;
  uniqueHllEst: string;
  trendPoints: string;
  status: 'breach' | 'watch' | 'normal';
}

interface CardinalityProps {
  data: DashboardData;
  tenantId: string;
}

export function Cardinality({ data: _data, tenantId }: CardinalityProps) {
  const [rows, setRows] = useState<CardinalityRow[]>([]);
  const [toast, setToast] = useState<string | null>(null);

  useEffect(() => {
    // Generate dynamic cardinality items based on tenant ID (Bug 11)
    if (tenantId.includes('staging')) {
      setRows([
        { service: 'analytics-query', attributeKey: 'client_ip', uniqueHllEst: '~1,240', trendPoints: '0,15 15,13 30,9 45,6 60,4', status: 'watch' },
        { service: 'gateway-proxy', attributeKey: 'user_agent', uniqueHllEst: '~420', trendPoints: '0,10 15,10 30,9 45,10 60,10', status: 'normal' }
      ]);
    } else if (tenantId.includes('alpha')) {
      setRows([
        { service: 'checkout-service', attributeKey: 'session_id', uniqueHllEst: '~28,450', trendPoints: '0,18 15,16 30,12 45,8 60,2', status: 'breach' },
        { service: 'order-processor', attributeKey: 'customer_email', uniqueHllEst: '~9,102', trendPoints: '0,14 15,12 30,11 45,9 60,5', status: 'breach' }
      ]);
    } else {
      setRows([
        { service: 'checkout-service', attributeKey: 'user_id_raw', uniqueHllEst: '~14,382', trendPoints: '0,16 15,14 30,10 45,6 60,3', status: 'breach' },
        { service: 'payments-api', attributeKey: 'raw_url', uniqueHllEst: '~8,120', trendPoints: '0,15 15,13 30,12 45,8 60,4', status: 'breach' },
        { service: 'inventory-worker', attributeKey: 'session_token', uniqueHllEst: '~640', trendPoints: '0,10 15,11 30,9 45,10 60,10', status: 'watch' },
        { service: 'auth-service', attributeKey: 'request_id', uniqueHllEst: '~92', trendPoints: '0,10 15,10 30,11 45,10 60,10', status: 'normal' }
      ]);
    }
  }, [tenantId]);

  const handleRedactClick = (row: CardinalityRow) => {
    setToast(`Remediation patch created for ${row.service} -> ${row.attributeKey}. View in Remediation panel.`);
    setTimeout(() => setToast(null), 2500);
  };

  const totalKeys = rows.length * 15 + 2;
  const maxLimit = 100;
  const progressPercent = Math.round((totalKeys / maxLimit) * 100);

  return (
    <section className="view active">
      {toast && (
        <div
          style={{
            position: 'fixed',
            bottom: '1rem',
            right: '1rem',
            background: 'var(--toast-bg)',
            border: '1px solid var(--toast-border)',
            padding: '12px 24px',
            borderRadius: '4px',
            color: 'var(--phosphor)',
            zIndex: 9999,
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            boxShadow: 'var(--shadow-sm)',
            fontSize: '13px'
          }}
        >
          <Check size={16} />
          <span>{toast}</span>
        </div>
      )}

      <div className="eyebrow">02 • cardinality detector • §8.1</div>

      {/* Dynamic Key-space Progress trackers (Bug 11) */}
      <div className="progress-wrap">
        <span className="progress-label">key-space</span>
        <div className="progress-track">
          <div className="progress-fill" style={{ width: `${progressPercent}%` }}></div>
          <div className="progress-cap" style={{ left: `${progressPercent}%` }}></div>
        </div>
        <span className="progress-label">{totalKeys} / {maxLimit} tracked keys ({progressPercent}%)</span>
      </div>

      <div className="panel panel-tight">
        <table>
          <thead>
            <tr>
              <th>service</th>
              <th>attribute key</th>
              <th>unique (hll est.)</th>
              <th>trend</th>
              <th>status</th>
              <th>action</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              // Unique keys instead of index loop fallbacks (Bug 20 & 11)
              <tr key={`${row.service}-${row.attributeKey}`}>
                <td className="mono-cell" style={{ fontWeight: '500' }}>{row.service}</td>
                <td className="mono-cell">{row.attributeKey}</td>
                <td className="mono-cell">{row.uniqueHllEst}</td>
                <td>
                  <svg width="60" height="20">
                    <polyline
                      points={row.trendPoints}
                      fill="none"
                      stroke={row.status === 'breach' ? 'var(--red)' : row.status === 'watch' ? 'var(--amber)' : 'var(--phosphor)'}
                      strokeWidth="1.5"
                    />
                  </svg>
                </td>
                <td>
                  <span className="status-chip">
                    <span className={`rled ${row.status === 'breach' ? 'r' : row.status === 'watch' ? 'a' : 'p'}`}></span>
                    {row.status}
                  </span>
                </td>
                <td>
                  {/* Strict disabled button bindings (Bug 11) */}
                  <button
                    className={`btn ${row.status === 'normal' ? 'btn-ghost' : ''}`}
                    onClick={() => handleRedactClick(row)}
                    disabled={row.status === 'normal'}
                  >
                    redact
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="footnote">
        key-space anomaly detected on checkout-service — dynamic key pattern active. tracking capped, truncation fallback active.
      </div>
    </section>
  );
}
