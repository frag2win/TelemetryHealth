import { useEffect, useState } from 'react';
import { Overview } from './components/views/Overview';
import { Cardinality } from './components/views/Cardinality';
import { TraceChains } from './components/views/TraceChains';
import { Coverage } from './components/views/Coverage';
import { Remediation } from './components/views/Remediation';
import { Loader2 } from 'lucide-react';

interface DashboardData {
  healthScore: number;
  metrics: {
    cardinality: { value: string; change: number };
    orphans: { value: string; change: number };
    coverage: { value: string; change: number };
  };
  remediation: {
    issueType: string;
    yaml: string;
  };
}

const titles: Record<string, string> = {
  overview: '01 / OVERVIEW',
  cardinality: '02 / CARDINALITY',
  tracechains: '03 / TRACE CHAINS',
  coverage: '04 / COVERAGE',
  remediation: '05 / REMEDIATION'
};

function App() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [activeView, setActiveView] = useState('overview');
  const [env, setEnv] = useState('production');

  useEffect(() => {
    fetch('http://localhost:8080/api/v1/tenant/tenant-123/health')
      .then((res) => {
        if (!res.ok) throw new Error('Failed to fetch data');
        return res.json();
      })
      .then((json) => setData(json))
      .catch((err) => setError(err.message));
  }, []);

  const navItem = (id: string, chan: string, label: string, ledClass: string) => (
    <button 
      className={`nav-item ${activeView === id ? 'active' : ''}`} 
      onClick={() => setActiveView(id)}
    >
      <span className="chan">{chan}</span>
      <span className="lbl">{label}</span>
      <span className={`led ${ledClass}`}></span>
    </button>
  );

  if (error) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', width: '100vw', color: 'var(--red)' }}>
        <h3>Error loading dashboard: {error}</h3>
      </div>
    );
  }

  if (!data) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', width: '100vw', color: 'var(--muted)' }}>
        <Loader2 size={48} className="animate-spin" style={{ animation: 'spin 1s linear infinite' }} />
      </div>
    );
  }

  const modifiedData = { ...data };
  if (env === 'staging') {
    modifiedData.healthScore = 91;
  } else {
    modifiedData.healthScore = data.healthScore || 78;
  }

  return (
    <>
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">TELEMETRY<span>HEALTH</span></div>
          <div className="brand-sub">pipeline health monitor</div>
        </div>
        <nav className="nav">
          {navItem('overview', '01', 'Overview', 'on-a')}
          {navItem('cardinality', '02', 'Cardinality', 'on-r')}
          {navItem('tracechains', '03', 'Trace chains', 'on-r')}
          {navItem('coverage', '04', 'Coverage', 'on-a')}
          {navItem('remediation', '05', 'Remediation', 'on-p')}
        </nav>
        <div className="sidebar-foot">
          tenant: acme-prod<br/>
          region: us-east-1<br/>
          v1.0.0-ga
        </div>
      </aside>

      <div className="main-content-area">
        <div className="topbar">
          <div className="topbar-title">
            {titles[activeView].split(' / ').map((part, i) => (
              <span key={i}>
                {i > 0 && <span className="dim"> / </span>}
                {part}
              </span>
            ))}
          </div>
          <div className="live"><span className="live-dot"></span>LIVE</div>
          <div className="spacer"></div>
          <div className="pillgroup">
            <button className={`pill ${env === 'production' ? 'active' : ''}`} onClick={() => setEnv('production')}>production</button>
            <button className={`pill ${env === 'staging' ? 'active' : ''}`} onClick={() => setEnv('staging')}>staging</button>
          </div>
          <div className="pillgroup">
            <button className="pill">1h</button>
            <button className="pill active">24h</button>
            <button className="pill">7d</button>
          </div>
        </div>

        <div className="content">
          {activeView === 'overview' && <Overview data={modifiedData} setView={setActiveView} />}
          {activeView === 'cardinality' && <Cardinality />}
          {activeView === 'tracechains' && <TraceChains />}
          {activeView === 'coverage' && <Coverage />}
          {activeView === 'remediation' && <Remediation apiRemediation={modifiedData.remediation} />}
        </div>
      </div>
    </>
  );
}

export default App;
