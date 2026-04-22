'use client';

import React, { useState, useCallback, useEffect, useRef } from 'react';
import ReactFlow, {
    Node,
    Edge,
    Controls,
    Background,
    BackgroundVariant,
    MiniMap,
    useNodesState,
    useEdgesState,
    ReactFlowProvider,
    MarkerType,
    Panel,
} from 'reactflow';
import 'reactflow/dist/style.css';

import { getLayoutedElements } from './layout.utils';
import type { BaseNode, LineageEdge } from './lineage.types';
import LineageNode from './LineageNode';
import EmptyState from '@/components/EmptyState';

interface LineageCanvasProps {
    nodes: BaseNode[];
    edges: LineageEdge[];
    onNodeClick?: (nodeId: string) => void;
    focusedNodeId?: string | null;
}

// ─── Edge styling by relationship type ───────────────────────────────────────

function buildEdgeStyle(edgeType: string, riskLevel?: string): Partial<Edge> {
    if (edgeType === 'EXPOSES') {
        const color = riskLevel === 'Critical' ? '#ef4444'
            : riskLevel === 'High' ? '#f97316'
            : riskLevel === 'Medium' ? '#eab308'
            : '#22c55e';
        return {
            type: 'smoothstep',
            animated: true,
            label: 'EXPOSES',
            labelStyle: { fontSize: 9, fontWeight: 700, fill: color },
            labelBgStyle: { fill: '#fff', fillOpacity: 0.85 },
            labelBgPadding: [3, 5] as [number, number],
            labelBgBorderRadius: 3,
            style: { stroke: color, strokeWidth: 2 },
            markerEnd: { type: MarkerType.ArrowClosed, color, width: 14, height: 14 },
        };
    }
    // SYSTEM_OWNS_ASSET
    return {
        type: 'smoothstep',
        animated: false,
        style: { stroke: '#3b82f6', strokeWidth: 1.5, strokeDasharray: '5 3' },
        markerEnd: { type: MarkerType.ArrowClosed, color: '#3b82f6', width: 12, height: 12 },
    };
}

// ─── Custom node types ────────────────────────────────────────────────────────

const nodeTypes = { lineageNode: LineageNode };

// ─── Canvas content ───────────────────────────────────────────────────────────

function LineageCanvasContent({ nodes: graphNodes, edges: graphEdges, onNodeClick, focusedNodeId }: LineageCanvasProps) {
    const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

    // Build richer edges: color by risk of target PII node
    const richEdges = React.useMemo(() => {
        const nodeMap: Record<string, BaseNode> = {};
        graphNodes.forEach(n => { nodeMap[n.id] = n; });

        return graphEdges.map(edge => {
            const target = nodeMap[edge.target];
            const riskLevel = (target?.metadata as any)?.risk_level;
            return {
                id: edge.id,
                source: edge.source,
                target: edge.target,
                data: edge,
                ...buildEdgeStyle(edge.type, riskLevel),
            };
        });
    }, [graphNodes, graphEdges]);

    // Calculate layout
    const { nodes: layoutedNodes, edges: layoutedEdges } = React.useMemo(() => {
        return getLayoutedElements(
            graphNodes.map(node => ({
                id: node.id,
                data: node,
                position: { x: 0, y: 0 },
                type: 'lineageNode',
                selected: node.id === selectedNodeId,
            })),
            richEdges
        );
    }, [graphNodes, richEdges, selectedNodeId]);

    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);

    useEffect(() => {
        setNodes(layoutedNodes);
        setEdges(layoutedEdges);
    }, [layoutedNodes, layoutedEdges, setNodes, setEdges]);

    const handleNodeClick = useCallback(
        (_: React.MouseEvent, node: Node) => {
            setSelectedNodeId(prev => prev === node.id ? null : node.id);
            onNodeClick?.(node.id);
        },
        [onNodeClick]
    );

    if (graphNodes.length === 0) {
        return (
            <EmptyState
                icon="🔗"
                title="No Lineage Data"
                description="No lineage graph available. Run a scan to populate."
            />
        );
    }

    return (
        <div style={{ height: '100%', width: '100%', position: 'relative' }}>
            <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onNodeClick={handleNodeClick}
                nodeTypes={nodeTypes}
                nodesDraggable
                nodesConnectable={false}
                nodesFocusable
                edgesFocusable={false}
                elementsSelectable
                fitView
                fitViewOptions={{ padding: 0.2, minZoom: 0.3, maxZoom: 1.5 }}
                minZoom={0.15}
                maxZoom={2}
                proOptions={{ hideAttribution: true }}
                defaultEdgeOptions={{
                    style: { stroke: '#94a3b8', strokeWidth: 1.5 },
                }}
                style={{ background: '#f8fafc' }}
            >
                {/* Subtle dot grid */}
                <Background
                    variant={BackgroundVariant.Dots}
                    gap={20}
                    size={1}
                    color="#cbd5e1"
                    style={{ opacity: 0.6 }}
                />

                {/* Zoom / fit controls */}
                <Controls
                    showInteractive={false}
                    style={{
                        background: '#fff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '8px',
                        boxShadow: '0 1px 4px rgba(0,0,0,0.07)',
                        bottom: 16,
                        left: 16,
                    }}
                />

                {/* Mini-map */}
                <MiniMap
                    nodeColor={(n) => {
                        const t = n.data?.type;
                        if (t === 'system') return '#3b82f6';
                        if (t === 'asset' || t === 'table' || t === 'file') return '#10b981';
                        if (t === 'pii_category') {
                            const r = n.data?.metadata?.risk_level;
                            return r === 'Critical' ? '#ef4444' : r === 'High' ? '#f97316' : r === 'Medium' ? '#eab308' : '#22c55e';
                        }
                        return '#94a3b8';
                    }}
                    style={{
                        background: '#fff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '8px',
                        boxShadow: '0 1px 4px rgba(0,0,0,0.07)',
                        bottom: 16,
                        right: 16,
                    }}
                    maskColor="rgba(241, 245, 249, 0.75)"
                    zoomable
                    pannable
                />

                {/* Legend panel */}
                <Panel position="top-right">
                    <GraphLegend nodes={graphNodes} />
                </Panel>

                {/* Stats panel */}
                <Panel position="top-left">
                    <div style={{
                        background: '#fff', border: '1px solid #e2e8f0',
                        borderRadius: '8px', padding: '8px 12px',
                        fontSize: '11px', color: '#64748b',
                        boxShadow: '0 1px 4px rgba(0,0,0,0.07)',
                        display: 'flex', gap: '14px',
                    }}>
                        <span><strong style={{ color: '#0f172a' }}>{graphNodes.length}</strong> nodes</span>
                        <span><strong style={{ color: '#0f172a' }}>{graphEdges.length}</strong> edges</span>
                        {selectedNodeId && (
                            <span style={{ color: '#3b82f6', cursor: 'pointer' }} onClick={() => setSelectedNodeId(null)}>
                                ✕ deselect
                            </span>
                        )}
                    </div>
                </Panel>
            </ReactFlow>
        </div>
    );
}

