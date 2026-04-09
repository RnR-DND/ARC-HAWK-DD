'use client';

import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { Shield, AlertTriangle, Scan, History, Plus, PlayCircle, Settings, FileText, ChevronRight, Activity, Zap, Database } from 'lucide-react';
import { GlobalLayout } from '@/components/layout/GlobalLayout';
import MetricCards from '@/components/ui/MetricCards';
import FindingsTable from '@/components/ui/FindingsTable';
import RiskChart from '@/components/ui/RiskChart';
import ScanStatusCard from '@/components/ui/ScanStatusCard';
import { dashboardApi, DashboardData } from '@/services/dashboard.api';
import { useWebSocket } from '@/hooks/useWebSocket';
import { AddSourceModal } from '@/components/sources/AddSourceModal';
import { ScanConfigModal } from '@/components/scans/ScanConfigModal';

// Fallback data for initial load or error
const FALLBACK_DATA: DashboardData = {
    metrics: { totalPII: 0, highRiskFindings: 0, assetsHit: 0, actionsRequired: 0 },
    recentFindings: [],
    riskDistribution: {},
    riskByAsset: {},
    riskByConfidence: {},
    latestScanId: null
};

// Skeleton component for loading state
const DashboardSkeleton = () => (
    <div className="max-w-7xl mx-auto p-6 space-y-8">
        <div className="h-20 bg-slate-100 rounded-xl animate-pulse" />
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            {[1, 2, 3, 4].map(i => (
                <div key={i} className="h-32 bg-slate-100 rounded-xl animate-pulse" />
            ))}
        </div>
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-8">
            <div className="xl:col-span-2 h-96 bg-slate-100 rounded-xl animate-pulse" />
            <div className="h-96 bg-slate-100 rounded-xl animate-pulse" />
        </div>
    </div>
);

