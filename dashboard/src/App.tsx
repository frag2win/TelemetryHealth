import { useEffect, useState } from 'react';
import { Overview } from './components/views/Overview';
import { Cardinality } from './components/views/Cardinality';
import { TraceChains } from './components/views/TraceChains';
import { Coverage } from './components/views/Coverage';
import { Remediation } from './components/views/Remediation';
import { AgentTraces } from './components/views/AgentTraces';
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
  tenantId?: string;
  version?: string;
}

const titles: Record<string, string> = {
  overview: '01 / OVERVIEW',
  cardinality: '02 / CARDINALITY',
  tracechains: '03 / TRACE CHAINS',
  coverage: '04 / COVERAGE',
  remediation: '05 / REMEDIATION',
  agenttraces: '06 / AI AGENTS'
};

function App() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [activeView, setActiveView] = useState('overview');
  const [env, setEnv] = useState('production');
  const [timeRange, setTimeRange] = useState('24h');

  useEffect(() => {
    const fetchData = () => {
      const fallbackData = {
        healthScore: 78,
        metrics: {
          cardinality: { value: "3", change: 1 },
          orphans: { value: "6.2%", change: 1.2 },
          coverage: { value: "1", change: -1 }
        },
        remediation: {
          issueType: "cardinality_spike",
          yaml: "apiVersion: telemetry.v1\nkind: Remediation\nspec:\n  action: drop_high_cardinality\n  target: user_id_raw"
        }
      };

      // Hybrid integration: Try real SigNoz / TelemetryHealth API first via Vite proxy, fallback to demo data if offline
      fetch(`/api/v1/tenant/${env}/health?range=${timeRange}`)
        .then(r => {
          if (!r.ok) throw new Error("API status not OK");
          return r.json();
        })
        .then(apiData => {
          if (apiData && typeof apiData.healthScore === 'number') {
            setData(apiData);
          } else {
            setData(fallbackData);
          }
        })
        .catch(() => {
          // Fallback so UI remains functional during demo
          setData(fallbackData);
        });
    };

    fetchData();
    const interval = setInterval(fetchData, 15000);

    return () => clearInterval(interval);
  }, [env, timeRange]);

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
          {navItem('agenttraces', '06', 'AI Agents', 'on-p')}
        </nav>
        <div className="sidebar-foot">
          tenant: {modifiedData.tenantId || 'acme-prod'}<br/>
          region: us-east-1<br/>
          {modifiedData.version || 'v1.0.0-ga'}
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
            <button className={`pill ${timeRange === '1h' ? 'active' : ''}`} onClick={() => setTimeRange('1h')}>1h</button>
            <button className={`pill ${timeRange === '24h' ? 'active' : ''}`} onClick={() => setTimeRange('24h')}>24h</button>
            <button className={`pill ${timeRange === '7d' ? 'active' : ''}`} onClick={() => setTimeRange('7d')}>7d</button>
          </div>
        </div>

        <div className="content">
          {activeView === 'overview' && <Overview data={modifiedData} setView={setActiveView} />}
          {activeView === 'cardinality' && <Cardinality />}
          {activeView === 'tracechains' && <TraceChains data={modifiedData} />}
          {activeView === 'coverage' && <Coverage data={modifiedData} />}
          {activeView === 'remediation' && <Remediation apiRemediation={modifiedData.remediation} />}
          {activeView === 'agenttraces' && <AgentTraces />}
        </div>
      </div>
    </>
  );
}

export default App;
