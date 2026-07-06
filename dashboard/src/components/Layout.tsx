import React from 'react';
import { Activity, ShieldAlert, Settings, Layers } from 'lucide-react';

interface LayoutProps {
  children: React.ReactNode;
}

const navItems = [
  { icon: Activity, label: 'Health Dashboard', active: true },
  { icon: ShieldAlert, label: 'Remediation', active: false },
  { icon: Layers, label: 'Tenants', active: false },
  { icon: Settings, label: 'Settings', active: false },
];

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  return (
    <div className="app-container">
      <aside className="sidebar">
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{ 
            width: '32px', height: '32px', 
            background: 'linear-gradient(135deg, var(--accent-blue), #8b5cf6)',
            borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center'
          }}>
            <Activity size={20} color="white" />
          </div>
          <h2 style={{ fontSize: '1.25rem', fontWeight: 700, letterSpacing: '-0.5px' }}>TelemetryHealth</h2>
        </div>

        <nav style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {navItems.map((item, idx) => (
            <button
              key={idx}
              style={{
                display: 'flex', alignItems: 'center', gap: '12px',
                width: '100%', padding: '12px 16px',
                borderRadius: '8px', border: 'none',
                background: item.active ? 'rgba(59, 130, 246, 0.1)' : 'transparent',
                color: item.active ? 'var(--accent-blue)' : 'var(--text-secondary)',
                cursor: 'pointer', textAlign: 'left',
                fontSize: '0.95rem', fontWeight: 500,
                transition: 'all 0.2s ease'
              }}
              onMouseEnter={(e) => {
                if (!item.active) e.currentTarget.style.color = 'var(--text-primary)';
              }}
              onMouseLeave={(e) => {
                if (!item.active) e.currentTarget.style.color = 'var(--text-secondary)';
              }}
            >
              <item.icon size={18} />
              {item.label}
            </button>
          ))}
        </nav>
      </aside>

      <main className="main-content">
        <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <h1 style={{ fontSize: '2rem', marginBottom: '8px' }}>Tenant Health Overview</h1>
            <p style={{ color: 'var(--text-secondary)' }}>Monitoring telemetry signals for tenant-123</p>
          </div>
          <div className="glass-panel" style={{ padding: '8px 16px', display: 'flex', alignItems: 'center', gap: '12px', borderRadius: '30px' }}>
            <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: 'var(--status-good)', boxShadow: '0 0 10px var(--status-good)' }} />
            <span style={{ fontSize: '0.9rem', fontWeight: 500 }}>System Healthy</span>
          </div>
        </header>

        {children}
      </main>
    </div>
  );
};
