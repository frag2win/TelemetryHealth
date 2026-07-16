import { useCallback } from 'react';
import {
  ReactFlow,
  MiniMap,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
} from '@xyflow/react';
import type { Connection, Edge, Node } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useTenantData, ErrorBanner, SkeletonLoader } from '../Shared';

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
    'topology',
    { nodes: [], edges: [] }
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(data?.nodes || []);
  const [edges, setEdges, onEdgesChange] = useEdgesState(data?.edges || []);

  const onConnect = useCallback(
    (params: Connection | Edge) => setEdges((eds) => addEdge(params, eds)),
    [setEdges],
  );

  // Update state when data arrives
  useCallback(() => {
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
        <h2 className="title">TELEMETRY PIPELINE TOPOLOGY</h2>
      </div>
      
      <div className="content-grid" style={{ height: '600px', width: '100%' }}>
        <ReactFlow
          nodes={data?.nodes || nodes}
          edges={data?.edges || edges}
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
