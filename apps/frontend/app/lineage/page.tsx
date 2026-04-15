'use client';

import React, { useEffect, useState } from 'react';
import InfoPanel from '@/components/InfoPanel';
import LineageCanvas from '@/modules/lineage/LineageCanvas';
import LoadingState from '@/components/LoadingState';
import { lineageApi } from '@/services/lineage.api';
import type { LineageGraphData } from '@/modules/lineage/lineage.types';
import { theme } from '@/design-system/theme';

// Node type config: icon, label, colors
const NODE_CONFIG: Record<string, { icon: string; label: string; bg: string; border: string; text: string; dot: string }> = {
    system:       { icon: '🌐', label: 'System',   bg: '#0f172a', border: '#3b82f6', text: '#93c5fd', dot: '#3b82f6' },
    asset:        { icon: '🗄️',  label: 'Asset',    bg: '#052e16', border: '#22c55e', text: '#86efac', dot: '#22c55e' },
    table:        { icon: '📋', label: 'Table',    bg: '#0c1a2e', border: '#38bdf8', text: '#7dd3fc', dot: '#38bdf8' },
    file:         { icon: '📂', label: 'File',     bg: '#1c1917', border: '#d4a373', text: '#d4a373', dot: '#d4a373' },
    pii_category: { icon: '🔴', label: 'PII Type', bg: '#2d0b3e', border: '#c084fc', text: '#f0abfc', dot: '#c084fc' },
};
const DEFAULT_NODE_CFG = { icon: '◉', label: 'Node', bg: '#1e293b', border: '#64748b', text: '#cbd5e1', dot: '#64748b' };

function nodeConfig(type: string) {
    return NODE_CONFIG[type] ?? DEFAULT_NODE_CFG;
}

function riskColor(score: number | undefined) {
    if (!score) return { bg: '#1e293b', text: '#94a3b8', bar: '#334155' };
    if (score >= 80) return { bg: '#3b0a0a', text: '#f87171', bar: '#ef4444' };
    if (score >= 60) return { bg: '#431407', text: '#fb923c', bar: '#f97316' };
    if (score >= 40) return { bg: '#422006', text: '#fbbf24', bar: '#f59e0b' };
    return { bg: '#052e16', text: '#4ade80', bar: '#22c55e' };
}

