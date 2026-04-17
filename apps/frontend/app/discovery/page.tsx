'use client';

import React, { useEffect, useState, useMemo, Suspense } from 'react';
import {
    LineChart,
    Line,
    XAxis,
    YAxis,
    Tooltip as RechartsTooltip,
    ResponsiveContainer,
    CartesianGrid,
} from 'recharts';
import {
    discoveryApi,
    OverviewSummary,
    Snapshot,
    DriftEvent,
    InventoryRow,
    RiskHotspot,
    GlossaryTerm,
} from '@/services/discoveryApi';
import { useSearchParams, useRouter } from 'next/navigation';

// Risk tier helpers
function riskColor(score: number): string {
    if (score >= 80) return '#ef4444';
    if (score >= 60) return '#f97316';
    if (score >= 40) return '#eab308';
    return '#22c55e';
}
function riskLabel(score: number): string {
    if (score >= 80) return 'Critical';
    if (score >= 60) return 'High';
    if (score >= 40) return 'Medium';
    return 'Low';
}

type ActiveTab = 'overview' | 'inventory' | 'risk' | 'glossary';

// ─── Tab bar ─────────────────────────────────────────────────────────────────

const TAB_LABELS: Record<ActiveTab, string> = {
    overview: 'Overview',
    inventory: 'Inventory',
    risk: 'Risk',
    glossary: 'Glossary',
};

