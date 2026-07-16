import { useCallback, useEffect, useMemo } from 'react';
import { Cpu, Search, Terminal, Server, AlertCircle } from 'lucide-react';
import {
  ReactFlow,
  MiniMap,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  Handle,
  Position
} from '@xyflow/react';
import type { Connection, Edge, Node } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useTenantData, ErrorBanner, SkeletonLoader } from '../Shared';

// --- Custom Premium Behavior Node ---
const BehaviorNode = ({ data }: any) => {
  const isError = data.status === 'error';
  const isWarning = data.status === 'warning';
  
  let Icon = Server;
  let color = 'var(--phosphor)'; // default green
  let bg = 'rgba(74, 222, 128, 0.05)';
  let border = 'rgba(74, 222, 128, 0.2)';

  if (data.type === 'planner') {
    Icon = Cpu;
    color = '#A78BFA'; // Purple
    bg = 'rgba(167, 139, 250, 0.08)';
    border = 'rgba(167, 139, 250, 0.3)';
  } else if (data.type === 'retriever') {
    Icon = Search;
    color = '#60A5FA'; // Blue
    bg = 'rgba(96, 165, 250, 0.08)';
    border = 'rgba(96, 165, 250, 0.3)';
  } else if (data.type === 'tool') {
    Icon = Terminal;
    color = '#FBBF24'; // Amber
    bg = 'rgba(251, 191, 36, 0.08)';
    border = 'rgba(251, 191, 36, 0.3)';
  }

  if (isError) {
    color = 'var(--red)';
    bg = 'rgba(248, 113, 113, 0.1)';
    border = 'rgba(248, 113, 113, 0.4)';
  } else if (isWarning) {
    color = 'var(--amber)';
    bg = 'rgba(251, 191, 36, 0.1)';
    border = 'rgba(251, 191, 36, 0.4)';
  }

  return (
    <div
      style={{
        background: 'var(--panel-bg, #0f1115)',
        border: `1px solid ${border}`,
        borderRadius: '8px',
        padding: '12px 16px',
        minWidth: '180px',
        boxShadow: `0 4px 20px ${bg}`,
        display: 'flex',
        flexDirection: 'column',
        gap: '8px',
        position: 'relative',
        overflow: 'hidden'
      }}
    >
      <Handle type="target" position={Position.Left} style={{ background: 'var(--bezel)' }} />
      
      {/* Top accent line */}
      <div style={{ position: 'absolute', top: 0, left: 0, right: 0, height: '3px', background: color }} />
      
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <div style={{ background: bg, padding: '6px', borderRadius: '6px', display: 'flex' }}>
          <Icon size={16} color={color} />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--paper)', fontFamily: 'var(--sans)' }}>
            {data.label}
          </span>
          <span style={{ fontSize: '10px', color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
            {data.type}
          </span>
        </div>
      </div>

      {data.detail && (
        <div style={{ fontSize: '10px', color: 'var(--muted)', fontFamily: 'var(--mono)', marginTop: '4px', borderTop: '1px solid var(--bezel)', paddingTop: '8px' }}>
          {data.detail}
        </div>
      )}

      {(isError || isWarning) && (
        <div style={{ position: 'absolute', top: '12px', right: '12px' }}>
          <AlertCircle size={14} color={color} />
        </div>
      )}

      <Handle type="source" position={Position.Right} style={{ background: 'var(--bezel)' }} />
    </div>
  );
};
// -------------------------------------

interface GraphData {
  nodes: Node[];
  edges: Edge[];
}

interface DigitalTwinProps {
  tenantId: string;
}

export function DigitalTwin({ tenantId }: DigitalTwinProps) {
  const { data, loading, error, errorMsg } = useTenantData<GraphData>(
    tenantId,
    'behavior',
    { nodes: [], edges: [] }
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(data?.nodes || []);
  const [edges, setEdges, onEdgesChange] = useEdgesState(data?.edges || []);

  const nodeTypes = useMemo(() => ({
    service: BehaviorNode,
    planner: BehaviorNode,
    retriever: BehaviorNode,
    tool: BehaviorNode,
    memory: BehaviorNode
  }), []);

  const onConnect = useCallback(
    (params: Connection | Edge) => setEdges((eds) => addEdge(params, eds)),
    [setEdges],
  );

  // Update state when data arrives
  useEffect(() => {
    if (data) {
      setNodes(data.nodes);
      setEdges(data.edges);
    }
  }, [data, setNodes, setEdges]);

  if (loading && !data) return <SkeletonLoader />;

  return (
    <div className="view-panel active">
      {error && <ErrorBanner message={errorMsg} />}
      <div className="panel-header">
        <h2 className="title">BEHAVIOR GRAPH</h2>
      </div>
      
      <div className="content-grid" style={{ height: '600px', width: '100%' }}>
        <ReactFlow
          nodes={data?.nodes || nodes}
          edges={data?.edges || edges}
          nodeTypes={nodeTypes}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          fitView
          colorMode="dark"
        >
          <Controls />
          <MiniMap />
          <Background gap={12} size={1} />
        </ReactFlow>
      </div>
    </div>
  );
}
