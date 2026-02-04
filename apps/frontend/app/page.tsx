'use client';

import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { Shield, AlertTriangle, Database, CheckCircle, RefreshCw, Play, Wifi, WifiOff } from 'lucide-react';
import MetricCards from '@/components/ui/MetricCards';
import FindingsTable from '@/components/ui/FindingsTable';
import RiskChart from '@/components/ui/RiskChart';
import ScanStatusCard from '@/components/ui/ScanStatusCard';
// @ts-ignore
import { dashboardApi, DashboardData } from '@/services/dashboard.api';
import { useWebSocket, useSystemStatus } from '@/hooks/useWebSocket';
import { AddSourceModal } from '@/components/sources/AddSourceModal';
import { ScanConfigModal } from '@/components/scans/ScanConfigModal';

const FALLBACK_DATA: DashboardData = {
    metrics: {
        totalPII: 0,
        highRiskFindings: 0,
        assetsHit: 0,
        actionsRequired: 0
    },
    recentFindings: [],
    riskDistribution: {},
    riskByAsset: {},
    riskByConfidence: {},
    latestScanId: null
};

// Skeleton component for loading state
const DashboardSkeleton = () => (
    <div className="min-h-screen bg-background p-6 space-y-8">
        <div className="flex justify-between items-center animate-pulse">
            <div className="h-10 w-64 bg-muted rounded"></div>
            <div className="h-10 w-32 bg-muted rounded"></div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {[...Array(4)].map((_, i) => (
                <div key={i} className="h-32 bg-muted rounded-xl animate-pulse"></div>
            ))}
        </div>
        <div className="h-64 bg-muted rounded-xl animate-pulse"></div>
    </div>
);

