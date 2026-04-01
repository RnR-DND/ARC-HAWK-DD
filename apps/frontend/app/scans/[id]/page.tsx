'use client';

import React, { useEffect, useState, use } from 'react';
import Link from 'next/link';
import { ArrowLeft, Calendar, Clock, Database, CheckCircle, XCircle, AlertTriangle, Loader2 } from 'lucide-react';
import { scansApi } from '@/services/scans.api';
import { format } from 'date-fns';

export default function ScanDetailPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = use(params);
    const [scan, setScan] = useState<any>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchScan = async () => {
            try {
                const data = await scansApi.getScan(id);
                setScan(data);
            } catch (err) {
                console.error('Failed to load scan details', err);
                setError('Failed to load scan details. The scan may not exist.');
            } finally {
                setLoading(false);
            }
        };

        if (id) {
            fetchScan();
        }
    }, [id]);

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

    if (loading) {
        return (
            <div className="flex items-center justify-center h-full bg-background text-muted-foreground">
                <Loader2 className="w-8 h-8 animate-spin mr-3" />
                <span>Loading scan details...</span>
            </div>
        );
    }

    if (error || !scan) {
        return (
            <div className="flex flex-col items-center justify-center h-full bg-background text-muted-foreground">
                <AlertTriangle className="w-12 h-12 text-red-500 mb-4" />
                <h3 className="text-xl font-semibold text-white mb-2">Error Loading Scan</h3>
                <p>{error || 'Scan not found'}</p>
                <Link href="/scans" className="mt-6 px-4 py-2 bg-muted rounded-lg hover:bg-accent transition-colors text-white">
                    Return to Scans
                </Link>
            </div>
        );
    }

    // Adapt metadata for display
    const piiSummary = scan.metadata?.pii_summary || []; // Assuming backend passes this structure, adaptable if needed

    return (
        <div className="flex flex-col h-full bg-slate-50">
            {/* Header */}
            <div className="bg-white border-b border-slate-200 px-8 py-6 shadow-sm">
                <div className="flex items-center gap-4 mb-4">
                    <Link
                        href="/scans"
                        className="p-2 -ml-2 text-slate-500 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        <ArrowLeft className="w-5 h-5" />
                    </Link>
                    <h1 className="text-2xl font-bold text-slate-900">{scan.profile_name || 'Unnamed Scan'}</h1>
                    <div className={`px-2 py-0.5 rounded text-xs font-semibold border ${scan.status === 'completed' ? 'bg-green-50 text-green-700 border-green-200' :
                            scan.status === 'failed' ? 'bg-red-50 text-red-700 border-red-200' :
                                'bg-blue-50 text-blue-700 border-blue-200'
                        }`}>
                        <span className="capitalize">{scan.status}</span>
                    </div>
                </div>

                <div className="flex items-center gap-8 text-sm text-slate-500">
                    <div className="flex items-center gap-2">
                        <span className="font-mono bg-slate-100 px-2 py-0.5 rounded text-slate-700 border border-slate-200">
                            {scan.id.substring(0, 8)}...
                        </span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Calendar className="w-4 h-4" />
                        <span>{formatDate(scan.scan_started_at)}</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Clock className="w-4 h-4" />
                        <span>{getDuration(scan.scan_started_at, scan.scan_completed_at, scan.status)}</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Database className="w-4 h-4" />
                        <span>Assets Scanned: {scan.total_assets}</span>
                    </div>
                </div>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-auto p-8">
                <div className="max-w-4xl mx-auto space-y-8">
                    {/* PII Detection Summary - Only show if we have summary data or mock integration if needed */}
                    {piiSummary.length > 0 ? (
                        <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm">
                            <div className="px-6 py-4 border-b border-slate-200 flex justify-between items-center">
                                <h2 className="text-lg font-semibold text-slate-800">PII Detection Summary</h2>
                                <span className="text-sm text-slate-500">
                                    Click a PII type to filter findings
                                </span>
                            </div>
                            <table className="w-full text-left text-sm">
                                <thead>
                                    <tr className="bg-slate-50 text-slate-500 border-b border-slate-200">
                                        <th className="px-6 py-3 font-medium">PII Type</th>
                                        <th className="px-6 py-3 font-medium text-right">Detected Count</th>
                                        <th className="px-6 py-3 font-medium">Status</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-100">
                                    {piiSummary.map((item: any) => (
                                        <tr
                                            key={item.type}
                                            className="hover:bg-slate-50 transition-colors cursor-pointer group"
                                        >
                                            <td className="px-6 py-4 font-medium text-slate-700">
                                                {item.type}
                                            </td>
                                            <td className="px-6 py-4 text-right font-mono text-slate-600">
                                                {item.count?.toLocaleString() || 0}
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2 text-emerald-600">
                                                    <CheckCircle className="w-4 h-4" />
                                                    <span className="font-medium">Detected</span>
                                                </div>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    ) : (
                        <div className="bg-white border border-slate-200 rounded-xl p-8 text-center shadow-sm">
                            <h3 className="text-lg font-medium text-slate-800 mb-2">No PII Summary Available</h3>
                            <p className="text-slate-500">This scan did not produce a detailed breakdown summary or no PII was found.</p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
