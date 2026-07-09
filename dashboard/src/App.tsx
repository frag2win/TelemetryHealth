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
  const [dataSource, setDataSource] = useState<'live'|'mock'>('live');
  const [theme, setTheme] = useState('dark');
  const [lastFetched, setLastFetched] = useState<Date | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey) {
        switch (e.key) {
          case '1': e.preventDefault(); setActiveView('overview'); break;
          case '2': e.preventDefault(); setActiveView('cardinality'); break;
          case '3': e.preventDefault(); setActiveView('tracechains'); break;
          case '4': e.preventDefault(); setActiveView('coverage'); break;
          case '5': e.preventDefault(); setActiveView('remediation'); break;
          case '6': e.preventDefault(); setActiveView('agenttraces'); break;
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

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

      fetch(`/api/v1/tenant/acme-${env}/health?range=${timeRange}`)
        .then(r => {
          if (!r.ok) throw new Error('API Error');
          return r.json();
        })
        .then(resData => {
          setData(resData);
          setDataSource('live');
          setLastFetched(new Date());
        })
        .catch(err => {
          console.error(err);
          setData(fallbackData);
          setDataSource('mock');
          setLastFetched(new Date());
        });
    };

    fetchData();
    const interval = setInterval(fetchData, 15000);

    return () => clearInterval(interval);
  }, [env, timeRange, refreshTrigger]);

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
          <div className={`pill ${dataSource === 'live' ? 'live' : 'mock'}`} style={{ marginLeft: '12px', border: '1px solid var(--panel-3)' }}>
            {dataSource === 'live' ? '🟢 Live' : '🔶 Mock fallback'}
          </div>
          <div className="spacer"></div>
          
          <div className="text-muted" style={{ fontSize: '12px', marginRight: '12px' }}>
            Last updated: {lastFetched?.toLocaleTimeString() || '...'}
          </div>
          
          <button className="btn" style={{ padding: '4px 8px', marginRight: '8px' }} onClick={() => setRefreshTrigger(t => t + 1)} title="Refresh data">
            🔄
          </button>
          
          <button className="btn" style={{ padding: '4px 8px', marginRight: '8px' }} onClick={() => {
            const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `telemetry-health-${env}-${timeRange}.json`;
            a.click();
          }} title="Export JSON">
            Export
          </button>
          
          <button className="btn" style={{ padding: '4px 8px', marginRight: '16px' }} onClick={() => {
            const newTheme = theme === 'dark' ? 'light' : 'dark';
            setTheme(newTheme);
            document.documentElement.setAttribute('data-theme', newTheme);
          }} title="Toggle Theme">
            {theme === 'dark' ? '☀️' : '🌙'}
          </button>
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
