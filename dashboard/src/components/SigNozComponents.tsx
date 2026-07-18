import React, { useState, useEffect } from 'react';
import { CheckCircle, XCircle, Loader, Cpu, BarChart2, Wifi, AlertTriangle, ChevronDown, ChevronUp, Copy, Check } from 'lucide-react';

// ──────────────────────────────────────────────────────────────────────────────
// IMPL-1: SigNoz Connection Status Badge (signoz.md §3)
// Polls GET /api/v1/signoz/health every 30s and shows a live indicator
// ──────────────────────────────────────────────────────────────────────────────
type SigNozStatus = 'connected' | 'disconnected' | 'checking';

interface SigNozHealthPayload {
  status: string;
  signoz_alertmanager: string;
  otlp_exporter: string;
  mock_mode: boolean;
}

export function SigNozStatusBadge() {
  const [signozStatus, setSignozStatus] = useState<SigNozStatus>('checking');
  const [mockMode, setMockMode] = useState(false);
  const [detail, setDetail] = useState('');

  useEffect(() => {
    const check = async () => {
      try {
        const res = await fetch('/api/v1/signoz/health', {
          headers: { 'Authorization': 'Bearer health-demo-key-2026' }
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data: SigNozHealthPayload = await res.json();
        setSignozStatus(data.status === 'healthy' ? 'connected' : 'disconnected');
        setMockMode(data.mock_mode ?? false);
        setDetail(data.signoz_alertmanager ?? '');
      } catch {
        setSignozStatus('disconnected');
      }
    };
    check();
    const id = setInterval(check, 30000);
    return () => clearInterval(id);
  }, []);

  const colors: Record<SigNozStatus, string> = {
    connected: 'var(--phosphor)',
    disconnected: 'var(--red)',
    checking: 'var(--amber)',
  };

  const icons: Record<SigNozStatus, React.ReactNode> = {
    connected: <CheckCircle size={11} />,
    disconnected: <XCircle size={11} />,
    checking: <Loader size={11} className="animate-spin" />,
  };

  const labels: Record<SigNozStatus, string> = {
    connected: mockMode ? 'SigNoz (demo)' : 'SigNoz',
    disconnected: 'SigNoz offline',
    checking: 'SigNoz…',
  };

  return (
    <div
      title={`Alertmanager: ${detail || 'unknown'} | Mock: ${mockMode}`}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '5px',
        fontSize: '11px',
        fontFamily: 'var(--mono)',
        color: colors[signozStatus],
        background: `${colors[signozStatus]}18`,
        border: `1px solid ${colors[signozStatus]}44`,
        borderRadius: '4px',
        padding: '3px 8px',
        cursor: 'default',
        transition: 'all 0.3s ease',
      }}
    >
      {icons[signozStatus]}
      {labels[signozStatus]}
    </div>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// IMPL-2: Alert Fired Indicator (signoz.md §2)
// Shows a prominent banner when health score < 50 indicating alert fired
// ──────────────────────────────────────────────────────────────────────────────
interface AlertFiredBannerProps {
  healthScore: number;
  tenantId: string;
}

export function AlertFiredBanner({ healthScore, tenantId }: AlertFiredBannerProps) {
  if (healthScore >= 50) return null;

  return (
    <div style={{
      display: 'flex',
      alignItems: 'flex-start',
      gap: '10px',
      padding: '10px 14px',
      background: 'rgba(255,77,77,0.10)',
      border: '1px solid var(--red)',
      borderRadius: '6px',
      marginTop: '12px',
      animation: 'pulse-border 2s ease infinite',
    }}>
      <AlertTriangle size={16} style={{ color: 'var(--red)', flexShrink: 0, marginTop: '1px' }} />
      <div>
        <div style={{ fontSize: '12px', fontWeight: 600, color: 'var(--red)', fontFamily: 'var(--mono)' }}>
          ⚠️ Alert fired to SigNoz Alertmanager
        </div>
        <div style={{ fontSize: '11px', color: 'var(--muted)', marginTop: '3px', fontFamily: 'var(--mono)' }}>
          Alert rule: <code style={{ color: 'var(--amber)' }}>telemetryhealth-health-degradation-{tenantId}</code>
          <br />
          Check: SigNoz UI → Alerts → Active Alerts
        </div>
      </div>
    </div>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// IMPL-3: MCP Tools Interactive Demo (signoz.md §4) — HIGH IMPACT
// Full interactive panel to invoke MCP tools and display JSON results
// ──────────────────────────────────────────────────────────────────────────────
interface McpToolResult {
  success?: boolean;
  data?: string;
  error?: string;
  [key: string]: unknown;
}

interface McpTool {
  id: string;
  icon: React.ReactNode;
  name: string;
  description: string;
  toolName: string;
  params: Record<string, unknown>;
}

interface McpToolsDemoProps {
  tenantId: string;
}

export function McpToolsDemo({ tenantId }: McpToolsDemoProps) {
  const [activeToolId, setActiveToolId] = useState<string | null>(null);
  const [results, setResults] = useState<Record<string, McpToolResult>>({});
  const [loading, setLoading] = useState<Record<string, boolean>>({});
  const [copied, setCopied] = useState<string | null>(null);

  const tools: McpTool[] = [
    {
      id: 'health',
      icon: <BarChart2 size={15} />,
      name: 'get_telemetry_health',
      description: 'Query real-time composite health score, cardinality metrics, and AI agent stats for a tenant.',
      toolName: 'get_telemetry_health',
      params: { tenant_id: tenantId },
    },
    {
      id: 'remediation',
      icon: <Cpu size={15} />,
      name: 'generate_remediation',
      description: 'Generate a verified, ready-to-deploy OTel Collector YAML patch for a given issue type.',
      toolName: 'generate_remediation',
      params: { issue_type: 'High Cardinality (user_id on checkout_service)' },
    },
  ];

  const callTool = async (tool: McpTool) => {
    setLoading(l => ({ ...l, [tool.id]: true }));
    setActiveToolId(tool.id);
    try {
      // Call via JSON-RPC 2.0 over HTTP (MCP streamable HTTP transport)
      const res = await fetch('/mcp', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: Date.now(),
          method: 'tools/call',
          params: { name: tool.toolName, arguments: tool.params },
        }),
      });
      if (res.ok) {
        const json = await res.json();
        const content = json?.result?.content?.[0]?.text;
        try {
          setResults(r => ({ ...r, [tool.id]: JSON.parse(content) }));
        } catch {
          setResults(r => ({ ...r, [tool.id]: { data: content } }));
        }
      } else {
        // Fallback to direct REST API when MCP proxy is not available
        const apiRes = await fetch(`/api/v1/tenant/${tenantId}/health`, {
          headers: { 'Authorization': 'Bearer health-demo-key-2026' }
        });
        const apiData = await apiRes.json();
        setResults(r => ({ ...r, [tool.id]: apiData }));
      }
    } catch {
      // Demo fallback with realistic data
      const fallback: Record<string, McpToolResult> = {
        health: {
          healthScore: 72,
          tenantId,
          version: 'v1.1.0-mcp',
          metrics: { cardinality: { value: '1.2M' }, orphans: { value: '6.2%' }, coverage: { value: '8' } },
          remediation: { issueType: 'high_cardinality', validated: true },
        },
        remediation: {
          data: `processors:\n  attributes/remediation:\n    action: delete\n    key: user_id\nservice:\n  pipelines:\n    traces:\n      processors: [attributes/remediation]`,
        },
      };
      setResults(r => ({ ...r, [tool.id]: fallback[tool.id] ?? { error: 'MCP endpoint unreachable in demo mode' } }));
    } finally {
      setLoading(l => ({ ...l, [tool.id]: false }));
    }
  };

  const copyResult = (toolId: string) => {
    const result = results[toolId];
    if (!result) return;
    navigator.clipboard.writeText(JSON.stringify(result, null, 2));
    setCopied(toolId);
    setTimeout(() => setCopied(null), 2000);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      {tools.map(tool => {
        const result = results[tool.id];
        const isLoading = loading[tool.id];
        const isActive = activeToolId === tool.id;

        return (
          <div
            key={tool.id}
            style={{
              background: 'var(--panel-2)',
              border: `1px solid ${isActive && result ? 'var(--phosphor)' : 'var(--bezel)'}`,
              borderRadius: '6px',
              overflow: 'hidden',
              transition: 'border-color 0.3s ease',
            }}
          >
            {/* Tool header */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '10px 14px' }}>
              <span style={{ color: 'var(--phosphor)' }}>{tool.icon}</span>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: '12px', fontWeight: 600, fontFamily: 'var(--mono)', color: 'var(--paper)' }}>
                  {tool.name}
                </div>
                <div style={{ fontSize: '11px', color: 'var(--muted)', marginTop: '1px' }}>
                  {tool.description}
                </div>
              </div>
              <button
                onClick={() => callTool(tool)}
                disabled={isLoading}
                style={{
                  display: 'flex', alignItems: 'center', gap: '5px',
                  padding: '5px 12px', fontSize: '11px', fontFamily: 'var(--mono)',
                  background: 'var(--phosphor)', color: 'var(--ink)',
                  border: 'none', borderRadius: '4px', cursor: 'pointer',
                  opacity: isLoading ? 0.6 : 1,
                  transition: 'opacity 0.2s',
                }}
              >
                {isLoading ? <Loader size={11} className="animate-spin" /> : <Wifi size={11} />}
                {isLoading ? 'Calling…' : 'Call Tool'}
              </button>
            </div>

            {/* Result panel */}
            {result && (
              <div style={{ borderTop: '1px solid var(--bezel)', position: 'relative' }}>
                <button
                  onClick={() => copyResult(tool.id)}
                  title="Copy JSON"
                  style={{
                    position: 'absolute', top: '8px', right: '10px',
                    background: 'var(--panel)', border: '1px solid var(--bezel)',
                    borderRadius: '4px', padding: '3px 6px', cursor: 'pointer',
                    color: copied === tool.id ? 'var(--phosphor)' : 'var(--muted)',
                    display: 'flex', alignItems: 'center', gap: '4px', fontSize: '10px',
                  }}
                >
                  {copied === tool.id ? <Check size={10} /> : <Copy size={10} />}
                  {copied === tool.id ? 'Copied' : 'Copy'}
                </button>
                <pre style={{
                  margin: 0, padding: '12px 14px',
                  fontSize: '11px', fontFamily: 'var(--mono)',
                  color: 'var(--phosphor)', background: 'var(--ink)',
                  overflowX: 'auto', maxHeight: '220px',
                  lineHeight: 1.5,
                }}>
                  {JSON.stringify(result, null, 2)}
                </pre>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// IMPL-5: Query Builder vs Raw SQL Demo (signoz.md §5)
// Educational display showing SigNoz best practices
// ──────────────────────────────────────────────────────────────────────────────
export function QueryBuilderDemo() {
  const [activeTab, setActiveTab] = useState<'builder' | 'sql'>('builder');

  const builderPayload = JSON.stringify({
    dataSource: 'metrics',
    queryType: 'builder',
    builderQuery: {
      queryName: 'A',
      stepInterval: 60,
      aggregateOperator: 'avg',
      aggregateAttribute: 'telemetryhealth_agent_health_score',
      groupBy: ['service_name', 'agent_id'],
      filters: {
        op: 'AND',
        items: [{ key: 'service_name', op: '=', value: 'ai-agent-service' }]
      }
    },
    start: Date.now() - 3600000,
    end: Date.now(),
  }, null, 2);

  const rawSql = `SELECT
  avg(value) AS health_score,
  service_name,
  agent_id
FROM telemetry_health.metrics
WHERE
  name = 'telemetryhealth_agent_health_score'
  AND timestamp >= now() - INTERVAL 1 HOUR
GROUP BY service_name, agent_id
ORDER BY health_score ASC;`;

  return (
    <div style={{ border: '1px solid var(--bezel)', borderRadius: '6px', overflow: 'hidden' }}>
      {/* Tabs */}
      <div style={{ display: 'flex', borderBottom: '1px solid var(--bezel)' }}>
        {[
          { id: 'builder', label: '✅ Query Builder (Recommended)' },
          { id: 'sql', label: '⚠️ Raw SQL (Legacy)' },
        ].map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id as 'builder' | 'sql')}
            style={{
              flex: 1, padding: '8px 12px',
              fontSize: '11px', fontFamily: 'var(--mono)',
              background: activeTab === tab.id ? 'var(--panel-2)' : 'var(--panel)',
              color: activeTab === tab.id ? 'var(--paper)' : 'var(--muted)',
              border: 'none', borderBottom: activeTab === tab.id ? '2px solid var(--phosphor)' : '2px solid transparent',
              cursor: 'pointer', transition: 'all 0.2s',
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>
      {/* Content */}
      <pre style={{
        margin: 0, padding: '14px',
        fontSize: '11px', fontFamily: 'var(--mono)',
        color: activeTab === 'builder' ? 'var(--phosphor)' : 'var(--amber)',
        background: 'var(--ink)', overflowX: 'auto', lineHeight: 1.6,
        maxHeight: '260px',
      }}>
        {activeTab === 'builder' ? builderPayload : rawSql}
      </pre>
      <div style={{ padding: '8px 14px', fontSize: '11px', color: 'var(--muted)', background: 'var(--panel-2)', borderTop: '1px solid var(--bezel)' }}>
        {activeTab === 'builder'
          ? '✅ Type-safe · validated · respects SigNoz semantic layer · works with alerting rules'
          : '⚠️ Bypasses SigNoz API layer · not recommended for production · no semantic validation'}
      </div>
    </div>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// IMPL-6: SigNoz Config Panel (signoz.md §6)
// Shows current SigNoz environment configuration fetched from backend
// ──────────────────────────────────────────────────────────────────────────────
interface SigNozConfigPayload {
  signoz_base_url: string;
  signoz_alertmanager_url: string;
  mcp_server_addr: string;
  otlp_endpoint: string;
  mock_mode: boolean;
}

export function SigNozConfigPanel() {
  const [config, setConfig] = useState<SigNozConfigPayload | null>(null);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    fetch('/api/v1/signoz/config', {
      headers: { 'Authorization': 'Bearer health-demo-key-2026' }
    })
      .then(r => r.ok ? r.json() : null)
      .then(d => { setConfig(d); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  const rows: Array<{ label: string; key: keyof SigNozConfigPayload; link?: boolean }> = [
    { label: 'SigNoz Base URL', key: 'signoz_base_url', link: true },
    { label: 'Alertmanager URL', key: 'signoz_alertmanager_url' },
    { label: 'MCP Server Address', key: 'mcp_server_addr' },
    { label: 'OTLP Endpoint', key: 'otlp_endpoint' },
    { label: 'Mock Mode', key: 'mock_mode' },
  ];

  return (
    <div style={{ border: '1px solid var(--bezel)', borderRadius: '6px', overflow: 'hidden' }}>
      <button
        onClick={() => setExpanded(e => !e)}
        style={{
          width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '10px 14px', background: 'var(--panel-2)', border: 'none', cursor: 'pointer',
          fontSize: '12px', fontFamily: 'var(--mono)', color: 'var(--paper)',
        }}
      >
        <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          🔧 SigNoz Configuration
          {config?.mock_mode && (
            <span style={{ fontSize: '10px', background: 'var(--amber-dim)', color: 'var(--amber)', padding: '1px 6px', borderRadius: '3px' }}>
              demo mode
            </span>
          )}
        </span>
        {expanded ? <ChevronUp size={14} style={{ color: 'var(--muted)' }} /> : <ChevronDown size={14} style={{ color: 'var(--muted)' }} />}
      </button>

      {expanded && (
        <div style={{ padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {loading ? (
            <div style={{ color: 'var(--muted)', fontSize: '12px', fontFamily: 'var(--mono)' }}>Loading configuration…</div>
          ) : !config ? (
            <div style={{ color: 'var(--red)', fontSize: '12px', fontFamily: 'var(--mono)' }}>Failed to load configuration</div>
          ) : (
            rows.map(row => {
              const val = config[row.key];
              const display = typeof val === 'boolean' ? (val ? 'true' : 'false') : String(val);
              return (
                <div key={row.key} style={{ display: 'flex', alignItems: 'center', gap: '12px', fontSize: '11px', fontFamily: 'var(--mono)' }}>
                  <span style={{ color: 'var(--muted)', width: '160px', flexShrink: 0 }}>{row.label}</span>
                  {row.link && typeof val === 'string' ? (
                    <a href={val} target="_blank" rel="noopener noreferrer" style={{ color: 'var(--phosphor)', textDecoration: 'none' }}>
                      {display} ↗
                    </a>
                  ) : (
                    <code style={{ color: typeof val === 'boolean' ? (val ? 'var(--amber)' : 'var(--phosphor)') : 'var(--paper)' }}>
                      {display}
                    </code>
                  )}
                </div>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}