function TabBar({ active, onChange }: { active: ActiveTab; onChange: (t: ActiveTab) => void }) {
    return (
        <div className="flex gap-1 mb-6 border-b border-gray-200">
            {(Object.keys(TAB_LABELS) as ActiveTab[]).map((tab) => (
                <button
                    key={tab}
                    onClick={() => onChange(tab)}
                    className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
                        active === tab
                            ? 'border-blue-600 text-blue-600'
                            : 'border-transparent text-gray-500 hover:text-gray-700'
                    }`}
                >
                    {TAB_LABELS[tab]}
                </button>
            ))}
        </div>
    );
}

// ─── KPI card ────────────────────────────────────────────────────────────────

interface KpiCardProps {
    label: string;
    value: string | number;
    accent?: boolean;
    color?: string;
}

function KpiCard({ label, value, accent, color }: KpiCardProps) {
    return (
        <div
            className={`p-4 rounded-lg border ${
                accent ? 'border-blue-200 bg-blue-50' : 'border-gray-200 bg-white'
            }`}
        >
            <div className="text-xs uppercase tracking-wide text-gray-500">{label}</div>
            <div
                className={`text-2xl font-bold mt-1 ${accent ? 'text-blue-700' : 'text-gray-900'}`}
                style={color ? { color } : undefined}
            >
                {value}
            </div>
        </div>
    );
}

function severityClass(s: string): string {
    switch (s) {
        case 'critical': return 'bg-red-100 text-red-800';
        case 'high':     return 'bg-orange-100 text-orange-800';
        case 'medium':   return 'bg-yellow-100 text-yellow-800';
        default:         return 'bg-gray-100 text-gray-700';
    }
}

function humanizeEvent(t: string): string {
    switch (t) {
        case 'asset_added':            return 'New asset/classification appeared';
        case 'asset_removed':          return 'Asset/classification removed';
        case 'classification_changed': return 'Classification changed';
        case 'risk_increased':         return 'Risk score increased';
        case 'risk_decreased':         return 'Risk score decreased';
        case 'finding_count_spike':    return 'PII finding count spiked';
        default:                       return t;
    }
}

// ─── Loading skeleton ─────────────────────────────────────────────────────────

function TableSkeleton({ rows = 5, cols = 6 }: { rows?: number; cols?: number }) {
    return (
        <div className="animate-pulse">
            {Array.from({ length: rows }).map((_, i) => (
                <div key={i} className="flex gap-4 px-4 py-3 border-b border-gray-100">
                    {Array.from({ length: cols }).map((_, j) => (
                        <div key={j} className="h-4 bg-gray-200 rounded flex-1" />
                    ))}
                </div>
            ))}
        </div>
    );
}

// ─── Overview tab ─────────────────────────────────────────────────────────────

interface OverviewTabProps {
    overview: OverviewSummary | null;
    latestSnapshot: Snapshot | null;
    drifts: DriftEvent[];
    snapshotLimit: number;
    setSnapshotLimit: (n: number) => void;
    sourceFilter: string;
    setSourceFilter: (s: string) => void;
    riskFilter: string;
    setRiskFilter: (s: string) => void;
    updateUrlParam: (key: string, value: string) => void;
    router: ReturnType<typeof useRouter>;
}

function OverviewTab({
    overview, latestSnapshot, drifts,
    snapshotLimit, setSnapshotLimit,
    sourceFilter, setSourceFilter,
    riskFilter, setRiskFilter,
    updateUrlParam, router,
}: OverviewTabProps) {
    return (
        <>
            {/* KPI grid */}
            <section className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
                <KpiCard label="Sources" value={overview?.source_count ?? 0} />
                <KpiCard label="Assets" value={overview?.asset_count ?? 0} />
                <KpiCard label="PII Findings" value={overview?.finding_count ?? 0} />
                <KpiCard
                    label="Composite Risk"
                    value={overview?.composite_risk_score?.toFixed(1) ?? '0.0'}
                    accent
                />
            </section>

            {/* Chart filters bar */}
            <section className="mb-4 flex flex-wrap items-center gap-3 p-3 bg-gray-50 rounded-lg border border-gray-200 text-sm">
                <span className="text-gray-500 font-medium">Filters:</span>
                <label className="flex items-center gap-1 text-gray-700">
                    Snapshots:
                    <select
                        value={snapshotLimit}
                        onChange={e => {
                            const v = parseInt(e.target.value, 10);
                            setSnapshotLimit(v);
                            updateUrlParam('snapshots', String(v));
                        }}
                        className="ml-1 px-2 py-1 rounded border border-gray-300 text-sm bg-white">
                        {[4, 8, 12, 24].map(n => <option key={n} value={n}>{n}</option>)}
                    </select>
                </label>
                <label className="flex items-center gap-1 text-gray-700">
                    Source:
                    <input
                        value={sourceFilter}
                        placeholder="e.g. postgresql"
                        onChange={e => { setSourceFilter(e.target.value); updateUrlParam('source', e.target.value); }}
                        className="ml-1 px-2 py-1 rounded border border-gray-300 text-sm bg-white w-32"
                    />
                </label>
                <label className="flex items-center gap-1 text-gray-700">
                    Risk tier:
                    <select
                        value={riskFilter}
                        onChange={e => { setRiskFilter(e.target.value); updateUrlParam('risk', e.target.value); }}
                        className="ml-1 px-2 py-1 rounded border border-gray-300 text-sm bg-white">
                        <option value="">All</option>
                        <option value="Critical">Critical (&ge;80)</option>
                        <option value="High">High (&ge;60)</option>
                        <option value="Medium">Medium (&ge;40)</option>
                        <option value="Low">Low</option>
                    </select>
                </label>
                {(sourceFilter || riskFilter || snapshotLimit !== 4) && (
                    <button
                        onClick={() => {
                            setSnapshotLimit(4); setSourceFilter(''); setRiskFilter('');
                            router.replace('?', { scroll: false });
                        }}
                        className="text-xs text-blue-600 hover:underline">
                        Clear filters
                    </button>
                )}
            </section>

            {/* Trend chart */}
            <section className="mb-8 p-4 bg-white rounded-lg border border-gray-200">
                <h2 className="text-sm font-semibold text-gray-700 mb-3">
                    PII Trend (last {snapshotLimit} snapshots)
                </h2>
                {overview?.trend_quarters && overview.trend_quarters.length > 0 ? (
                    <ResponsiveContainer width="100%" height={220}>
                        <LineChart data={overview.trend_quarters.slice(-snapshotLimit)}>
                            <CartesianGrid strokeDasharray="3 3" stroke="#eee" />
                            <XAxis dataKey="label" tick={{ fontSize: 11 }} />
                            <YAxis tick={{ fontSize: 11 }} />
                            <RechartsTooltip />
                            <Line type="monotone" dataKey="finding_count" stroke="#3b82f6" strokeWidth={2} name="Findings" />
                            <Line type="monotone" dataKey="composite_risk_score" stroke="#ef4444" strokeWidth={2} name="Risk Score" />
                        </LineChart>
                    </ResponsiveContainer>
                ) : (
                    <div className="text-sm text-gray-500 py-8 text-center">
                        Not enough snapshots to plot a trend yet. Run a few more to see history.
                    </div>
                )}
            </section>

            {/* Hotspots heatmap + Drift */}
            <section className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="p-4 bg-white rounded-lg border border-gray-200">
                    <h2 className="text-sm font-semibold text-gray-700 mb-4">PII Density Heatmap</h2>
                    {(() => {
                        const maxScore = 100;
                        const hotspots = (overview?.top_hotspots ?? [])
                            .filter(h => !sourceFilter || (h.classification ?? '').toLowerCase().includes(sourceFilter.toLowerCase()))
                            .filter(h => !riskFilter || riskLabel(h.score) === riskFilter);
                        return hotspots.length > 0 ? (
                            <div className="space-y-3">
                                {hotspots.map(h => {
                                    const color = riskColor(h.score);
                                    const pct = Math.max(4, Math.round((h.score / maxScore) * 100));
                                    return (
                                        <div key={h.asset_id}>
                                            <div className="flex justify-between text-xs mb-1">
                                                <span className="text-gray-800 font-medium truncate max-w-[60%]">
                                                    {h.asset_name || h.asset_id.slice(0, 8)}
                                                </span>
                                                <span className="flex items-center gap-2">
                                                    <span className="text-gray-500">{h.classification}</span>
                                                    <span style={{ color, fontWeight: 700 }} className="font-mono">{h.score.toFixed(1)}</span>
                                                    <span style={{ backgroundColor: `${color}20`, color, fontSize: '10px', fontWeight: 700, padding: '1px 6px', borderRadius: '999px' }}>
                                                        {riskLabel(h.score)}
                                                    </span>
                                                </span>
                                            </div>
                                            <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                                                <div style={{ width: `${pct}%`, height: '100%', backgroundColor: color, borderRadius: '999px', transition: 'width 0.4s ease' }} />
                                            </div>
                                        </div>
                                    );
                                })}
                                <div className="flex gap-4 mt-3 pt-3 border-t border-gray-100 text-xs text-gray-500">
                                    {[['#ef4444', 'Critical ≥80'], ['#f97316', 'High ≥60'], ['#eab308', 'Medium ≥40'], ['#22c55e', 'Low']].map(([c, l]) => (
                                        <span key={l} className="flex items-center gap-1">
                                            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: c, display: 'inline-block' }} />
                                            {l}
                                        </span>
                                    ))}
                                </div>
                            </div>
                        ) : (
                            <div className="text-sm text-gray-500 py-4">
                                {sourceFilter || riskFilter ? 'No hotspots match the current filters.' : 'No hotspots yet.'}
                            </div>
                        );
                    })()}
                </div>

                <div className="p-4 bg-white rounded-lg border border-gray-200">
                    <h2 className="text-sm font-semibold text-gray-700 mb-3">Recent Drift</h2>
                    {drifts.length > 0 ? (
                        <ul className="space-y-2 text-sm">
                            {drifts.map((d) => (
                                <li key={d.id} className="flex items-start gap-2">
                                    <span className={`px-2 py-0.5 text-xs rounded ${severityClass(d.severity)}`}>
                                        {d.severity}
                                    </span>
                                    <div className="flex-1">
                                        <div className="font-medium text-gray-800">{humanizeEvent(d.event_type)}</div>
                                        <div className="text-xs text-gray-500">{new Date(d.detected_at).toLocaleString()}</div>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <div className="text-sm text-gray-500 py-4">No drift detected since the last snapshot.</div>
                    )}
                </div>
            </section>

            {latestSnapshot && (
                <footer className="mt-6 text-xs text-gray-500">
                    Snapshot {latestSnapshot.id.slice(0, 8)} ·{' '}
                    {latestSnapshot.duration_ms ? `${(latestSnapshot.duration_ms / 1000).toFixed(1)}s` : 'pending'}{' '}
                    · {latestSnapshot.status}
                </footer>
            )}
        </>
    );
}

// ─── Inventory tab ────────────────────────────────────────────────────────────

function complianceStatus(sensitivity: number): string {
    if (sensitivity >= 80) return 'Non-compliant';
    if (sensitivity >= 50) return 'At risk';
    return 'Compliant';
}

function complianceBadgeClass(sensitivity: number): string {
    if (sensitivity >= 80) return 'bg-red-100 text-red-800';
    if (sensitivity >= 50) return 'bg-yellow-100 text-yellow-800';
    return 'bg-green-100 text-green-800';
}

function InventoryTab({ router }: { router: ReturnType<typeof useRouter> }) {
    const [data, setData] = useState<{ items: InventoryRow[]; count: number } | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [search, setSearch] = useState('');
    const [pendingSearch, setPendingSearch] = useState('');
    const [page, setPage] = useState(0);
    const PAGE_SIZE = 50;

    const fetchInventory = async (searchQuery: string, pageNum: number) => {
        setLoading(true);
        setError(null);
        try {
            const result = await discoveryApi.listInventory({
                search: searchQuery || undefined,
                limit: PAGE_SIZE,
                offset: pageNum * PAGE_SIZE,
            });
            setData(result);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load inventory');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchInventory('', 0);
    }, []);

    const handleSearch = () => {
        setSearch(pendingSearch);
        setPage(0);
        fetchInventory(pendingSearch, 0);
    };

    const handlePageChange = (newPage: number) => {
        setPage(newPage);
        fetchInventory(search, newPage);
    };

    const totalPages = data ? Math.ceil(data.count / PAGE_SIZE) : 0;

    return (
        <div>
            {/* Search bar */}
            <div className="flex gap-2 mb-4">
                <input
                    type="text"
                    value={pendingSearch}
                    onChange={e => setPendingSearch(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && handleSearch()}
                    placeholder="Search assets by name, type, classification..."
                    className="flex-1 px-3 py-2 rounded-md border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <button
                    onClick={handleSearch}
                    className="px-4 py-2 rounded-md bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium"
                >
                    Search
                </button>
                {search && (
                    <button
                        onClick={() => { setPendingSearch(''); setSearch(''); setPage(0); fetchInventory('', 0); }}
                        className="px-3 py-2 rounded-md border border-gray-300 text-sm text-gray-600 hover:bg-gray-50"
                    >
                        Clear
                    </button>
                )}
            </div>

            {error && (
                <div className="mb-4 p-3 rounded bg-red-50 border border-red-200 text-red-800 text-sm">{error}</div>
            )}

            <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <table className="w-full text-sm">
                    <thead className="bg-gray-50 border-b border-gray-200">
                        <tr>
                            <th className="px-4 py-3 text-left text-xs font-bold text-gray-500 uppercase tracking-wider">Asset Name</th>
                            <th className="px-4 py-3 text-left text-xs font-bold text-gray-500 uppercase tracking-wider">Type</th>
                            <th className="px-4 py-3 text-left text-xs font-bold text-gray-500 uppercase tracking-wider">PII Count</th>
                            <th className="px-4 py-3 text-left text-xs font-bold text-gray-500 uppercase tracking-wider">Risk Score</th>
                            <th className="px-4 py-3 text-left text-xs font-bold text-gray-500 uppercase tracking-wider">Last Scanned</th>
                            <th className="px-4 py-3 text-left text-xs font-bold text-gray-500 uppercase tracking-wider">Compliance</th>
                        </tr>
                    </thead>
                    <tbody>
                        {loading ? (
                            <tr><td colSpan={6} className="p-0"><TableSkeleton rows={8} cols={6} /></td></tr>
                        ) : data?.items.length === 0 ? (
                            <tr>
                                <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                                    No assets found{search ? ` matching "${search}"` : ''}.
                                </td>
                            </tr>
                        ) : (
                            data?.items.map((row) => (
                                <tr
                                    key={row.id}
                                    onClick={() => router.push(`/assets/${row.asset_id}`)}
                                    className="border-b border-gray-100 hover:bg-blue-50 cursor-pointer transition-colors"
                                >
                                    <td className="px-4 py-3 font-medium text-gray-900 truncate max-w-[200px]">
                                        {row.asset_name || row.asset_id.slice(0, 12)}
                                    </td>
                                    <td className="px-4 py-3 text-gray-600 font-mono text-xs">
                                        {row.classification}
                                    </td>
                                    <td className="px-4 py-3 text-gray-900 font-semibold">
                                        {row.finding_count}
                                    </td>
                                    <td className="px-4 py-3">
                                        <span style={{ color: riskColor(row.sensitivity), fontWeight: 700 }}>
                                            {row.sensitivity.toFixed(1)}
                                        </span>
                                        <span className="ml-1 text-xs text-gray-400">{riskLabel(row.sensitivity)}</span>
                                    </td>
                                    <td className="px-4 py-3 text-gray-500 text-xs">
                                        {row.last_scanned_at
                                            ? new Date(row.last_scanned_at).toLocaleString()
                                            : '—'}
                                    </td>
                                    <td className="px-4 py-3">
                                        <span className={`px-2 py-0.5 rounded text-xs font-semibold ${complianceBadgeClass(row.sensitivity)}`}>
                                            {complianceStatus(row.sensitivity)}
                                        </span>
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
                <div className="mt-4 flex items-center justify-between text-sm text-gray-600">
                    <span>
                        Showing {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, data?.count ?? 0)} of {data?.count ?? 0}
                    </span>
                    <div className="flex gap-2">
                        <button
                            disabled={page === 0}
                            onClick={() => handlePageChange(page - 1)}
                            className="px-3 py-1 rounded border border-gray-300 disabled:opacity-40 hover:bg-gray-50"
                        >
                            Prev
                        </button>
                        <button
                            disabled={page >= totalPages - 1}
                            onClick={() => handlePageChange(page + 1)}
                            className="px-3 py-1 rounded border border-gray-300 disabled:opacity-40 hover:bg-gray-50"
                        >
                            Next
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}

// ─── Risk tab ─────────────────────────────────────────────────────────────────

function RiskTab({ router }: { router: ReturnType<typeof useRouter> }) {
    const [riskOv, setRiskOv] = useState<any>(null);
    const [hotspots, setHotspots] = useState<RiskHotspot[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetch = async () => {
            setLoading(true);
            setError(null);
            try {
                const [ov, hs] = await Promise.all([
                    discoveryApi.getRiskOverview(),
                    discoveryApi.getRiskHotspots(10),
                ]);
                setRiskOv(ov);
                setHotspots(hs.hotspots ?? []);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load risk data');
            } finally {
                setLoading(false);
            }
        };
        fetch();
    }, []);

    if (loading) {
        return (
            <div className="animate-pulse space-y-4">
                <div className="grid grid-cols-3 gap-4">
                    {[0, 1, 2].map(i => <div key={i} className="h-24 bg-gray-100 rounded-lg" />)}
                </div>
                <div className="h-64 bg-gray-100 rounded-lg" />
            </div>
        );
    }

    if (error) {
        return <div className="p-3 rounded bg-red-50 border border-red-200 text-red-800 text-sm">{error}</div>;
    }

    // Derive KPI values defensively from whatever the backend returns
    const avgRisk: number = riskOv?.avg_risk_score ?? riskOv?.average_risk_score
        ?? (hotspots.length > 0 ? hotspots.reduce((s, h) => s + h.score, 0) / hotspots.length : 0);
    const highRiskCount: number = riskOv?.high_risk_count ?? riskOv?.high_risk_assets
        ?? hotspots.filter(h => h.score >= 60).length;
    const criticalCount: number = riskOv?.critical_count ?? riskOv?.critical_assets
        ?? hotspots.filter(h => h.score >= 80).length;

    return (
        <div className="space-y-6">
            {/* KPI cards */}
            <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <KpiCard
                    label="Average Risk Score"
                    value={avgRisk.toFixed(1)}
                    color={riskColor(avgRisk)}
                />
                <KpiCard
                    label="High-Risk Assets"
                    value={highRiskCount}
                    color="#f97316"
                />
                <KpiCard
                    label="Critical Assets"
                    value={criticalCount}
                    color="#ef4444"
                />
            </section>

            {/* Hotspots list */}
            <section className="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <div className="px-4 py-3 border-b border-gray-200">
                    <h2 className="text-sm font-semibold text-gray-700">Top Risky Assets</h2>
                    <p className="text-xs text-gray-500 mt-0.5">Sorted by risk score (highest first)</p>
                </div>
                {hotspots.length === 0 ? (
                    <div className="px-4 py-8 text-center text-gray-500 text-sm">No risk hotspots found.</div>
                ) : (
                    <ul className="divide-y divide-gray-100">
                        {hotspots.map((h, idx) => {
                            const color = riskColor(h.score);
                            const pct = Math.max(4, Math.round((h.score / 100) * 100));
                            return (
                                <li
                                    key={h.asset_id}
                                    onClick={() => router.push(`/assets/${h.asset_id}`)}
                                    className="flex items-center gap-4 px-4 py-3 hover:bg-blue-50 cursor-pointer transition-colors"
                                >
                                    <span className="w-6 text-xs text-gray-400 font-mono text-right shrink-0">{idx + 1}</span>
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center justify-between mb-1">
                                            <span className="font-medium text-gray-900 truncate text-sm">
                                                {h.asset_name || h.asset_id.slice(0, 12)}
                                            </span>
                                            <span className="flex items-center gap-2 ml-2 shrink-0">
                                                <span className="text-xs text-gray-500">{h.finding_count} findings</span>
                                                <span style={{ color, fontWeight: 700 }} className="font-mono text-sm">{h.score.toFixed(1)}</span>
                                                <span style={{ backgroundColor: `${color}20`, color, fontSize: '10px', fontWeight: 700, padding: '1px 6px', borderRadius: '999px' }}>
                                                    {riskLabel(h.score)}
                                                </span>
                                            </span>
                                        </div>
                                        <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
                                            <div style={{ width: `${pct}%`, height: '100%', backgroundColor: color, borderRadius: '999px' }} />
                                        </div>
                                    </div>
                                </li>
                            );
                        })}
                    </ul>
                )}
            </section>
        </div>
    );
}

// ─── Glossary tab ─────────────────────────────────────────────────────────────

const RISK_LEVEL_CLASS: Record<string, string> = {
    critical: 'bg-red-100 text-red-800',
    high:     'bg-orange-100 text-orange-800',
    medium:   'bg-yellow-100 text-yellow-800',
    low:      'bg-green-100 text-green-800',
};

function GlossaryTab() {
    const [terms, setTerms] = useState<GlossaryTerm[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [search, setSearch] = useState('');

    useEffect(() => {
        const fetch = async () => {
            setLoading(true);
            setError(null);
            try {
                const data = await discoveryApi.getGlossary();
                setTerms(data.terms ?? []);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load glossary');
            } finally {
                setLoading(false);
            }
        };
        fetch();
    }, []);

    const filtered = useMemo(() => {
        if (!search) return terms;
        const lower = search.toLowerCase();
        return terms.filter(t =>
            t.name.toLowerCase().includes(lower) ||
            t.description.toLowerCase().includes(lower) ||
            (t.dpdpa_section ?? '').toLowerCase().includes(lower) ||
            t.regulation_refs.some(r => r.toLowerCase().includes(lower))
        );
    }, [terms, search]);

    if (loading) {
        return (
            <div className="animate-pulse grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {Array.from({ length: 6 }).map((_, i) => (
                    <div key={i} className="h-40 bg-gray-100 rounded-lg" />
                ))}
            </div>
        );
    }

    if (error) {
        return <div className="p-3 rounded bg-red-50 border border-red-200 text-red-800 text-sm">{error}</div>;
    }

    return (
        <div>
            <div className="mb-4">
                <input
                    type="text"
                    value={search}
                    onChange={e => setSearch(e.target.value)}
                    placeholder="Search terms, regulations, DPDPA sections..."
                    className="w-full max-w-md px-3 py-2 rounded-md border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
            </div>

            {filtered.length === 0 ? (
                <div className="py-12 text-center text-gray-500 text-sm">
                    {terms.length === 0 ? 'No glossary terms available.' : `No terms match "${search}".`}
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {filtered.map((term) => {
                        const riskClass = RISK_LEVEL_CLASS[(term.risk_level ?? '').toLowerCase()] ?? 'bg-gray-100 text-gray-700';
                        return (
                            <div key={term.id} className="p-4 bg-white rounded-lg border border-gray-200 flex flex-col gap-2">
                                <div className="flex items-start justify-between gap-2">
                                    <h3 className="font-semibold text-gray-900 text-sm leading-tight">{term.name}</h3>
                                    {term.risk_level && (
                                        <span className={`px-2 py-0.5 rounded text-xs font-bold shrink-0 ${riskClass}`}>
                                            {term.risk_level}
                                        </span>
                                    )}
                                </div>

                                <p className="text-xs text-gray-600 leading-relaxed flex-1">{term.description}</p>

                                {term.dpdpa_section && (
                                    <div className="text-xs text-gray-500">
                                        <span className="font-semibold text-gray-700">DPDPA: </span>
                                        {term.dpdpa_section}
                                    </div>
                                )}

                                {term.regulation_refs.length > 0 && (
                                    <div className="flex flex-wrap gap-1 mt-1">
                                        {term.regulation_refs.map((ref, i) => (
                                            <span key={i} className="px-1.5 py-0.5 bg-blue-50 text-blue-700 rounded text-xs font-mono">
                                                {ref}
                                            </span>
                                        ))}
                                    </div>
                                )}

                                {term.examples && term.examples.length > 0 && (
                                    <div className="text-xs text-gray-400 italic">
                                        e.g. {term.examples.slice(0, 2).join(', ')}
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}

// ─── Main page ────────────────────────────────────────────────────────────────

function DiscoveryOverviewPageInner() {
    const router = useRouter();
    const searchParams = useSearchParams();

    const [activeTab, setActiveTab] = useState<ActiveTab>('overview');

    // Overview state
    const [overview, setOverview] = useState<OverviewSummary | null>(null);
    const [latestSnapshot, setLatestSnapshot] = useState<Snapshot | null>(null);
    const [drifts, setDrifts] = useState<DriftEvent[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [snapshotInProgress, setSnapshotInProgress] = useState(false);
    const [reportStatus, setReportStatus] = useState<string | null>(null);

    // Chart filters
    const [snapshotLimit, setSnapshotLimit] = useState<number>(
        parseInt(searchParams?.get('snapshots') ?? '4', 10)
    );
    const [sourceFilter, setSourceFilter] = useState<string>(
        searchParams?.get('source') ?? ''
    );
    const [riskFilter, setRiskFilter] = useState<string>(
        searchParams?.get('risk') ?? ''
    );

    const updateUrlParam = (key: string, value: string) => {
        const params = new URLSearchParams(searchParams?.toString() ?? '');
        if (value) params.set(key, value); else params.delete(key);
        router.replace(`?${params.toString()}`, { scroll: false });
    };

    const fetchAll = async () => {
        setLoading(true);
        setError(null);
        try {
            const [ov, snaps, dr] = await Promise.all([
                discoveryApi.getOverview(),
                discoveryApi.listSnapshots(1, 0),
                discoveryApi.getDriftTimeline(10).catch(() => ({ events: [] as DriftEvent[], count: 0, snapshot_id: '' })),
            ]);
            setOverview(ov);
            setLatestSnapshot(snaps.items?.[0] ?? null);
            setDrifts(dr.events ?? []);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load discovery dashboard');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchAll();
    }, []);

    const handleTriggerSnapshot = async () => {
        setSnapshotInProgress(true);
        setError(null);
        try {
            await discoveryApi.triggerSnapshot();
            await fetchAll();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Snapshot failed');
        } finally {
            setSnapshotInProgress(false);
        }
    };

    const handleGenerateReport = async () => {
        setReportStatus('Generating...');
        try {
            const job = await discoveryApi.generateReport('html');
            for (let i = 0; i < 10; i++) {
                await new Promise((resolve) => setTimeout(resolve, 1000));
                const status = await discoveryApi.getReport(job.report_id);
                if (status.status === 'completed') {
                    setReportStatus('Report ready');
                    window.open(discoveryApi.downloadReportUrl(job.report_id), '_blank');
                    return;
                }
                if (status.status === 'failed') {
                    setReportStatus(`Report failed: ${status.error}`);
                    return;
                }
            }
            setReportStatus('Report still running. Check Reports tab.');
        } catch (err) {
            setReportStatus(err instanceof Error ? err.message : 'Report generation failed');
        }
    };

    if (loading && !overview) {
        return (
            <div className="p-8">
                <div className="animate-pulse text-gray-500">Loading discovery dashboard...</div>
            </div>
        );
    }

    return (
        <div className="p-6 md:p-8 max-w-7xl mx-auto">
            <header className="mb-6 flex items-start justify-between flex-wrap gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900">Data Discovery</h1>
                    <p className="text-gray-600 text-sm mt-1">
                        Unified view of every PII signal across your sources. Generate a board-ready
                        report in one click.
                    </p>
                    {overview?.last_snapshot_at && (
                        <p className="text-xs text-gray-500 mt-1">
                            Last snapshot: {new Date(overview.last_snapshot_at).toLocaleString()}
                        </p>
                    )}
                </div>
                <div className="flex gap-2">
                    <button
                        onClick={handleTriggerSnapshot}
                        disabled={snapshotInProgress}
                        className="px-4 py-2 rounded-md bg-gray-100 hover:bg-gray-200 text-gray-900 text-sm font-medium disabled:opacity-50"
                    >
                        {snapshotInProgress ? 'Snapshotting...' : 'Take Snapshot Now'}
                    </button>
                    <button
                        onClick={handleGenerateReport}
                        className="px-4 py-2 rounded-md bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium"
                    >
                        Generate Board Report
                    </button>
                </div>
            </header>

            {error && (
                <div className="mb-4 p-3 rounded bg-red-50 border border-red-200 text-red-800 text-sm">{error}</div>
            )}
            {reportStatus && (
                <div className="mb-4 p-3 rounded bg-blue-50 border border-blue-200 text-blue-800 text-sm">{reportStatus}</div>
            )}

            <TabBar active={activeTab} onChange={setActiveTab} />

            {activeTab === 'overview' && (
                <OverviewTab
                    overview={overview}
                    latestSnapshot={latestSnapshot}
                    drifts={drifts}
                    snapshotLimit={snapshotLimit}
                    setSnapshotLimit={setSnapshotLimit}
                    sourceFilter={sourceFilter}
                    setSourceFilter={setSourceFilter}
                    riskFilter={riskFilter}
                    setRiskFilter={setRiskFilter}
                    updateUrlParam={updateUrlParam}
                    router={router}
                />
            )}

            {activeTab === 'inventory' && <InventoryTab router={router} />}

            {activeTab === 'risk' && <RiskTab router={router} />}

            {activeTab === 'glossary' && <GlossaryTab />}
        </div>
    );
}

export default function DiscoveryOverviewPage() {
    return (
        <Suspense fallback={<div className="p-8 space-y-4">{[...Array(4)].map((_, i) => <div key={i} className="h-24 bg-slate-100 rounded-xl animate-pulse" />)}</div>}>
            <DiscoveryOverviewPageInner />
        </Suspense>
    );
}
