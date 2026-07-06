import { useEffect, useState } from 'react';
import { Layout } from './components/Layout';
import { HealthGauge } from './components/HealthGauge';
import { MetricCard } from './components/MetricCard';
import { RemediationPanel } from './components/RemediationPanel';
import { Hash, GitBranch, Radio, Loader2 } from 'lucide-react';

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

function App() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Fetch data from Go API
    fetch('http://localhost:8080/api/v1/tenant/tenant-123/health')
      .then((res) => {
        if (!res.ok) throw new Error('Failed to fetch data');
        return res.json();
      })
      .then((json) => setData(json))
      .catch((err) => setError(err.message));
  }, []);

  if (error) {
    return (
      <Layout>
        <div className="glass-panel" style={{ padding: '40px', textAlign: 'center', color: 'var(--status-crit)' }}>
          <h3>Error loading dashboard</h3>
          <p>{error}</p>
        </div>
      </Layout>
    );
  }

  if (!data) {
    return (
      <Layout>
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', color: 'var(--text-secondary)' }}>
          <Loader2 size={48} className="animate-spin" style={{ animation: 'spin 1s linear infinite' }} />
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="grid-dashboard">
        <HealthGauge score={data.healthScore} />
        
        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
          <div className="grid-metrics">
            <MetricCard 
              title="Cardinality Estimate" 
              value={data.metrics.cardinality.value} 
              change={data.metrics.cardinality.change} 
              icon={Hash} 
              status="warn" 
              delay="0.1s" 
            />
            <MetricCard 
              title="Orphaned Traces" 
              value={data.metrics.orphans.value} 
              change={data.metrics.orphans.change} 
              icon={GitBranch} 
              status="good" 
              delay="0.2s" 
            />
            <MetricCard 
              title="Active Services" 
              value={data.metrics.coverage.value} 
              change={data.metrics.coverage.change} 
              icon={Radio} 
              status="good" 
              delay="0.3s" 
            />
          </div>
          
          <RemediationPanel 
            issueType={data.remediation.issueType} 
            yamlConfig={data.remediation.yaml} 
          />
        </div>
      </div>
    </Layout>
  );
}

export default App;
