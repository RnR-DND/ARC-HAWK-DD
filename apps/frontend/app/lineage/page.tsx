'use client';

import React, { useEffect, useState } from 'react';
import InfoPanel from '@/components/InfoPanel';
import LineageCanvas from '@/modules/lineage/LineageCanvas';
import LoadingState from '@/components/LoadingState';
import { lineageApi, fetchLineage, syncLineage } from '@/services/lineage.api';
import { findingsApi } from '@/services/findings.api';
import type { LineageGraphData } from '@/modules/lineage/lineage.types';
import { theme } from '@/design-system/theme';

// ─── Severity helpers ────────────────────────────────────────────────────────

const SEVERITY_STYLE: Record<string, { bg: string; text: string; border: string }> = {
    Critical: { bg: '#fef2f2', text: '#b91c1c', border: '#fecaca' },
    High:     { bg: '#fff7ed', text: '#c2410c', border: '#fed7aa' },
    Medium:   { bg: '#fefce8', text: '#92400e', border: '#fde68a' },
    Low:      { bg: '#f0fdf4', text: '#166534', border: '#bbf7d0' },
};

const RISK_BAR: Record<string, string> = {
    Critical: '#ef4444',
    High:     '#f97316',
    Medium:   '#eab308',
    Low:      '#22c55e',
};

function SeverityBadge({ level }: { level: string }) {
    const s = SEVERITY_STYLE[level] ?? { bg: '#f8fafc', text: '#64748b', border: '#e2e8f0' };
    return (
        <span style={{
            background: s.bg, color: s.text, border: `1px solid ${s.border}`,
            borderRadius: '4px', padding: '1px 6px', fontSize: '11px', fontWeight: 700,
        }}>
            {level}
        </span>
    );
}

// ─── Confidence bar ───────────────────────────────────────────────────────────

function ConfBar({ score }: { score: number }) {
    const pct = Math.round(score * 100);
    const color = score >= 0.85 ? '#22c55e' : score >= 0.65 ? '#f59e0b' : '#ef4444';
    return (
        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
            <div style={{ width: '40px', height: '4px', background: '#e2e8f0', borderRadius: '2px', overflow: 'hidden' }}>
                <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: '2px' }} />
            </div>
            <span style={{ fontSize: '11px', color: '#64748b', fontWeight: 600 }}>{pct}%</span>
        </div>
    );
}

// ─── Types ───────────────────────────────────────────────────────────────────

interface FieldOccurrence {
    findingId: string;
    assetPath: string;   // e.g. "public.customers.aadhaar_no" or "/data/export.csv:col_2"
    field: string;       // last segment
    severity: string;
    confidence: number;
    sampleText: string;
    reviewStatus: string;
}

interface PIIGroup {
    piiType: string;
    riskLevel: string;
    dpdpaCategory: string;
    findingCount: number;
    avgConfidence: number;
    fields: FieldOccurrence[];   // unique field paths from findings
}

interface AssetGroup {
    assetId: string;
    assetLabel: string;
    assetPath: string;
    environment: string;
    piiGroups: PIIGroup[];
}

interface SystemGroup {
    systemId: string;
    systemLabel: string;
    host: string;
    assets: AssetGroup[];
}

// ─── Path parsing ────────────────────────────────────────────────────────────

function splitPath(assetPath: string): { dir: string; field: string } {
    if (!assetPath) return { dir: '—', field: '—' };
    const sep = Math.max(assetPath.lastIndexOf('/'), assetPath.lastIndexOf(':'), assetPath.lastIndexOf('.'));
    if (sep <= 0) return { dir: assetPath, field: assetPath };
    return { dir: assetPath.substring(0, sep), field: assetPath.substring(sep + 1) };
}

// ─── Main component ──────────────────────────────────────────────────────────

