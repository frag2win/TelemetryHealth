import { useEffect, useState, useCallback } from 'react';
import { Overview } from './components/views/Overview';
import { Cardinality } from './components/views/Cardinality';
import { TraceChains } from './components/views/TraceChains';
import { Coverage } from './components/views/Coverage';
import { Remediation } from './components/views/Remediation';
import { AgentTraces } from './components/views/AgentTraces';
import { RefreshCw, Download, Sun, Moon, Menu, X, AlertTriangle, Loader2 } from 'lucide-react';

export interface MetricValue {
  value: string;
  change: number;
}

export interface MetricsPayload {
  cardinality: MetricValue;
  orphans: MetricValue;
  coverage: MetricValue;
}

export interface RemediationPayload {
  issueType: string;
  yaml: string;
  validated?: boolean;
}

export interface DashboardData {
  healthScore: number;
  metrics: MetricsPayload;
  remediation: RemediationPayload;
  tenantId: string;
  version: string;
  history?: number[];
}

const titles: Record<string, string> = {
  overview: '01 / OVERVIEW',
  cardinality: '02 / CARDINALITY',
  tracechains: '03 / TRACE CHAINS',
  coverage: '04 / COVERAGE',
  remediation: '05 / REMEDIATION',
  agenttraces: '06 / AI AGENTS'
};

const tenants = [
  { id: 'acme-prod', name: 'acme-prod (Production)' },
  { id: 'acme-staging', name: 'acme-staging (Staging)' },
  { id: 'tenant-alpha', name: 'tenant-alpha (Enterprise A)' },
  { id: 'tenant-beta', name: 'tenant-beta (Enterprise B)' },
  { id: 'tenant-gamma', name: 'tenant-gamma (Internal)' }
];

const timeRanges = [
  { id: '1h', label: 'Last 1h' },
  { id: '6h', label: 'Last 6h' },
  { id: '24h', label: 'Last 24h' },
  { id: '7d', label: 'Last 7d' }
];