export default function Home() {
    const [data, setData] = useState<DashboardData | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isAddSourceOpen, setIsAddSourceOpen] = useState(false);
    const [isScanConfigOpen, setIsScanConfigOpen] = useState(false);

    // WebSocket for real-time updates.
    // Normalize the env var so operators can set NEXT_PUBLIC_WS_URL without a scheme
    // (e.g. "host:8080") and still get a valid ws:// or wss:// URL.
    const wsUrl = (() => {
        const raw = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws';
        if (/^wss?:\/\//.test(raw)) return raw;
        if (typeof window !== 'undefined' && window.location.protocol === 'https:') {
            return `wss://${raw}`;
        }
        return `ws://${raw}`;
    })();
    const { lastMessage, isConnected: wsConnected } = useWebSocket({ url: wsUrl });
    const [liveFindings, setLiveFindings] = useState<any[]>([]);

    useEffect(() => {
        fetchDashboardData();
    }, []);

    // Handle real-time updates
    useEffect(() => {
        if (lastMessage) {
            try {
                const msg = JSON.parse(lastMessage.data);
                if (msg.type === 'new_finding') {
                    setLiveFindings(prev => [msg.data, ...prev].slice(0, 5));
                    // Optimistically update counts
                    setData(prev => prev ? {
                        ...prev,
                        metrics: {
                            ...prev.metrics,
                            totalPII: prev.metrics.totalPII + 1,
                            highRiskFindings: ['High', 'Critical'].includes(msg.data.severity)
                                ? prev.metrics.highRiskFindings + 1
                                : prev.metrics.highRiskFindings
                        }
                    } : null);
                } else if (msg.type === 'scan_complete' || msg.type === 'scan_progress') {
                    // Refresh data on major scan events
                    fetchDashboardData();
                }
            } catch (e) {
                console.error('Error parsing WS message:', e);
            }
        }
    }, [lastMessage]);

    const fetchDashboardData = async () => {
        try {
            setLoading(true);
            const dashboardData = await dashboardApi.getDashboardData();
            setData(dashboardData);
            setError(null);
        } catch (err) {
            console.error('Error fetching dashboard data:', err);
            setError('Failed to load dashboard data. Showing cached view.');
        } finally {
            setLoading(false);
        }
    };

    const displayData = data || FALLBACK_DATA;

    if (loading && !data) {
        return <DashboardSkeleton />;
    }

    // Empty state for first-time users
    if (!loading && displayData.metrics.totalPII === 0 && !error) {
        return (
            <div className="min-h-[80vh] flex flex-col items-center justify-center p-6 text-center">
                    <div className="max-w-2xl space-y-8">
                        <div className="p-6 bg-blue-50 rounded-full w-24 h-24 flex items-center justify-center mx-auto mb-6 ring-8 ring-blue-50/50">
                            <Shield className="w-12 h-12 text-blue-600" />
                        </div>
                        <h1 className="text-4xl font-bold text-slate-900 tracking-tight">
                            Welcome to ARC Hawk
                        </h1>
                        <p className="text-xl text-slate-600 leading-relaxed">
                            Your enterprise-grade PII discovery and remediation platform is ready. <br />
                            Connect a data source to start scanning for sensitive information.
                        </p>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-12">
                            <button
                                onClick={() => setIsAddSourceOpen(true)}
                                className="group p-6 bg-white border border-slate-200 rounded-xl hover:shadow-lg hover:border-blue-500/30 transition-all text-left"
                            >
                                <div className="bg-blue-100 w-12 h-12 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                                    <Database className="w-6 h-6 text-blue-600" />
                                </div>
                                <h3 className="font-semibold text-slate-900 text-lg mb-2">Connect Source</h3>
                                <p className="text-slate-500 text-sm">Add a database, file system, or cloud storage bucket.</p>
                            </button>

                            <button
                                onClick={() => setIsScanConfigOpen(true)}
                                className="group p-6 bg-white border border-slate-200 rounded-xl hover:shadow-lg hover:border-emerald-500/30 transition-all text-left"
                            >
                                <div className="bg-emerald-100 w-12 h-12 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                                    <PlayCircle className="w-6 h-6 text-emerald-600" />
                                </div>
                                <h3 className="font-semibold text-slate-900 text-lg mb-2">Start Scan</h3>
                                <p className="text-slate-500 text-sm">Configure and launch your first PII discovery scan.</p>
                            </button>
                        </div>
                    </div>
                    <AddSourceModal isOpen={isAddSourceOpen} onClose={() => setIsAddSourceOpen(false)} />
                    <ScanConfigModal isOpen={isScanConfigOpen} onClose={() => setIsScanConfigOpen(false)} />
                </div>
        )
    }

    return (
        <div className="min-h-screen bg-white">
            <div className="max-w-7xl mx-auto p-6 space-y-8">
                    {/* Header */}
                    <motion.div
                        initial={{ opacity: 0, y: -20 }}
                        animate={{ opacity: 1, y: 0 }}
                        className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4"
                    >
                        <div>
                            <h1 className="text-3xl font-bold text-slate-950 tracking-tight">Security Overview</h1>
                            <p className="text-slate-600 mt-1">Real-time PII detection and risk analysis</p>
                        </div>
                        <div className="flex items-center gap-3">
                        </div>
                    </motion.div>

                    {/* Modals */}
                    <AddSourceModal isOpen={isAddSourceOpen} onClose={() => setIsAddSourceOpen(false)} />
                    <ScanConfigModal isOpen={isScanConfigOpen} onClose={() => setIsScanConfigOpen(false)} />

                    {/* Error Banner */}
                    {error && (
                        <motion.div
                            initial={{ opacity: 0, height: 0 }}
                            animate={{ opacity: 1, height: 'auto' }}
                            className="bg-amber-50 border border-amber-200 text-amber-800 px-4 py-3 rounded-lg flex items-center gap-3"
                        >
                            <AlertTriangle className="w-5 h-5 text-amber-600 shrink-0" />
                            <span className="text-sm">{error}</span>
                            <button onClick={fetchDashboardData} className="ml-auto text-amber-700 hover:text-amber-900 text-sm font-medium whitespace-nowrap">
                                Retry
                            </button>
                        </motion.div>
                    )}

                    {/* Metrics Cards */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: 0.1 }}
                    >
                        <MetricCards
                            {...displayData.metrics}
                            loading={loading}
                        />
                    </motion.div>

                    {/* Scan Status */}
                    {displayData.latestScanId && (
                        <motion.div
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.2 }}
                        >
                            <ScanStatusCard scanId={displayData.latestScanId} />
                        </motion.div>
                    )}

                    {/* Live Findings Ticker */}
                    {wsConnected && liveFindings.length > 0 && (
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            className="bg-white border border-slate-200 rounded-lg p-3 flex items-center gap-4 shadow-sm overflow-hidden"
                        >
                            <div className="flex items-center gap-2 px-2 border-r border-slate-200">
                                <span className="relative flex h-3 w-3">
                                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
                                    <span className="relative inline-flex rounded-full h-3 w-3 bg-red-500"></span>
                                </span>
                                <span className="text-sm font-semibold text-slate-700 whitespace-nowrap">Live Feed</span>
                            </div>
                            <div className="flex-1 overflow-hidden">
                                <div className="animate-marquee flex gap-8">
                                    {liveFindings.map((f, i) => (
                                        <span key={i} className="text-sm font-mono text-slate-600 flex items-center gap-2">
                                            <span className={`w-2 h-2 rounded-full ${f.severity === 'Critical' ? 'bg-red-500' : 'bg-amber-500'
                                                }`} />
                                            Found {f.type} in {f.source}
                                        </span>
                                    ))}
                                </div>
                            </div>
                        </motion.div>
                    )}

                    {/* Main Content Grid */}
                    <div className="grid grid-cols-1 xl:grid-cols-3 gap-8">
                        {/* Findings Table */}
                        <motion.div
                            className="xl:col-span-2"
                            initial={{ opacity: 0, x: -20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: 0.3 }}
                        >
                            <div className="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
                                <div className="p-6 border-b border-slate-100 flex items-center justify-between">
                                    <div>
                                        <h2 className="text-lg font-bold text-slate-900 flex items-center gap-2">
                                            <Activity className="w-5 h-5 text-blue-600" />
                                            Recent Findings
                                        </h2>
                                        <p className="text-sm text-slate-600">Latest discovered PII instances</p>
                                    </div>
                                    <button className="text-sm text-blue-600 hover:text-blue-700 font-medium flex items-center gap-1 hover:gap-2 transition-all">
                                        View All <ChevronRight className="w-4 h-4" />
                                    </button>
                                </div>
                                <FindingsTable findings={displayData.recentFindings} loading={loading} />
                            </div>
                        </motion.div>

                        {/* Risk Analysis */}
                        <motion.div
                            initial={{ opacity: 0, x: 20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: 0.3 }}
                        >
                            <RiskChart
                                byPiiType={displayData.riskDistribution}
                                byAsset={displayData.riskByAsset}
                                byConfidence={displayData.riskByConfidence}
                                loading={loading}
                            />
                        </motion.div>
                    </div>

                    {/* Quick Actions */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: 0.4 }}
                    >
                        <h3 className="text-lg font-bold text-slate-800 mb-4 px-1">Quick Actions</h3>
                        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
                            <button
                                onClick={() => setIsScanConfigOpen(true)}
                                className="p-4 bg-white border border-slate-200 hover:border-blue-300 hover:shadow-md rounded-xl transition-all group text-left"
                            >
                                <div className="w-10 h-10 bg-blue-50 text-blue-600 rounded-lg flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                                    <PlayCircle className="w-6 h-6" />
                                </div>
                                <div className="font-semibold text-slate-800">New Scan</div>
                                <div className="text-xs text-slate-500 mt-1">Configure and start scanning</div>
                            </button>

                            <button
                                onClick={() => setIsAddSourceOpen(true)}
                                className="p-4 bg-white border border-slate-200 hover:border-emerald-300 hover:shadow-md rounded-xl transition-all group text-left"
                            >
                                <div className="w-10 h-10 bg-emerald-50 text-emerald-600 rounded-lg flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                                    <Plus className="w-6 h-6" />
                                </div>
                                <div className="font-semibold text-slate-800">Add Source</div>
                                <div className="text-xs text-slate-500 mt-1">Connect new data source</div>
                            </button>

                            <button className="p-4 bg-white border border-slate-200 hover:border-purple-300 hover:shadow-md rounded-xl transition-all group text-left">
                                <div className="w-10 h-10 bg-purple-50 text-purple-600 rounded-lg flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                                    <FileText className="w-6 h-6" />
                                </div>
                                <div className="font-semibold text-slate-800">Generate Report</div>
                                <div className="text-xs text-slate-500 mt-1">Export compliance status</div>
                            </button>

                            <button className="p-4 bg-white border border-slate-200 hover:border-slate-300 hover:shadow-md rounded-xl transition-all group text-left">
                                <div className="w-10 h-10 bg-slate-100 text-slate-600 rounded-lg flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                                    <Settings className="w-6 h-6" />
                                </div>
                                <div className="font-semibold text-slate-800">Settings</div>
                                <div className="text-xs text-slate-500 mt-1">System configuration</div>
                            </button>
                        </div>
                    </motion.div>
                </div>
        </div>
    );
}
