'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { Play, Clock, CheckCircle, AlertCircle, Calendar, Loader2, Trash2 } from 'lucide-react';
import { scansApi } from '@/services/scans.api';
import { format } from 'date-fns';

import { ScanConfigModal } from '@/components/scans/ScanConfigModal';

export default function ScansPage() {
    const [scans, setScans] = useState<any[]>([]);
    const [loading, setLoading] = useState(true);
    const [showScanConfigModal, setShowScanConfigModal] = useState(false);
    const [deletingId, setDeletingId] = useState<string | null>(null);

    const handleDeleteScan = async (e: React.MouseEvent, scanId: string) => {
        e.preventDefault();
        e.stopPropagation();
        if (!confirm('Delete this scan and all its findings?')) return;
        setDeletingId(scanId);
        try {
            await scansApi.deleteScan(scanId);
            setScans(prev => prev.filter(s => s.id !== scanId));
        } catch (error) {
            console.error('Failed to delete scan:', error);
        } finally {
            setDeletingId(null);
        }
    };

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

    const getDuration = (start: string, end?: string, status?: string) => {
        if (!start || start.startsWith('0001-01-01')) return '-';
        
        const startTime = new Date(start).getTime();
        let endTime = new Date().getTime();
        
        if (end && !end.startsWith('0001-01-01')) {
            endTime = new Date(end).getTime();
        } else if (status === 'completed' || status === 'failed') {
            return 'Unknown'; // Old scan missing end time
        }
        
        const diffSeconds = Math.floor((endTime - startTime) / 1000);
        if (diffSeconds < 60) return `${diffSeconds}s`;
        const minutes = Math.floor(diffSeconds / 60);
        const seconds = diffSeconds % 60;
        return `${minutes}m ${seconds}s`;
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
                    <h1 className="text-2xl font-bold text-slate-900">Scans</h1>
                    <p className="text-slate-500 mt-1">Manage and review PII detection scans.</p>
                </div>
                <button
                    onClick={() => setShowScanConfigModal(true)}
                    className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-medium transition-colors"
                >
                    <Play className="w-4 h-4" />
                    <span>New Scan</span>
                </button>
            </div>

            <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm">
                {loading ? (
                    <div className="flex items-center justify-center p-12 text-slate-500">
                        <Loader2 className="w-8 h-8 animate-spin mr-3" />
                        <span>Loading scan history...</span>
                    </div>
                ) : scans.length === 0 ? (
                    <div className="flex flex-col items-center justify-center p-12 text-slate-500">
                        <div className="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mb-4">
                            <Clock className="w-8 h-8 text-slate-400" />
                        </div>
                        <h3 className="text-lg font-medium text-slate-900 mb-1">No Scans Found</h3>
                        <p className="text-sm text-slate-500">Run your first scan to see results here.</p>
                    </div>
                ) : (
                    <table className="w-full text-left text-sm">
                        <thead>
                            <tr className="bg-slate-50 text-slate-600 border-b border-slate-200">
                                <th className="px-6 py-4 font-semibold">Scan Name</th>
                                <th className="px-6 py-4 font-semibold">Date</th>
                                <th className="px-6 py-4 font-semibold">Status</th>
                                <th className="px-6 py-4 font-semibold">Duration</th>
                                <th className="px-6 py-4 font-semibold text-right">Findings</th>
                                <th className="px-6 py-4 font-semibold w-12"></th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {scans.map((scan) => (
                                <tr
                                    key={scan.id}
                                    className="group hover:bg-slate-50 transition-colors cursor-pointer"
                                >
                                    <td className="px-6 py-4">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            <div className="font-semibold text-blue-600 group-hover:text-blue-700 transition-colors">
                                                {scan.profile_name || 'Unnamed Scan'}
                                            </div>
                                            <div className="text-xs text-slate-500 mt-0.5">{scan.id}</div>
                                        </Link>
                                    </td>
                                    <td className="px-6 py-4 text-slate-600">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            <div className="flex items-center gap-2">
                                                <Calendar className="w-4 h-4 text-slate-400" />
                                                {formatDate(scan.scan_started_at)}
                                            </div>
                                        </Link>
                                    </td>
                                    <td className="px-6 py-4">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            <div className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold border ${scan.status === 'completed'
                                                ? 'bg-green-50 text-green-700 border-green-200'
                                                : scan.status === 'failed'
                                                    ? 'bg-red-50 text-red-700 border-red-200'
                                                    : 'bg-blue-50 text-blue-700 border-blue-200'
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
                                    <td className="px-6 py-4 text-slate-600">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            <div className="flex items-center gap-2">
                                                <Clock className="w-4 h-4 text-slate-400" />
                                                {getDuration(scan.scan_started_at, scan.scan_completed_at, scan.status)}
                                            </div>
                                        </Link>
                                    </td>
                                    <td className="px-6 py-4 text-right font-mono text-slate-700">
                                        <Link href={`/scans/${scan.id}`} className="block">
                                            {scan.total_findings?.toLocaleString() || 0}
                                        </Link>
                                    </td>
                                    <td className="px-3 py-4 text-center">
                                        <button
                                            onClick={(e) => handleDeleteScan(e, scan.id)}
                                            disabled={deletingId === scan.id}
                                            className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50"
                                            title="Delete scan"
                                        >
                                            {deletingId === scan.id ? (
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                            ) : (
                                                <Trash2 className="w-4 h-4" />
                                            )}
                                        </button>
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
