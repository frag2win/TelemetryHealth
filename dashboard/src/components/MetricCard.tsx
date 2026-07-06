import React from 'react';
import { DivideIcon as LucideIcon } from 'lucide-react';

interface MetricCardProps {
  title: string;
  value: string | number;
  change: number;
  icon: typeof LucideIcon;
  status: 'good' | 'warn' | 'crit';
  delay: string;
}

export const MetricCard: React.FC<MetricCardProps> = ({ title, value, change, icon: Icon, status, delay }) => {
  const statusColors = {
    good: 'var(--status-good)',
    warn: 'var(--status-warn)',
    crit: 'var(--status-crit)',
  };

  const isPositive = change >= 0;
  
  return (
    <div className="glass-panel animate-fade-in" style={{ padding: '24px', animationDelay: delay, opacity: 0 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '24px' }}>
        <div style={{ 
          padding: '10px', 
          borderRadius: '12px', 
          background: `rgba(255, 255, 255, 0.05)`,
          color: statusColors[status]
        }}>
          <Icon size={24} />
        </div>
        <div style={{ 
          display: 'flex', alignItems: 'center', gap: '4px',
          padding: '4px 8px', borderRadius: '20px',
          background: isPositive ? 'rgba(16, 185, 129, 0.1)' : 'rgba(239, 68, 68, 0.1)',
          color: isPositive ? 'var(--status-good)' : 'var(--status-crit)',
          fontSize: '0.85rem', fontWeight: 600
        }}>
          {isPositive ? '+' : ''}{change}%
        </div>
      </div>
      
      <div>
        <h4 style={{ color: 'var(--text-secondary)', fontSize: '0.95rem', fontWeight: 500, marginBottom: '8px' }}>
          {title}
        </h4>
        <div style={{ fontSize: '2.5rem', fontWeight: 700, color: 'var(--text-primary)' }}>
          {value}
        </div>
      </div>
    </div>
  );
};
