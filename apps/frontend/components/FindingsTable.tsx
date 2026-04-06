'use client';

import React, { useState } from 'react';
import { FindingWithDetails } from '@/types';
import { findingsApi } from '@/services/findings.api';
import { FindingDetailDrawer } from './findings/FindingDetailDrawer';

interface FindingsTableProps {
    findings: FindingWithDetails[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
    onPageChange: (page: number) => void;
    onFilterChange: (filters: { severity?: string; search?: string }) => void;
    onRemediate?: (id: string, action: 'MASK' | 'DELETE') => void;
    onMarkFalsePositive?: (id: string) => Promise<void> | void;
}

export default function FindingsTable({
    findings,
    total,
    page,
    pageSize,
    totalPages,
    onPageChange,
    onFilterChange,
    onRemediate,
    onMarkFalsePositive
}: FindingsTableProps) {
    const [selectedFinding, setSelectedFinding] = useState<FindingWithDetails | null>(null);
    const [isDrawerOpen, setIsDrawerOpen] = useState(false);

    const handleRowClick = (finding: FindingWithDetails) => {
        setSelectedFinding(finding);
        setIsDrawerOpen(true);
    };

    const handleRemediate = (id: string, action: 'MASK' | 'DELETE') => {
        if (onRemediate) {
            onRemediate(id, action);
        }
    };

    const handleMarkFalsePositive = async (id: string) => {
        if (onMarkFalsePositive) {
            await onMarkFalsePositive(id);
        }
    };

    return (
        <div>
            <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                    <thead>
                        <tr className="bg-slate-50 text-slate-600 border-b border-slate-200">
                            <th className="px-4 py-3 font-semibold">Asset</th>
                            <th className="px-4 py-3 font-semibold">Object/Path</th>
                            <th className="px-4 py-3 font-semibold">Field</th>
                            <th className="px-4 py-3 font-semibold">PII Type</th>
                            <th className="px-4 py-3 font-semibold">Risk</th>
                            <th className="px-4 py-3 font-semibold">Conf</th>
                            <th className="px-4 py-3 font-semibold">Status</th>
                            <th className="px-4 py-3 font-semibold text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                        {findings.length === 0 ? (
                            <tr>
                                <td colSpan={8} className="text-center py-12 text-slate-500">
                                    No findings match the current filters
                                </td>
                            </tr>
                        ) : (
                            findings.map((finding) => {
                                const classification = finding.classifications[0];
                                const piiType = classification?.classification_type || 'Unknown';
                                const confidence = classification?.confidence_score || 0;

                                const fullPath = finding.asset_path || '';
                                const lastSeparatorIndex = Math.max(fullPath.lastIndexOf('/'), fullPath.lastIndexOf('.'));
                                const path = lastSeparatorIndex > -1 ? fullPath.substring(0, lastSeparatorIndex) : 'Root';
                                const field = lastSeparatorIndex > -1 ? fullPath.substring(lastSeparatorIndex + 1) : fullPath;

                                return (
                                    <tr
                                        key={finding.id}
                                        onClick={() => handleRowClick(finding)}
                                        className="hover:bg-slate-50 cursor-pointer transition-colors group"
                                    >
                                        <td className="px-4 py-3 font-medium text-slate-900">
                                            {finding.asset_name}
                                        </td>
                                        <td className="px-4 py-3 text-slate-500 text-xs font-mono truncate max-w-[150px]" title={path}>
                                            {path}
                                        </td>
                                        <td className="px-4 py-3 text-blue-600 text-xs font-mono font-medium">
                                            {field}
                                        </td>
                                        <td className="px-4 py-3 text-slate-600">
                                            {piiType}
                                        </td>
                                        <td className="px-4 py-3">
                                            <span className={`
                                                px-2 py-0.5 rounded text-xs font-bold border
                                                ${finding.severity === 'Critical' ? 'bg-red-50 text-red-700 border-red-200' : ''}
                                                ${finding.severity === 'High' ? 'bg-orange-50 text-orange-700 border-orange-200' : ''}
                                                ${finding.severity === 'Medium' ? 'bg-yellow-50 text-yellow-700 border-yellow-200' : ''}
                                                ${finding.severity === 'Low' ? 'bg-emerald-50 text-emerald-700 border-emerald-200' : ''}
                                            `}>
                                                {finding.severity}
                                            </span>
                                        </td>
                                        <td className="px-4 py-3 font-mono text-xs text-slate-600">
                                            {(confidence * 100).toFixed(0)}%
                                        </td>
                                        <td className="px-4 py-3">
                                            <span className="px-2 py-0.5 rounded text-xs font-semibold bg-green-50 text-green-700 border border-green-200">
                                                Active
                                            </span>
                                        </td>
                                        <td className="px-4 py-3 text-right">
                                            <div className="flex items-center justify-end gap-2">
                                                <button
                                                    onClick={(e) => { e.stopPropagation(); }}
                                                    className="px-2 py-1 text-xs font-medium text-slate-600 hover:text-slate-900 bg-slate-100 rounded border border-slate-200 hover:border-slate-300 transition-colors"
                                                >
                                                    Lineage
                                                </button>
                                                <button
                                                    onClick={(e) => { e.stopPropagation(); handleRemediate(finding.id, 'MASK'); }}
                                                    className="px-2 py-1 text-xs font-medium text-blue-600 hover:text-blue-700 bg-blue-50 hover:bg-blue-100 rounded border border-blue-200 transition-colors"
                                                >
                                                    Mask
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                );
                            })
                        )}
                    </tbody>
                </table>
            </div>

            {totalPages > 1 && (
                <div className="border-t border-slate-200 px-4 py-3 flex items-center justify-between">
                    <span className="text-sm text-slate-500">
                        Showing {((page - 1) * pageSize) + 1}-{Math.min(page * pageSize, total)} of {total.toLocaleString()} findings
                    </span>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={() => onPageChange(page - 1)}
                            disabled={page <= 1}
                            className={`px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
                                page <= 1
                                    ? 'bg-slate-50 text-slate-400 border-slate-200 cursor-not-allowed'
                                    : 'bg-white text-slate-700 border-slate-200 hover:bg-slate-50'
                            }`}
                        >
                            Previous
                        </button>
                        <span className="text-sm text-slate-600 px-2">
                            Page {page} of {totalPages}
                        </span>
                        <button
                            onClick={() => onPageChange(page + 1)}
                            disabled={page >= totalPages}
                            className={`px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
                                page >= totalPages
                                    ? 'bg-slate-50 text-slate-400 border-slate-200 cursor-not-allowed'
                                    : 'bg-white text-slate-700 border-slate-200 hover:bg-slate-50'
                            }`}
                        >
                            Next
                        </button>
                    </div>
                </div>
            )}

            <FindingDetailDrawer
                finding={selectedFinding}
                isOpen={isDrawerOpen}
                onClose={() => setIsDrawerOpen(false)}
                onMarkFalsePositive={handleMarkFalsePositive}
                onRemediate={handleRemediate}
            />
        </div>
    );
}
