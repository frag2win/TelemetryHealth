import { useCallback, useEffect } from 'react';
import {
  ReactFlow,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
} from '@xyflow/react';
import type { Connection, Edge } from '@xyflow/react';
import { useTenantData, ErrorBanner, SkeletonLoader } from '../Shared';
import type { GraphData } from '../Shared';

interface RootCauseGraphProps {
  tenantId: string;
  issueId: string;
}

export function RootCauseGraph({ tenantId, issueId }: RootCauseGraphProps) {
  const { data, loading, error, errorMsg } = useTenantData<GraphData>(
    tenantId,
    `root-cause?issue_id=${issueId}`,
    { nodes: [], edges: [] }
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(data?.nodes || []);
  const [edges, setEdges, onEdgesChange] = useEdgesState(data?.edges || []);

  const onConnect = useCallback(
    (params: Connection | Edge) => setEdges((eds: Edge[]) => addEdge(params, eds)),
    [setEdges],
  );

  // Sync graph state when API data arrives (was a dead useCallback — Bug 1 fix)
  useEffect(() => {
    if (data) {
      setNodes(data.nodes);
      setEdges(data.edges);
    }
  }, [data, setNodes, setEdges]);

  if (loading && !data) return <SkeletonLoader />;

  return (
    <div style={{ height: '400px', width: '100%', border: '1px solid #333', borderRadius: '4px', marginTop: '1rem' }}>
      {error && <ErrorBanner message={errorMsg ?? 'Error loading root cause graph'} />}
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        fitView
        colorMode="dark"
      >
        <Controls />
        <Background gap={12} size={1} />
      </ReactFlow>
    </div>
  );
}
