'use client';

import React, { useEffect, useState } from 'react';
import { Search, Download } from 'lucide-react';
import FindingsTable from '@/components/FindingsTable';
import LoadingState from '@/components/LoadingState';
import { findingsApi } from '@/services/findings.api';
import { RemediationConfirmationModal } from '@/components/remediation/RemediationConfirmationModal';
import { remediationApi } from '@/services/remediation.api';

import type { FindingWithDetails, FindingsResponse } from '@/types';

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

    // Facets state — populated once on mount from API
    const [facets, setFacets] = useState<{ pii_types: string[]; assets: string[]; severities: string[] }>({
        pii_types: [],
        assets: [],
        severities: ['Critical', 'High', 'Medium', 'Low'],
    });

    // Selected finding for detail drawer (passed down to FindingsTable)
    const [selectedFinding, setSelectedFinding] = useState<FindingWithDetails | null>(null);

    // Fetch facets once on mount
    useEffect(() => {
        findingsApi.getFacets().then(setFacets).catch(() => {})
    }, [])

    useEffect(() => {
        fetchFindings();
    }, [page, searchTerm, severityFilter, statusFilter, assetFilter, piiTypeFilter]);


    const fetchFindings = async () => {
        try {
            setLoading(true);
            setError(null);

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
            // Preserve original finding.id so remediation API receives a valid UUID
            const explodedFindings: FindingWithDetails[] = [];

            result.findings.forEach(finding => {
                if (finding.matches && finding.matches.length > 0) {
                    finding.matches.forEach((match: string) => {
                        explodedFindings.push({
                            ...finding,
                            matches: [match],
                            // Keep original ID — FindingsTable uses array index as React key
                        });
                    });
                } else {
                    explodedFindings.push(finding);
                }
            });

            // Supplement facets from loaded findings when getFacets returns nothing
            setFacets(prev => {
                const piiTypes = prev.pii_types.length > 0
                    ? prev.pii_types
                    : [...new Set(result.findings.flatMap(f => f.classifications?.map((c: any) => c.classification_type) ?? []))].filter(Boolean) as string[];
                const assets = prev.assets.length > 0
                    ? prev.assets
                    : [...new Set(result.findings.map(f => f.asset_name).filter(Boolean))] as string[];
                return { ...prev, pii_types: piiTypes, assets };
            });

            const data: FindingsResponse = {
                findings: explodedFindings,
                total: result.total,
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
        setPage(1);
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
                user_id: 'current-user'
            });

            setRemediationState(prev => ({ ...prev, isOpen: false }));
            fetchFindings();
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
            fetchFindings();
        } catch (error) {
            console.error('Failed to mark false positive:', error);
            setError('Failed to update finding');
        }
    };

    return (
        <div className="flex flex-col h-full bg-white">
            {/* Header */}
            <div className="flex items-center justify-between px-8 py-6 border-b border-slate-200 bg-white">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Findings Explorer</h1>
                    <p className="text-slate-500 mt-1">Detailed breakdown of PII detections and security risks.</p>
                </div>
                {findingsData && findingsData.findings.length > 0 && (
                    <button
                        onClick={() => {
                            const { exportToCSV } = require('@/utils/export');
                            exportToCSV(findingsData.findings, 'findings');
                        }}
                        className="flex items-center gap-2 px-4 py-2 bg-white border border-slate-200 text-slate-700 rounded-lg hover:bg-slate-50 hover:text-slate-900 transition-colors text-sm font-medium"
                    >
                        <Download className="w-4 h-4" />
                        Export CSV
                    </button>
                )}
            </div>

            {/* Sticky Filters Bar */}
            <div className="sticky top-0 z-20 bg-white border-b border-slate-200 px-8 py-3 flex items-center gap-4 overflow-x-auto">
                <div className="flex items-center gap-2 text-sm text-slate-500 font-medium whitespace-nowrap">
                    <span>Filters:</span>
                </div>

                {/* PII Type Filter */}
                <select
                    className="bg-white border border-slate-200 text-slate-700 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    value={piiTypeFilter}
                    onChange={(e) => setPiiTypeFilter(e.target.value)}
                >
                    <option value="">All PII Types</option>
                    {facets.pii_types.map(t => <option key={t} value={t}>{t}</option>)}
                </select>

                {/* Asset Filter */}
                <select
                    className="bg-white border border-slate-200 text-slate-700 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    value={assetFilter}
                    onChange={(e) => setAssetFilter(e.target.value)}
                >
                    <option value="">All Assets</option>
                    {facets.assets.map(a => <option key={a} value={a}>{a}</option>)}
                </select>

                {/* Risk Filter */}
                <select
                    className="bg-white border border-slate-200 text-slate-700 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    value={severityFilter}
                    onChange={(e) => setSeverityFilter(e.target.value)}
                >
                    <option value="">All Severities</option>
                    {facets.severities.map(s => <option key={s} value={s}>{s}</option>)}
                </select>

                {/* Status Filter */}
                <select
                    className="bg-white border border-slate-200 text-slate-700 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value)}
                >
                    <option value="">Status: All</option>
                    <option value="Active">Active</option>
                    <option value="Suppressed">Suppressed</option>
                    <option value="Remediated">Remediated</option>
                </select>

                {/* Search */}
                <div className="ml-auto relative">
                    <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
                    <input
                        type="text"
                        placeholder="Search path/field..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="bg-white border border-slate-200 text-slate-700 text-sm rounded-lg pl-9 pr-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 w-64"
                    />
                </div>
            </div>

            {/* Findings Content */}
            <div className="flex-1 overflow-auto p-8">
                {error && (
                    <div className="bg-red-50 border border-red-200 text-red-700 p-4 rounded-lg mb-6 text-sm">
                        {error}
                    </div>
                )}

                {loading && !findingsData ? (
                    <LoadingState message="Loading findings..." />
                ) : (
                    <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm">
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
                                onRowClick={setSelectedFinding}
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
