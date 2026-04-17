'use client';

import React, { useState, useEffect } from 'react';
import { History, Shield, EyeOff, Trash2, CheckCircle, RotateCcw } from 'lucide-react';
import { remediationApi, type RemediationEvent } from '@/services/remediation.api';

export default function HistoryPage() {
    const [history, setHistory] = useState<RemediationEvent[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [rollbackConfirmId, setRollbackConfirmId] = useState<string | null>(null);
    const [rollingBackId, setRollingBackId] = useState<string | null>(null);

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

    const handleRollback = async (id: string) => {
        setRollingBackId(id);
        setRollbackConfirmId(null);
        try {
            await remediationApi.rollback(id);
            await loadHistory();
        } catch (err) {
            console.error('Failed to rollback:', err);
        } finally {
            setRollingBackId(null);
        }
    };

    if (loading) {
        return (
            <div className="p-8 space-y-6">
                <div className="flex items-center justify-between">
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-3">
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
                        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-3">
                            <History className="w-6 h-6 text-muted-foreground" />
                            Remediation History
                        </h1>
                        <p className="text-muted-foreground mt-1">Audit log of all remediation actions and policy enforcements.</p>
                    </div>
                </div>
                <div className="bg-card border border-border rounded-xl p-12 text-center">
                    <div className="text-red-600 mb-4">{error}</div>
                    <button
                        onClick={loadHistory}
                        className="px-4 py-2 bg-slate-100 hover:bg-slate-200 rounded text-slate-700"
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
                    <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-3">
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
                            <th className="px-6 py-4 font-medium"></th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                        {history.length === 0 ? (
                            <tr>
                                <td colSpan={7} className="px-6 py-12 text-center text-muted-foreground">
                                    No remediation actions found
                                </td>
                            </tr>
                        ) : (
                            history.map((event) => (
                                <React.Fragment key={event.id}>
                                    <tr className="hover:bg-muted transition-colors">
                                        <td className="px-6 py-4 text-slate-500 font-mono text-xs">
                                            {new Date(event.executed_at).toLocaleString()}
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-2">
                                                {event.action === 'MASK' ? (
                                                    <div className="p-1 rounded bg-blue-50 text-blue-600">
                                                        <EyeOff className="w-4 h-4" />
                                                    </div>
                                                ) : (
                                                    <div className="p-1 rounded bg-red-50 text-red-600">
                                                        <Trash2 className="w-4 h-4" />
                                                    </div>
                                                )}
                                                <span className={`font-medium ${event.action === 'DELETE' ? 'text-red-600' : 'text-blue-600'}`}>
                                                    {event.action}
                                                </span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 text-slate-700">
                                            {event.target}
                                        </td>
                                        <td className="px-6 py-4 text-muted-foreground">
                                            {event.executed_by}
                                        </td>
                                        <td className="px-6 py-4">
                                            {event.scan_id ? (
                                                <span className="px-2 py-1 rounded bg-slate-100 text-slate-600 border border-slate-200 text-xs font-mono">
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
                                        <td className="px-6 py-4">
                                            {event.status === 'COMPLETED' && (
                                                <button
                                                    onClick={() => setRollbackConfirmId(event.id)}
                                                    disabled={rollingBackId === event.id}
                                                    className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-600 bg-white border border-slate-200 rounded-md hover:bg-slate-50 hover:border-slate-300 disabled:opacity-50"
                                                >
                                                    {rollingBackId === event.id ? (
                                                        <RotateCcw className="w-3 h-3 animate-spin" />
                                                    ) : (
                                                        <RotateCcw className="w-3 h-3" />
                                                    )}
                                                    Rollback
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                    {rollbackConfirmId === event.id && (
                                        <tr>
                                            <td colSpan={7} className="px-6 py-3 bg-amber-50 border-t border-amber-100">
                                                <div className="flex items-center gap-4 text-sm">
                                                    <RotateCcw className="w-4 h-4 text-amber-500 shrink-0" />
                                                    <span className="text-amber-700 font-medium flex-1">
                                                        Roll back this action? This will attempt to reverse the changes.
                                                    </span>
                                                    <button
                                                        onClick={() => setRollbackConfirmId(null)}
                                                        className="px-3 py-1.5 text-sm text-slate-600 bg-white border border-slate-200 rounded-lg hover:bg-slate-50"
                                                    >
                                                        Cancel
                                                    </button>
                                                    <button
                                                        onClick={() => handleRollback(event.id)}
                                                        className="px-3 py-1.5 text-sm text-white bg-amber-600 rounded-lg hover:bg-amber-700"
                                                    >
                                                        Confirm Rollback
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    )}
                                </React.Fragment>
                            ))
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
