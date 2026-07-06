
import { Layout } from './components/Layout';
import { HealthGauge } from './components/HealthGauge';
import { MetricCard } from './components/MetricCard';
import { RemediationPanel } from './components/RemediationPanel';
import { Hash, GitBranch, Radio } from 'lucide-react';

const MOCK_YAML = `
processors:
  attributes/remediation:
    actions:
      - key: "user_id"
        action: "delete"
`;

function App() {
  return (
    <Layout>
      <div className="grid-dashboard">
        <HealthGauge score={84} />
        
        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
          <div className="grid-metrics">
            <MetricCard 
              title="Cardinality Estimate" 
              value="1.2M" 
              change={14.5} 
              icon={Hash} 
              status="warn" 
              delay="0.1s" 
            />
            <MetricCard 
              title="Orphaned Traces" 
              value="432" 
              change={-5.2} 
              icon={GitBranch} 
              status="good" 
              delay="0.2s" 
            />
            <MetricCard 
              title="Active Services" 
              value="14" 
              change={0} 
              icon={Radio} 
              status="good" 
              delay="0.3s" 
            />
          </div>
          
          <RemediationPanel 
            issueType="High Cardinality (user_id on checkout_service)" 
            yamlConfig={MOCK_YAML} 
          />
        </div>
      </div>
    </Layout>
  );
}

export default App;
