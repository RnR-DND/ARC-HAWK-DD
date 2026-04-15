'use client';

import React, { useEffect, useState } from 'react';
import InfoPanel from '@/components/InfoPanel';
import LineageCanvas from '@/modules/lineage/LineageCanvas';
import LoadingState from '@/components/LoadingState';
import { lineageApi } from '@/services/lineage.api';
import type { LineageGraphData } from '@/modules/lineage/lineage.types';
import { theme } from '@/design-system/theme';

export default function DashboardPage() {
    const [lineageData, setLineageData] = useState<LineageGraphData | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Graph state
    const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
    const [focusedAssetId, setFocusedAssetId] = useState<string | null>(null);

    useEffect(() => {
        // Parse URL params for assetId
        if (typeof window !== 'undefined') {
            const params = new URLSearchParams(window.location.search);
            const assetId = params.get('assetId');
            if (assetId) {
                setFocusedAssetId(assetId);
            }
        }
        fetchData();
    }, []);

    const fetchData = async () => {
        try {
            setLoading(true);
            setError(null);
            const graphData = await lineageApi.getLineage(undefined, undefined);
            setLineageData(graphData);
        } catch (err: any) {
            console.error('Error fetching data:', err);
            setError(err.message || 'Failed to fetch data from backend. Make sure the server is running.');
        } finally {
            setLoading(false);
        }
    };

    // Calculate metrics
    const avgRiskScore = lineageData?.nodes?.length
        ? Math.round(lineageData.nodes.reduce((sum, n) => {
            if (n.type === 'asset' && 'risk_score' in n.metadata) {
                return sum + (n.metadata.risk_score as number);
            }
            return sum;
        }, 0) / lineageData.nodes.length)
        : 0;

    // View mode: graph or path tree
    const [viewMode, setViewMode] = useState<'graph' | 'path'>('graph');

    // Filter nodes based on search
    const [searchQuery, setSearchQuery] = useState('');

    const filteredNodes = React.useMemo(() => {
        const nodes = lineageData?.nodes || [];
        if (!searchQuery) return nodes;
        const lower = searchQuery.toLowerCase();
        return nodes.filter(n =>
            n.label.toLowerCase().includes(lower) ||
            n.type.toLowerCase().includes(lower)
        );
    }, [lineageData, searchQuery]);

    // Filter edges
    const filteredEdges = React.useMemo(() => {
        const edges = lineageData?.edges || [];
        if (!searchQuery) return edges;
        const nodeIds = new Set(filteredNodes.map(n => n.id));
        return edges.filter(e =>
            nodeIds.has(e.source) && nodeIds.has(e.target)
        );
    }, [lineageData, filteredNodes, searchQuery]);

    if (loading && !lineageData) {
        return <LoadingState fullScreen message="Loading ARC-Hawk dashboard..." />;
    }

    return (
        <div style={{ padding: '0', minHeight: '100vh', backgroundColor: theme.colors.background.primary }}>
            <div style={{ padding: '32px', maxWidth: '1600px', margin: '0 auto', height: 'calc(100vh - 80px)', display: 'flex', flexDirection: 'column' }}>
                {error && (
                    <div
                        style={{
                            padding: '16px 24px',
                            backgroundColor: `${theme.colors.status.error}15`,
                            border: `1px solid ${theme.colors.status.error}40`,
                            borderRadius: '12px',
                            color: theme.colors.status.error,
                            marginBottom: '24px',
                            fontWeight: 600,
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px',
                        }}
                    >
                        <span>⚠️</span>
                        <span>{error}</span>
                    </div>
                )}

                {/* Lineage Graph Section */}
                <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
                    {/* Section Header */}
                    <div
                        style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            marginBottom: '24px',
                        }}
                    >
                        <h2
                            style={{
                                fontSize: '24px',
                                fontWeight: 800,
                                color: theme.colors.text.primary,
                                margin: 0,
                                letterSpacing: '-0.02em',
                            }}
                        >
                            Data Lineage
                        </h2>

                        <span
                            style={{
                                fontSize: '14px',
                                color: theme.colors.text.secondary,
                                fontWeight: 600,
                                backgroundColor: theme.colors.background.card,
                                padding: '6px 12px',
                                borderRadius: '20px',
                                border: `1px solid ${theme.colors.border.subtle}`,
                            }}
                        >
                            🔗 Neo4j Semantic Graph
                        </span>
                    </div>

                    {/* View mode tabs */}
                    <div style={{ display: 'flex', gap: '8px', marginBottom: '16px' }}>
                        <button
                            onClick={() => setViewMode('graph')}
                            style={{
                                padding: '8px 16px',
                                borderRadius: '8px',
                                fontSize: '14px',
                                fontWeight: 500,
                                border: 'none',
                                cursor: 'pointer',
                                backgroundColor: viewMode === 'graph' ? theme.colors.primary.DEFAULT : theme.colors.background.card,
                                color: viewMode === 'graph' ? '#fff' : theme.colors.text.secondary,
                            }}
                        >
                            Graph View
                        </button>
                        <button
                            onClick={() => setViewMode('path')}
                            style={{
                                padding: '8px 16px',
                                borderRadius: '8px',
                                fontSize: '14px',
                                fontWeight: 500,
                                border: 'none',
                                cursor: 'pointer',
                                backgroundColor: viewMode === 'path' ? theme.colors.primary.DEFAULT : theme.colors.background.card,
                                color: viewMode === 'path' ? '#fff' : theme.colors.text.secondary,
                            }}
                        >
                            Path View
                        </button>
                    </div>

                    {/* Lineage Canvas */}
                    {viewMode === 'graph' && (lineageData ? (
                        <div style={{
                            flex: 1,
                            border: `1px solid ${theme.colors.border.default}`,
                            borderRadius: '12px',
                            overflow: 'hidden',
                            backgroundColor: theme.colors.background.card,
                            boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -1px rgba(0, 0, 0, 0.03)'
                        }}>
                            <LineageCanvas
                                nodes={filteredNodes}
                                edges={filteredEdges}
                                onNodeClick={setSelectedNodeId}
                                focusedNodeId={focusedAssetId}
                            />
                        </div>
                    ) : (
                        <LoadingState message="Initializing visualization..." />
                    ))}

                    {/* Path View */}
                    {viewMode === 'path' && (() => {
                        const nodes = lineageData?.nodes || [];
                        const edges = lineageData?.edges || [];

                        // Build adjacency: parent → children
                        const children: Record<string, string[]> = {};
                        edges.forEach(e => {
                            if (!children[e.source]) children[e.source] = [];
                            children[e.source].push(e.target);
                        });
                        // Build reverse: child → parent
                        const parent: Record<string, string> = {};
                        edges.forEach(e => { parent[e.target] = e.source; });

                        const nodeMap: Record<string, typeof nodes[0]> = {};
                        nodes.forEach(n => { nodeMap[n.id] = n; });

                        // Build full path for a node by walking up to root
                        function getAncestorPath(nodeId: string): typeof nodes[0][] {
                            const path: typeof nodes[0][] = [];
                            let cur: string | undefined = nodeId;
                            const visited = new Set<string>();
                            while (cur && nodeMap[cur] && !visited.has(cur)) {
                                visited.add(cur);
                                path.unshift(nodeMap[cur]);
                                cur = parent[cur];
                            }
                            return path;
                        }

                        // Show PII category nodes with their full ancestor chain
                        const piiNodes = nodes.filter(n => n.type === 'pii_category');
                        // If no PII nodes, show all nodes grouped by type
                        const showNodes = piiNodes.length > 0 ? piiNodes : nodes;

                        const typeColors: Record<string, { bg: string; text: string }> = {
                            system:       { bg: '#1e3a5f', text: '#93c5fd' },
                            asset:        { bg: '#1e4d2b', text: '#86efac' },
                            pii_category: { bg: '#4c1d3f', text: '#f0abfc' },
                            file:         { bg: '#2d2d2d', text: '#d1d5db' },
                            table:        { bg: '#1c3557', text: '#7dd3fc' },
                        };

                        const getColor = (type: string) => typeColors[type] ?? { bg: '#2d2d2d', text: '#d1d5db' };

                        return (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', overflowY: 'auto', maxHeight: 'calc(100vh - 320px)' }}>
                                {showNodes.map(node => {
                                    const chain = getAncestorPath(node.id);
                                    return (
                                        <div
                                            key={node.id}
                                            style={{
                                                backgroundColor: theme.colors.background.card,
                                                border: `1px solid ${theme.colors.border.subtle}`,
                                                borderRadius: '10px',
                                                padding: '12px 16px',
                                            }}
                                        >
                                            {/* Breadcrumb chain */}
                                            <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: '4px' }}>
                                                {chain.map((n, i) => {
                                                    const col = getColor(n.type);
                                                    return (
                                                        <React.Fragment key={n.id}>
                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                                <span style={{
                                                                    fontSize: '10px',
                                                                    padding: '2px 6px',
                                                                    borderRadius: '4px',
                                                                    backgroundColor: col.bg,
                                                                    color: col.text,
                                                                    fontWeight: 600,
                                                                    textTransform: 'uppercase',
                                                                    letterSpacing: '0.05em',
                                                                }}>
                                                                    {n.type}
                                                                </span>
                                                                <span style={{
                                                                    fontSize: '13px',
                                                                    color: i === chain.length - 1 ? theme.colors.text.primary : theme.colors.text.secondary,
                                                                    fontFamily: 'monospace',
                                                                    fontWeight: i === chain.length - 1 ? 600 : 400,
                                                                }}>
                                                                    {n.label}
                                                                </span>
                                                            </div>
                                                            {i < chain.length - 1 && (
                                                                <span style={{ color: theme.colors.text.muted, fontSize: '14px', fontWeight: 300 }}>›</span>
                                                            )}
                                                        </React.Fragment>
                                                    );
                                                })}
                                            </div>
                                            {/* Finding count badge */}
                                            {'finding_count' in node.metadata && node.metadata.finding_count ? (
                                                <div style={{ marginTop: '8px', display: 'flex', gap: '8px', alignItems: 'center' }}>
                                                    <span style={{
                                                        fontSize: '11px',
                                                        color: '#f87171',
                                                        backgroundColor: 'rgba(248,113,113,0.1)',
                                                        padding: '2px 8px',
                                                        borderRadius: '4px',
                                                        fontWeight: 600,
                                                    }}>
                                                        {String(node.metadata.finding_count)} findings
                                                    </span>
                                                    {(node.metadata as Record<string, unknown>).risk_score != null && (
                                                        <span style={{
                                                            fontSize: '11px',
                                                            color: '#fb923c',
                                                            backgroundColor: 'rgba(251,146,60,0.1)',
                                                            padding: '2px 8px',
                                                            borderRadius: '4px',
                                                            fontWeight: 600,
                                                        }}>
                                                            Risk {Number((node.metadata as Record<string, unknown>).risk_score).toFixed(0)}
                                                        </span>
                                                    )}
                                                </div>
                                            ) : null}
                                        </div>
                                    );
                                })}
                                {!nodes.length && (
                                    <p style={{ textAlign: 'center', color: theme.colors.text.muted, padding: '48px 0', fontSize: '14px' }}>
                                        No lineage data available. Run a scan to populate.
                                    </p>
                                )}
                            </div>
                        );
                    })()}
                </div>

                {/* Findings Link */}
                <div style={{ marginTop: '24px', textAlign: 'center' }}>
                    <a href="/findings" style={{ color: theme.colors.primary.DEFAULT, fontSize: '14px', fontWeight: 600, textDecoration: 'none' }}>
                        View Detailed Findings &rarr;
                    </a>
                </div>

                {/* InfoPanel */}
                {selectedNodeId && (
                    <InfoPanel
                        nodeId={selectedNodeId}
                        nodeData={lineageData?.nodes.find(n => n.id === selectedNodeId)}
                        onClose={() => setSelectedNodeId(null)}
                    />
                )}
            </div>

            <style jsx global>{`
                @keyframes spin {
                    to { transform: rotate(360deg); }
                }
            `}</style>
        </div>
    );
}
