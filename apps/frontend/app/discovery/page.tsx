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
import { discoveryApi, OverviewSummary, Snapshot, DriftEvent } from '@/services/discoveryApi';
import { useSearchParams, useRouter } from 'next/navigation';

/**
 * Discovery Overview — the front door of the Data Discovery Module.
 *
 * Renders four KPI cards (sources, assets, findings, composite risk),
 * a 4-quarter trend chart, the top-5 risk hotspots, the latest drift events,
 * and a "Generate Board Report" action.
 *
 * Data flows:
 *   1. On mount: GET /api/v1/discovery/overview → KPI cards + trend + hotspots
 *   2. On mount: GET /api/v1/discovery/snapshots?limit=1 → most recent snapshot
 *   3. On mount: GET /api/v1/discovery/drift/timeline?limit=10 → drift list
 *   4. Click "Take Snapshot Now" → POST /snapshots/trigger then refresh
 *   5. Click "Generate Board Report" → POST /reports/generate, poll status, open download URL
 */
// Risk tier helpers for the heatmap
function riskColor(score: number): string {
    if (score >= 80) return '#ef4444'; // Critical
    if (score >= 60) return '#f97316'; // High
    if (score >= 40) return '#eab308'; // Medium
    return '#22c55e';                  // Low
}
function riskLabel(score: number): string {
    if (score >= 80) return 'Critical';
    if (score >= 60) return 'High';
    if (score >= 40) return 'Medium';
    return 'Low';
}

function DiscoveryOverviewPageInner() {
    const router = useRouter();
    const searchParams = useSearchParams();

    const [overview, setOverview] = useState<OverviewSummary | null>(null);
    const [latestSnapshot, setLatestSnapshot] = useState<Snapshot | null>(null);
    const [drifts, setDrifts] = useState<DriftEvent[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [snapshotInProgress, setSnapshotInProgress] = useState(false);
    const [reportStatus, setReportStatus] = useState<string | null>(null);

    // Chart filters — persisted in URL params for shareable views
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
            setLatestSnapshot(snaps.items[0] ?? null);
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
            // Poll up to 10 times with 1s gap.
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
                <div className="mb-4 p-3 rounded bg-red-50 border border-red-200 text-red-800 text-sm">
                    {error}
                </div>
            )}
            {reportStatus && (
                <div className="mb-4 p-3 rounded bg-blue-50 border border-blue-200 text-blue-800 text-sm">
                    {reportStatus}
                </div>
            )}

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
                        onChange={e => {
                            setSourceFilter(e.target.value);
                            updateUrlParam('source', e.target.value);
                        }}
                        className="ml-1 px-2 py-1 rounded border border-gray-300 text-sm bg-white w-32"
                    />
                </label>
                <label className="flex items-center gap-1 text-gray-700">
                    Risk tier:
                    <select
                        value={riskFilter}
                        onChange={e => {
                            setRiskFilter(e.target.value);
                            updateUrlParam('risk', e.target.value);
                        }}
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
                            <Line
                                type="monotone"
                                dataKey="finding_count"
                                stroke="#3b82f6"
                                strokeWidth={2}
                                name="Findings"
                            />
                            <Line
                                type="monotone"
                                dataKey="composite_risk_score"
                                stroke="#ef4444"
                                strokeWidth={2}
                                name="Risk Score"
                            />
                        </LineChart>
                    </ResponsiveContainer>
                ) : (
                    <div className="text-sm text-gray-500 py-8 text-center">
                        Not enough snapshots to plot a trend yet. Run a few more to see history.
                    </div>
                )}
            </section>

            {/* Hotspots heatmap + Drift side by side */}
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
                                                    <span
                                                        style={{ color, fontWeight: 700 }}
                                                        className="font-mono">
                                                        {h.score.toFixed(1)}
                                                    </span>
                                                    <span
                                                        style={{
                                                            backgroundColor: `${color}20`,
                                                            color,
                                                            fontSize: '10px',
                                                            fontWeight: 700,
                                                            padding: '1px 6px',
                                                            borderRadius: '999px',
                                                        }}>
                                                        {riskLabel(h.score)}
                                                    </span>
                                                </span>
                                            </div>
                                            <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                                                <div
                                                    style={{
                                                        width: `${pct}%`,
                                                        height: '100%',
                                                        backgroundColor: color,
                                                        borderRadius: '999px',
                                                        transition: 'width 0.4s ease',
                                                    }}
                                                />
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
                                    <span
                                        className={`px-2 py-0.5 text-xs rounded ${severityClass(
                                            d.severity
                                        )}`}
                                    >
                                        {d.severity}
                                    </span>
                                    <div className="flex-1">
                                        <div className="font-medium text-gray-800">
                                            {humanizeEvent(d.event_type)}
                                        </div>
                                        <div className="text-xs text-gray-500">
                                            {new Date(d.detected_at).toLocaleString()}
                                        </div>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <div className="text-sm text-gray-500 py-4">
                            No drift detected since the last snapshot.
                        </div>
                    )}
                </div>
            </section>

            {latestSnapshot && (
                <footer className="mt-6 text-xs text-gray-500">
                    Snapshot {latestSnapshot.id.slice(0, 8)} ·{' '}
                    {latestSnapshot.duration_ms
                        ? `${(latestSnapshot.duration_ms / 1000).toFixed(1)}s`
                        : 'pending'}{' '}
                    · {latestSnapshot.status}
                </footer>
            )}
        </div>
    );
}

interface KpiCardProps {
    label: string;
    value: string | number;
    accent?: boolean;
}

function KpiCard({ label, value, accent }: KpiCardProps) {
    return (
        <div
            className={`p-4 rounded-lg border ${
                accent ? 'border-blue-200 bg-blue-50' : 'border-gray-200 bg-white'
            }`}
        >
            <div className="text-xs uppercase tracking-wide text-gray-500">{label}</div>
            <div
                className={`text-2xl font-bold mt-1 ${accent ? 'text-blue-700' : 'text-gray-900'}`}
            >
                {value}
            </div>
        </div>
    );
}

function severityClass(s: string): string {
    switch (s) {
        case 'critical':
            return 'bg-red-100 text-red-800';
        case 'high':
            return 'bg-orange-100 text-orange-800';
        case 'medium':
            return 'bg-yellow-100 text-yellow-800';
        default:
            return 'bg-gray-100 text-gray-700';
    }
}

function humanizeEvent(t: string): string {
    switch (t) {
        case 'asset_added':
            return 'New asset/classification appeared';
        case 'asset_removed':
            return 'Asset/classification removed';
        case 'classification_changed':
            return 'Classification changed';
        case 'risk_increased':
            return 'Risk score increased';
        case 'risk_decreased':
            return 'Risk score decreased';
        case 'finding_count_spike':
            return 'PII finding count spiked';
        default:
            return t;
    }
}

export default function DiscoveryOverviewPage() {
    return (
        <Suspense fallback={<div className="p-8 space-y-4">{[...Array(4)].map((_, i) => <div key={i} className="h-24 bg-slate-100 rounded-xl animate-pulse" />)}</div>}>
            <DiscoveryOverviewPageInner />
        </Suspense>
    );
}
