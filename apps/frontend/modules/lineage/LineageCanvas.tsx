'use client';

import React, { useState, useCallback, useEffect } from 'react';
import ReactFlow, {
    Node,
    Controls,
    Background,
    MiniMap,
    useNodesState,
    useEdgesState,
    ReactFlowProvider,
} from 'reactflow';
import 'reactflow/dist/style.css';

import { getLayoutedElements } from './layout.utils';

import {
    BaseNode,
    LineageEdge,
} from './lineage.types';
import LineageNode from './LineageNode';
import LineageLegend from './LineageLegend';
import { colors } from '@/design-system/colors';
import EmptyState from '@/components/EmptyState';

interface LineageCanvasProps {
    nodes: BaseNode[];
    edges: LineageEdge[];
    onNodeClick?: (nodeId: string) => void;
    focusedNodeId?: string | null;
}

// Custom node types
const nodeTypes = {
    lineageNode: LineageNode,
};

function LineageCanvasContent({ nodes: graphNodes, edges: graphEdges, onNodeClick, focusedNodeId }: LineageCanvasProps) {
    const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

    // Calculate layout
    const { nodes: layoutedNodes, edges: layoutedEdges } = React.useMemo(() => {
        return getLayoutedElements(
            graphNodes.map(node => ({
                id: node.id,
                select: () => { },
                data: node, // base node data
                position: { x: 0, y: 0 }, // initial position
                type: 'lineageNode',
            })),
            graphEdges.map(edge => ({
                id: edge.id,
                source: edge.source,
                target: edge.target,
                type: 'smoothstep',
                animated: true,
                data: edge,
            }))
        );
    }, [graphNodes, graphEdges]);

    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);

    // Sync layout with state
    useEffect(() => {
        setNodes(layoutedNodes);
        setEdges(layoutedEdges);
    }, [layoutedNodes, layoutedEdges, setNodes, setEdges]);

    const handleNodeClick = useCallback(
        (_: React.MouseEvent, node: Node) => {
            setSelectedNodeId(node.id);
            onNodeClick && onNodeClick(node.id);
        },
        [onNodeClick]
    );

    if (graphNodes.length === 0) {
        return (
            <EmptyState
                icon="🔗"
                title="No Lineage Data"
                description="No lineage graph available. Please run a scan to populate the graph."
            />
        );
    }

    const isLargeGraph = graphNodes.length > 50;

    return (
        <>
            {isLargeGraph && (
                <div style={{
                    padding: '12px 16px',
                    marginBottom: '16px',
                    background: '#FEF9C3',
                    border: '1px solid #FDE047',
                    borderRadius: '8px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '12px',
                    fontSize: '14px',
                    color: '#854D0E'
                }}>
                    <span style={{ fontSize: '18px' }}>ℹ️</span>
                    <div>
                        <strong>Large Graph Detected</strong>
                        <span style={{ marginLeft: '8px' }}>
                            ({graphNodes.length} nodes) - Expand nodes selectively for better performance
                        </span>
                    </div>
                </div>
            )}
            <div
                style={{
                    height: 'calc(100vh - 250px)',
                    background: '#0f172a',
                    borderRadius: '12px',
                    overflow: 'hidden',
                    border: '1px solid #334155',
                    position: 'relative',
                }}
            >
                <ReactFlow
                    nodes={nodes}
                    edges={edges}
                    onNodesChange={onNodesChange}
                    onEdgesChange={onEdgesChange}
                    onNodeClick={handleNodeClick}
                    nodeTypes={nodeTypes}
                    nodesDraggable={true}
                    nodesConnectable={false}
                    nodesFocusable={true}
                    edgesFocusable={false}
                    elementsSelectable={true}
                    fitView
                    fitViewOptions={{
                        padding: 0.15,
                        minZoom: 0.5,
                        maxZoom: 1.5,
                    }}
                    minZoom={0.2}
                    maxZoom={2}
                    attributionPosition="bottom-left"
                    proOptions={{ hideAttribution: true }}
                    defaultEdgeOptions={{
                        style: {
                            stroke: '#475569',
                            strokeWidth: 1.5,
                        },
                        animated: false,
                    }}
                >
                    <Controls
                        showInteractive={false}
                        style={{
                            background: '#1e293b',
                            border: '1px solid #334155',
                            borderRadius: '8px',
                        }}
                    />
                    <Background
                        color="#334155"
                        gap={20}
                        size={0.5}
                        style={{ opacity: 0.3 }}
                    />
                    <MiniMap
                        nodeColor={(n) => {
                            const nodeType = n.data.type;
                            if (nodeType === 'system') return '#3b82f6';
                            if (nodeType === 'asset' || nodeType === 'file' || nodeType === 'table')
                                return '#a855f7';
                            if (nodeType === 'pii_category') {
                                const risk = n.data.metadata?.risk_score || 0;
                                return risk >= 70 ? '#ef4444' : risk >= 40 ? '#f97316' : '#22c55e';
                            }
                            return '#64748b';
                        }}
                        style={{
                            background: '#1e293b',
                            border: '1px solid #334155',
                            borderRadius: '8px',
                        }}
                        maskColor="rgba(15, 23, 42, 0.8)"
                    />
                </ReactFlow>

                <LineageLegend />
            </div>
        </>
    );
}

export default function LineageCanvas(props: LineageCanvasProps) {
    return (
        <ReactFlowProvider>
            <LineageCanvasContent {...props} />
        </ReactFlowProvider>
    );
}
