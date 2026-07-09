import { useState, useEffect } from 'react';

export function Coverage({ data }: { data?: any }) {
  const [coverageData, setCoverageData] = useState<any[]>([]);
  const coverageCount = data?.metrics?.coverage?.value || 1;

  useEffect(() => {
    fetch('/api/v1/tenant/acme-prod/coverage')
      .then(r => r.json())
      .then(setCoverageData)
      .catch(console.error);
  }, []);

  return (
    <section className="view active">
      <div className="eyebrow">04 &#183; coverage &#183; sampling gap detector &#183; §8.3</div>
      <div className="tag-row">
        <span className="tag">active services <b>{coverageCount}</b></span>
      </div>
      <div className="panel panel-tight">
        <table>
          <thead>
            <tr>
              <th>Service</th>
              <th>Status</th>
              <th className="align-right">Last seen</th>
            </tr>
          </thead>
          <tbody>
            {coverageData.length > 0 ? coverageData.map((cov, i) => (
              <tr key={i}>
                <td>{cov.service}</td>
                <td><span className={`badge ${cov.status === 'silent' ? 'badge-err' : 'badge-ok'}`}>{cov.status}</span></td>
                <td className="align-right">{cov.lastSeen}</td>
              </tr>
            )) : (
              <tr><td colSpan={3}>No coverage data found.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
