'use client';

import React, { useState, useEffect } from 'react';
import { History, Shield, EyeOff, Trash2, CheckCircle, RotateCcw } from 'lucide-react';
import { remediationApi, type RemediationEvent } from '@/services/remediation.api';

export default function HistoryPage() {
    const [history, setHistory] = useState<RemediationEvent[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        loadHistory();
    }, []);

    const loadHistory = async () => {
        try {
            setLoading(true);
            setError(null);
            const data = await remediationApi.getRemediationHistory({ limit: 50 });
            setHistory(data.history || []);
        } catch (err) {
            console.error('Failed to load remediation history:', err);
            setError('Failed to load remediation history');
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return (
            <div className="p-8 space-y-6">
                <div className="flex items-center justify-between">
                    <div>
                        <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                            <History className="w-6 h-6 text-muted-foreground" />
                            Remediation History
                        </h1>
                        <p className="text-muted-foreground mt-1">Audit log of all remediation actions and policy enforcements.</p>
                    </div>
                </div>
                <div className="bg-card border border-border rounded-xl p-12 text-center">
                    <div className="text-muted-foreground">Loading remediation history...</div>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-8 space-y-6">
                <div className="flex items-center justify-between">
                    <div>
                        <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                            <History className="w-6 h-6 text-muted-foreground" />
                            Remediation History
                        </h1>
                        <p className="text-muted-foreground mt-1">Audit log of all remediation actions and policy enforcements.</p>
                    </div>
                </div>
                <div className="bg-card border border-border rounded-xl p-12 text-center">
                    <div className="text-red-400 mb-4">{error}</div>
                    <button
                        onClick={loadHistory}
                        className="px-4 py-2 bg-muted hover:bg-accent rounded text-white"
                    >
                        Retry
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="p-8 space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <History className="w-6 h-6 text-muted-foreground" />
                        Remediation History
                    </h1>
                    <p className="text-muted-foreground mt-1">Audit log of all remediation actions and policy enforcements.</p>
                </div>
            </div>

            <div className="bg-card border border-border rounded-xl overflow-hidden shadow-sm">
                <table className="w-full text-left text-sm">
                    <thead>
                        <tr className="bg-secondary text-muted-foreground border-b border-border">
                            <th className="px-6 py-4 font-medium">Date</th>
                            <th className="px-6 py-4 font-medium">Action</th>
                            <th className="px-6 py-4 font-medium">Target Asset</th>
                            <th className="px-6 py-4 font-medium">Executed By</th>
                            <th className="px-6 py-4 font-medium">Scan Context</th>
                            <th className="px-6 py-4 font-medium">Status</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                        {history.length === 0 ? (
                            <tr>
                                <td colSpan={6} className="px-6 py-12 text-center text-muted-foreground">
                                    No remediation actions found
                                </td>
                            </tr>
                        ) : (
                            history.map((event) => (
                                <tr key={event.id} className="hover:bg-muted transition-colors">
                                    <td className="px-6 py-4 text-slate-300 font-mono text-xs">
                                        {new Date(event.executed_at).toLocaleString()}
                                    </td>
                                    <td className="px-6 py-4">
                                        <div className="flex items-center gap-2">
                                            {event.action === 'MASK' ? (
                                                <div className="p-1 rounded bg-blue-500/10 text-blue-400">
                                                    <EyeOff className="w-4 h-4" />
                                                </div>
                                            ) : (
                                                <div className="p-1 rounded bg-red-500/10 text-red-400">
                                                    <Trash2 className="w-4 h-4" />
                                                </div>
                                            )}
                                            <span className={`font-medium ${event.action === 'DELETE' ? 'text-red-400' : 'text-blue-400'}`}>
                                                {event.action}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 text-slate-200">
                                        {event.target}
                                    </td>
                                    <td className="px-6 py-4 text-muted-foreground">
                                        {event.executed_by}
                                    </td>
                                    <td className="px-6 py-4">
                                        {event.scan_id ? (
                                            <span className="px-2 py-1 rounded bg-muted text-slate-300 border border-border text-xs font-mono">
                                                {event.scan_id}
                                            </span>
                                        ) : (
                                            <span className="text-slate-500 text-xs">N/A</span>
                                        )}
                                    </td>
                                    <td className="px-6 py-4">
                                        {event.status === 'ROLLED_BACK' ? (
                                            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-amber-500/10 text-amber-500 border border-amber-500/20">
                                                <RotateCcw className="w-3 h-3" />
                                                Rolled Back
                                            </span>
                                        ) : event.status === 'FAILED' ? (
                                            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-red-500/10 text-red-500 border border-red-500/20">
                                                Failed
                                            </span>
                                        ) : (
                                            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-green-500/10 text-green-500 border border-green-500/20">
                                                <CheckCircle className="w-3 h-3" />
                                                Applied
                                            </span>
                                        )}
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
