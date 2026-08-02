import { useState, useEffect, useRef } from 'react';
import { Terminal, Play, Pause, Trash2, Download, Search, Filter, ArrowUpRight, Copy, Check } from 'lucide-react';

export interface LogEntry {
  id: string;
  timestamp: string;
  level: 'INFO' | 'WARN' | 'ERROR' | 'DEBUG';
  service: string;
  message: string;
  traceId?: string;
  attributes?: Record<string, string | number>;
}

interface LiveTerminalProps {
  tenantId: string;
  setView?: (v: string) => void;
}

const INITIAL_LOGS: LogEntry[] = [
  {
    id: 'log-101',
    timestamp: new Date(Date.now() - 12000).toISOString().split('T')[1].slice(0, 8),
    level: 'INFO',
    service: 'control-plane-api',
    message: 'OTLP ingest gateway listening on 0.0.0.0:4317 (gRPC) and 0.0.0.0:4318 (HTTP)',
    attributes: { component: 'otel-collector', transport: 'grpc' }
  },
  {
    id: 'log-102',
    timestamp: new Date(Date.now() - 10000).toISOString().split('T')[1].slice(0, 8),
    level: 'INFO',
    service: 'checkout-service',
    message: 'Processed payload batch #4812 — 142 spans exported to SigNoz ClickHouse',
    traceId: 'trace-991',
    attributes: { span_count: 142, duration_ms: 18.4 }
  },
  {
    id: 'log-103',
    timestamp: new Date(Date.now() - 8000).toISOString().split('T')[1].slice(0, 8),
    level: 'WARN',
    service: 'payments-api',
    message: 'Orphan span detected: parent context missing for span_id=0x8f2a1b9',
    traceId: 'trace-992',
    attributes: { orphan_rate: '18.2%', threshold: '5.0%' }
  },
  {
    id: 'log-104',
    timestamp: new Date(Date.now() - 6000).toISOString().split('T')[1].slice(0, 8),
    level: 'ERROR',
    service: 'checkout-service',
    message: 'High cardinality alert fired: user_id_raw exceeds 10,000 unique HLL hashes',
    attributes: { attribute_key: 'user_id_raw', cardinality_count: 14382 }
  },
  {
    id: 'log-105',
    timestamp: new Date(Date.now() - 4000).toISOString().split('T')[1].slice(0, 8),
    level: 'INFO',
    service: 'ai-agent-runner',
    message: 'Tool call execution completed: retriever.query() returned 4 document chunks',
    traceId: 'trace-token-limit',
    attributes: { tokens_burned: 1204, model: 'gpt-4o-mini' }
  },
  {
    id: 'log-106',
    timestamp: new Date(Date.now() - 2000).toISOString().split('T')[1].slice(0, 8),
    level: 'DEBUG',
    service: 'remediation-engine',
    message: 'Circuit breaker state evaluated: FAIL_OPEN condition check passed',
    attributes: { window_ms: 100, burst_limit: 20 }
  }
];

const SIMULATED_SERVICES = ['checkout-service', 'payments-api', 'inventory-worker', 'ai-agent-runner', 'remediation-engine', 'otel-collector'];
const SIMULATED_MESSAGES = [
  { level: 'INFO', msg: 'OTLP batch export successful — 88 spans committed to ClickHouse', traceId: 'trace-991' },
  { level: 'WARN', msg: 'High memory usage on worker thread: 84% heap utilization', traceId: 'trace-992' },
  { level: 'ERROR', msg: 'Upstream HTTP 502 Bad Gateway from payment gateway endpoint', traceId: 'trace-992' },
  { level: 'DEBUG', msg: 'Health score matrix evaluated: overall score = 78 (HEALTHY)', traceId: undefined },
  { level: 'INFO', msg: 'MCP server tool call executed: query_active_alerts()', traceId: 'trace-token-limit' }
];

