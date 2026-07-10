import { useState, useEffect } from 'react';
import { AlertCircle, ShieldAlert } from 'lucide-react';
import type { DashboardData } from '../../App';

interface CoverageItem {
  service: string;
  status: string;
  lastSeen: string;
}

interface CoverageProps {
  data: DashboardData;
  tenantId: string;
}

export function Coverage({ data, tenantId }: CoverageProps) {
  const [coverageData, setCoverageData] = useState<CoverageItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<boolean>(false);
  const coverageCount = data?.metrics?.coverage?.value || '1';

  useEffect(() => {
    setLoading(true);
    setError(false);
    
    fetch(`http://localhost:8080/api/v1/tenant/${tenantId}/coverage`)
      .then((r) => {
        if (!r.ok) throw new Error('Failed to load coverage');
        return r.json();
      })
      .then(setCoverageData)
      .catch((err) => {
        console.error(err);
        setError(true);
        // Fallback mock data
        setCoverageData([
          { service: 'inventory-worker', status: 'silent', lastSeen: '14m ago' },
          { service: 'auth-service', status: 'active', lastSeen: '1s ago' },
          { service: 'checkout-service', status: 'active', lastSeen: '2s ago' }
        ]);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [tenantId]);

  return (
    <section className="view active">
      <div className="eyebrow">04 • coverage • sampling gap detector • §8.3</div>
      
      <div className="tag-row">
        <span className="tag">
          active services <b style={{ color: 'var(--phosphor)' }}>{coverageCount}</b>
        </span>
      </div>

      {error && (
        <div style={{ background: 'rgba(239, 68, 68, 0.08)', padding: '10px 16px', borderRadius: '4px', border: '1px solid var(--red)', color: 'var(--red)', marginBottom: '14px', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <AlertCircle size={14} />
          <span>Error loading live coverage. Showing simulated local fallback.</span>
        </div>
      )}

      <div className="panel panel-tight">
        {loading ? (
          <div className="animate-pulse" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <div style={{ height: '14px', background: 'var(--bezel-soft)', borderRadius: '4px', width: '50%' }}></div>
            <div style={{ height: '14px', background: 'var(--bezel-soft)', borderRadius: '4px', width: '70%' }}></div>
          </div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Service</th>
                <th>Status</th>
                <th className="align-right">Last seen</th>
              </tr>
            </thead>
            <tbody>
              {coverageData.length > 0 ? (
                coverageData.map((cov, i) => (
                  <tr key={i}>
                    <td className="mono-cell" style={{ fontWeight: '500' }}>{cov.service}</td>
                    <td>
                      <span className={`badge ${cov.status === 'silent' ? 'badge-err' : 'badge-ok'}`} style={{
                        background: cov.status === 'silent' ? 'var(--red-dim)' : 'var(--phosphor-dim)',
                        color: cov.status === 'silent' ? 'var(--red)' : 'var(--phosphor)'
                      }}>
                        {cov.status === 'silent' ? (
                          <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                            <ShieldAlert size={10} /> {cov.status}
                          </span>
                        ) : (
                          cov.status
                        )}
                      </span>
                    </td>
                    <td className="align-right mono-cell">{cov.lastSeen}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={3} style={{ textAlign: 'center', color: 'var(--muted)', padding: '20px' }}>
                    No coverage data found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>
    </section>
  );
}
