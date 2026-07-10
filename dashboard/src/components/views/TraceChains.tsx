import { useState, useEffect } from 'react';
import type { DashboardData } from '../../App';
import { useTenantData, ErrorBanner, SkeletonLoader } from '../Shared';
import { Activity, AlertTriangle, HelpCircle } from 'lucide-react';

interface TraceOrphanData {
  orphanRate: string;
  topOrphanedService: string;
  missingParents: number;
}

interface TraceChainsProps {
  data: DashboardData;
  tenantId: string;
}

interface ServiceNode {
  id: string;
  label: string;
  x: number;
  y: number;
  status: 'healthy' | 'degraded' | 'critical';
  rate: string;
  errors: string;
  latency: string;
  ops: string[];
}

interface ServiceLink {
  source: string;
  target: string;
  isBroken?: boolean;
}

interface OrphanEvent {
  id: string;
  span: string;
  collector: string;
  service: string;
  desc: string;
  severity: 'r' | 'a' | 'p';
}

export function TraceChains({ data: _data, tenantId }: TraceChainsProps) {
  const [selectedNode, setSelectedNode] = useState<string | null>(null);

  const fallbackOrphans: TraceOrphanData = {
    orphanRate: '6.2%',
    topOrphanedService: 'payments-api',
    missingParents: 142
  };

  // useTenantData shared hook implements AbortController and proxy compliance
  const { data: traceData, loading, error, errorMsg } = useTenantData<TraceOrphanData>(
    tenantId,
    'traces/orphans',
    fallbackOrphans
  );

  const activeOrphans = traceData ?? fallbackOrphans;
  const topService = activeOrphans.topOrphanedService;

  // Auto-select the breached node on mount or update
  useEffect(() => {
    setSelectedNode('payments');
  }, [topService]);

  const nodes: ServiceNode[] = [
    {
      id: 'gateway',
      label: 'api-gateway',
      x: 60,
      y: 100,
      status: 'healthy',
      rate: '2,480 req/s',
      errors: '0.01%',
      latency: '24ms',
      ops: ['GET /api/v1/health', 'POST /api/v1/remediation/apply', 'GET /api/v1/tenant/acme-prod/health']
    },
    {
      id: 'auth',
      label: 'auth-service',
      x: 210,
      y: 40,
      status: 'healthy',
      rate: '450 req/s',
      errors: '0.00%',
      latency: '8ms',
      ops: ['POST /auth/login', 'POST /auth/verify', 'GET /auth/session']
    },
    {
      id: 'checkout',
      label: 'checkout-service',
      x: 210,
      y: 160,
      status: 'degraded',
      rate: '1,210 req/s',
      errors: '1.82%',
      latency: '145ms',
      ops: ['POST /checkout/create', 'GET /checkout/active', 'POST /checkout/validate']
    },
    {
      id: 'payments',
      label: topService,
      x: 380,
      y: 160,
      status: 'critical',
      rate: '120 req/s',
      errors: activeOrphans.orphanRate,
      latency: '820ms',
      ops: ['POST /charge/card', 'GET /payment/history', 'POST /refund/request']
    },
    {
      id: 'inventory',
      label: 'inventory-worker',
      x: 380,
      y: 40,
      status: 'healthy',
      rate: '80 req/s',
      errors: '0.00%',
      latency: '12ms',
      ops: ['POST /stock/reserve', 'GET /inventory/check']
    }
  ];

  const links: ServiceLink[] = [
    { source: 'gateway', target: 'auth' },
    { source: 'gateway', target: 'checkout' },
    { source: 'checkout', target: 'payments', isBroken: true },
    { source: 'checkout', target: 'inventory' }
  ];

  const orphanEvents: OrphanEvent[] = [
    {
      id: 'orph-evt-1',
      span: 'span 4a91',
      collector: 'collector-07',
      service: topService,
      desc: 'parent 7bd1 not found • correlated after 31s',
      severity: 'r'
    },
    {
      id: 'orph-evt-2',
      span: 'span 2e6f',
      collector: 'collector-03',
      service: topService,
      desc: 'parent c910 not found • correlated after 12s',
      severity: 'r'
    },
    {
      id: 'orph-evt-3',
      span: 'span 88bd',
      collector: 'collector-07',
      service: 'checkout-service',
      desc: 'late arrival • resolved within window',
      severity: 'a'
    }
  ];

  const activeNode = nodes.find(n => n.id === selectedNode);

  const getLinkPoints = (link: ServiceLink) => {
    const srcNode = nodes.find(n => n.id === link.source);
    const tgtNode = nodes.find(n => n.id === link.target);
    if (!srcNode || !tgtNode) return { x1: 0, y1: 0, x2: 0, y2: 0 };
    return { x1: srcNode.x, y1: srcNode.y, x2: tgtNode.x, y2: tgtNode.y };
  };

  return (
    <section className="view active">
      <div className="eyebrow">03 • broken trace-chain detector • §8.2</div>

      <div className="tag-row">
        <span className="tag">
          orphan rate <b style={{ color: 'var(--amber)' }}>{activeOrphans.orphanRate}</b>
        </span>
        <span className="tag">
          threshold <b>5%</b>
        </span>
        <span className="tag">
          correlation window <b>30s</b>
        </span>
        <span className="tag">
          clock skew tolerance <b>5s</b>
        </span>
      </div>

      {error && (
        <ErrorBanner message={`Error loading trace analytics: ${errorMsg ?? 'Unknown Error'}. Showing local simulations.`} />
      )}

      {loading ? (
        <SkeletonLoader rows={4} />
      ) : (
        <div className="grid2" style={{ gridTemplateColumns: '1.2fr 0.8fr', gap: '16px' }}>
          {/* Topology Interactive Panel */}
          <div className="panel" style={{ position: 'relative' }}>
            <div className="metric-label" style={{ marginBottom: '14px' }}>
              Service Topology Map (Click nodes to inspect)
            </div>

            <svg viewBox="0 0 460 220" style={{ width: '100%', height: '220px', overflow: 'visible' }}>
              {/* Draw Links */}
              {links.map((link, idx) => {
                const pts = getLinkPoints(link);
                const isSelected = selectedNode === link.source || selectedNode === link.target;
                return (
                  <g key={idx}>
                    <line
                      x1={pts.x1}
                      y1={pts.y1}
                      x2={pts.x2}
                      y2={pts.y2}
                      stroke={link.isBroken ? 'var(--red)' : isSelected ? 'var(--phosphor)' : 'var(--bezel)'}
                      strokeWidth={link.isBroken ? '2' : isSelected ? '2' : '1.5'}
                      strokeDasharray={link.isBroken ? '4 4' : undefined}
                      className={link.isBroken ? 'trace-line broken' : undefined}
                    />
                    {link.isBroken && (
                      <circle r="4" fill="var(--red)">
                        <animateMotion
                          dur="2.5s"
                          repeatCount="indefinite"
                          path={`M ${pts.x1} ${pts.y1} L ${pts.x2} ${pts.y2}`}
                        />
                      </circle>
                    )}
                  </g>
                );
              })}

              {/* Draw Nodes */}
              {nodes.map((node) => {
                const isSelected = selectedNode === node.id;
                const strokeColor =
                  node.status === 'critical'
                    ? 'var(--red)'
                    : node.status === 'degraded'
                    ? 'var(--amber)'
                    : 'var(--phosphor)';
                
                return (
                  <g
                    key={node.id}
                    transform={`translate(${node.x},${node.y})`}
                    style={{ cursor: 'pointer' }}
                    onClick={() => setSelectedNode(node.id)}
                  >
                    {/* Pulsing ring for critical/degraded nodes */}
                    {node.status !== 'healthy' && (
                      <circle r={isSelected ? '22' : '18'} fill="none" stroke={strokeColor} strokeWidth="1" opacity="0.6">
                        <animate attributeName="r" values={isSelected ? '20;26;20' : '16;22;16'} dur="2s" repeatCount="indefinite" />
                        <animate attributeName="opacity" values="0.8;0;0.8" dur="2s" repeatCount="indefinite" />
                      </circle>
                    )}

                    {/* Node Core */}
                    <circle
                      r={isSelected ? '16' : '12'}
                      fill="var(--panel)"
                      stroke={isSelected ? 'var(--paper)' : strokeColor}
                      strokeWidth={isSelected ? '3' : '2'}
                      style={{ transition: 'all 0.15s ease' }}
                    />

                    {/* Service Label Text */}
                    <text
                      y="26"
                      textAnchor="middle"
                      className="trace-text"
                      style={{
                        fontWeight: isSelected ? '700' : '500',
                        fill: isSelected ? 'var(--paper)' : 'var(--muted)',
                        fontSize: '10px'
                      }}
                    >
                      {node.label}
                    </text>
                  </g>
                );
              })}
            </svg>
          </div>

          {/* Details RED Metrics Side Drawer */}
          <div className="panel" style={{ display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
            {activeNode ? (
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px' }}>
                  <span
                    className="metric-label"
                    style={{ textTransform: 'uppercase', color: 'var(--paper)', fontSize: '12px' }}
                  >
                    {activeNode.label}
                  </span>
                  <span
                    className={`badge ${
                      activeNode.status === 'critical'
                        ? 'badge-err'
                        : activeNode.status === 'degraded'
                        ? 'badge-warn'
                        : 'badge-ok'
                    }`}
                  >
                    {activeNode.status}
                  </span>
                </div>

                {/* RED Metrics Grid */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '16px' }}>
                  <div style={{ background: 'var(--panel-2)', padding: '10px', borderRadius: '4px' }}>
                    <div style={{ fontSize: '10px', color: 'var(--muted)', textTransform: 'uppercase' }}>throughput</div>
                    <div style={{ fontSize: '14px', fontWeight: '600', color: 'var(--paper)', marginTop: '4px' }}>
                      {activeNode.rate}
                    </div>
                  </div>
                  <div style={{ background: 'var(--panel-2)', padding: '10px', borderRadius: '4px' }}>
                    <div style={{ fontSize: '10px', color: 'var(--muted)', textTransform: 'uppercase' }}>avg latency</div>
                    <div style={{ fontSize: '14px', fontWeight: '600', color: 'var(--paper)', marginTop: '4px' }}>
                      {activeNode.latency}
                    </div>
                  </div>
                  <div style={{ background: 'var(--panel-2)', padding: '10px', borderRadius: '4px', gridColumn: 'span 2' }}>
                    <div style={{ fontSize: '10px', color: 'var(--muted)', textTransform: 'uppercase' }}>error/orphan rate</div>
                    <div
                      style={{
                        fontSize: '14px',
                        fontWeight: '600',
                        color: activeNode.status === 'critical' ? 'var(--red)' : 'var(--paper)',
                        marginTop: '4px',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '6px'
                      }}
                    >
                      <Activity size={12} />
                      {activeNode.errors}
                    </div>
                  </div>
                </div>

                <div style={{ fontSize: '11px', color: 'var(--muted)', marginBottom: '6px', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Recent Operations
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', maxHeight: '100px', overflowY: 'auto' }}>
                  {activeNode.ops.map((op) => (
                    <div
                      key={op}
                      style={{
                        fontFamily: 'var(--mono)',
                        fontSize: '11px',
                        background: 'var(--bezel-soft)',
                        padding: '6px 8px',
                        borderRadius: '4px',
                        color: 'var(--muted-2)'
                      }}
                    >
                      {op}
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div style={{ textAlign: 'center', color: 'var(--muted)', padding: '40px 0' }}>
                <HelpCircle size={32} style={{ marginBottom: '10px', opacity: 0.5 }} />
                <div>Select a node on the topology map to view metrics</div>
              </div>
            )}

            {activeNode?.status === 'critical' && (
              <div
                style={{
                  marginTop: '16px',
                  background: 'var(--red-dim)',
                  border: '1px solid var(--red)',
                  borderRadius: '4px',
                  padding: '10px',
                  fontSize: '11px',
                  color: 'var(--red)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px'
                }}
              >
                <AlertTriangle size={14} />
                <span>Clock skew skewing spans beyond arrival limits. Apply OTel config.</span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Recent events panel */}
      <h2 className="section-title">Recent Orphan Events</h2>
      <div className="panel panel-tight">
        {loading ? (
          <SkeletonLoader rows={3} />
        ) : (
          orphanEvents.map((evt) => (
            <div className="rack-row" key={evt.id}>
              <span className={`rled ${evt.severity}`}></span>
              <div style={{ flex: 1 }}>
                <div className="rack-svc" style={{ fontSize: '12px' }}>
                  {evt.span} • {evt.collector} ({evt.service})
                </div>
                <div className="rack-desc">{evt.desc}</div>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
