'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Shield, Search, RefreshCw, Clock, User, FileText, Filter } from 'lucide-react';
import { auditApi, type AuditLogEntry } from '@/services/audit.api';

const ACTION_COLORS: Record<string, string> = {
    LOGIN_SUCCESS:         'bg-green-100 text-green-800',
    LOGIN_FAILED:          'bg-red-100 text-red-800',
    USER_REGISTERED:       'bg-blue-100 text-blue-800',
    PASSWORD_CHANGED:      'bg-yellow-100 text-yellow-800',
    REMEDIATION_EXECUTED:  'bg-purple-100 text-purple-800',
    REMEDIATION_ROLLED_BACK: 'bg-orange-100 text-orange-800',
    PATTERN_CREATED:       'bg-teal-100 text-teal-800',
    PATTERN_UPDATED:       'bg-cyan-100 text-cyan-800',
    PATTERN_DELETED:       'bg-red-100 text-red-800',
    ASSET_ACCESSED:        'bg-slate-100 text-slate-700',
    ASSET_CREATED:         'bg-blue-100 text-blue-800',
    ASSET_DELETED:         'bg-red-100 text-red-800',
};

function actionBadge(action: string) {
    const cls = ACTION_COLORS[action] ?? 'bg-slate-100 text-slate-700';
    return (
        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
            {action}
        </span>
    );
}

function formatTime(ts: string) {
    try {
        return new Date(ts).toLocaleString();
    } catch {
        return ts;
    }
}

export default function AuditPage() {
    const [logs, setLogs] = useState<AuditLogEntry[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [search, setSearch] = useState('');
    const [actionFilter, setActionFilter] = useState('');
    const [resourceFilter, setResourceFilter] = useState('');

    const loadLogs = useCallback(async () => {
        try {
            setLoading(true);
            setError(null);
            const data = await auditApi.getLogs({ limit: 200 });
            setLogs(data);
        } catch (err) {
            console.error('Failed to load audit logs:', err);
            setError('Failed to load audit logs');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadLogs();
    }, [loadLogs]);

    const filtered = logs.filter(log => {
        const searchLower = search.toLowerCase();
        const matchSearch = !search ||
            log.action.toLowerCase().includes(searchLower) ||
            log.resource_type.toLowerCase().includes(searchLower) ||
            log.resource_id.toLowerCase().includes(searchLower) ||
            (log.user_id || '').toLowerCase().includes(searchLower);
        const matchAction = !actionFilter || log.action === actionFilter;
        const matchResource = !resourceFilter || log.resource_type === resourceFilter;
        return matchSearch && matchAction && matchResource;
    });

    const uniqueActions = Array.from(new Set(logs.map(l => l.action))).sort();
    const uniqueResources = Array.from(new Set(logs.map(l => l.resource_type))).sort();

    return (
        <div className="p-8 space-y-6 bg-white min-h-screen">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-3">
                        <Shield className="w-6 h-6 text-slate-600" />
                        Audit Logs
                    </h1>
                    <p className="text-slate-500 mt-1 text-sm">
                        Tamper-evident hash-chained event log. Every action recorded here is cryptographically linked to its predecessor.
                    </p>
                </div>
                <button
                    onClick={loadLogs}
                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50"
                >
                    <RefreshCw className="w-4 h-4" />
                    Refresh
                </button>
            </div>

            {/* Filters */}
            <div className="flex flex-wrap gap-3">
                <div className="relative flex-1 min-w-48">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                    <input
                        type="text"
                        placeholder="Search by action, resource, user..."
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                        className="w-full pl-9 pr-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900 bg-white"
                    />
                </div>
                <div className="flex items-center gap-2">
                    <Filter className="w-4 h-4 text-slate-400" />
                    <select
                        value={actionFilter}
                        onChange={e => setActionFilter(e.target.value)}
                        className="text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900 bg-white"
                    >
                        <option value="">All actions</option>
                        {uniqueActions.map(a => (
                            <option key={a} value={a}>{a}</option>
                        ))}
                    </select>
                    <select
                        value={resourceFilter}
                        onChange={e => setResourceFilter(e.target.value)}
                        className="text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900 bg-white"
                    >
                        <option value="">All resources</option>
                        {uniqueResources.map(r => (
                            <option key={r} value={r}>{r}</option>
                        ))}
                    </select>
                </div>
            </div>

            {/* Stats bar */}
            <div className="flex gap-6 text-sm text-slate-500">
                <span>{logs.length} total events</span>
                <span>{filtered.length} shown</span>
            </div>

            {/* Table */}
            {loading ? (
                <div className="space-y-3">
                    {[...Array(8)].map((_, i) => (
                        <div key={i} className="h-12 bg-slate-100 rounded-lg animate-pulse" />
                    ))}
                </div>
            ) : error ? (
                <div className="p-6 text-center text-red-600 bg-red-50 rounded-lg border border-red-100">
                    {error}
                </div>
            ) : filtered.length === 0 ? (
                <div className="p-12 text-center text-slate-400 bg-slate-50 rounded-lg border border-slate-100">
                    No audit log entries match your filters.
                </div>
            ) : (
                <div className="border border-slate-200 rounded-lg overflow-hidden">
                    <table className="w-full text-sm">
                        <thead className="bg-slate-50 border-b border-slate-200">
                            <tr>
                                <th className="text-left px-4 py-3 font-medium text-slate-600 w-40">
                                    <span className="flex items-center gap-1"><Clock className="w-3.5 h-3.5" />Time</span>
                                </th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Action</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">
                                    <span className="flex items-center gap-1"><FileText className="w-3.5 h-3.5" />Resource</span>
                                </th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">
                                    <span className="flex items-center gap-1"><User className="w-3.5 h-3.5" />User</span>
                                </th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Details</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {filtered.map(log => (
                                <tr key={log.id} className="hover:bg-slate-50 transition-colors">
                                    <td className="px-4 py-3 text-slate-500 whitespace-nowrap font-mono text-xs">
                                        {formatTime(log.event_time)}
                                    </td>
                                    <td className="px-4 py-3">
                                        {actionBadge(log.action)}
                                    </td>
                                    <td className="px-4 py-3">
                                        <span className="text-slate-700 font-medium">{log.resource_type}</span>
                                        <span className="text-slate-400 ml-2 font-mono text-xs truncate max-w-32 inline-block align-bottom">
                                            {log.resource_id}
                                        </span>
                                    </td>
                                    <td className="px-4 py-3 text-slate-600 font-mono text-xs truncate max-w-40">
                                        {log.user_id || '—'}
                                    </td>
                                    <td className="px-4 py-3 text-slate-400 text-xs">
                                        {log.ip_address && (
                                            <span className="mr-3">IP: {log.ip_address}</span>
                                        )}
                                        {log.metadata && Object.keys(log.metadata).length > 0 && (
                                            <details className="inline">
                                                <summary className="cursor-pointer text-blue-500 hover:text-blue-700">
                                                    metadata
                                                </summary>
                                                <pre className="mt-1 text-xs bg-slate-100 rounded p-2 max-w-xs overflow-auto text-slate-700">
                                                    {JSON.stringify(log.metadata, null, 2)}
                                                </pre>
                                            </details>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}
