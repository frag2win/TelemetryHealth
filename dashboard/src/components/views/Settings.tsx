import { Moon, Sun, Sliders, CheckCircle2 } from 'lucide-react';

interface SettingsProps {
  theme: string;
  setTheme: (theme: string) => void;
  selectedTenantId: string;
  setSelectedTenantId: (tenantId: string) => void;
  timeRange: string;
  setTimeRange: (timeRange: string) => void;
  dataSource: 'live' | 'mock';
  setDataSource: (dataSource: 'live' | 'mock') => void;
}

export function Settings({
  theme,
  setTheme,
  selectedTenantId,
  setSelectedTenantId,
  timeRange,
  setTimeRange,
  dataSource,
  setDataSource
}: SettingsProps) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', padding: '4px 0' }}>
      
      {/* Visual Theme & Appearance Panel */}
      <div style={{ background: 'var(--panel)', border: '1px solid var(--bezel)', borderRadius: '8px', padding: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '14px' }}>
          <Sun size={18} style={{ color: 'var(--phosphor)' }} />
          <h2 style={{ fontSize: '14px', fontWeight: 600, margin: 0, color: 'var(--paper)' }}>
            Appearance & Theme
          </h2>
        </div>
        <p style={{ fontSize: '12px', color: 'var(--muted)', marginBottom: '16px' }}>
          Select dashboard visual mode and color palette theme.
        </p>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '14px' }}>
          {/* Dark Mode Card */}
          <div
            onClick={() => setTheme('dark')}
            style={{
              padding: '16px',
              borderRadius: '8px',
              border: `2px solid ${theme === 'dark' ? 'var(--phosphor)' : 'var(--bezel)'}`,
              background: 'var(--ink)',
              cursor: 'pointer',
              display: 'flex',
              flexDirection: 'column',
              gap: '10px',
              transition: 'border-color 0.2s ease'
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontWeight: 600, fontSize: '13px', color: 'var(--paper)' }}>
                <Moon size={16} style={{ color: '#a855f7' }} />
                <span>Dark Obsidian</span>
              </div>
              {theme === 'dark' && <CheckCircle2 size={16} style={{ color: 'var(--phosphor)' }} />}
            </div>
            <div style={{ fontSize: '11px', color: 'var(--muted)' }}>
              High contrast obsidian dark theme designed for operations monitoring.
            </div>
          </div>

          {/* Light Mode Card */}
          <div
            onClick={() => setTheme('light')}
            style={{
              padding: '16px',
              borderRadius: '8px',
              border: `2px solid ${theme === 'light' ? 'var(--phosphor)' : 'var(--bezel)'}`,
              background: 'var(--paper)',
              color: '#000',
              cursor: 'pointer',
              display: 'flex',
              flexDirection: 'column',
              gap: '10px',
              transition: 'border-color 0.2s ease'
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontWeight: 600, fontSize: '13px', color: '#000' }}>
                <Sun size={16} style={{ color: '#f59e0b' }} />
                <span>Light Studio</span>
              </div>
              {theme === 'light' && <CheckCircle2 size={16} style={{ color: 'var(--phosphor)' }} />}
            </div>
            <div style={{ fontSize: '11px', color: '#4b5563' }}>
              Bright studio theme optimized for daytime readability and presentations.
            </div>
          </div>
        </div>
      </div>

      {/* Cluster & Telemetry Configuration Panel */}
      <div style={{ background: 'var(--panel)', border: '1px solid var(--bezel)', borderRadius: '8px', padding: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '14px' }}>
          <Sliders size={18} style={{ color: 'var(--phosphor)' }} />
          <h2 style={{ fontSize: '14px', fontWeight: 600, margin: 0, color: 'var(--paper)' }}>
            Pipeline & Environment Settings
          </h2>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          {/* Data Source */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingBottom: '12px', borderBottom: '1px solid var(--bezel)' }}>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 500, color: 'var(--paper)' }}>Ingestion Analytics Engine</div>
              <div style={{ fontSize: '11px', color: 'var(--muted)' }}>Switch between live ClickHouse queries and local simulation</div>
            </div>
            <select
              value={dataSource}
              onChange={(e) => setDataSource(e.target.value as 'live' | 'mock')}
              className="select-dropdown"
              style={{ minWidth: '150px' }}
            >
              <option value="live">🟢 Live ClickHouse</option>
              <option value="mock">🔶 Local Simulator</option>
            </select>
          </div>

          {/* Time Window */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingBottom: '12px', borderBottom: '1px solid var(--bezel)' }}>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 500, color: 'var(--paper)' }}>Default Metrics Window</div>
              <div style={{ fontSize: '11px', color: 'var(--muted)' }}>Lookback time range for composite health calculations</div>
            </div>
            <select
              value={timeRange}
              onChange={(e) => setTimeRange(e.target.value)}
              className="select-dropdown"
              style={{ minWidth: '150px' }}
            >
              <option value="1h">Last 1 hour</option>
              <option value="6h">Last 6 hours</option>
              <option value="24h">Last 24 hours</option>
              <option value="7d">Last 7 days</option>
            </select>
          </div>

          {/* Target Tenant */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <div style={{ fontSize: '13px', fontWeight: 500, color: 'var(--paper)' }}>Active Tenant Environment</div>
              <div style={{ fontSize: '11px', color: 'var(--muted)' }}>Multi-tenant mTLS SPIFFE SAN identity context</div>
            </div>
            <select
              value={selectedTenantId}
              onChange={(e) => setSelectedTenantId(e.target.value)}
              className="select-dropdown"
              style={{ minWidth: '180px' }}
            >
              <option value="00000000-0000-0000-0000-000000000001">acme-prod (Production)</option>
              <option value="00000000-0000-0000-0000-000000000002">acme-staging (Staging)</option>
              <option value="tenant-alpha">tenant-alpha (Enterprise A)</option>
              <option value="tenant-beta">tenant-beta (Enterprise B)</option>
            </select>
          </div>
        </div>
      </div>

    </div>
  );
}