export default function Home() {
    const [data, setData] = useState<DashboardData | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [lastUpdated, setLastUpdated] = useState<Date>(new Date());
    const [liveFindings, setLiveFindings] = useState<any[]>([]);

    // Modal states
    const [showAddSourceModal, setShowAddSourceModal] = useState(false);
    const [showScanConfigModal, setShowScanConfigModal] = useState(false);

    // Real-time WebSocket connection
    const systemStatus = useSystemStatus();
    const { isConnected: wsConnected } = useWebSocket({
        onMessage: (message) => {
            switch (message.type) {
                case 'new_finding':
                    // Add new finding to live findings
                    setLiveFindings(prev => [message.data, ...prev.slice(0, 9)]);
                    // Refresh dashboard data
                    fetchDashboardData();
                    break;
                case 'scan_progress':
                    // Update scan progress (could show in UI)
                    console.log('Scan progress:', message.data);
                    break;
                case 'scan_complete':
                    // Refresh dashboard when scan completes
                    fetchDashboardData();
                    break;
            }
        }
    });

    useEffect(() => {
        fetchDashboardData();
        // Auto-refresh every 60 seconds (less frequent with real-time updates)
        const interval = setInterval(fetchDashboardData, 60000);
        return () => clearInterval(interval);
    }, []);

    const fetchDashboardData = async () => {
        try {
            setLoading(true);
            const dashboardData = await dashboardApi.getDashboardData();
            setData(dashboardData);
            setError(null);
            setLastUpdated(new Date());
        } catch (err) {
            console.error('Failed to load dashboard:', err);
            setError('Failed to load dashboard data. Using fallback.');
            setData(FALLBACK_DATA);
        } finally {
            setLoading(false);
        }
    };

    if (loading && !data) {
        return <DashboardSkeleton />;
    }

    const displayData = data || FALLBACK_DATA;

    // Empty state for no data sources
    if (!loading && !displayData.latestScanId && displayData.metrics.totalPII === 0) {
        return (
            <div className="min-h-screen bg-background">
                <div className="max-w-7xl mx-auto p-6">
                    {/* Modals */}
                    <AddSourceModal
                        isOpen={showAddSourceModal}
                        onClose={() => setShowAddSourceModal(false)}
                    />
                    <ScanConfigModal
                        isOpen={showScanConfigModal}
                        onClose={() => setShowScanConfigModal(false)}
                        onRunScan={() => {
                            setTimeout(fetchDashboardData, 1000);
                        }}
                    />

                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        className="text-center py-20"
                    >
                        <div className="mx-auto w-24 h-24 bg-gradient-to-br from-blue-500/20 to-purple-500/20 rounded-full flex items-center justify-center mb-8">
                            <Shield className="w-12 h-12 text-blue-400" />
                        </div>
                        <h1 className="text-3xl font-bold text-white mb-4">
                            Welcome to ARC-Hawk
                        </h1>
                        <p className="text-xl text-muted-foreground mb-8 max-w-2xl mx-auto">
                            Your enterprise-grade PII governance platform. Start by connecting your data sources and running your first scan to discover sensitive information.
                        </p>

                        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-4xl mx-auto mb-12">
                            <motion.div
                                initial={{ opacity: 0, x: -20 }}
                                animate={{ opacity: 1, x: 0 }}
                                transition={{ delay: 0.2 }}
                                className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6"
                            >
                                <div className="p-3 bg-blue-500/20 rounded-lg w-fit mb-4">
                                    <Database className="w-6 h-6 text-blue-400" />
                                </div>
                                <h3 className="text-lg font-semibold text-white mb-2">Connect Data Sources</h3>
                                <p className="text-muted-foreground text-sm">
                                    Link databases, file systems, cloud storage, and APIs to scan for PII.
                                </p>
                            </motion.div>

                            <motion.div
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.4 }}
                                className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6"
                            >
                                <div className="p-3 bg-emerald-500/20 rounded-lg w-fit mb-4">
                                    <Play className="w-6 h-6 text-emerald-400" />
                                </div>
                                <h3 className="text-lg font-semibold text-white mb-2">Run Your First Scan</h3>
                                <p className="text-muted-foreground text-sm">
                                    Execute comprehensive PII discovery across all connected sources.
                                </p>
                            </motion.div>

                            <motion.div
                                initial={{ opacity: 0, x: 20 }}
                                animate={{ opacity: 1, x: 0 }}
                                transition={{ delay: 0.6 }}
                                className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6"
                            >
                                <div className="p-3 bg-purple-500/20 rounded-lg w-fit mb-4">
                                    <CheckCircle className="w-6 h-6 text-purple-400" />
                                </div>
                                <h3 className="text-lg font-semibold text-white mb-2">Review & Remediate</h3>
                                <p className="text-muted-foreground text-sm">
                                    Analyze findings and take automated remediation actions.
                                </p>
                            </motion.div>
                        </div>

                        <motion.div
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.8 }}
                            className="flex flex-col sm:flex-row gap-4 justify-center"
                        >
                            <button
                                onClick={() => setShowAddSourceModal(true)}
                                className="px-8 py-3 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 text-white rounded-lg font-medium transition-all shadow-lg hover:shadow-xl"
                            >
                                Get Started
                            </button>
                            <button
                                onClick={() => window.open('https://docs.arc-hawk.io', '_blank')}
                                className="px-8 py-3 bg-secondary hover:bg-accent text-slate-300 hover:text-white rounded-lg font-medium transition-all border border-border/50 hover:border-slate-500/50"
                            >
                                View Documentation
                            </button>
                        </motion.div>
                    </motion.div>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-background">
            <div className="max-w-7xl mx-auto p-6 space-y-8">
                {/* Header */}
                <motion.div
                    initial={{ opacity: 0, y: -20 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="flex items-center justify-between"
                >
                    <div>
                        <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
                            ARC-Hawk Dashboard
                        </h1>
                        <p className="text-muted-foreground mt-2">
                            Real-time PII governance and risk management
                        </p>
                    </div>

                    <div className="flex items-center gap-4">
                        {/* WebSocket Connection Status */}
                        <div className="flex items-center gap-2">
                            {wsConnected ? (
                                <Wifi className="w-4 h-4 text-green-400" />
                            ) : (
                                <WifiOff className="w-4 h-4 text-red-400" />
                            )}
                            <span className={`text-sm ${wsConnected ? 'text-green-400' : 'text-red-400'}`}>
                                {wsConnected ? 'Live' : 'Offline'}
                            </span>
                        </div>

                        <div className="text-right">
                            <div className="text-sm text-muted-foreground">Last updated</div>
                            <div className="text-white font-medium">
                                {lastUpdated.toLocaleTimeString()}
                            </div>
                        </div>
                        <button
                            onClick={fetchDashboardData}
                            className="p-3 bg-muted hover:bg-accent rounded-lg transition-colors"
                            title="Refresh data"
                        >
                            <RefreshCw className="w-5 h-5 text-slate-300" />
                        </button>
                    </div>
                </motion.div>

                {/* Modals */}
                <AddSourceModal
                    isOpen={showAddSourceModal}
                    onClose={() => setShowAddSourceModal(false)}
                />
                <ScanConfigModal
                    isOpen={showScanConfigModal}
                    onClose={() => setShowScanConfigModal(false)}
                    onRunScan={() => {
                        setTimeout(fetchDashboardData, 1000);
                    }}
                />

                {/* Error Banner */}
                {error && (
                    <motion.div
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        className="bg-gradient-to-r from-amber-500/10 to-orange-500/10 border border-amber-500/30 rounded-xl p-4"
                    >
                        <div className="flex items-center gap-3">
                            <AlertTriangle className="w-5 h-5 text-amber-400" />
                            <div>
                                <h3 className="text-amber-400 font-medium">Connection Issue</h3>
                                <p className="text-amber-300/80 text-sm">{error}</p>
                            </div>
                        </div>
                    </motion.div>
                )}

                {/* Metrics Cards */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.1 }}
                >
                    <MetricCards
                        totalPII={displayData.metrics.totalPII}
                        highRiskFindings={displayData.metrics.highRiskFindings}
                        assetsHit={displayData.metrics.assetsHit}
                        actionsRequired={displayData.metrics.actionsRequired}
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

                {/* Live Findings */}
                {wsConnected && liveFindings.length > 0 && (
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: 0.25 }}
                        className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6"
                    >
                        <div className="flex items-center gap-3 mb-4">
                            <div className="w-8 h-8 bg-green-500/20 rounded-lg flex items-center justify-center">
                                <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                            </div>
                            <div>
                                <h3 className="text-lg font-semibold text-white">Live Findings</h3>
                                <p className="text-sm text-muted-foreground">Real-time PII detections from active scans</p>
                            </div>
                        </div>

                        <div className="space-y-3">
                            {liveFindings.slice(0, 5).map((finding, index) => (
                                <motion.div
                                    key={`${finding.id}-${index}`}
                                    initial={{ opacity: 0, x: -20 }}
                                    animate={{ opacity: 1, x: 0 }}
                                    transition={{ delay: index * 0.1 }}
                                    className="flex items-center justify-between p-3 bg-accent rounded-lg border border-border/30"
                                >
                                    <div className="flex items-center gap-3">
                                        <div className={`w-2 h-2 rounded-full ${finding.severity === 'Critical' ? 'bg-red-400' :
                                            finding.severity === 'High' ? 'bg-orange-400' :
                                                'bg-yellow-400'
                                            }`}></div>
                                        <div>
                                            <div className="text-sm font-medium text-white">
                                                {finding.pii_type} in {finding.asset_name}
                                            </div>
                                            <div className="text-xs text-muted-foreground">
                                                {finding.asset_path}
                                            </div>
                                        </div>
                                    </div>
                                    <div className="text-xs text-slate-500">
                                        {new Date().toLocaleTimeString()}
                                    </div>
                                </motion.div>
                            ))}
                        </div>
                    </motion.div>
                )}

                {/* Main Content Grid */}
                <div className="grid grid-cols-1 xl:grid-cols-3 gap-8">
                    {/* Findings Table */}
                    <motion.div
                        initial={{ opacity: 0, x: -20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.3 }}
                        className="xl:col-span-2"
                    >
                        <FindingsTable
                            findings={displayData.recentFindings.map(f => ({
                                id: f.id,
                                assetName: f.assetName,
                                assetPath: f.assetPath,
                                field: f.field,
                                piiType: f.piiType,
                                confidence: f.confidence,
                                risk: f.risk === 'High' ? 'Critical' : f.risk === 'Medium' ? 'High' : 'Low' as const,
                                sourceType: f.sourceType === 'Filesystem' ? 'File' : f.sourceType === 'S3' ? 'Cloud' : 'Database' as const,
                            }))}
                            loading={loading}
                        />
                    </motion.div>

                    {/* Risk Analysis */}
                    <motion.div
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.4 }}
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
                    transition={{ delay: 0.5 }}
                    className="bg-card backdrop-blur-sm border border-border/30 rounded-xl p-6"
                >
                    <div className="flex items-center justify-between mb-6">
                        <div>
                            <h3 className="text-lg font-semibold text-white">Quick Actions</h3>
                            <p className="text-muted-foreground text-sm mt-1">Common tasks to manage your PII governance</p>
                        </div>
                        <div className="text-xs text-slate-500 bg-card/50 px-2 py-1 rounded">
                            {displayData.latestScanId ? 'System Active' : 'Setup Required'}
                        </div>
                    </div>

                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <motion.button
                            whileHover={{ scale: 1.02, y: -2 }}
                            whileTap={{ scale: 0.98 }}
                            onClick={() => setShowScanConfigModal(true)}
                            className="group relative flex flex-col items-center gap-3 p-6 bg-gradient-to-br from-blue-500/10 to-blue-600/10 hover:from-blue-500/20 hover:to-blue-600/20 border border-blue-500/20 hover:border-blue-500/40 rounded-xl transition-all duration-200"
                            title="Start a new comprehensive scan of all connected data sources"
                        >
                            <div className="p-3 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                                <Shield className="w-6 h-6 text-blue-400 group-hover:text-blue-300" />
                            </div>
                            <div className="text-center">
                                <span className="text-sm font-medium text-slate-300 group-hover:text-white block">New Scan</span>
                                <span className="text-xs text-slate-500 group-hover:text-muted-foreground block mt-1">Discover PII</span>
                            </div>
                        </motion.button>

                        <motion.button
                            whileHover={{ scale: 1.02, y: -2 }}
                            whileTap={{ scale: 0.98 }}
                            onClick={() => setShowAddSourceModal(true)}
                            className="group relative flex flex-col items-center gap-3 p-6 bg-gradient-to-br from-emerald-500/10 to-emerald-600/10 hover:from-emerald-500/20 hover:to-emerald-600/20 border border-emerald-500/20 hover:border-emerald-500/40 rounded-xl transition-all duration-200"
                            title="Connect a new data source (database, cloud storage, etc.)"
                        >
                            <div className="p-3 bg-emerald-500/20 rounded-lg group-hover:bg-emerald-500/30 transition-colors">
                                <Database className="w-6 h-6 text-emerald-400 group-hover:text-emerald-300" />
                            </div>
                            <div className="text-center">
                                <span className="text-sm font-medium text-slate-300 group-hover:text-white block">Add Source</span>
                                <span className="text-xs text-slate-500 group-hover:text-muted-foreground block mt-1">Connect Data</span>
                            </div>
                        </motion.button>

                        <div className="relative group/btn cursor-pointer">
                            <motion.button
                                whileHover={{ scale: 1.02, y: -2 }}
                                whileTap={{ scale: 0.98 }}
                                onClick={() => window.location.href = '/remediation'}
                                className="w-full h-full group relative flex flex-col items-center gap-3 p-6 bg-gradient-to-br from-amber-500/10 to-amber-600/10 hover:from-amber-500/20 hover:to-amber-600/20 border border-amber-500/20 hover:border-amber-500/40 rounded-xl transition-all duration-200"
                                title="Manage remediation tasks"
                            >
                                <div className="p-3 bg-amber-500/20 rounded-lg transition-colors group-hover:bg-amber-500/30">
                                    <CheckCircle className="w-6 h-6 text-amber-500 group-hover:text-amber-400" />
                                </div>
                                <div className="text-center">
                                    <span className="text-sm font-medium text-slate-300 group-hover:text-white block">Remediate</span>
                                    <span className="text-xs text-slate-500 group-hover:text-muted-foreground block mt-1">
                                        Action Center
                                    </span>
                                </div>
                            </motion.button>
                        </div>
                    </div>
                </motion.div>
            </div>
        </div>
    );
}
