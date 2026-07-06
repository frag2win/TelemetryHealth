import React, { useState } from 'react';
import { Copy, CheckCircle2, Wrench } from 'lucide-react';

interface RemediationPanelProps {
  issueType: string;
  yamlConfig: string;
}

export const RemediationPanel: React.FC<RemediationPanelProps> = ({ issueType, yamlConfig }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(yamlConfig.trim());
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="glass-panel animate-fade-in" style={{ animationDelay: '0.3s', opacity: 0, padding: '32px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '24px' }}>
        <div style={{ padding: '8px', background: 'rgba(59, 130, 246, 0.1)', color: 'var(--accent-blue)', borderRadius: '8px' }}>
          <Wrench size={20} />
        </div>
        <div>
          <h3 style={{ fontSize: '1.1rem', fontWeight: 600 }}>Active Remediation Plan</h3>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>Issue detected: {issueType}</p>
        </div>
      </div>
      
      <div style={{ 
        position: 'relative', 
        background: '#0d1117', 
        borderRadius: '12px', 
        border: '1px solid rgba(255, 255, 255, 0.05)',
        overflow: 'hidden'
      }}>
        <div style={{ 
          display: 'flex', justifyContent: 'space-between', alignItems: 'center', 
          padding: '12px 16px', background: 'rgba(255, 255, 255, 0.02)',
          borderBottom: '1px solid rgba(255, 255, 255, 0.05)'
        }}>
          <span style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', fontFamily: 'monospace' }}>otel-collector-config.yaml</span>
          <button 
            onClick={handleCopy}
            style={{ 
              background: 'transparent', border: 'none', color: copied ? 'var(--status-good)' : 'var(--text-secondary)',
              cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.85rem'
            }}
          >
            {copied ? <CheckCircle2 size={16} /> : <Copy size={16} />}
            {copied ? 'Copied' : 'Copy YAML'}
          </button>
        </div>
        
        <pre style={{ margin: 0, padding: '24px', overflowX: 'auto' }}>
          <code style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace', fontSize: '0.9rem', color: '#e2e8f0', lineHeight: 1.5 }}>
            {yamlConfig.trim()}
          </code>
        </pre>
      </div>
    </div>
  );
};
