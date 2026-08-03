import { useState, useEffect } from 'react';
import { getAuthHeaders } from '../../auth';
import { Play, Clock, CheckCircle, XCircle, AlertTriangle, RefreshCw } from 'lucide-react';
import { McpToolsDemo, QueryBuilderDemo, SigNozConfigPanel } from '../SigNozComponents';

interface ReplayEvent {
  TraceID: string;
  SpanID: string;
  ParentSpanID: string;
  ServiceName: string;
  OperationName: string;
  StartTime: string;
  EndTime: string;
  Status: string;
  Attributes: Record<string, string>;
}

interface ReplayPayload {
  tenant_id: string;
  trace_id: string;
  mode: string;
  events: ReplayEvent[];
}

interface SigNozIntegrationProps {
  tenantId: string;
}

// ──────────────────────────────────────────────────────────────────────────────
// IMPL-3 (replay half): Replay timeline component
// ──────────────────────────────────────────────────────────────────────────────
function ReplayTimeline({ tenantId }: { tenantId: string }) {
  const [replay, setReplay] = useState<ReplayPayload | null>(null);
  const [loading, setLoading] = useState(false);
  const [traceInput, setTraceInput] = useState('');
  const [error, setError] = useState<string | null>(null);

  const fetchReplay = async (traceId?: string) => {
    setLoading(true);
    setReplay(null); // Visually clear the timeline so the user knows it's loading
    setError(null);
    try {
      const url = traceId
        ? `/api/v1/tenant/${tenantId}/replay?trace_id=${encodeURIComponent(traceId)}`
        : `/api/v1/tenant/${tenantId}/replay`;
      const res = await fetch(url, { headers: getAuthHeaders() });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: ReplayPayload & { loadedAt?: string } = await res.json();
      data.loadedAt = new Date().toLocaleTimeString();
      setReplay(data);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load replay');
    } finally {
      setLoading(false);
    }
  };

  // Automatically fetch new data when the tenant (enterprise) changes
  useEffect(() => {
    fetchReplay(undefined);
  }, [tenantId]);

  const getStatusColor = (status: string) => {
    if (status === 'ERROR') return 'var(--red)';
    if (status === 'UNSET') return 'var(--amber)';
    return 'var(--phosphor)';
  };

  const getStatusIcon = (status: string) => {
    if (status === 'ERROR') return <XCircle size={12} style={{ color: 'var(--red)' }} />;
    if (status === 'UNSET') return <AlertTriangle size={12} style={{ color: 'var(--amber)' }} />;
    return <CheckCircle size={12} style={{ color: 'var(--phosphor)' }} />;
  };

  const getDurationMs = (start: string, end: string) => {
    const ms = new Date(end).getTime() - new Date(start).getTime();
    return isNaN(ms) ? '—' : `${ms}ms`;
  };

  // Compute timeline bar widths relative to total span time
  const computeBar = (events: ReplayEvent[], evt: ReplayEvent) => {
    if (!events.length) return { left: '0%', width: '10%' };
    const times = events.flatMap(e => [new Date(e.StartTime).getTime(), new Date(e.EndTime).getTime()]).filter(t => !isNaN(t));
    const minT = Math.min(...times);
    const maxT = Math.max(...times);
    const range = maxT - minT || 1;
    const left = ((new Date(evt.StartTime).getTime() - minT) / range) * 100;
    const width = ((new Date(evt.EndTime).getTime() - new Date(evt.StartTime).getTime()) / range) * 100;
    return { left: `${Math.max(0, left)}%`, width: `${Math.max(2, width)}%` };
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      {/* Controls */}
      <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
        <input
          type="text"
          placeholder="trace-992 (optional — blank = latest)"
          value={traceInput}
          onChange={e => setTraceInput(e.target.value)}
          style={{
            flex: 1, padding: '6px 10px', fontSize: '11px', fontFamily: 'var(--mono)',
            background: 'var(--panel-2)', border: '1px solid var(--bezel)',
            borderRadius: '4px', color: 'var(--paper)', outline: 'none',
          }}
        />
        <button
          onClick={() => fetchReplay(traceInput.trim() || undefined)}
          disabled={loading}
          style={{
            display: 'flex', alignItems: 'center', gap: '5px',
            padding: '6px 12px', fontSize: '11px', fontFamily: 'var(--mono)',
            background: 'var(--phosphor)', color: 'var(--ink)',
            border: 'none', borderRadius: '4px', cursor: 'pointer',
            opacity: loading ? 0.6 : 1,
          }}
        >
          {loading ? <RefreshCw size={11} className="animate-spin" /> : <Play size={11} />}
          {loading ? 'Loading…' : 'Load Replay'}
        </button>
      </div>

      {error && (
        <div style={{ fontSize: '11px', color: 'var(--red)', fontFamily: 'var(--mono)', padding: '6px 10px', background: 'var(--red-dim)', borderRadius: '4px' }}>
          {error}
        </div>
      )}

      {replay && (
        <>
          {/* Header */}
          <div style={{ display: 'flex', gap: '16px', fontSize: '11px', fontFamily: 'var(--mono)', color: 'var(--muted)' }}>
            <span>Trace: <code style={{ color: 'var(--amber)' }}>{replay.trace_id}</code></span>
            <span>Mode: <code style={{ color: 'var(--phosphor)' }}>{replay.mode}</code></span>
            <span>Spans: <code style={{ color: 'var(--paper)' }}>{replay.events?.length ?? 0}</code></span>
            {/* @ts-ignore - added loadedAt ad-hoc */}
            {replay.loadedAt && <span>Updated: <code style={{ color: 'var(--paper)' }}>{replay.loadedAt}</code></span>}
          </div>

          {/* Gantt-style timeline */}
          <div style={{ border: '1px solid var(--bezel)', borderRadius: '6px', overflow: 'hidden' }}>
            <div style={{
              padding: '6px 12px', background: 'var(--panel-2)',
              borderBottom: '1px solid var(--bezel)',
              display: 'grid', gridTemplateColumns: '20px 180px 90px 80px 1fr',
              gap: '8px', fontSize: '10px', color: 'var(--muted)', fontFamily: 'var(--mono)',
            }}>
              <span></span>
              <span>Operation</span>
              <span>Service</span>
              <span>Duration</span>
              <span>Timeline</span>
            </div>
            {(replay.events ?? []).map((evt, i) => {
              const bar = computeBar(replay.events, evt);
              return (
                <div
                  key={evt.SpanID || i}
                  style={{
                    padding: '6px 12px',
                    borderBottom: i < (replay.events?.length ?? 0) - 1 ? '1px solid var(--bezel)' : 'none',
                    display: 'grid', gridTemplateColumns: '20px 180px 90px 80px 1fr',
                    gap: '8px', alignItems: 'center', fontSize: '11px',
                    background: i % 2 === 0 ? 'var(--panel)' : 'transparent',
                  }}
                >
                  <span title={evt.Status}>{getStatusIcon(evt.Status)}</span>
                  <span style={{
                    fontFamily: 'var(--mono)', color: getStatusColor(evt.Status),
                    overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                  }} title={evt.OperationName}>
                    {evt.OperationName}
                  </span>
                  <span style={{ color: 'var(--muted)', fontSize: '10px', fontFamily: 'var(--mono)' }}>
                    {evt.ServiceName}
                  </span>
                  <span style={{ color: 'var(--paper)', fontFamily: 'var(--mono)', fontSize: '10px', display: 'flex', alignItems: 'center', gap: '3px' }}>
                    <Clock size={9} style={{ color: 'var(--muted)' }} />
                    {getDurationMs(evt.StartTime, evt.EndTime)}
                  </span>
                  {/* Timeline bar */}
                  <div style={{ position: 'relative', height: '10px', background: 'var(--panel-2)', borderRadius: '2px' }}>
                    <div style={{
                      position: 'absolute', top: 0, left: bar.left, width: bar.width,
                      height: '100%', background: getStatusColor(evt.Status),
                      borderRadius: '2px', opacity: 0.7,
                      transition: 'all 0.3s ease',
                    }} />
                  </div>
                </div>
              );
            })}
          </div>

          {/* Attributes of first span if any */}
          {replay.events?.[0]?.Attributes && Object.keys(replay.events[0].Attributes).length > 0 && (
            <div style={{ fontSize: '11px', fontFamily: 'var(--mono)', color: 'var(--muted)' }}>
              Root span attributes:{' '}
              {Object.entries(replay.events[0].Attributes).map(([k, v]) => (
                <span key={k} style={{ marginRight: '12px' }}>
                  <code style={{ color: 'var(--amber)' }}>{k}</code>
                  <span style={{ color: 'var(--muted)' }}>: </span>
                  <code style={{ color: 'var(--paper)' }}>{v}</code>
                </span>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Main SigNoz Integration view — all 4 IMPL sections composed
// ──────────────────────────────────────────────────────────────────────────────
export function SigNozIntegration({ tenantId }: SigNozIntegrationProps) {
  const sections = [
    { id: 'replay', label: '01 / AGENT REPLAY TIMELINE', content: <ReplayTimeline tenantId={tenantId} /> },
    { id: 'mcp', label: '02 / MCP TOOL EXPLORER', content: <McpToolsDemo tenantId={tenantId} /> },
    { id: 'query', label: '03 / QUERY BUILDER vs RAW SQL', content: <QueryBuilderDemo /> },
    { id: 'config', label: '04 / SIGNOZ CONFIGURATION', content: <SigNozConfigPanel /> },
  ];

  const [activeSection, setActiveSection] = useState<string>('replay');

  return (
    <section className="view active">
      {/* Section tabs */}
      <div style={{ display: 'flex', gap: '4px', marginBottom: '20px', flexWrap: 'wrap' }}>
        {sections.map(s => (
          <button
            key={s.id}
            onClick={() => setActiveSection(s.id)}
            style={{
              padding: '5px 12px', fontSize: '11px', fontFamily: 'var(--mono)',
              background: activeSection === s.id ? 'var(--phosphor)' : 'var(--panel-2)',
              color: activeSection === s.id ? 'var(--ink)' : 'var(--muted)',
              border: `1px solid ${activeSection === s.id ? 'var(--phosphor)' : 'var(--bezel)'}`,
              borderRadius: '4px', cursor: 'pointer', transition: 'all 0.2s',
              fontWeight: activeSection === s.id ? 600 : 400,
            }}
          >
            {s.label}
          </button>
        ))}
      </div>

      {/* Active section content */}
      {sections.map(s => (
        <div key={s.id} style={{ display: activeSection === s.id ? 'block' : 'none' }}>
          <div className="panel" style={{ padding: '20px' }}>
            {s.content}
          </div>
        </div>
      ))}
    </section>
  );
}
