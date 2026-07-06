import React, { useEffect, useState } from 'react';

interface HealthGaugeProps {
  score: number;
}

export const HealthGauge: React.FC<HealthGaugeProps> = ({ score }) => {
  const [animatedScore, setAnimatedScore] = useState(0);

  useEffect(() => {
    const timer = setTimeout(() => setAnimatedScore(score), 100);
    return () => clearTimeout(timer);
  }, [score]);

  // Determine color based on score
  let color = 'var(--status-good)';
  let glow = 'rgba(16, 185, 129, 0.4)';
  if (score < 70) {
    color = 'var(--status-crit)';
    glow = 'rgba(239, 68, 68, 0.4)';
  } else if (score < 90) {
    color = 'var(--status-warn)';
    glow = 'rgba(245, 158, 11, 0.4)';
  }

  const radius = 90;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (animatedScore / 100) * circumference;

  return (
    <div className="glass-panel flex-center" style={{ padding: '40px', flexDirection: 'column', height: '100%' }}>
      <h3 style={{ alignSelf: 'flex-start', marginBottom: 'auto', color: 'var(--text-secondary)', fontSize: '1rem' }}>
        Composite Health Score
      </h3>
      
      <div style={{ position: 'relative', width: '240px', height: '240px', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '32px 0' }}>
        <svg width="240" height="240" style={{ transform: 'rotate(-90deg)', filter: `drop-shadow(0 0 12px ${glow})` }}>
          {/* Background circle */}
          <circle
            cx="120" cy="120" r={radius}
            fill="transparent"
            stroke="rgba(255, 255, 255, 0.05)"
            strokeWidth="16"
          />
          {/* Animated progress circle */}
          <circle
            cx="120" cy="120" r={radius}
            fill="transparent"
            stroke={color}
            strokeWidth="16"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            style={{ transition: 'stroke-dashoffset 1.5s cubic-bezier(0.4, 0, 0.2, 1), stroke 0.5s ease' }}
          />
        </svg>
        
        <div style={{ position: 'absolute', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
          <span style={{ fontSize: '4rem', fontWeight: 700, lineHeight: 1, color: 'var(--text-primary)' }}>
            {Math.round(animatedScore)}
          </span>
          <span style={{ fontSize: '1rem', color: color, fontWeight: 600, marginTop: '8px' }}>
            / 100
          </span>
        </div>
      </div>
      
      <div style={{ alignSelf: 'flex-start', marginTop: 'auto' }}>
        <p style={{ fontSize: '0.9rem', color: 'var(--text-secondary)' }}>
          Based on cardinality, trace correlation, and coverage metrics across 14 services.
        </p>
      </div>
    </div>
  );
};