// ─── Legend ───────────────────────────────────────────────────────────────────

// PII type → risk color (matches LineageNode.tsx accent bar)
const RISK_COLOR: Record<string, string> = {
    Critical: '#ef4444',
    High:     '#f97316',
    Medium:   '#eab308',
    Low:      '#22c55e',
};

interface GraphLegendProps {
    nodes: BaseNode[];
}

function GraphLegend({ nodes }: GraphLegendProps) {
    // Collect unique PII types from the rendered graph, preserving their risk
    // color so the legend swatch matches the node's visible accent bar.
    const piiTypes = React.useMemo(() => {
        const seen = new Map<string, string>(); // pii_type → risk color
        nodes.forEach(n => {
            if (n.type !== 'pii_category') return;
            const meta = n.metadata as Record<string, any>;
            const label: string = meta?.pii_type || n.label;
            if (!label || seen.has(label)) return;
            const risk = meta?.risk_level as string | undefined;
            seen.set(label, RISK_COLOR[risk ?? ''] ?? '#64748b');
        });
        return Array.from(seen, ([label, color]) => ({ label, color }))
            .sort((a, b) => a.label.localeCompare(b.label));
    }, [nodes]);

    return (
        <div style={{
            background: '#fff', border: '1px solid #e2e8f0',
            borderRadius: '8px', padding: '10px 14px',
            boxShadow: '0 1px 4px rgba(0,0,0,0.07)',
            fontSize: '11px', color: '#475569',
            display: 'flex', flexDirection: 'column', gap: '6px',
            minWidth: '160px', maxWidth: '220px',
            maxHeight: '60vh', overflowY: 'auto',
        }}>
            <div style={{ fontWeight: 700, color: '#0f172a', fontSize: '11px', marginBottom: '2px' }}>Legend</div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '7px' }}>
                    <div style={{ width: '10px', height: '10px', borderRadius: '2px', background: '#3b82f6', flexShrink: 0 }} />
                    <span>System</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '7px' }}>
                    <div style={{ width: '10px', height: '10px', borderRadius: '2px', background: '#10b981', flexShrink: 0 }} />
                    <span>Asset</span>
                </div>
            </div>

            {piiTypes.length > 0 && (
                <div style={{ borderTop: '1px solid #f1f5f9', paddingTop: '6px', marginTop: '2px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
                    <div style={{ fontSize: '10px', fontWeight: 600, color: '#94a3b8', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                        PII Types
                    </div>
                    {piiTypes.map(item => (
                        <div key={item.label} style={{ display: 'flex', alignItems: 'center', gap: '7px' }}>
                            <div style={{ width: '10px', height: '10px', borderRadius: '2px', background: item.color, flexShrink: 0 }} />
                            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#0f172a', wordBreak: 'break-all' }}>
                                {item.label}
                            </span>
                        </div>
                    ))}
                </div>
            )}

            <div style={{ borderTop: '1px solid #f1f5f9', paddingTop: '6px', marginTop: '2px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '7px' }}>
                    <div style={{ width: '18px', height: '2px', background: '#3b82f6', borderRadius: '1px', flexShrink: 0, borderTop: '1px dashed #3b82f6' }} />
                    <span>Owns</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '7px' }}>
                    <div style={{ width: '18px', height: '2px', background: '#ef4444', borderRadius: '1px', flexShrink: 0 }} />
                    <span>Exposes PII</span>
                </div>
            </div>
        </div>
    );
}

// ─── Exported wrapper ─────────────────────────────────────────────────────────

export default function LineageCanvas(props: LineageCanvasProps) {
    return (
        <ReactFlowProvider>
            <LineageCanvasContent {...props} />
        </ReactFlowProvider>
    );
}
