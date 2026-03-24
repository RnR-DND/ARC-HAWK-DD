'use client';

import React, { useState } from 'react';
import { FindingWithDetails, SignalScore } from '@/types';
import { findingsApi } from '@/services/findings.api';
import { theme, getRiskColor } from '@/design-system/theme';
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
            // Close drawer if action was regarding the selected finding
            // setIsDrawerOpen(false); // Optional: keep open or close? Keep open to see result usually better but if list updates finding might be gone.
        }
    };

    return (
        <div>
            {/* Table Header Only - Filters are in Page */}
            <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                    <thead>
                        <tr className="bg-secondary text-muted-foreground border-b border-border">
                            <th className="px-4 py-3 font-medium">Asset</th>
                            <th className="px-4 py-3 font-medium">Object/Path</th>
                            <th className="px-4 py-3 font-medium">Field</th>
                            <th className="px-4 py-3 font-medium">PII Type</th>
                            <th className="px-4 py-3 font-medium">Risk</th>
                            <th className="px-4 py-3 font-medium">Conf</th>
                            <th className="px-4 py-3 font-medium">Status</th>
                            <th className="px-4 py-3 font-medium text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                        {findings.length === 0 ? (
                            <tr>
                                <td colSpan={8} className="text-center py-12 text-muted-foreground">
                                    No findings match the current filters
                                </td>
                            </tr>
                        ) : (
                            findings.map((finding) => {
                                const classification = finding.classifications[0];
                                const piiType = classification?.classification_type || 'Unknown';
                                const confidence = classification?.confidence_score || 0;

                                // Logic to split path and field
                                const fullPath = finding.asset_path || '';
                                const lastSeparatorIndex = Math.max(fullPath.lastIndexOf('/'), fullPath.lastIndexOf('.'));
                                const path = lastSeparatorIndex > -1 ? fullPath.substring(0, lastSeparatorIndex) : 'Root';
                                const field = lastSeparatorIndex > -1 ? fullPath.substring(lastSeparatorIndex + 1) : fullPath;

                                return (
                                    <tr
                                        key={finding.id}
                                        onClick={() => handleRowClick(finding)}
                                        className="hover:bg-muted cursor-pointer transition-colors group"
                                    >
                                        <td className="px-4 py-3 font-medium text-foreground">
                                            {finding.asset_name}
                                        </td>
                                        <td className="px-4 py-3 text-muted-foreground text-xs font-mono truncate max-w-[150px]" title={path}>
                                            {path}
                                        </td>
                                        <td className="px-4 py-3 text-blue-600 text-xs font-mono font-medium">
                                            {field}
                                        </td>
                                        <td className="px-4 py-3 text-muted-foreground">
                                            {piiType}
                                        </td>
                                        <td className="px-4 py-3">
                                            <span className={`
                                                px-2 py-0.5 rounded text-xs font-bold border
                                                ${finding.severity === 'Critical' ? 'bg-red-500/10 text-red-500 border-red-500/20' : ''}
                                                ${finding.severity === 'High' ? 'bg-orange-500/10 text-orange-500 border-orange-500/20' : ''}
                                                ${finding.severity === 'Medium' ? 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20' : ''}
                                                ${finding.severity === 'Low' ? 'bg-blue-500/10 text-blue-500 border-blue-500/20' : ''}
                                            `}>
                                                {finding.severity}
                                            </span>
                                        </td>
                                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                                            {(confidence * 100).toFixed(0)}%
                                        </td>
                                        <td className="px-4 py-3">
                                            <span className="px-2 py-0.5 rounded text-xs font-medium bg-green-500/10 text-green-400 border border-green-500/20">
                                                Active
                                            </span>
                                        </td>
                                        <td className="px-4 py-3 text-right">
                                            <div className="flex items-center justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                                <button
                                                    onClick={(e) => { e.stopPropagation(); /* View lineage logic */ }}
                                                    className="px-2 py-1 text-xs font-medium text-muted-foreground hover:text-foreground bg-muted rounded border border-border hover:border-slate-400 transition-colors"
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
                <div style={{ borderTop: `1px solid ${theme.colors.border.default}`, paddingTop: 16, display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
                    <button
                        onClick={() => onPageChange(page - 1)}
                        disabled={page <= 1}
                        style={{
                            border: `1px solid ${theme.colors.border.default}`,
                            borderRadius: 4,
                            padding: '6px 12px',
                            background: page <= 1 ? theme.colors.background.secondary : theme.colors.background.card,
                            color: page <= 1 ? theme.colors.text.muted : theme.colors.text.primary
                        }}
                    >
                        Previous
                    </button>
                    <span style={{ fontSize: 14, color: theme.colors.text.secondary, alignSelf: 'center' }}>
                        Page {page} of {totalPages}
                    </span>
                    <button
                        onClick={() => onPageChange(page + 1)}
                        disabled={page >= totalPages}
                        style={{
                            border: `1px solid ${theme.colors.border.default}`,
                            borderRadius: 4,
                            padding: '6px 12px',
                            background: page >= totalPages ? theme.colors.background.secondary : theme.colors.background.card,
                            color: page >= totalPages ? theme.colors.text.muted : theme.colors.text.primary
                        }}
                    >
                        Next
                    </button>
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