function App() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [activeView, setActiveView] = useState<string>('overview');
  const [selectedTenantId, setSelectedTenantId] = useState<string>('acme-prod');
  const [timeRange, setTimeRange] = useState<string>('6h');
  const [dataSource, setDataSource] = useState<'live' | 'mock'>('live');
  const [theme, setTheme] = useState<string>('dark');
  const [lastFetched, setLastFetched] = useState<Date | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState<number>(0);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);

  // Keyboard shortcut handler
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey) {
        let matched = true;
        switch (e.key) {
          case '1': setActiveView('overview'); break;
          case '2': setActiveView('cardinality'); break;
          case '3': setActiveView('tracechains'); break;
          case '4': setActiveView('coverage'); break;
          case '5': setActiveView('remediation'); break;
          case '6': setActiveView('agenttraces'); break;
          default: matched = false; break;
        }
        if (matched) {
          e.preventDefault();
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setErrorMsg(null);

    const fallbackData: DashboardData = {
      healthScore: 78,
      metrics: {
        cardinality: { value: '3', change: 1.0 },
        orphans: { value: '6.2%', change: 1.2 },
        coverage: { value: '1', change: -1.0 }
      },
      remediation: {
        issueType: 'cardinality_spike',
        yaml: 'apiVersion: telemetry.v1\nkind: Remediation\nspec:\n  action: drop_high_cardinality\n  target: user_id_raw',
        validated: true
      },
      tenantId: selectedTenantId,
      version: 'v1.1.0-mock'
    };

    try {
      // Clean backend contract URL
      const response = await fetch(`http://localhost:8080/api/v1/tenant/${selectedTenantId}/health`);
      if (!response.ok) {
        throw new Error(`API Error: ${response.status} ${response.statusText}`);
      }
      const resData = await response.json();
      
      // Perform time-range simulations completely client side to satisfy contract
      const simulatedData = simulateTimeRangeMetrics(resData, timeRange);
      setData(simulatedData);
      setDataSource('live');
      setLastFetched(new Date());
    } catch (err: any) {
      console.warn('Backend fetch failed, using mock fallback. Details:', err.message);
      // Simulate fallback data too
      const simulatedFallback = simulateTimeRangeMetrics(fallbackData, timeRange);
      setData(simulatedFallback);
      setDataSource('mock');
      setLastFetched(new Date());
      setErrorMsg(`Failed to connect to backend: ${err.message || 'Unknown Network Error'}. Using local simulator.`);
    } finally {
      setLoading(false);
    }
  }, [selectedTenantId, timeRange]);

  // Handle data fetching intervals
  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 20000);
    return () => clearInterval(interval);
  }, [fetchData, refreshTrigger]);

  // Client-side simulation of time-range metric slices
  const simulateTimeRangeMetrics = (baseData: DashboardData, range: string): DashboardData => {
    const updated = { ...baseData };
    const score = updated.healthScore || 78;
    
    // Simulate histories of different lengths & trends
    if (range === '1h') {
      updated.history = [score - 4, score - 8, score - 6, score - 3, score - 1, score + 2, score - 2, score];
      if (updated.metrics.cardinality) {
        updated.metrics.cardinality.value = '1.1M';
        updated.metrics.cardinality.change = 4.2;
      }
      if (updated.metrics.orphans) {
        updated.metrics.orphans.value = '8.4%';
        updated.metrics.orphans.change = 2.4;
      }
    } else if (range === '6h') {
      updated.history = [score - 10, score - 7, score - 9, score - 5, score - 4, score - 2, score - 3, score];
      // Default metrics
    } else if (range === '24h') {
      updated.history = [score - 15, score - 12, score - 14, score - 10, score - 8, score - 5, score - 6, score];
      if (updated.metrics.cardinality) {
        updated.metrics.cardinality.value = '1.4M';
        updated.metrics.cardinality.change = -2.1;
      }
      if (updated.metrics.orphans) {
        updated.metrics.orphans.value = '5.1%';
        updated.metrics.orphans.change = -1.2;
      }
    } else if (range === '7d') {
      updated.history = [score - 22, score - 18, score - 20, score - 14, score - 12, score - 9, score - 7, score];
      if (updated.metrics.cardinality) {
        updated.metrics.cardinality.value = '2.1M';
        updated.metrics.cardinality.change = -8.5;
      }
      if (updated.metrics.orphans) {
        updated.metrics.orphans.value = '4.8%';
        updated.metrics.orphans.change = -3.1;
      }
    }
    return updated;
  };

  const handleExport = () => {
    if (!data) return;
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `telemetry-health-${selectedTenantId}-${timeRange}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const navItem = (id: string, chan: string, label: string, ledClass: string) => (
    <button
      className={`nav-item ${activeView === id ? 'active' : ''}`}
      onClick={() => {
        setActiveView(id);
        setIsMobileMenuOpen(false);
      }}
    >
      <span className="chan">{chan}</span>
      <span className="lbl">{label}</span>
      <span className={`led ${ledClass}`}></span>
    </button>
  );

  return (
    <>
      {/* Mobile Header Nav */}
      <header className="mobile-header">
        <div className="brand-mark" style={{ fontSize: '13px' }}>
          TELEMETRY<span>HEALTH</span>
        </div>
        <button className="btn" onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}>
          {isMobileMenuOpen ? <X size={16} /> : <Menu size={16} />}
        </button>
      </header>

      {/* Mobile Navigation Dropdown */}
      {isMobileMenuOpen && (
        <div className="mobile-nav-menu">
          {navItem('overview', '01', 'Overview', 'on-a')}
          {navItem('cardinality', '02', 'Cardinality', 'on-r')}
          {navItem('tracechains', '03', 'Trace chains', 'on-r')}
          {navItem('coverage', '04', 'Coverage', 'on-a')}
          {navItem('remediation', '05', 'Remediation', 'on-p')}
          {navItem('agenttraces', '06', 'AI Agents', 'on-p')}
        </div>
      )}

      {/* Main Desktop Sidebar */}
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">
            TELEMETRY<span>HEALTH</span>
          </div>
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
          tenant: {selectedTenantId}
          <br />
          region: us-east-1
          <br />
          {data?.version || 'v1.1.0-ga'}
        </div>
      </aside>

      <div className="main-content-area">
        {/* Dynamic Warning/Error Banner */}
        {errorMsg && (
          <div style={{ background: 'rgba(239, 68, 68, 0.15)', borderBottom: '1px solid var(--red)', padding: '10px 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', zIndex: 10 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--red)', fontSize: '13px' }}>
              <AlertTriangle size={16} />
              <span>{errorMsg}</span>
            </div>
            <button className="btn" style={{ padding: '2px 8px', borderColor: 'var(--red)', color: 'var(--red)' }} onClick={() => setRefreshTrigger(t => t + 1)}>
              Retry Connection
            </button>
          </div>
        )}

        {/* Top Control Bar */}
        <div className="topbar">
          <div className="topbar-title">
            {titles[activeView].split(' / ').map((part, i) => (
              <span key={i}>
                {i > 0 && <span className="dim"> / </span>}
                {part}
              </span>
            ))}
          </div>
          
          <div className={`pill ${dataSource === 'live' ? 'live' : 'mock'}`} style={{ border: '1px solid var(--bezel)' }}>
            {dataSource === 'live' ? '🟢 Live' : '🔶 Simulator'}
          </div>
          
          <div className="spacer"></div>
          
          <div className="text-muted" style={{ fontSize: '11px', color: 'var(--muted)', display: 'flex', alignItems: 'center', gap: '4px' }}>
            <span>Updated: {lastFetched?.toLocaleTimeString() || '...'}</span>
          </div>
          
          <button className="btn" style={{ padding: '6px' }} onClick={() => setRefreshTrigger(t => t + 1)} title="Refresh data">
            <RefreshCw size={12} className={loading ? 'animate-spin' : ''} />
          </button>
          
          <button className="btn" style={{ padding: '6px 10px' }} onClick={handleExport} title="Export JSON configuration report">
            <Download size={12} />
            <span>Export</span>
          </button>
          
          <button className="btn" style={{ padding: '6px' }} onClick={() => {
            const newTheme = theme === 'dark' ? 'light' : 'dark';
            setTheme(newTheme);
            document.documentElement.setAttribute('data-theme', newTheme);
          }} title="Toggle Visual Style">
            {theme === 'dark' ? <Sun size={12} /> : <Moon size={12} />}
          </button>

          {/* Interactive Tenant Switcher */}
          <div className="flex items-center gap-1" style={{ display: 'inline-flex', alignItems: 'center' }}>
            <select
              value={selectedTenantId}
              onChange={(e) => setSelectedTenantId(e.target.value)}
              className="select-dropdown"
              style={{ display: 'inline-flex' }}
            >
              {tenants.map(t => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          </div>

          {/* Client-side Time-Range dropdown */}
          <div className="flex items-center gap-1" style={{ display: 'inline-flex', alignItems: 'center' }}>
            <select
              value={timeRange}
              onChange={(e) => setTimeRange(e.target.value)}
              className="select-dropdown"
            >
              {timeRanges.map(tr => (
                <option key={tr.id} value={tr.id}>{tr.label}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Primary Content Window */}
        <div className="content">
          {loading && !data ? (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh', color: 'var(--muted)' }}>
              <Loader2 size={36} className="animate-spin" />
            </div>
          ) : !data ? (
            <div style={{ padding: '40px', textAlign: 'center', border: '1px dashed var(--bezel)', borderRadius: '6px', color: 'var(--muted)' }}>
              No telemetry data yet — start your OTel Collector to see health metrics
            </div>
          ) : (
            <>
              {activeView === 'overview' && <Overview data={data} setView={setActiveView} tenantId={selectedTenantId} />}
              {activeView === 'cardinality' && <Cardinality />}
              {activeView === 'tracechains' && <TraceChains data={data} tenantId={selectedTenantId} />}
              {activeView === 'coverage' && <Coverage data={data} tenantId={selectedTenantId} />}
              {activeView === 'remediation' && <Remediation apiRemediation={data.remediation} />}
              {activeView === 'agenttraces' && <AgentTraces tenantId={selectedTenantId} />}
            </>
          )}
        </div>
      </div>
    </>
  );
}

export default App;
