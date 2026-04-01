import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { Clock, CheckCircle, Play, Pause, AlertCircle } from 'lucide-react';

interface ScanStatusCardProps {
    scanId: string | null;
}

const scanStatuses = {
    'idle': { label: 'Idle', color: 'text-slate-500', bg: 'bg-slate-100', ring: 'ring-slate-200', icon: Pause },
    'running': { label: 'Running', color: 'text-blue-600', bg: 'bg-blue-100', ring: 'ring-blue-200', icon: Play },
    'completed': { label: 'Completed', color: 'text-emerald-600', bg: 'bg-emerald-100', ring: 'ring-emerald-200', icon: CheckCircle },
    'failed': { label: 'Failed', color: 'text-red-600', bg: 'bg-red-100', ring: 'ring-red-200', icon: AlertCircle },
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

    const startTime = scanData?.created_at && !scanData.created_at.startsWith('0001-01-01') ? new Date(scanData.created_at) : null;
    const endTime = scanData?.completed_at && !scanData.completed_at.startsWith('0001-01-01') ? new Date(scanData.completed_at) : null;
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
            case 'idle': return 'text-slate-500';
            case 'completed': return 'text-emerald-600';
            case 'running': return 'text-blue-600';
            case 'failed': return 'text-red-600';
        }
        return 'text-slate-500';
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
            className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm"
        >
            <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-4">
                    <div className={`p-3 rounded-lg ${scanStatuses[status as keyof typeof scanStatuses].bg} ring-1 ${scanStatuses[status as keyof typeof scanStatuses].ring}`}>
                        <StatusIcon className={`w-6 h-6 ${scanStatuses[status as keyof typeof scanStatuses].color}`} />
                    </div>
                    <div>
                        <h3 className="text-lg font-bold text-slate-800">Scan Status</h3>
                        <p className={`text-sm font-medium ${getStatusColor()}`}>
                            {scanStatuses[status as keyof typeof scanStatuses].label}
                        </p>
                        <p className="text-slate-500 text-sm mt-1">
                            {getStatusDescription()}
                        </p>
                    </div>
                </div>

                {scanId && (
                    <div className="text-right">
                        <div className="text-xs text-slate-500 mb-1">Latest Scan</div>
                        <div className="text-sm font-mono text-slate-600 bg-slate-100 px-2 py-1 rounded border border-slate-200">
                            {scanId}
                        </div>
                    </div>
                )}
            </div>

            {/* Progress Bar */}
            <div className="mb-4">
                <div className="flex items-center justify-between mb-2">
                    <span className="text-sm text-slate-500 font-medium">Progress</span>
                    <span className="text-sm text-slate-700 font-bold">{progress}%</span>
                </div>
                <div className="w-full bg-slate-100 rounded-full h-2">
                    <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${progress}%` }}
                        transition={{ duration: 1, ease: "easeOut" }}
                        className={`h-2 rounded-full ${status === 'completed' ? 'bg-emerald-500' : status === 'running' ? 'bg-blue-500' : 'bg-slate-400'
                            }`}
                    />
                </div>
            </div>

            {/* Time Information */}
            <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                    <div className="text-slate-500 mb-1">Started</div>
                    <div className="text-slate-700 font-medium flex items-center gap-2">
                        <Clock className="w-4 h-4 text-slate-400" />
                        {startTime ? startTime.toLocaleTimeString() : 'N/A'}
                    </div>
                </div>

                {endTime && (
                    <div>
                        <div className="text-slate-500 mb-1">
                            {status === 'completed' ? 'Completed' : 'Failed'}
                        </div>
                        <div className="text-slate-700 font-medium flex items-center gap-2">
                            <CheckCircle className="w-4 h-4 text-emerald-500" />
                            {endTime ? endTime.toLocaleTimeString() : 'N/A'}
                        </div>
                    </div>
                )}

                {status === 'completed' && (
                    <div>
                        <div className="text-slate-500 mb-1">Duration</div>
                        <div className="text-slate-700 font-medium">
                            {endTime && startTime ? (
                                (endTime.getTime() - startTime.getTime()) / 1000 < 60 
                                    ? `${Math.floor((endTime.getTime() - startTime.getTime()) / 1000)}s total`
                                    : `${Math.floor((endTime.getTime() - startTime.getTime()) / 1000 / 60)}m ${Math.floor(((endTime.getTime() - startTime.getTime()) / 1000) % 60)}s total`
                            ) : 'N/A'}
                        </div>
                    </div>
                )}
            </div>

            {/* Recommended Actions */}
            <div className="mb-6 mt-6">
                <h4 className="text-sm font-semibold text-slate-800 mb-3">Recommended Next Steps</h4>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    {recommendedActions.map((action, index) => (
                        <motion.button
                            key={action.label}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: index * 0.1 }}
                            className={`p-3 rounded-lg border transition-all duration-200 text-left group ${action.priority === 'high'
                                ? 'bg-blue-50 border-blue-200 hover:bg-blue-100 hover:border-blue-300'
                                : action.priority === 'medium'
                                    ? 'bg-white border-slate-200 hover:bg-slate-50 hover:border-slate-300'
                                    : 'bg-slate-50 border-slate-200 hover:bg-slate-100'
                                }`}
                            title={action.description}
                        >
                            <div className={`text-sm font-semibold mb-1 ${action.priority === 'high' ? 'text-blue-700' : 'text-slate-700'
                                }`}>
                                {action.label}
                            </div>
                            <div className="text-xs text-slate-500 group-hover:text-slate-700 transition-colors">
                                {action.description}
                            </div>
                            {action.priority === 'high' && (
                                <div className="mt-2">
                                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700 border border-blue-200">
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
                        className="flex-1 bg-red-50 hover:bg-red-100 text-red-600 hover:text-red-700 px-4 py-2.5 rounded-lg transition-all duration-200 border border-red-200 hover:border-red-300 disabled:opacity-50 disabled:cursor-not-allowed font-medium"
                    >
                        {cancelling ? 'Cancelling...' : 'Cancel Scan'}
                    </button>
                )}
                <button className="flex-1 bg-white hover:bg-slate-50 text-slate-700 hover:text-slate-900 px-4 py-2.5 rounded-lg transition-all duration-200 border border-slate-200 hover:border-slate-300 font-medium shadow-sm">
                    View Details
                </button>
                {status !== 'running' && (
                    <button className="flex-1 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2.5 rounded-lg transition-all duration-200 shadow hover:shadow-lg font-medium">
                        {status === 'completed' ? 'New Scan' : status === 'idle' ? 'Start Scan' : 'Retry Scan'}
                    </button>
                )}
            </div>
        </motion.div>
    );
}