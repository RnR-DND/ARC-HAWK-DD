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

                    {/* Lineage Canvas */}
                    {lineageData ? (
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
                    )}
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
