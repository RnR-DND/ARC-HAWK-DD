'use client';

import React, { useEffect, useState } from 'react';
import Topbar from '@/components/Topbar';
import FindingsTable from '@/components/FindingsTable';
import LoadingState from '@/components/LoadingState';
import { findingsApi } from '@/services/findings.api';
import { theme } from '@/design-system/theme';
import type { FindingWithDetails, FindingsResponse } from '@/types';
import { RemediationConfirmationModal } from '@/components/remediation/RemediationConfirmationModal';
import { remediationApi } from '@/services/remediation.api';

export default function FindingsPage() {
    const [findingsData, setFindingsData] = useState<FindingsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Filter state
    const [page, setPage] = useState(1);
    const [searchTerm, setSearchTerm] = useState('');
    const [severityFilter, setSeverityFilter] = useState('');
    const [statusFilter, setStatusFilter] = useState('');
    const [assetFilter, setAssetFilter] = useState('');
    const [piiTypeFilter, setPiiTypeFilter] = useState('');

    useEffect(() => {
        fetchFindings();
    }, [page, searchTerm, severityFilter, statusFilter, assetFilter, piiTypeFilter]);


    const fetchFindings = async () => {
        try {
            setLoading(true);
            setError(null);

            // Note: The current API service might need updating if it doesn't support all filters
            // But for now we use what we have
            const result = await findingsApi.getFindings({
                page,
                page_size: 20,
                severity: severityFilter || undefined,
                status: statusFilter || undefined,
                asset: assetFilter || undefined,
                pii_type: piiTypeFilter || undefined,
                search: searchTerm || undefined
            });

            // Explode findings: One row per match
            const explodedFindings: FindingWithDetails[] = [];

            result.findings.forEach(finding => {
                if (finding.matches && finding.matches.length > 0) {
                    finding.matches.forEach((match: string) => {
                        // Create a clone of the finding for this specific match
                        explodedFindings.push({
                            ...finding,
                            // Override matches to display only this specific match
                            matches: [match],
                            // Update ID to be unique for React keys (optional but good practice)
                            id: `${finding.id}-${match}`
                        });
                    });
                } else {
                    // Fallback if no matches array
                    explodedFindings.push(finding);
                }
            });

            // Polyfill legacy pagination structure
            const data: FindingsResponse = {
                findings: explodedFindings,
                total: result.total, // Total findings count remains (or should update?) 
                // Actually total should reflect number of rows, but backend pagination is by finding.
                // It's acceptable to show exploded rows for the current page.
                page: page,
                page_size: 20,
                total_pages: Math.ceil(result.total / 20)
            };

            setFindingsData(data);
        } catch (err: any) {
            console.error('Error fetching findings:', err);
            setError('Failed to load findings. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    const handleFilterChange = (filters: { severity?: string; search?: string }) => {
        if (filters.search !== undefined) setSearchTerm(filters.search);
        if (filters.severity !== undefined) setSeverityFilter(filters.severity);
        setPage(1); // Reset to first page on filter change
    };

    const [remediationState, setRemediationState] = useState<{
        isOpen: boolean;
        findingId: string | null;
        action: 'MASK' | 'DELETE';
    }>({
        isOpen: false,
        findingId: null,
        action: 'MASK'
    });

    const handleRemediateRequest = (id: string, action: 'MASK' | 'DELETE') => {
        setRemediationState({
            isOpen: true,
            findingId: id,
            action: action
        });
    };


    const handleRemediationConfirm = async (options: { createRollback: boolean; notifyOwner: boolean }) => {
        if (!remediationState.findingId) return;

        try {
            await remediationApi.executeRemediation({
                finding_ids: [remediationState.findingId],
                action_type: remediationState.action,
                user_id: 'current-user' // In real app, get from auth context
            });

            // Refresh findings
            fetchFindings();
            setRemediationState(prev => ({ ...prev, isOpen: false }));
        } catch (error) {
            console.error('Remediation failed:', error);
            setError('Failed to execute remediation');
        }
    };

    const handleMarkFalsePositive = async (id: string) => {
        try {
            await findingsApi.submitFeedback(id, {
                feedback_type: 'FALSE_POSITIVE',
                comments: 'Marked via UI'
            });
            fetchFindings(); // Refresh list to see status change
        } catch (error) {
            console.error('Failed to mark false positive:', error);
            setError('Failed to update finding');
        }
    };

    return (
        <div className="flex flex-col h-full bg-background">
            {/* Header with Title and Global Actions */}
            <div className="flex items-center justify-between px-8 py-6 border-b border-border bg-card">
                <div>
                    <h1 className="text-2xl font-bold text-white">Findings Explorer</h1>
                    <p className="text-muted-foreground mt-1">Detailed breakdown of PII detections and display security risks.</p>
                </div>
                {findingsData && findingsData.findings.length > 0 && (
                    <button
                        onClick={() => {
                            const { exportToCSV } = require('@/utils/export');
                            exportToCSV(findingsData.findings, 'findings');
                        }}
                        className="px-4 py-2 bg-muted border border-border text-slate-300 rounded-lg hover:bg-accent hover:text-white transition-colors text-sm font-medium flex items-center gap-2"
                    >
                        📊 Export CSV
                    </button>
                )}
            </div>

            {/* Sticky Filters Bar */}
            <div className="sticky top-0 z-20 bg-card border-b border-border px-8 py-3 flex items-center gap-4 overflow-x-auto">
                <div className="flex items-center gap-2 text-sm text-muted-foreground font-medium whitespace-nowrap">
                    <span className="text-slate-500">Filters:</span>
                </div>

                {/* Scan Filter (Mock) */}
                <select className="bg-muted border border-border text-slate-300 text-sm rounded px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-blue-500">
                    <option>All Scans</option>
                    <option>SCAN_021 (Latest)</option>
                    <option>SCAN_020</option>
                </select>

                {/* PII Type Filter */}
                <select
                    className="bg-muted border border-border text-slate-300 text-sm rounded px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    value={piiTypeFilter}
                    onChange={(e) => setPiiTypeFilter(e.target.value)}
                >
                    <option value="">PII Type: All</option>
                    <option value="PAN">PAN</option>
                    <option value="Aadhaar">Aadhaar</option>
                    <option value="Email">Email</option>
                </select>

                {/* Asset Filter */}
                <select
                    className="bg-muted border border-border text-slate-300 text-sm rounded px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    value={assetFilter}
                    onChange={(e) => setAssetFilter(e.target.value)}
                >
                    <option value="">Asset: All</option>
                    <option value="DB-Prod">DB-Prod</option>
                    <option value="S3-Logs">S3-Logs</option>
                </select>

                {/* Risk Filter */}
                <select
                    className="bg-muted border border-border text-slate-300 text-sm rounded px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    value={severityFilter}
                    onChange={(e) => setSeverityFilter(e.target.value)}
                >
                    <option value="">Risk: All</option>
                    <option value="Critical">Critical</option>
                    <option value="High">High</option>
                    <option value="Medium">Medium</option>
                </select>

                {/* Status Filter */}
                <select
                    className="bg-muted border border-border text-slate-300 text-sm rounded px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value)}
                >
                    <option value="">Status: All</option>
                    <option value="Active">Active</option>
                    <option value="Suppressed">Suppressed</option>
                    <option value="Remediated">Remediated</option>
                </select>

                {/* Search */}
                <input
                    type="text"
                    placeholder="Search path/field..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="ml-auto bg-muted border border-border text-slate-200 text-sm rounded px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-blue-500 w-64"
                />
            </div>

            {/* Findings Content */}
            <div className="flex-1 overflow-auto p-8">
                {error && (
                    <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-4 rounded-lg mb-6">
                        {error}
                    </div>
                )}

                {loading && !findingsData ? (
                    <LoadingState message="Loading findings..." />
                ) : (
                    <div className="bg-card border border-border rounded-xl overflow-hidden shadow-sm">
                        {findingsData ? (
                            <FindingsTable
                                findings={findingsData.findings}
                                total={findingsData.total}
                                page={findingsData.page}
                                pageSize={findingsData.page_size}
                                totalPages={findingsData.total_pages}
                                onPageChange={setPage}
                                onFilterChange={handleFilterChange}
                                onRemediate={handleRemediateRequest}
                                onMarkFalsePositive={handleMarkFalsePositive}
                            />
                        ) : (
                            <div className="text-center py-12 text-slate-500">
                                No findings data available.
                            </div>
                        )}
                    </div>
                )}
            </div>

            <RemediationConfirmationModal
                isOpen={remediationState.isOpen}
                onClose={() => setRemediationState(prev => ({ ...prev, isOpen: false }))}
                onConfirm={handleRemediationConfirm}
                findingId={remediationState.findingId}
                actionType={remediationState.action}
            />
        </div>
    );
}
