import { useState, useEffect, useRef, useCallback } from 'react';
import { getAuthHeaders } from '../auth';
import { Info, ArrowUpRight, ArrowDownRight, Check, AlertTriangle } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { DashboardData } from '../App';
import type { Node, Edge } from '@xyflow/react';

export interface ViewProps {
  data: DashboardData;
  tenantId: string;
}

// Shared graph data type used by RootCauseGraph and DigitalTwin (Dup 4 fix)
export interface GraphData {
  nodes: Node[];
  edges: Edge[];
}

// Shared status color resolver (Dup 2 fix)
export function getStatusColor(status: string): string {
  if (status === 'error') return 'var(--red)';
  if (status === 'warning') return 'var(--amber)';
  return 'var(--phosphor)';
}

// Shared Toast notification component (Dup 1 fix)
interface ToastProps {
  message: string;
  isError?: boolean;
}

export function Toast({ message, isError = false }: ToastProps) {
  return (
    <div
      style={{
        position: 'fixed',
        bottom: '1rem',
        right: '1rem',
        background: 'var(--toast-bg)',
        border: `1px solid ${isError ? 'var(--red)' : 'var(--toast-border)'}`,
        padding: '12px 24px',
        borderRadius: '4px',
        color: isError ? 'var(--red)' : 'var(--phosphor)',
        zIndex: 9999,
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        boxShadow: 'var(--shadow-sm)',
        fontSize: '13px'
      }}
    >
      {isError ? <AlertTriangle size={16} /> : <Check size={16} />}
      <span style={{ fontWeight: '500' }}>{message}</span>
    </div>
  );
}

// 1. useTenantData custom hook implementing AbortController and proxy compliance
export function useTenantData<T>(tenantId: string, endpoint: string, fallbackData: T) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState<number>(0);

  const fallbackRef = useRef(fallbackData);
  useEffect(() => {
    fallbackRef.current = fallbackData;
  }, [fallbackData]);

  const refetch = useCallback(() => {
    setRefreshTrigger(prev => prev + 1);
  }, []);

  useEffect(() => {
    setData(null);
    setLoading(true);
    setError(false);
    setErrorMsg(null);

    const controller = new AbortController();
    const { signal } = controller;

    // Strict proxy relative path enforcement
    const url = endpoint.startsWith('/') ? endpoint : `/api/v1/tenant/${tenantId}/${endpoint}`;

    fetch(url, { 
      signal,
      headers: getAuthHeaders()
    })
      .then((r) => {
        if (!r.ok) throw new Error(`API error: ${r.status} ${r.statusText}`);
        return r.json();
      })
      .then((resData) => {
        setData(resData);
        setError(false);
        setLoading(false);
      })
      .catch((err: any) => {
        if (err.name === 'AbortError') {
          return;
        }
        console.warn(`Fetch to ${url} failed, loading mock fallback. Details:`, err.message);
        setError(true);
        setErrorMsg(err.message ?? 'Network error');
        setData(fallbackRef.current);
        setLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, [tenantId, endpoint, refreshTrigger]);

  return { data, loading, error, errorMsg, refetch };
}

// 2. Shared Error Banner component
interface ErrorBannerProps {
  message: string;
}

export function ErrorBanner({ message }: ErrorBannerProps) {
  return (
    <div className="error-banner">
      <AlertTriangle size={16} className="error-banner-icon" />
      <span className="error-banner-text">{message}</span>
    </div>
  );
}

// 3. Shared Loading Skeleton component
interface SkeletonLoaderProps {
  rows?: number;
}

export function SkeletonLoader({ rows = 3 }: SkeletonLoaderProps) {
  return (
    <div className="animate-pulse" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          style={{
            height: '14px',
            background: 'var(--bezel-soft)',
            borderRadius: '4px',
            width: `${50 + (i % 3) * 15}%`
          }}
        ></div>
      ))}
    </div>
  );
}

// 4. Shared Metric component
export interface MetricProps {
  label: string;
  value: string | number;
  sub: string;
  percent: number;
  color: string;
  tooltip: string;
  change: number;
  icon?: LucideIcon;
  isInteractive?: boolean;
  isActive?: boolean;
  onClick?: () => void;
}

export function Metric({ label, value, sub, percent, color, tooltip, change, icon: Icon, isInteractive, isActive, onClick }: MetricProps) {
  // Inverted color coding rule: coverage gap increases are bad (red)
  const isCoverageGap = label.toLowerCase().includes('coverage gaps');
  const isGood = isCoverageGap ? change <= 0 : (label.toLowerCase().includes('coverage') ? change >= 0 : change <= 0);
  const isNeutral = change === 0;

  return (
    <div
      className={`panel metric ${isInteractive ? 'metric-interactive' : ''}`}
      onClick={isInteractive ? onClick : undefined}
      style={{
        cursor: isInteractive ? 'pointer' : 'default',
        borderColor: isActive ? 'var(--phosphor)' : undefined,
        background: isActive ? 'var(--panel-2)' : undefined
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div className="metric-label">
          {Icon && <Icon size={16} aria-hidden="true" />}
          <span>{label}</span>
        </div>
        <div title={tooltip} style={{ cursor: 'help', color: 'var(--muted)' }}>
          <Info size={12} />
        </div>
      </div>
      <div className="metric-val" style={{ color: `var(--${color})` }}>
        <span>{value}</span>
        {!isNeutral && (
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              fontSize: '12px',
              fontWeight: '500',
              color: isGood ? 'var(--phosphor)' : 'var(--red)',
              marginLeft: '4px'
            }}
          >
            {change > 0 ? <ArrowUpRight size={14} /> : <ArrowDownRight size={14} />}
            {Math.abs(change)}%
          </span>
        )}
      </div>
      <div className="metric-sub">{sub}</div>
      <div className="metric-bar">
        <div style={{ width: `${percent}%`, background: `var(--${color})` }}></div>
      </div>
    </div>
  );
}