export function LiveTerminal({ tenantId, setView }: LiveTerminalProps) {
  const [logs, setLogs] = useState<LogEntry[]>(INITIAL_LOGS);
  const [isStreaming, setIsStreaming] = useState<boolean>(true);
  const [selectedLevel, setSelectedLevel] = useState<string>('ALL');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new logs arrive (if streaming is active)
  useEffect(() => {
    if (isStreaming) {
      terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, isStreaming]);

  // Live log generator simulator / backend poll fallback
  useEffect(() => {
    if (!isStreaming) return;

    const interval = setInterval(() => {
      const randomTemplate = SIMULATED_MESSAGES[Math.floor(Math.random() * SIMULATED_MESSAGES.length)];
      const randomService = SIMULATED_SERVICES[Math.floor(Math.random() * SIMULATED_SERVICES.length)];
      const now = new Date().toISOString().split('T')[1].slice(0, 8);

      const newLog: LogEntry = {
        id: `log-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
        timestamp: now,
        level: randomTemplate.level as LogEntry['level'],
        service: randomService,
        message: randomTemplate.msg,
        traceId: randomTemplate.traceId,
        attributes: { tenant_id: tenantId, env: 'production' }
      };

      setLogs((prev) => [...prev.slice(-150), newLog]); // Keep max 150 lines
    }, 2500);

    return () => clearInterval(interval);
  }, [isStreaming, tenantId]);

  const filteredLogs = logs.filter((log) => {
    const matchesLevel = selectedLevel === 'ALL' || log.level === selectedLevel;
    const matchesSearch =
      searchQuery === '' ||
      log.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
      log.service.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (log.traceId && log.traceId.toLowerCase().includes(searchQuery.toLowerCase()));
    return matchesLevel && matchesSearch;
  });

  const handleCopyLog = (log: LogEntry) => {
    const text = `[${log.timestamp}] [${log.level}] [${log.service}] ${log.message}`;
    navigator.clipboard.writeText(text);
    setCopiedId(log.id);
    setTimeout(() => setCopiedId(null), 1500);
  };

  const handleDownloadLogs = () => {
    const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(logs, null, 2));
    const downloadAnchor = document.createElement('a');
    downloadAnchor.setAttribute('href', dataStr);
    downloadAnchor.setAttribute('download', `telemetry_logs_${tenantId}_${Date.now()}.json`);
    document.body.appendChild(downloadAnchor);
    downloadAnchor.click();
    downloadAnchor.remove();
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', height: 'calc(100vh - 120px)', minHeight: '550px' }}>
      
      {/* Terminal Header & Control Toolbar */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          background: 'var(--panel)',
          border: '1px solid var(--bezel)',
          borderRadius: '8px',
          padding: '12px 16px',
          gap: '12px',
          flexWrap: 'wrap'
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Terminal size={18} style={{ color: 'var(--phosphor)' }} />
          <div>
            <h2 style={{ fontSize: '14px', fontWeight: 600, margin: 0, color: 'var(--paper)', display: 'flex', alignItems: 'center', gap: '8px' }}>
              Live Telemetry Log Stream
              <span
                style={{
                  fontSize: '10px',
                  padding: '2px 6px',
                  borderRadius: '4px',
                  background: isStreaming ? 'rgba(34, 197, 94, 0.15)' : 'rgba(239, 68, 68, 0.15)',
                  color: isStreaming ? '#22c55e' : '#ef4444',
                  fontFamily: 'var(--mono)'
                }}
              >
                {isStreaming ? '● LIVE STREAMING' : '⏸ PAUSED'}
              </span>
            </h2>
            <div style={{ fontSize: '11px', color: 'var(--muted)', marginTop: '2px' }}>
              Real-time OTLP log tailing, span trace links, and anomaly telemetry feed
            </div>
          </div>
        </div>

        {/* Action Buttons & Controls */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          
          {/* Pause / Stream Toggle */}
          <button
            onClick={() => setIsStreaming(!isStreaming)}
            className="btn"
            style={{
              borderColor: isStreaming ? 'var(--bezel)' : 'var(--phosphor)',
              color: isStreaming ? 'var(--paper)' : 'var(--phosphor)'
            }}
          >
            {isStreaming ? <Pause size={14} /> : <Play size={14} />}
            <span>{isStreaming ? 'Pause' : 'Resume'}</span>
          </button>

          {/* Clear Logs */}
          <button onClick={() => setLogs([])} className="btn btn-icon" title="Clear log buffer" aria-label="Clear logs">
            <Trash2 size={14} />
          </button>

          {/* Download Logs */}
          <button onClick={handleDownloadLogs} className="btn btn-icon" title="Export logs as JSON" aria-label="Export logs">
            <Download size={14} />
          </button>
        </div>
      </div>

      {/* Filter & Search Bar */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '12px',
          background: 'var(--ink)',
          border: '1px solid var(--bezel)',
          borderRadius: '6px',
          padding: '8px 12px'
        }}
      >
        {/* Search Input */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flex: 1, maxHeight: '32px' }}>
          <Search size={14} style={{ color: 'var(--muted-2)' }} />
          <input
            type="text"
            placeholder="Search logs by keyword, service, or trace_id..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{
              background: 'transparent',
              border: 'none',
              outline: 'none',
              color: 'var(--paper)',
              fontSize: '12px',
              fontFamily: 'var(--mono)',
              width: '100%'
            }}
          />
        </div>

        {/* Severity Filter Buttons */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          <Filter size={12} style={{ color: 'var(--muted-2)', marginRight: '4px' }} />
          {['ALL', 'ERROR', 'WARN', 'INFO', 'DEBUG'].map((level) => (
            <button
              key={level}
              onClick={() => setSelectedLevel(level)}
              style={{
                background: selectedLevel === level ? 'var(--panel-2)' : 'transparent',
                border: `1px solid ${selectedLevel === level ? 'var(--bezel)' : 'transparent'}`,
                color:
                  selectedLevel === level
                    ? level === 'ERROR'
                      ? 'var(--red)'
                      : level === 'WARN'
                      ? 'var(--amber)'
                      : level === 'INFO'
                      ? 'var(--phosphor)'
                      : 'var(--paper)'
                    : 'var(--muted-2)',
                fontSize: '10px',
                fontFamily: 'var(--mono)',
                padding: '3px 8px',
                borderRadius: '4px',
                cursor: 'pointer',
                transition: 'all 0.15s ease'
              }}
            >
              {level}
            </button>
          ))}
        </div>
      </div>

      {/* Terminal View Output Container */}
      <div
        style={{
          flex: 1,
          background: '#090a0f',
          border: '1px solid var(--bezel)',
          borderRadius: '8px',
          padding: '14px 16px',
          fontFamily: 'var(--mono)',
          fontSize: '11px',
          lineHeight: 1.6,
          overflowY: 'auto',
          boxShadow: 'inset 0 0 12px rgba(0, 0, 0, 0.8)',
          display: 'flex',
          flexDirection: 'column',
          gap: '6px'
        }}
      >
        {filteredLogs.length === 0 ? (
          <div style={{ padding: '40px', textAlign: 'center', color: 'var(--muted-2)' }}>
            No log records matching filter query.
          </div>
        ) : (
          filteredLogs.map((log) => {
            const levelColor =
              log.level === 'ERROR'
                ? '#ef4444'
                : log.level === 'WARN'
                ? '#f59e0b'
                : log.level === 'INFO'
                ? '#22c55e'
                : '#3b82f6';

            return (
              <div
                key={log.id}
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: '12px',
                  padding: '4px 6px',
                  borderRadius: '4px',
                  transition: 'background 0.1s ease',
                  borderLeft: `2px solid ${levelColor}`
                }}
                className="log-row-hover"
              >
                {/* Timestamp */}
                <span style={{ color: 'var(--muted-2)', flexShrink: 0, width: '65px' }}>{log.timestamp}</span>

                {/* Level Badge */}
                <span
                  style={{
                    color: levelColor,
                    fontWeight: 700,
                    width: '50px',
                    flexShrink: 0
                  }}
                >
                  [{log.level}]
                </span>

                {/* Service Name */}
                <span style={{ color: 'var(--paper)', fontWeight: 600, width: '130px', flexShrink: 0 }}>
                  {log.service}
                </span>

                {/* Message Body */}
                <span style={{ color: log.level === 'ERROR' ? '#fca5a5' : '#d1d5db', flex: 1, wordBreak: 'break-word' }}>
                  {log.message}

                  {/* Attributes Tag */}
                  {log.attributes && (
                    <span style={{ color: 'var(--muted-2)', marginLeft: '8px', fontSize: '10px' }}>
                      {Object.entries(log.attributes)
                        .map(([k, v]) => `${k}=${v}`)
                        .join(' ')}
                    </span>
                  )}
                </span>

                {/* Trace ID Link */}
                {log.traceId && (
                  <button
                    onClick={() => setView && setView(log.traceId === 'trace-token-limit' ? 'agenttraces' : 'tracechains')}
                    style={{
                      background: 'rgba(34, 197, 94, 0.1)',
                      border: '1px solid rgba(34, 197, 94, 0.2)',
                      color: 'var(--phosphor)',
                      fontSize: '10px',
                      fontFamily: 'var(--mono)',
                      padding: '1px 6px',
                      borderRadius: '3px',
                      cursor: 'pointer',
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '4px',
                      flexShrink: 0
                    }}
                    title="Jump to trace in Trace Chains"
                  >
                    <span>{log.traceId}</span>
                    <ArrowUpRight size={10} />
                  </button>
                )}

                {/* Quick Copy Button */}
                <button
                  onClick={() => handleCopyLog(log)}
                  style={{
                    background: 'transparent',
                    border: 'none',
                    color: 'var(--muted-2)',
                    cursor: 'pointer',
                    padding: '2px',
                    borderRadius: '3px',
                    flexShrink: 0
                  }}
                  title="Copy log line"
                >
                  {copiedId === log.id ? <Check size={12} style={{ color: '#22c55e' }} /> : <Copy size={12} />}
                </button>
              </div>
            );
          })
        )}
        <div ref={terminalEndRef} />
      </div>
    </div>
  );
}
