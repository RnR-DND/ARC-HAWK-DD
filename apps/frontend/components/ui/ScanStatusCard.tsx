import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { Clock, CheckCircle, Play, Pause, AlertCircle } from 'lucide-react';

interface ScanStatusCardProps {
    scanId: string | null;
}

const scanStatuses = {
    'idle': { label: 'Idle', color: 'text-muted-foreground', bg: 'bg-slate-500/10', icon: Pause },
    'running': { label: 'Running', color: 'text-blue-400', bg: 'bg-blue-500/10', icon: Play },
    'completed': { label: 'Completed', color: 'text-emerald-400', bg: 'bg-emerald-500/10', icon: CheckCircle },
    'failed': { label: 'Failed', color: 'text-red-400', bg: 'bg-red-500/10', icon: AlertCircle },
};

type ScanStatus = 'idle' | 'running' | 'completed' | 'failed';

export default function ScanStatusCard({ scanId }: ScanStatusCardProps) {
    const [scanData, setScanData] = useState<any>(null);
    const [loading, setLoading] = useState(true);
    const [cancelling, setCancelling] = useState(false);

    useEffect(() => {
        if (scanId) {
            fetchScanData();
            // Poll for updates every 5 seconds if scan is running
            const interval = setInterval(() => {
                if (scanData?.status === 'running') {
                    fetchScanData();
                }
            }, 5000);
            return () => clearInterval(interval);
        } else {
            setLoading(false);
        }
    }, [scanId, scanData?.status]);

    const fetchScanData = async () => {
        try {
            const res = await fetch(`/api/v1/scans/${scanId}/status`);
            if (res.ok) {
                const data = await res.json();
                setScanData(data);
            }
        } catch (error) {
            console.error('Failed to fetch scan data:', error);
        } finally {
            setLoading(false);
        }
    };

    const handleCancelScan = async () => {
        if (!scanId || cancelling) return;

        setCancelling(true);
        try {
            const res = await fetch(`/api/v1/scans/${scanId}/cancel`, {
                method: 'POST',
            });
            if (res.ok) {
                await fetchScanData(); // Refresh scan data
            }
        } catch (error) {
            console.error('Failed to cancel scan:', error);
        } finally {
            setCancelling(false);
        }
    };

    // Determine status from scan data or default to idle
    const status: ScanStatus = scanData?.status === 'completed' ? 'completed' :
        scanData?.status === 'running' ? 'running' :
            scanData?.status === 'failed' ? 'failed' :
                scanData?.status === 'cancelled' ? 'failed' :
                    'idle';

    const startTime = scanData?.created_at ? new Date(scanData.created_at) : null;
    const endTime = scanData?.completed_at ? new Date(scanData.completed_at) : null;
    const progress = scanData?.progress !== undefined ? scanData.progress : (status === 'completed' ? 100 : 0);

    const StatusIcon = scanStatuses[status as keyof typeof scanStatuses].icon;

    const getStatusDescription = () => {
        switch (status) {
            case 'idle':
                return 'System ready. No active scans running.';
            case 'completed':
                return 'Scan completed. Findings are ready for review.';
            case 'running':
                return 'Scan in progress. Discovering PII across your data sources.';
            case 'failed':
                return scanData?.status === 'cancelled' ? 'Scan was cancelled.' : 'Scan failed. Please check logs for details.';
        }
    };

    const getStatusColor = () => {
        switch (status) {
            case 'idle': return 'text-muted-foreground';
            case 'completed': return 'text-emerald-400';
            case 'running': return 'text-blue-400';
            case 'failed': return 'text-red-400';
        }
        return 'text-muted-foreground';
    };

    const getRecommendedActions = () => {
        switch (status) {
            case 'idle':
                return [
                    { label: 'Add Data Source', description: 'Connect databases, files, or cloud storage', priority: 'high' as const },
                    { label: 'Configure Scan', description: 'Set scan parameters and rules', priority: 'medium' as const },
                    { label: 'Start Scan', description: 'Begin comprehensive PII discovery', priority: 'high' as const }
                ];
            case 'completed':
                return [
                    { label: 'Review Findings', description: 'Examine discovered PII instances', priority: 'high' as const },
                    { label: 'Generate Report', description: 'Create compliance documentation', priority: 'medium' as const },
                    { label: 'Start New Scan', description: 'Scan additional data sources', priority: 'medium' as const }
                ];
            case 'running':
                return [
                    { label: 'Monitor Progress', description: 'Track scan execution in real-time', priority: 'high' as const },
                    { label: 'View Partial Results', description: 'See findings as they are discovered', priority: 'medium' as const },
                ];
        }
        return [];
    };

    const recommendedActions = getRecommendedActions();

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6"
        >
            <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-4">
                    <div className={`p-3 rounded-lg ${scanStatuses[status as keyof typeof scanStatuses].bg} ring-2 ring-white/10`}>
                        <StatusIcon className={`w-6 h-6 ${scanStatuses[status as keyof typeof scanStatuses].color}`} />
                    </div>
                    <div>
                        <h3 className="text-lg font-semibold text-white">Scan Status</h3>
                        <p className={`text-sm font-medium ${getStatusColor()}`}>
                            {scanStatuses[status as keyof typeof scanStatuses].label}
                        </p>
                        <p className="text-muted-foreground text-sm mt-1">
                            {getStatusDescription()}
                        </p>
                    </div>
                </div>

                {scanId && (
                    <div className="text-right">
                        <div className="text-xs text-slate-500 mb-1">Latest Scan</div>
                        <div className="text-sm font-mono text-slate-300 bg-card/50 px-2 py-1 rounded">
                            {scanId}
                        </div>
                    </div>
                )}
            </div>

            {/* Progress Bar */}
            <div className="mb-4">
                <div className="flex items-center justify-between mb-2">
                    <span className="text-sm text-muted-foreground">Progress</span>
                    <span className="text-sm text-slate-300">{progress}%</span>
                </div>
                <div className="w-full bg-accent rounded-full h-2">
                    <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${progress}%` }}
                        transition={{ duration: 1, ease: "easeOut" }}
                        className={`h-2 rounded-full ${status === 'completed' ? 'bg-emerald-500' : status === 'running' ? 'bg-blue-500' : 'bg-slate-500'
                            }`}
                    />
                </div>
            </div>

            {/* Time Information */}
            <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                    <div className="text-slate-500 mb-1">Started</div>
                    <div className="text-slate-300 flex items-center gap-2">
                        <Clock className="w-4 h-4" />
                        {startTime ? startTime.toLocaleTimeString() : 'N/A'}
                    </div>
                </div>

                {endTime && (
                    <div>
                        <div className="text-slate-500 mb-1">
                            {status === 'completed' ? 'Completed' : 'Failed'}
                        </div>
                        <div className="text-slate-300 flex items-center gap-2">
                            <CheckCircle className="w-4 h-4" />
                            {endTime ? endTime.toLocaleTimeString() : 'N/A'}
                        </div>
                    </div>
                )}

                {status === 'completed' && (
                    <div>
                        <div className="text-slate-500 mb-1">Duration</div>
                        <div className="text-slate-300">
                            {endTime && startTime ? `${Math.floor((endTime.getTime() - startTime.getTime()) / 1000 / 60)}m total` : 'N/A'}
                        </div>
                    </div>
                )}
            </div>

            {/* Recommended Actions */}
            <div className="mb-6 mt-6">
                <h4 className="text-sm font-medium text-slate-300 mb-3">Recommended Next Steps</h4>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    {recommendedActions.map((action, index) => (
                        <motion.button
                            key={action.label}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: index * 0.1 }}
                            className={`p-3 rounded-lg border transition-all duration-200 text-left group ${action.priority === 'high'
                                ? 'bg-blue-500/10 border-blue-500/30 hover:bg-blue-500/20 hover:border-blue-500/50'
                                : action.priority === 'medium'
                                    ? 'bg-accent border-border/50 hover:bg-slate-600/50 hover:border-slate-500/50'
                                    : 'bg-secondary border-border/50 hover:bg-accent hover:border-border/50'
                                }`}
                            title={action.description}
                        >
                            <div className={`text-sm font-medium mb-1 ${action.priority === 'high' ? 'text-blue-400' : 'text-slate-300'
                                }`}>
                                {action.label}
                            </div>
                            <div className="text-xs text-muted-foreground group-hover:text-slate-300 transition-colors">
                                {action.description}
                            </div>
                            {action.priority === 'high' && (
                                <div className="mt-2">
                                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-500/20 text-blue-400 border border-blue-500/30">
                                        Priority
                                    </span>
                                </div>
                            )}
                        </motion.button>
                    ))}
                </div>
            </div>

            {/* Action Buttons */}
            <div className="flex gap-3">
                {status === 'running' && (
                    <button
                        onClick={handleCancelScan}
                        disabled={cancelling}
                        className="flex-1 bg-red-600/20 hover:bg-red-600/30 text-red-400 hover:text-red-300 px-4 py-2.5 rounded-lg transition-all duration-200 border border-red-600/50 hover:border-red-500/50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {cancelling ? 'Cancelling...' : 'Cancel Scan'}
                    </button>
                )}
                <button className="flex-1 bg-accent hover:bg-slate-600/50 text-slate-300 hover:text-white px-4 py-2.5 rounded-lg transition-all duration-200 border border-border/50 hover:border-slate-500/50">
                    View Details
                </button>
                {status !== 'running' && (
                    <button className="flex-1 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2.5 rounded-lg transition-all duration-200 shadow-lg hover:shadow-xl">
                        {status === 'completed' ? 'New Scan' : status === 'idle' ? 'Start Scan' : 'Retry Scan'}
                    </button>
                )}
            </div>
        </motion.div>
    );
}