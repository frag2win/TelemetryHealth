import type { DashboardData } from '../../App';
import { useTenantData, ErrorBanner, SkeletonLoader } from '../Shared';

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
  const fallbackCoverage: CoverageItem[] = [
    { service: 'inventory-worker', status: 'silent', lastSeen: '14m ago' },
    { service: 'auth-service', status: 'active', lastSeen: '1s ago' }
  ];

  // useTenantData shared hook implements AbortController and proxy compliance
  const { data: coverageData, loading, error, errorMsg } = useTenantData<CoverageItem[]>(
    tenantId,
    'coverage',
    fallbackCoverage
  );

  const coverageItems = coverageData ?? [];

  // Resolve props vs fetch desynchronization (Bug 23): align count badge dynamically
  const activeCount = !loading && coverageItems.length > 0
    ? coverageItems.filter(c => c.status === 'active').length.toString()
    : (data.metrics?.coverage?.value ?? '1');

  return (
    <section className="view active">
      <div className="eyebrow">04 • coverage • sampling gap detector • §8.3</div>
      <div className="tag-row">
        <span className="tag">
          active services <b style={{ color: 'var(--phosphor)' }}>{activeCount}</b>
        </span>
      </div>

      {error && (
        <ErrorBanner message={`Error loading coverage data: ${errorMsg ?? 'Unknown Failure'}. Showing simulated fallback.`} />
      )}

      <div className="panel panel-tight">
        {loading ? (
          <SkeletonLoader rows={2} />
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
              {coverageItems.length > 0 ? (
                coverageItems.map((cov) => (
                  // Swapped key from index to unique service name (Bug 20)
                  <tr key={cov.service}>
                    <td className="mono-cell" style={{ fontWeight: '500' }}>{cov.service}</td>
                    <td>
                      <span className={`badge ${cov.status === 'silent' ? 'badge-err' : 'badge-ok'}`} style={{
                        background: cov.status === 'silent' ? 'var(--red-dim)' : 'var(--phosphor-dim)',
                        color: cov.status === 'silent' ? 'var(--red)' : 'var(--phosphor)'
                      }}>
                        {cov.status}
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
