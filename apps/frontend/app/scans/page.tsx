'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { Play, Clock, CheckCircle, AlertCircle, Calendar, Loader2 } from 'lucide-react';
import { scansApi } from '@/services/scans.api';
import { format } from 'date-fns';

import { ScanConfigModal } from '@/components/scans/ScanConfigModal';

export default function ScansPage() {
    const [scans, setScans] = useState<any[]>([]);
    const [loading, setLoading] = useState(true);
    const [showScanConfigModal, setShowScanConfigModal] = useState(false);

    const fetchScans = async () => {
        try {
            const data = await scansApi.getScans();
            setScans(data);
        } catch (error) {
            console.error('Failed to load scans', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchScans();
    }, []);

    const formatDate = (dateString: string) => {
        try {
            return format(new Date(dateString), 'MMM d, yyyy h:mm a');
        } catch (e) {
            return dateString;
        }
    };

    const getDuration = (start: string, end?: string) => {
        if (!start) return '-';
        const startTime = new Date(start).getTime();
        const endTime = end ? new Date(end).getTime() : new Date().getTime();
        const minutes = Math.floor((endTime - startTime) / 60000);
        return `${minutes}m`;
    };

    return (
        <div className="p-8">
            <ScanConfigModal
                isOpen={showScanConfigModal}
                onClose={() => setShowScanConfigModal(false)}
                onRunScan={() => {
                    // Update list shortly after scan starts
                    setTimeout(fetchScans, 1000);
                }}
            />

            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-2xl font-bold text-white">Scans</h1>
                    <p className="text-muted-foreground mt-1">Manage and review PII detection scans.</p>
                </div>
                <button
                    onClick={() => setShowScanConfigModal(true)}
                    className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-medium transition-colors"
                >
                    <Play className="w-4 h-4" />
                    <span>New Scan</span>
                </button>
            </div>

            <div className="bg-card border border-border rounded-xl overflow-hidden shadow-sm">
                {loading ? (
                    <div className="flex items-center justify-center p-12 text-muted-foreground">
                        <Loader2 className="w-8 h-8 animate-spin mr-3" />
                        <span>Loading scan history...</span>
                    </div>
                ) : scans.length === 0 ? (
                    <div className="flex flex-col items-center justify-center p-12 text-muted-foreground">
                        <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mb-4">
                            <Clock className="w-8 h-8 text-slate-500" />
                        </div>
                        <h3 className="text-lg font-medium text-white mb-1">No Scans Found</h3>
                        <p className="text-sm">Run your first scan to see results here.</p>
                    </div>
                ) : (
                    <table className="w-full text-left text-sm">
                        <thead>
                            <tr className="bg-secondary text-muted-foreground border-b border-border">
                                <th className="px-6 py-4 font-medium">Scan Name</th>
                                <th className="px-6 py-4 font-medium">Date</th>
                                <th className="px-6 py-4 font-medium">Status</th>
                                <th className="px-6 py-4 font-medium">Duration</th>
                                <th className="px-6 py-4 font-medium text-right">Findings</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                            {scans.map((scan) => (
                                <tr
                                    key={scan.id}
                                    className="group hover:bg-secondary transition-colors cursor-pointer"
                                >
                                    <td className="px-6 py-4">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            <div className="font-semibold text-blue-400 group-hover:text-blue-300 transition-colors">
                                                {scan.profile_name || 'Unnamed Scan'}
                                            </div>
                                            <div className="text-xs text-slate-500 mt-0.5">{scan.id}</div>
                                        </Link>
                                    </td>
                                    <td className="px-6 py-4 text-slate-300">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            <div className="flex items-center gap-2">
                                                <Calendar className="w-4 h-4 text-slate-500" />
                                                {formatDate(scan.scan_started_at)}
                                            </div>
                                        </Link>
                                    </td>
                                    <td className="px-6 py-4">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            <div className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${scan.status === 'completed'
                                                ? 'bg-green-500/10 text-green-400 border-green-500/20'
                                                : scan.status === 'failed'
                                                    ? 'bg-red-500/10 text-red-400 border-red-500/20'
                                                    : 'bg-blue-500/10 text-blue-400 border-blue-500/20'
                                                }`}>
                                                {scan.status === 'completed' ? (
                                                    <CheckCircle className="w-3 h-3" />
                                                ) : scan.status === 'failed' ? (
                                                    <AlertCircle className="w-3 h-3" />
                                                ) : (
                                                    <Loader2 className="w-3 h-3 animate-spin" />
                                                )}
                                                <span className="capitalize">{scan.status}</span>
                                            </div>
                                        </Link>
                                    </td>
                                    <td className="px-6 py-4 text-slate-300">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            <div className="flex items-center gap-2">
                                                <Clock className="w-4 h-4 text-slate-500" />
                                                {getDuration(scan.scan_started_at, scan.scan_completed_at)}
                                            </div>
                                        </Link>
                                    </td>
                                    <td className="px-6 py-4 text-right font-mono text-slate-300">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            {scan.total_findings?.toLocaleString() || 0}
                                        </Link>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>
        </div>
    );
}