export default function LineagePage() {
    const [lineageData, setLineageData] = useState<LineageGraphData | null>(null);
    const [aggregations, setAggregations] = useState<{
        total_assets: number; total_pii_types: number; total_systems: number;
        by_pii_type: { count: number; pii_type: string }[];
    } | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
    const [focusedAssetId, setFocusedAssetId] = useState<string | null>(null);
    const [syncing, setSyncing] = useState(false);
    const [syncStatus, setSyncStatus] = useState<string | null>(null);
    const [viewMode, setViewMode] = useState<'path' | 'graph'>('path');
    const [searchQuery, setSearchQuery] = useState('');
    const [expandedSystems, setExpandedSystems] = useState<Set<string>>(new Set());
    const [expandedAssets, setExpandedAssets] = useState<Set<string>>(new Set());
    const [expandedPII, setExpandedPII] = useState<Set<string>>(new Set());
    const [systemGroups, setSystemGroups] = useState<SystemGroup[]>([]);

    useEffect(() => {
        fetchData();
    }, []);

    const fetchData = async () => {
        const assetId = typeof window !== 'undefined'
            ? (new URLSearchParams(window.location.search).get('assetId') ?? undefined)
            : undefined;
        if (assetId) setFocusedAssetId(assetId);

        try {
            setLoading(true);
            setError(null);

            // Fetch lineage + findings in parallel; forward assetId filter when present
            const [fullLineage, findingsResult] = await Promise.all([
                fetchLineage(undefined, undefined, assetId),
                findingsApi.getFindings({ page: 1, page_size: 200, sort_by: 'severity', sort_order: 'desc' }),
            ]);

            const graphData = {
                nodes: fullLineage.hierarchy?.nodes || [],
                edges: fullLineage.hierarchy?.edges || [],
            };
            setLineageData(graphData);

            // Extract aggregations
            if (fullLineage.aggregations) {
                setAggregations(fullLineage.aggregations as any);
            }

            // Build system groups from lineage + findings
            const nodes = graphData.nodes || [];
            const edges = graphData.edges || [];

            // Build maps
            const nodeMap: Record<string, typeof nodes[0]> = {};
            nodes.forEach(n => { nodeMap[n.id] = n; });

            // parent map: target → source
            const parentMap: Record<string, string> = {};
            edges.forEach(e => { parentMap[e.target] = e.source; });

            // children: source → targets
            const childrenMap: Record<string, string[]> = {};
            edges.forEach(e => {
                if (!childrenMap[e.source]) childrenMap[e.source] = [];
                childrenMap[e.source].push(e.target);
            });

            // Edge map keyed by `source__target`. Per-(asset,pii) finding_count
            // lives on the EXPOSES edge's metadata; the PII_Category node is
            // MERGEd globally by pii_type so its finding_count gets overwritten
            // by the last-synced asset and cannot be trusted for per-asset display.
            const edgeByEndpoints: Record<string, typeof edges[0]> = {};
            edges.forEach(e => { edgeByEndpoints[`${e.source}__${e.target}`] = e; });

            // Map findings by (assetId, subCategory) → occurrences. We key by the
            // classification's sub_category (e.g. "IN_PAN") so findings from
            // multiple pattern sources ("PAN", "presidio:IN_PAN") collapse into
            // the same PII group. Fallback to pattern_name if no classification.
            const findingsByAssetPII: Record<string, FieldOccurrence[]> = {};
            (findingsResult.findings || []).forEach((f: any) => {
                const subCat = f.classifications?.[0]?.sub_category as string | undefined;
                const piiKey = subCat || f.pattern_name;
                const key = `${f.asset_id}__${piiKey}`;
                if (!findingsByAssetPII[key]) findingsByAssetPII[key] = [];
                const { dir, field } = splitPath(f.asset_path || '');
                findingsByAssetPII[key].push({
                    findingId: f.id,
                    assetPath: f.asset_path || '',
                    field: field || f.pattern_name || '—',
                    severity: f.severity,
                    confidence: f.confidence_score ?? 0,
                    sampleText: (f.sample_text || '').substring(0, 60),
                    reviewStatus: f.review_status || 'pending',
                });
            });

            // Build system groups
            const systemNodes = nodes.filter(n => n.type === 'system');
            const groups: SystemGroup[] = systemNodes.map(sys => {
                const assetIds = childrenMap[sys.id] || [];
                const assets: AssetGroup[] = assetIds.map(aid => {
                    const assetNode = nodeMap[aid];
                    if (!assetNode) return null;
                    const piiIds = childrenMap[aid] || [];
                    const piiGroups: PIIGroup[] = piiIds.map(pid => {
                        const piiNode = nodeMap[pid];
                        if (!piiNode || piiNode.type !== 'pii_category') return null;
                        const meta = piiNode.metadata as Record<string, any>;
                        const piiType = meta.pii_type || piiNode.label;
                        const key = `${aid}__${piiType}`;
                        const fields = findingsByAssetPII[key] || [];
                        // Prefer the EXPOSES edge's per-asset count over the
                        // node's shared count (see edgeByEndpoints comment).
                        const edgeMeta = (edgeByEndpoints[`${aid}__${pid}`]?.metadata ?? {}) as Record<string, any>;
                        const findingCount = edgeMeta.finding_count ?? fields.length ?? meta.finding_count ?? 0;
                        const avgConfidence = edgeMeta.avg_confidence ?? meta.avg_confidence ?? 0;
                        return {
                            piiType,
                            riskLevel: meta.risk_level || 'Medium',
                            dpdpaCategory: meta.dpdpa_category || '—',
                            findingCount,
                            avgConfidence,
                            fields,
                        } as PIIGroup;
                    }).filter(Boolean) as PIIGroup[];

                    const assetMeta = assetNode.metadata as Record<string, any>;
                    return {
                        assetId: aid,
                        assetLabel: assetNode.label,
                        assetPath: assetMeta.path || '',
                        environment: assetMeta.environment || 'production',
                        piiGroups,
                    } as AssetGroup;
                }).filter(Boolean) as AssetGroup[];

                const sysMeta = sys.metadata as Record<string, any>;
                return {
                    systemId: sys.id,
                    systemLabel: sys.label,
                    host: sysMeta.host || sys.label,
                    assets,
                } as SystemGroup;
            });

            setSystemGroups(groups);
            // Expand all systems by default
            setExpandedSystems(new Set(groups.map(g => g.systemId)));
        } catch (err: any) {
            setError(err.message || 'Failed to load lineage data.');
        } finally {
            setLoading(false);
        }
    };

    const handleSyncLineage = async () => {
        setSyncing(true);
        setSyncStatus(null);
        try {
            const result = await syncLineage();
            setSyncStatus(result.message ?? 'Sync complete');
            await fetchData();
        } catch (err) {
            setSyncStatus(err instanceof Error ? err.message : 'Sync failed');
        } finally {
            setSyncing(false);
        }
    };

    const toggleSystem = (id: string) => setExpandedSystems(prev => {
        const s = new Set(prev);
        s.has(id) ? s.delete(id) : s.add(id);
        return s;
    });

    const toggleAsset = (id: string) => setExpandedAssets(prev => {
        const s = new Set(prev);
        s.has(id) ? s.delete(id) : s.add(id);
        return s;
    });

    const togglePII = (id: string) => setExpandedPII(prev => {
        const s = new Set(prev);
        s.has(id) ? s.delete(id) : s.add(id);
        return s;
    });

    // Filter system groups by focused asset + search query. When the page
    // is opened from an asset detail (?assetId=X) the lineage is scoped to
    // that single asset so users see just its flow, not the whole graph.
    const filteredGroups = React.useMemo(() => {
        let groups = systemGroups;
        if (focusedAssetId) {
            groups = groups
                .map(sg => ({
                    ...sg,
                    assets: sg.assets.filter(a => a.assetId === focusedAssetId),
                }))
                .filter(sg => sg.assets.length > 0);
        }
        if (!searchQuery) return groups;
        const lower = searchQuery.toLowerCase();
        return groups
            .map(sg => ({
                ...sg,
                assets: sg.assets
                    .map(a => ({
                        ...a,
                        piiGroups: a.piiGroups.filter(p =>
                            p.piiType.toLowerCase().includes(lower) ||
                            p.dpdpaCategory.toLowerCase().includes(lower) ||
                            a.assetLabel.toLowerCase().includes(lower) ||
                            sg.systemLabel.toLowerCase().includes(lower)
                        ),
                    }))
                    .filter(a => a.piiGroups.length > 0),
            }))
            .filter(sg => sg.assets.length > 0);
    }, [systemGroups, searchQuery, focusedAssetId]);

    // Stats
    const stats = {
        systems: aggregations?.total_systems ?? systemGroups.length,
        assets: aggregations?.total_assets ?? systemGroups.reduce((s, g) => s + g.assets.length, 0),
        piiTypes: aggregations?.total_pii_types ?? 0,
        totalFindings: systemGroups.reduce((s, g) =>
            s + g.assets.reduce((sa, a) =>
                sa + a.piiGroups.reduce((sp, p) => sp + p.findingCount, 0), 0), 0),
    };

    // Lineage graph filtered nodes/edges. When focusedAssetId is set we walk
    // the edge list from the focused asset in both directions (parents and
    // descendants) so the graph view matches what Path view renders.
    const filteredNodes = React.useMemo(() => {
        const allNodes = lineageData?.nodes || [];
        const allEdges = lineageData?.edges || [];

        let scopedIds: Set<string> | null = null;
        if (focusedAssetId) {
            scopedIds = new Set<string>([focusedAssetId]);
            // Two separate one-way walks from the focused asset:
            //  • upwards: visit only parents, then parents-of-parents (never re-descend)
            //  • downwards: visit only children, then children-of-children
            // If both walks shared a queue, walking up to the system would
            // then walk DOWN to all sibling assets — which is exactly the
            // "full system graph" regression that shipped in v1.
            const parentOf: Record<string, string[]> = {};
            const childrenOf: Record<string, string[]> = {};
            allEdges.forEach(e => {
                (childrenOf[e.source] ||= []).push(e.target);
                (parentOf[e.target] ||= []).push(e.source);
            });

            const upQueue = [focusedAssetId];
            while (upQueue.length) {
                const cur = upQueue.shift()!;
                for (const p of parentOf[cur] || []) {
                    if (!scopedIds.has(p)) { scopedIds.add(p); upQueue.push(p); }
                }
            }

            const downQueue = [focusedAssetId];
            while (downQueue.length) {
                const cur = downQueue.shift()!;
                for (const ch of childrenOf[cur] || []) {
                    if (!scopedIds.has(ch)) { scopedIds.add(ch); downQueue.push(ch); }
                }
            }
        }

        const nodes = scopedIds ? allNodes.filter(n => scopedIds!.has(n.id)) : allNodes;
        if (!searchQuery) return nodes;
        const lower = searchQuery.toLowerCase();
        return nodes.filter(n => n.label.toLowerCase().includes(lower) || n.type.toLowerCase().includes(lower));
    }, [lineageData, searchQuery, focusedAssetId]);

    const filteredEdges = React.useMemo(() => {
        const edges = lineageData?.edges || [];
        const nodeIds = new Set(filteredNodes.map(n => n.id));
        // Keep only edges where both endpoints survived the node filter.
        return edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));
    }, [lineageData, filteredNodes]);

    if (loading && !lineageData) {
        return <LoadingState fullScreen message="Loading data lineage..." />;
    }

    return (
        <div style={{ minHeight: '100vh', backgroundColor: theme.colors.background.secondary, fontFamily: theme.fonts.sans }}>
            <div style={{ maxWidth: '1400px', margin: '0 auto', padding: '32px 24px' }}>

                {/* ── Header ── */}
                <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: '28px', flexWrap: 'wrap', gap: '16px' }}>
                    <div>
                        <h1 style={{ fontSize: '24px', fontWeight: 800, margin: 0, color: theme.colors.text.primary, letterSpacing: '-0.02em' }}>
                            Data Lineage
                        </h1>
                        <p style={{ margin: '4px 0 0', fontSize: '14px', color: theme.colors.text.secondary }}>
                            End-to-end PII flow — trace every finding to its source field
                        </p>
                    </div>
                    <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
                        <div style={{ position: 'relative' }}>
                            <span style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', fontSize: '13px', color: '#94a3b8' }}>🔍</span>
                            <input
                                type="text"
                                placeholder="Filter by PII type, asset..."
                                value={searchQuery}
                                onChange={e => setSearchQuery(e.target.value)}
                                style={{
                                    border: `1px solid ${theme.colors.border.default}`,
                                    borderRadius: '8px', padding: '8px 12px 8px 30px',
                                    fontSize: '13px', color: theme.colors.text.primary,
                                    background: '#fff', outline: 'none', width: '220px',
                                }}
                            />
                        </div>
                        <button
                            onClick={fetchData}
                            style={{
                                border: `1px solid ${theme.colors.border.default}`,
                                borderRadius: '8px', padding: '8px 14px',
                                fontSize: '13px', color: theme.colors.text.secondary,
                                background: '#fff', cursor: 'pointer', fontWeight: 500,
                            }}
                        >
                            ↺ Refresh
                        </button>
                        <button
                            onClick={handleSyncLineage}
                            disabled={syncing}
                            style={{
                                border: `1px solid ${theme.colors.border.default}`,
                                borderRadius: '8px', padding: '8px 14px',
                                fontSize: '13px', color: syncing ? theme.colors.text.muted : '#fff',
                                background: syncing ? theme.colors.background.secondary : theme.colors.primary.DEFAULT,
                                cursor: syncing ? 'not-allowed' : 'pointer', fontWeight: 600,
                                opacity: syncing ? 0.7 : 1,
                            }}
                        >
                            {syncing ? '⏳ Syncing...' : '⚡ Sync Lineage'}
                        </button>
                    </div>
                </div>

                {error && (
                    <div style={{
                        background: '#fef2f2', border: '1px solid #fecaca',
                        borderRadius: '10px', padding: '12px 16px', marginBottom: '20px',
                        color: '#b91c1c', fontSize: '13px', display: 'flex', gap: '8px',
                    }}>
                        <span>⚠️</span><span>{error}</span>
                    </div>
                )}
                {syncStatus && (
                    <div style={{
                        background: '#eff6ff', border: '1px solid #bfdbfe',
                        borderRadius: '10px', padding: '12px 16px', marginBottom: '20px',
                        color: '#1e40af', fontSize: '13px', display: 'flex', gap: '8px',
                    }}>
                        <span>ℹ️</span><span>{syncStatus}</span>
                    </div>
                )}

                {/* ── Stats row ── */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '12px', marginBottom: '24px' }}>
                    {[
                        { label: 'Systems', value: stats.systems, icon: '🌐', color: theme.colors.primary.DEFAULT },
                        { label: 'Assets', value: stats.assets, icon: '🗄️', color: '#10b981' },
                        { label: 'PII Types', value: stats.piiTypes, icon: '🔖', color: '#8b5cf6' },
                        { label: 'Total Findings', value: stats.totalFindings.toLocaleString(), icon: '⚠️', color: '#ef4444' },
                    ].map(card => (
                        <div key={card.label} style={{
                            background: '#fff',
                            border: `1px solid ${theme.colors.border.default}`,
                            borderRadius: '12px', padding: '16px 20px',
                            borderTop: `3px solid ${card.color}`,
                        }}>
                            <div style={{ fontSize: '18px', marginBottom: '6px' }}>{card.icon}</div>
                            <div style={{ fontSize: '22px', fontWeight: 700, color: theme.colors.text.primary, lineHeight: 1 }}>{card.value}</div>
                            <div style={{ fontSize: '11px', color: theme.colors.text.muted, marginTop: '4px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>{card.label}</div>
                        </div>
                    ))}
                </div>

                {/* ── View tabs ── */}
                <div style={{ display: 'flex', gap: '4px', marginBottom: '20px', background: theme.colors.background.tertiary, borderRadius: '10px', padding: '4px', width: 'fit-content' }}>
                    {(['path', 'graph'] as const).map(mode => (
                        <button
                            key={mode}
                            onClick={() => setViewMode(mode)}
                            style={{
                                padding: '7px 18px', borderRadius: '7px',
                                fontSize: '13px', fontWeight: 600, border: 'none', cursor: 'pointer',
                                background: viewMode === mode ? theme.colors.primary.DEFAULT : 'transparent',
                                color: viewMode === mode ? '#fff' : theme.colors.text.secondary,
                                transition: 'all 0.15s',
                            }}
                        >
                            {mode === 'path' ? '🗺 Path View' : '🕸 Graph View'}
                        </button>
                    ))}
                </div>

                {/* ── Graph View ── */}
                {viewMode === 'graph' && (
                    <div style={{
                        border: `1px solid ${theme.colors.border.default}`,
                        borderRadius: '14px', overflow: 'hidden', height: 'calc(100vh - 320px)', minHeight: '520px',
                        background: '#fff', boxShadow: theme.shadows.sm,
                    }}>
                        {lineageData ? (
                            <LineageCanvas
                                nodes={filteredNodes}
                                edges={filteredEdges}
                                onNodeClick={setSelectedNodeId}
                                focusedNodeId={focusedAssetId}
                            />
                        ) : <LoadingState message="Building graph..." />}
                    </div>
                )}

                {/* ── Path View ── */}
                {viewMode === 'path' && (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                        {filteredGroups.length === 0 ? (
                            <div style={{
                                textAlign: 'center', padding: '80px 0',
                                color: theme.colors.text.muted, fontSize: '15px',
                                background: '#fff', border: `1px solid ${theme.colors.border.default}`,
                                borderRadius: '12px',
                            }}>
                                <div style={{ fontSize: '36px', marginBottom: '12px' }}>🔍</div>
                                {systemGroups.length === 0
                                    ? 'No lineage data yet. Run a scan to populate.'
                                    : 'No results match your search.'}
                            </div>
                        ) : (
                            filteredGroups.map(sg => (
                                <div key={sg.systemId} style={{
                                    background: '#fff',
                                    border: `1px solid ${theme.colors.border.default}`,
                                    borderRadius: '12px', overflow: 'hidden',
                                    boxShadow: theme.shadows.sm,
                                }}>
                                    {/* System header */}
                                    <div
                                        onClick={() => toggleSystem(sg.systemId)}
                                        style={{
                                            display: 'flex', alignItems: 'center',
                                            justifyContent: 'space-between',
                                            padding: '14px 20px',
                                            cursor: 'pointer',
                                            background: theme.colors.background.tertiary,
                                            borderBottom: expandedSystems.has(sg.systemId)
                                                ? `1px solid ${theme.colors.border.default}` : 'none',
                                            userSelect: 'none',
                                        }}
                                    >
                                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                            <span style={{
                                                background: '#dbeafe', color: '#1d4ed8',
                                                border: '1px solid #bfdbfe',
                                                borderRadius: '6px', padding: '3px 8px',
                                                fontSize: '11px', fontWeight: 700,
                                                textTransform: 'uppercase', letterSpacing: '0.05em',
                                            }}>
                                                🌐 System
                                            </span>
                                            <div>
                                                <span style={{ fontSize: '14px', fontWeight: 700, color: theme.colors.text.primary }}>
                                                    {sg.systemLabel}
                                                </span>
                                                <span style={{ fontSize: '12px', color: theme.colors.text.muted, marginLeft: '8px', fontFamily: theme.fonts.mono }}>
                                                    {sg.host}
                                                </span>
                                            </div>
                                        </div>
                                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                            <span style={{
                                                background: theme.colors.background.secondary,
                                                border: `1px solid ${theme.colors.border.default}`,
                                                borderRadius: '20px', padding: '2px 10px',
                                                fontSize: '12px', color: theme.colors.text.secondary, fontWeight: 600,
                                            }}>
                                                {sg.assets.length} asset{sg.assets.length !== 1 ? 's' : ''}
                                            </span>
                                            <span style={{ color: theme.colors.text.muted, fontSize: '12px' }}>
                                                {expandedSystems.has(sg.systemId) ? '▲' : '▼'}
                                            </span>
                                        </div>
                                    </div>

                                    {/* Assets */}
                                    {expandedSystems.has(sg.systemId) && (
                                        <div style={{ padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
                                            {sg.assets.map(asset => (
                                                <div key={asset.assetId} style={{
                                                    border: `1px solid ${theme.colors.border.default}`,
                                                    borderRadius: '10px', overflow: 'hidden',
                                                }}>
                                                    {/* Asset header */}
                                                    <div
                                                        onClick={() => toggleAsset(asset.assetId)}
                                                        style={{
                                                            display: 'flex', alignItems: 'center',
                                                            justifyContent: 'space-between',
                                                            padding: '10px 16px',
                                                            cursor: 'pointer',
                                                            background: '#f8fafc',
                                                            borderBottom: expandedAssets.has(asset.assetId)
                                                                ? `1px solid ${theme.colors.border.default}` : 'none',
                                                            userSelect: 'none',
                                                        }}
                                                    >
                                                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                            <span style={{
                                                                background: '#dcfce7', color: '#15803d',
                                                                border: '1px solid #bbf7d0',
                                                                borderRadius: '5px', padding: '2px 7px',
                                                                fontSize: '10px', fontWeight: 700,
                                                                textTransform: 'uppercase',
                                                            }}>
                                                                🗄️ Asset
                                                            </span>
                                                            <span style={{ fontSize: '13px', fontWeight: 600, color: theme.colors.text.primary }}>
                                                                {asset.assetLabel}
                                                            </span>
                                                            {asset.assetPath && (
                                                                <span style={{ fontSize: '11px', color: theme.colors.text.muted, fontFamily: theme.fonts.mono }}>
                                                                    {asset.assetPath}
                                                                </span>
                                                            )}
                                                            <span style={{
                                                                background: '#f0fdf4', color: '#166534',
                                                                border: '1px solid #bbf7d0',
                                                                borderRadius: '12px', padding: '1px 7px',
                                                                fontSize: '10px', fontWeight: 600,
                                                            }}>
                                                                {asset.environment}
                                                            </span>
                                                        </div>
                                                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                            <span style={{
                                                                background: '#fef2f2', color: '#b91c1c',
                                                                border: '1px solid #fecaca',
                                                                borderRadius: '12px', padding: '1px 8px',
                                                                fontSize: '11px', fontWeight: 600,
                                                            }}>
                                                                {asset.piiGroups.length} PII type{asset.piiGroups.length !== 1 ? 's' : ''}
                                                            </span>
                                                            <span style={{ color: theme.colors.text.muted, fontSize: '11px' }}>
                                                                {expandedAssets.has(asset.assetId) ? '▲' : '▼'}
                                                            </span>
                                                        </div>
                                                    </div>

                                                    {/* PII groups */}
                                                    {expandedAssets.has(asset.assetId) && (
                                                        <div style={{ padding: '10px 14px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                                            {asset.piiGroups.map(pii => {
                                                                const piiKey = `${asset.assetId}__${pii.piiType}`;
                                                                const riskBar = RISK_BAR[pii.riskLevel] ?? '#94a3b8';
                                                                const sStyle = SEVERITY_STYLE[pii.riskLevel] ?? SEVERITY_STYLE['Medium'];
                                                                return (
                                                                    <div key={piiKey} style={{
                                                                        border: `1px solid ${theme.colors.border.default}`,
                                                                        borderLeft: `3px solid ${riskBar}`,
                                                                        borderRadius: '8px', overflow: 'hidden',
                                                                    }}>
                                                                        {/* PII header */}
                                                                        <div
                                                                            onClick={() => togglePII(piiKey)}
                                                                            style={{
                                                                                display: 'flex', alignItems: 'center',
                                                                                justifyContent: 'space-between',
                                                                                padding: '9px 14px',
                                                                                cursor: 'pointer',
                                                                                background: sStyle.bg,
                                                                                userSelect: 'none',
                                                                            }}
                                                                        >
                                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                                                                                <span style={{
                                                                                    fontSize: '13px', fontWeight: 700,
                                                                                    color: theme.colors.text.primary,
                                                                                    fontFamily: theme.fonts.mono,
                                                                                }}>
                                                                                    {pii.piiType}
                                                                                </span>
                                                                                <SeverityBadge level={pii.riskLevel} />
                                                                                <span style={{ fontSize: '11px', color: theme.colors.text.muted }}>
                                                                                    DPDPA: <strong style={{ color: theme.colors.text.secondary }}>{pii.dpdpaCategory}</strong>
                                                                                </span>
                                                                            </div>
                                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                                                                <span style={{
                                                                                    background: '#fff', border: `1px solid ${sStyle.border}`,
                                                                                    borderRadius: '12px', padding: '2px 8px',
                                                                                    fontSize: '11px', fontWeight: 700, color: sStyle.text,
                                                                                }}>
                                                                                    {pii.findingCount} finding{pii.findingCount !== 1 ? 's' : ''}
                                                                                </span>
                                                                                <ConfBar score={pii.avgConfidence} />
                                                                                <span style={{ color: theme.colors.text.muted, fontSize: '11px' }}>
                                                                                    {expandedPII.has(piiKey) ? '▲' : '▼'}
                                                                                </span>
                                                                            </div>
                                                                        </div>

                                                                        {/* Field-level detail */}
                                                                        {expandedPII.has(piiKey) && (
                                                                            <div style={{ background: '#fff' }}>
                                                                                {/* How-to-fix banner */}
                                                                                <div style={{
                                                                                    padding: '8px 14px',
                                                                                    background: '#eff6ff',
                                                                                    borderBottom: `1px solid #bfdbfe`,
                                                                                    fontSize: '12px', color: '#1e40af',
                                                                                    display: 'flex', alignItems: 'center', gap: '6px',
                                                                                }}>
                                                                                    <span>💡</span>
                                                                                    <span>
                                                                                        Connect to <strong style={{ fontFamily: theme.fonts.mono }}>{sg.host}</strong> and
                                                                                        apply masking, deletion, or access-control on the fields below.
                                                                                    </span>
                                                                                </div>

                                                                                {pii.fields.length === 0 ? (
                                                                                    <div style={{ padding: '12px 16px', color: theme.colors.text.muted, fontSize: '13px' }}>
                                                                                        No field-level detail loaded. Try scanning again.
                                                                                    </div>
                                                                                ) : (
                                                                                    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '12px' }}>
                                                                                        <thead>
                                                                                            <tr style={{ background: '#f8fafc', borderBottom: `1px solid ${theme.colors.border.default}` }}>
                                                                                                <th style={{ padding: '8px 14px', textAlign: 'left', fontWeight: 600, color: theme.colors.text.secondary, width: '35%' }}>Field / Path</th>
                                                                                                <th style={{ padding: '8px 12px', textAlign: 'left', fontWeight: 600, color: theme.colors.text.secondary }}>Severity</th>
                                                                                                <th style={{ padding: '8px 12px', textAlign: 'left', fontWeight: 600, color: theme.colors.text.secondary }}>Confidence</th>
                                                                                                <th style={{ padding: '8px 12px', textAlign: 'left', fontWeight: 600, color: theme.colors.text.secondary }}>Sample (masked)</th>
                                                                                                <th style={{ padding: '8px 12px', textAlign: 'left', fontWeight: 600, color: theme.colors.text.secondary }}>Status</th>
                                                                                            </tr>
                                                                                        </thead>
                                                                                        <tbody>
                                                                                            {pii.fields.map((f, fi) => (
                                                                                                <tr
                                                                                                    key={`${f.findingId}-${fi}`}
                                                                                                    style={{
                                                                                                        borderBottom: `1px solid ${theme.colors.border.subtle}`,
                                                                                                        background: fi % 2 === 0 ? '#fff' : '#f8fafc',
                                                                                                    }}
                                                                                                >
                                                                                                    <td style={{ padding: '8px 14px' }}>
                                                                                                        <div style={{ fontFamily: theme.fonts.mono, color: theme.colors.primary.DEFAULT, fontSize: '12px', fontWeight: 600 }}>
                                                                                                            {f.field}
                                                                                                        </div>
                                                                                                        {f.assetPath && f.assetPath !== f.field && (
                                                                                                            <div style={{ color: theme.colors.text.muted, fontSize: '11px', marginTop: '2px', fontFamily: theme.fonts.mono, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '260px' }}>
                                                                                                                {f.assetPath}
                                                                                                            </div>
                                                                                                        )}
                                                                                                    </td>
                                                                                                    <td style={{ padding: '8px 12px' }}>
                                                                                                        <SeverityBadge level={f.severity} />
                                                                                                    </td>
                                                                                                    <td style={{ padding: '8px 12px' }}>
                                                                                                        <ConfBar score={f.confidence} />
                                                                                                    </td>
                                                                                                    <td style={{ padding: '8px 12px', fontFamily: theme.fonts.mono, color: theme.colors.text.muted, maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                                                                                        {f.sampleText || '—'}
                                                                                                    </td>
                                                                                                    <td style={{ padding: '8px 12px' }}>
                                                                                                        <span style={{
                                                                                                            background: f.reviewStatus === 'confirmed' ? '#f0fdf4' : f.reviewStatus === 'false_positive' ? '#f8fafc' : '#fefce8',
                                                                                                            color: f.reviewStatus === 'confirmed' ? '#166534' : f.reviewStatus === 'false_positive' ? '#64748b' : '#92400e',
                                                                                                            border: `1px solid ${f.reviewStatus === 'confirmed' ? '#bbf7d0' : f.reviewStatus === 'false_positive' ? '#e2e8f0' : '#fde68a'}`,
                                                                                                            borderRadius: '4px', padding: '1px 6px',
                                                                                                            fontSize: '10px', fontWeight: 600, textTransform: 'capitalize',
                                                                                                        }}>
                                                                                                            {f.reviewStatus || 'pending'}
                                                                                                        </span>
                                                                                                    </td>
                                                                                                </tr>
                                                                                            ))}
                                                                                        </tbody>
                                                                                    </table>
                                                                                )}
                                                                            </div>
                                                                        )}
                                                                    </div>
                                                                );
                                                            })}
                                                        </div>
                                                    )}
                                                </div>
                                            ))}
                                        </div>
                                    )}
                                </div>
                            ))
                        )}
                    </div>
                )}

                {/* Footer */}
                <div style={{ marginTop: '28px', textAlign: 'center' }}>
                    <a
                        href="/findings"
                        style={{ color: theme.colors.primary.DEFAULT, fontSize: '14px', fontWeight: 600, textDecoration: 'none' }}
                    >
                        View detailed findings explorer →
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