export default function LineagePage() {
    const [lineageData, setLineageData] = useState<LineageGraphData | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
    const [focusedAssetId, setFocusedAssetId] = useState<string | null>(null);
    const [viewMode, setViewMode] = useState<'graph' | 'path'>('path');
    const [searchQuery, setSearchQuery] = useState('');
    const [expandedGroup, setExpandedGroup] = useState<string | null>(null);

    useEffect(() => {
        if (typeof window !== 'undefined') {
            const params = new URLSearchParams(window.location.search);
            const assetId = params.get('assetId');
            if (assetId) setFocusedAssetId(assetId);
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
            setError(err.message || 'Failed to load lineage data.');
        } finally {
            setLoading(false);
        }
    };

    const filteredNodes = React.useMemo(() => {
        const nodes = lineageData?.nodes || [];
        if (!searchQuery) return nodes;
        const lower = searchQuery.toLowerCase();
        return nodes.filter(n =>
            n.label.toLowerCase().includes(lower) ||
            n.type.toLowerCase().includes(lower)
        );
    }, [lineageData, searchQuery]);

    const filteredEdges = React.useMemo(() => {
        const edges = lineageData?.edges || [];
        if (!searchQuery) return edges;
        const nodeIds = new Set(filteredNodes.map(n => n.id));
        return edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));
    }, [lineageData, filteredNodes, searchQuery]);

    // Stats
    const stats = React.useMemo(() => {
        const nodes = lineageData?.nodes || [];
        const systems = nodes.filter(n => n.type === 'system').length;
        const assets = nodes.filter(n => n.type === 'asset').length;
        const piiTypes = nodes.filter(n => n.type === 'pii_category').length;
        const riskScores = nodes
            .map(n => (n.metadata as Record<string, unknown>).risk_score as number)
            .filter(Boolean);
        const avgRisk = riskScores.length
            ? Math.round(riskScores.reduce((a, b) => a + b, 0) / riskScores.length)
            : 0;
        const totalFindings = nodes.reduce((sum, n) => {
            const fc = (n.metadata as Record<string, unknown>).finding_count as number;
            return sum + (fc || 0);
        }, 0);
        return { systems, assets, piiTypes, avgRisk, totalFindings, total: nodes.length };
    }, [lineageData]);

    // Path view: build hierarchy
    const { parentMap, childrenMap, nodeMap } = React.useMemo(() => {
        const nodes = lineageData?.nodes || [];
        const edges = lineageData?.edges || [];
        const nm: Record<string, typeof nodes[0]> = {};
        nodes.forEach(n => { nm[n.id] = n; });
        const cm: Record<string, string[]> = {};
        const pm: Record<string, string> = {};
        edges.forEach(e => {
            if (!cm[e.source]) cm[e.source] = [];
            cm[e.source].push(e.target);
            pm[e.target] = e.source;
        });
        return { parentMap: pm, childrenMap: cm, nodeMap: nm };
    }, [lineageData]);

    function getAncestorPath(nodeId: string): any[] {
        const path: any[] = [];
        let cur: string | undefined = nodeId;
        const visited = new Set<string>();
        while (cur && nodeMap[cur] && !visited.has(cur)) {
            visited.add(cur);
            path.unshift(nodeMap[cur]);
            cur = parentMap[cur];
        }
        return path;
    }

    // Group PII nodes by their root system
    const groupedPaths = React.useMemo(() => {
        const nodes = lineageData?.nodes || [];
        const piiNodes = nodes.filter(n => n.type === 'pii_category');
        const groups: Record<string, { system: string; systemId: string; paths: { chain: any[]; leaf: any }[] }> = {};

        (piiNodes.length > 0 ? piiNodes : nodes).forEach(node => {
            const chain = getAncestorPath(node.id);
            const systemNode = chain.find(n => n.type === 'system');
            const groupKey = systemNode?.id ?? '__ungrouped__';
            const groupLabel = systemNode?.label ?? 'Unknown System';
            if (!groups[groupKey]) groups[groupKey] = { system: groupLabel, systemId: groupKey, paths: [] };
            groups[groupKey].paths.push({ chain, leaf: node });
        });
        return Object.values(groups);
    }, [lineageData, nodeMap, parentMap]);

    if (loading && !lineageData) {
        return <LoadingState fullScreen message="Loading data lineage..." />;
    }

    const statCards = [
        { label: 'Systems', value: stats.systems, icon: '🌐', color: '#3b82f6' },
        { label: 'Assets', value: stats.assets, icon: '🗄️', color: '#22c55e' },
        { label: 'PII Types', value: stats.piiTypes, icon: '🔴', color: '#c084fc' },
        { label: 'Findings', value: stats.totalFindings.toLocaleString(), icon: '⚠️', color: '#f97316' },
        { label: 'Avg Risk', value: stats.avgRisk ? `${stats.avgRisk}` : '—', icon: '📊', color: riskColor(stats.avgRisk).bar },
    ];

    return (
        <div style={{ minHeight: '100vh', backgroundColor: '#0a0f1e', color: '#e2e8f0', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif' }}>
            <div style={{ maxWidth: '1400px', margin: '0 auto', padding: '32px 24px' }}>

                {/* Header */}
                <div style={{ marginBottom: '32px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '16px' }}>
                        <div>
                            <h1 style={{ fontSize: '28px', fontWeight: 800, margin: 0, letterSpacing: '-0.03em', color: '#f1f5f9' }}>
                                Data Lineage
                            </h1>
                            <p style={{ margin: '6px 0 0', fontSize: '14px', color: '#64748b' }}>
                                End-to-end PII flow from source systems to detection
                            </p>
                        </div>
                        <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                            {/* Search */}
                            <div style={{ position: 'relative' }}>
                                <span style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', fontSize: '14px', color: '#475569' }}>🔍</span>
                                <input
                                    type="text"
                                    placeholder="Filter nodes..."
                                    value={searchQuery}
                                    onChange={e => setSearchQuery(e.target.value)}
                                    style={{
                                        background: '#1e293b',
                                        border: '1px solid #334155',
                                        borderRadius: '8px',
                                        padding: '8px 12px 8px 32px',
                                        fontSize: '13px',
                                        color: '#e2e8f0',
                                        outline: 'none',
                                        width: '200px',
                                    }}
                                />
                            </div>
                            {/* Refresh */}
                            <button
                                onClick={fetchData}
                                style={{ background: '#1e293b', border: '1px solid #334155', borderRadius: '8px', padding: '8px 14px', fontSize: '13px', color: '#94a3b8', cursor: 'pointer' }}
                            >
                                ↺ Refresh
                            </button>
                        </div>
                    </div>
                </div>

                {error && (
                    <div style={{ background: '#3b0a0a', border: '1px solid #7f1d1d', borderRadius: '10px', padding: '14px 18px', marginBottom: '24px', color: '#f87171', fontSize: '14px', display: 'flex', gap: '8px' }}>
                        <span>⚠️</span><span>{error}</span>
                    </div>
                )}

                {/* Stats Row */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '12px', marginBottom: '28px' }}>
                    {statCards.map(card => (
                        <div key={card.label} style={{
                            background: '#131929',
                            border: `1px solid #1e293b`,
                            borderRadius: '12px',
                            padding: '16px 20px',
                            display: 'flex',
                            flexDirection: 'column',
                            gap: '6px',
                            position: 'relative',
                            overflow: 'hidden',
                        }}>
                            <div style={{ position: 'absolute', top: 0, left: 0, right: 0, height: '3px', background: card.color, borderRadius: '12px 12px 0 0' }} />
                            <div style={{ fontSize: '20px' }}>{card.icon}</div>
                            <div style={{ fontSize: '22px', fontWeight: 700, color: '#f1f5f9', lineHeight: 1 }}>{card.value}</div>
                            <div style={{ fontSize: '11px', color: '#64748b', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>{card.label}</div>
                        </div>
                    ))}
                </div>

                {/* View tabs */}
                <div style={{ display: 'flex', gap: '4px', marginBottom: '20px', background: '#131929', borderRadius: '10px', padding: '4px', width: 'fit-content' }}>
                    {(['path', 'graph'] as const).map(mode => (
                        <button
                            key={mode}
                            onClick={() => setViewMode(mode)}
                            style={{
                                padding: '7px 20px',
                                borderRadius: '7px',
                                fontSize: '13px',
                                fontWeight: 600,
                                border: 'none',
                                cursor: 'pointer',
                                background: viewMode === mode ? '#3b82f6' : 'transparent',
                                color: viewMode === mode ? '#fff' : '#64748b',
                                transition: 'all 0.15s',
                            }}
                        >
                            {mode === 'path' ? '🗺 Path View' : '🕸 Graph View'}
                        </button>
                    ))}
                </div>

                {/* Graph View */}
                {viewMode === 'graph' && (
                    <div style={{ border: '1px solid #1e293b', borderRadius: '14px', overflow: 'hidden', height: '600px', background: '#0d1525' }}>
                        {lineageData ? (
                            <LineageCanvas
                                nodes={filteredNodes}
                                edges={filteredEdges}
                                onNodeClick={setSelectedNodeId}
                                focusedNodeId={focusedAssetId}
                            />
                        ) : (
                            <LoadingState message="Building graph..." />
                        )}
                    </div>
                )}

                {/* Path View */}
                {viewMode === 'path' && (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                        {groupedPaths.length === 0 ? (
                            <div style={{ textAlign: 'center', padding: '80px 0', color: '#475569', fontSize: '15px' }}>
                                <div style={{ fontSize: '40px', marginBottom: '12px' }}>🔍</div>
                                No lineage data available. Run a scan first.
                            </div>
                        ) : (
                            groupedPaths.map(group => {
                                const sysCfg = nodeConfig('system');
                                const isExpanded = expandedGroup === null || expandedGroup === group.systemId;
                                return (
                                    <div key={group.systemId} style={{
                                        background: '#0d1525',
                                        border: '1px solid #1e293b',
                                        borderRadius: '14px',
                                        overflow: 'hidden',
                                    }}>
                                        {/* System group header */}
                                        <div
                                            onClick={() => setExpandedGroup(expandedGroup === group.systemId ? null : group.systemId)}
                                            style={{
                                                display: 'flex',
                                                alignItems: 'center',
                                                justifyContent: 'space-between',
                                                padding: '14px 20px',
                                                cursor: 'pointer',
                                                background: 'linear-gradient(90deg, #0f172a 0%, #0d1525 100%)',
                                                borderBottom: isExpanded ? '1px solid #1e293b' : 'none',
                                            }}
                                        >
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                                <span style={{
                                                    background: sysCfg.bg,
                                                    border: `1px solid ${sysCfg.border}`,
                                                    borderRadius: '8px',
                                                    padding: '4px 10px',
                                                    fontSize: '12px',
                                                    color: sysCfg.text,
                                                    fontWeight: 700,
                                                    display: 'flex',
                                                    alignItems: 'center',
                                                    gap: '5px',
                                                }}>
                                                    {sysCfg.icon} SYSTEM
                                                </span>
                                                <span style={{ fontSize: '15px', fontWeight: 700, color: '#f1f5f9' }}>
                                                    {group.system}
                                                </span>
                                            </div>
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                                <span style={{
                                                    background: '#1e293b',
                                                    border: '1px solid #334155',
                                                    borderRadius: '20px',
                                                    padding: '2px 10px',
                                                    fontSize: '12px',
                                                    color: '#94a3b8',
                                                    fontWeight: 600,
                                                }}>
                                                    {group.paths.length} path{group.paths.length !== 1 ? 's' : ''}
                                                </span>
                                                <span style={{ color: '#475569', fontSize: '16px' }}>{isExpanded ? '▲' : '▼'}</span>
                                            </div>
                                        </div>

                                        {/* Paths */}
                                        {isExpanded && (
                                            <div style={{ padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                                {group.paths.map(({ chain, leaf }, idx) => {
                                                    const riskScore = (leaf.metadata as Record<string, unknown>).risk_score as number | undefined;
                                                    const findingCount = (leaf.metadata as Record<string, unknown>).finding_count as number | undefined;
                                                    const rc = riskColor(riskScore);
                                                    return (
                                                        <div
                                                            key={`${leaf.id}-${idx}`}
                                                            style={{
                                                                background: '#131929',
                                                                border: `1px solid #1e293b`,
                                                                borderLeft: `3px solid ${rc.bar}`,
                                                                borderRadius: '8px',
                                                                padding: '12px 16px',
                                                                display: 'flex',
                                                                alignItems: 'center',
                                                                justifyContent: 'space-between',
                                                                gap: '12px',
                                                                flexWrap: 'wrap',
                                                                cursor: 'pointer',
                                                                transition: 'border-color 0.15s',
                                                            }}
                                                            onClick={() => setSelectedNodeId(leaf.id)}
                                                        >
                                                            {/* Breadcrumb chain */}
                                                            <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: '4px', flex: 1, minWidth: 0 }}>
                                                                {chain.map((n, i) => {
                                                                    const cfg = nodeConfig(n.type);
                                                                    const isLast = i === chain.length - 1;
                                                                    return (
                                                                        <React.Fragment key={n.id}>
                                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                                                                                {/* Type pill */}
                                                                                <span style={{
                                                                                    background: cfg.bg,
                                                                                    border: `1px solid ${cfg.border}40`,
                                                                                    borderRadius: '4px',
                                                                                    padding: '1px 5px',
                                                                                    fontSize: '9px',
                                                                                    color: cfg.text,
                                                                                    fontWeight: 700,
                                                                                    textTransform: 'uppercase',
                                                                                    letterSpacing: '0.05em',
                                                                                    flexShrink: 0,
                                                                                }}>
                                                                                    {cfg.icon} {cfg.label}
                                                                                </span>
                                                                                {/* Node label */}
                                                                                <span style={{
                                                                                    fontSize: isLast ? '13px' : '12px',
                                                                                    color: isLast ? '#f1f5f9' : '#94a3b8',
                                                                                    fontFamily: 'monospace',
                                                                                    fontWeight: isLast ? 700 : 400,
                                                                                    overflow: 'hidden',
                                                                                    textOverflow: 'ellipsis',
                                                                                    whiteSpace: 'nowrap',
                                                                                    maxWidth: isLast ? '160px' : '120px',
                                                                                }}>
                                                                                    {n.label}
                                                                                </span>
                                                                            </div>
                                                                            {!isLast && (
                                                                                <span style={{ color: '#334155', fontSize: '16px', margin: '0 2px', flexShrink: 0 }}>→</span>
                                                                            )}
                                                                        </React.Fragment>
                                                                    );
                                                                })}
                                                            </div>

                                                            {/* Stats badges */}
                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexShrink: 0 }}>
                                                                {findingCount ? (
                                                                    <span style={{
                                                                        background: '#3b0a0a',
                                                                        border: '1px solid #7f1d1d',
                                                                        borderRadius: '20px',
                                                                        padding: '2px 10px',
                                                                        fontSize: '11px',
                                                                        color: '#fca5a5',
                                                                        fontWeight: 700,
                                                                        whiteSpace: 'nowrap',
                                                                    }}>
                                                                        ⚠️ {findingCount} finding{findingCount !== 1 ? 's' : ''}
                                                                    </span>
                                                                ) : null}
                                                                {riskScore != null ? (
                                                                    <div style={{
                                                                        display: 'flex',
                                                                        alignItems: 'center',
                                                                        gap: '6px',
                                                                        background: rc.bg,
                                                                        border: `1px solid ${rc.bar}40`,
                                                                        borderRadius: '20px',
                                                                        padding: '3px 10px',
                                                                    }}>
                                                                        <div style={{ width: '28px', height: '4px', background: '#1e293b', borderRadius: '2px', overflow: 'hidden' }}>
                                                                            <div style={{ width: `${Math.min(riskScore, 100)}%`, height: '100%', background: rc.bar, borderRadius: '2px' }} />
                                                                        </div>
                                                                        <span style={{ fontSize: '11px', color: rc.text, fontWeight: 700 }}>
                                                                            {Math.round(riskScore)}
                                                                        </span>
                                                                    </div>
                                                                ) : null}
                                                            </div>
                                                        </div>
                                                    );
                                                })}
                                            </div>
                                        )}
                                    </div>
                                );
                            })
                        )}
                    </div>
                )}

                {/* Footer link */}
                <div style={{ marginTop: '28px', display: 'flex', justifyContent: 'center' }}>
                    <a href="/findings" style={{ color: '#3b82f6', fontSize: '14px', fontWeight: 600, textDecoration: 'none', display: 'flex', alignItems: 'center', gap: '6px' }}>
                        View Detailed Findings <span>→</span>
                    </a>
                </div>
            </div>

            {/* Node detail panel */}
            {selectedNodeId && (
                <InfoPanel
                    nodeId={selectedNodeId}
                    nodeData={lineageData?.nodes.find(n => n.id === selectedNodeId)}
                    onClose={() => setSelectedNodeId(null)}
                />
            )}
        </div>
    );
}
