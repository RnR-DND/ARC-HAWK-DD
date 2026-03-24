import React, { useState, useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Eye, AlertTriangle, Database, File, ExternalLink, ChevronDown, ChevronUp, Search, Filter, X, Shield } from 'lucide-react';
import { RemediationConfirmationModal } from '@/components/remediation/RemediationConfirmationModal';
import { remediationApi } from '@/services/remediation.api';

interface Finding {
    id: string;
    assetName: string;
    assetPath: string;
    field: string;
    piiType: string;
    confidence: number;
    risk: 'Critical' | 'High' | 'Medium' | 'Low' | 'Info';
    sourceType: 'Database' | 'File' | 'Cloud' | 'API';
}

interface FindingsTableProps {
    findings: Finding[];
    loading?: boolean;
}

const riskConfig = {
    Critical: { color: 'text-red-700 bg-red-50 border-red-200', icon: AlertTriangle },
    High: { color: 'text-orange-700 bg-orange-50 border-orange-200', icon: AlertTriangle },
    Medium: { color: 'text-yellow-700 bg-yellow-50 border-yellow-200', icon: AlertTriangle },
    Low: { color: 'text-emerald-700 bg-emerald-50 border-emerald-200', icon: AlertTriangle },
    Info: { color: 'text-blue-700 bg-blue-50 border-blue-200', icon: Eye },
};

const sourceIcons = {
    Database: Database,
    File: File,
    Cloud: ExternalLink,
    API: ExternalLink,
};

export default function FindingsTable({ findings, loading = false }: FindingsTableProps) {
    const [sortField, setSortField] = useState<string>('confidence');
    const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');
    const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
    const [searchQuery, setSearchQuery] = useState('');
    const [riskFilter, setRiskFilter] = useState<string>('all');
    const [sourceFilter, setSourceFilter] = useState<string>('all');

    // Remediation State
    const [showRemediationModal, setShowRemediationModal] = useState(false);
    const [selectedFindingId, setSelectedFindingId] = useState<string | null>(null);
    const [remediationAction, setRemediationAction] = useState<'MASK' | 'DELETE'>('MASK');

    const handleSort = (field: string) => {
        if (sortField === field) {
            setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
        } else {
            setSortField(field);
            setSortDirection('desc');
        }
    };

    const toggleRowExpansion = (id: string) => {
        const newExpanded = new Set(expandedRows);
        if (newExpanded.has(id)) {
            newExpanded.delete(id);
        } else {
            newExpanded.add(id);
        }
        setExpandedRows(newExpanded);
    };

    const handleRemediateClick = (findingId: string, action: 'MASK' | 'DELETE' = 'MASK') => {
        setSelectedFindingId(findingId);
        setRemediationAction(action);
        setShowRemediationModal(true);
    };

    const executeRemediation = async (options: any) => {
        if (!selectedFindingId) return;

        // Call backend API
        await remediationApi.executeRemediation({
            finding_ids: [selectedFindingId],
            action_type: remediationAction,
            user_id: 'current-user-id' // Should catch from context in real app
        });

        // Close modal handled by modal itself on success, or we might need to refresh list
        // Ideally we trigger a refresh here, but for now we just allow the modal flow to complete
    };

    // Filter and search logic
    const filteredFindings = useMemo(() => {
        return findings.filter(finding => {
            const matchesSearch = searchQuery === '' ||
                finding.assetName.toLowerCase().includes(searchQuery.toLowerCase()) ||
                finding.piiType.toLowerCase().includes(searchQuery.toLowerCase()) ||
                finding.field.toLowerCase().includes(searchQuery.toLowerCase());

            const matchesRisk = riskFilter === 'all' || finding.risk.toLowerCase() === riskFilter.toLowerCase();
            const matchesSource = sourceFilter === 'all' || finding.sourceType.toLowerCase() === sourceFilter.toLowerCase();

            return matchesSearch && matchesRisk && matchesSource;
        });
    }, [findings, searchQuery, riskFilter, sourceFilter]);

    const sortedFindings = [...filteredFindings].sort((a, b) => {
        let aValue = a[sortField as keyof Finding];
        let bValue = b[sortField as keyof Finding];

        if (typeof aValue === 'string') aValue = aValue.toLowerCase();
        if (typeof bValue === 'string') bValue = bValue.toLowerCase();

        if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1;
        if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1;
        return 0;
    });

    const clearFilters = () => {
        setSearchQuery('');
        setRiskFilter('all');
        setSourceFilter('all');
    };

    if (loading) {
        return (
            <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
                <div className="animate-pulse space-y-4">
                    <div className="h-6 w-48 bg-slate-100 rounded" />
                    <div className="space-y-3">
                        {[1, 2, 3, 4, 5].map(i => (
                            <div key={i} className="h-16 bg-slate-50 rounded-lg" />
                        ))}
                    </div>
                </div>
            </div>
        );
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm"
        >
            <RemediationConfirmationModal
                isOpen={showRemediationModal}
                onClose={() => setShowRemediationModal(false)}
                onConfirm={executeRemediation}
                findingId={selectedFindingId}
                actionType={remediationAction}
            />

            <div className="p-6 border-b border-slate-100 space-y-4">
                {/* Header with stats */}
                <div className="flex items-center justify-between">
                    <div>
                        <h2 className="text-xl font-bold text-slate-800">PII Findings</h2>
                        <p className="text-slate-500 text-sm mt-1">
                            {filteredFindings.length} of {findings.length} findings
                            {(searchQuery || riskFilter !== 'all' || sourceFilter !== 'all') && (
                                <span className="ml-2 text-blue-600 font-medium">(filtered)</span>
                            )}
                        </p>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                        <div className="px-3 py-1 bg-red-50 border border-red-100 rounded-lg text-red-700 font-medium">
                            {findings.filter(f => f.risk === 'Critical').length} Critical
                        </div>
                        <div className="px-3 py-1 bg-orange-50 border border-orange-100 rounded-lg text-orange-700 font-medium">
                            {findings.filter(f => f.risk === 'High').length} High
                        </div>
                    </div>
                </div>

                {/* Search and Filters */}
                <div className="flex flex-col sm:flex-row gap-4">
                    {/* Search */}
                    <div className="flex-1 relative">
                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-slate-400" />
                        <input
                            type="text"
                            placeholder="Search findings by asset, PII type, or field..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="w-full pl-10 pr-4 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-medium"
                        />
                    </div>

                    {/* Filters */}
                    <div className="flex gap-2">
                        <select
                            value={riskFilter}
                            onChange={(e) => setRiskFilter(e.target.value)}
                            className="px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500/20 hover:bg-slate-100 transition-colors"
                        >
                            <option value="all">All Risks</option>
                            <option value="critical">Critical</option>
                            <option value="high">High</option>
                            <option value="medium">Medium</option>
                            <option value="low">Low</option>
                            <option value="info">Info</option>
                        </select>

                        <select
                            value={sourceFilter}
                            onChange={(e) => setSourceFilter(e.target.value)}
                            className="px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500/20 hover:bg-slate-100 transition-colors"
                        >
                            <option value="all">All Sources</option>
                            <option value="database">Database</option>
                            <option value="file">File</option>
                            <option value="cloud">Cloud</option>
                            <option value="api">API</option>
                        </select>

                        {(searchQuery || riskFilter !== 'all' || sourceFilter !== 'all') && (
                            <button
                                onClick={clearFilters}
                                className="px-3 py-2 bg-slate-100 hover:bg-slate-200 text-slate-600 hover:text-slate-900 rounded-lg transition-colors flex items-center gap-2 border border-slate-200 font-medium"
                            >
                                <X className="w-4 h-4" />
                                Clear
                            </button>
                        )}
                    </div>
                </div>
            </div>

            <div className="overflow-x-auto">
                <table className="w-full">
                    <thead className="bg-slate-50 border-b border-slate-200">
                        <tr>
                            {[
                                { key: 'assetName', label: 'Asset' },
                                { key: 'piiType', label: 'PII Type' },
                                { key: 'confidence', label: 'Confidence' },
                                { key: 'risk', label: 'Risk Level' },
                                { key: 'sourceType', label: 'Source' },
                            ].map(({ key, label }) => (
                                <th
                                    key={key}
                                    className="px-6 py-4 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider cursor-pointer hover:text-blue-600 transition-colors select-none"
                                    onClick={() => handleSort(key)}
                                >
                                    <div className="flex items-center gap-2">
                                        {label}
                                        {sortField === key && (
                                            sortDirection === 'asc' ?
                                                <ChevronUp className="w-4 h-4 text-blue-500" /> :
                                                <ChevronDown className="w-4 h-4 text-blue-500" />
                                        )}
                                    </div>
                                </th>
                            ))}
                            <th className="px-6 py-4 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">
                                Actions
                            </th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                        {sortedFindings.map((finding) => {
                            const RiskIcon = riskConfig[finding.risk].icon;
                            const SourceIcon = sourceIcons[finding.sourceType];
                            const isExpanded = expandedRows.has(finding.id);

                            return (
                                <React.Fragment key={finding.id}>
                                    <motion.tr
                                        initial={{ opacity: 0 }}
                                        animate={{ opacity: 1 }}
                                        className={`transition-colors cursor-pointer ${isExpanded ? 'bg-blue-50/50' : 'hover:bg-slate-50'}`}
                                        onClick={() => toggleRowExpansion(finding.id)}
                                    >
                                        <td className="px-6 py-4">
                                            <div>
                                                <div className="text-slate-900 font-semibold">{finding.assetName}</div>
                                                <div className="text-slate-500 text-sm truncate max-w-xs mt-0.5">
                                                    {finding.assetPath}
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-2">
                                                <div className="px-2.5 py-1 rounded-md text-xs font-semibold bg-slate-100 text-slate-700 border border-slate-200">
                                                    {finding.piiType}
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-2">
                                                <div className="h-1.5 w-12 bg-slate-100 rounded-full overflow-hidden">
                                                    <div
                                                        className={`h-full rounded-full ${finding.confidence > 0.9 ? 'bg-emerald-500' : finding.confidence > 0.7 ? 'bg-blue-500' : 'bg-orange-500'}`}
                                                        style={{ width: `${finding.confidence * 100}%` }}
                                                    />
                                                </div>
                                                <span className="text-slate-700 font-medium text-sm">
                                                    {Math.round(finding.confidence * 100)}%
                                                </span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${riskConfig[finding.risk].color}`}>
                                                <RiskIcon className="w-3.5 h-3.5" />
                                                {finding.risk}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-2 text-slate-600">
                                                <SourceIcon className="w-4 h-4 text-slate-400" />
                                                <span className="text-sm font-medium">{finding.sourceType}</span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-2">
                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        console.log('View details for:', finding.id);
                                                    }}
                                                    className="p-2 hover:bg-blue-50 text-slate-400 hover:text-blue-600 rounded-lg transition-colors"
                                                    title="View Details"
                                                >
                                                    <Eye className="w-4 h-4" />
                                                </button>

                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        handleRemediateClick(finding.id, 'MASK');
                                                    }}
                                                    className="flex items-center gap-1.5 px-3 py-1.5 bg-white border border-slate-200 hover:border-purple-300 hover:bg-purple-50 text-slate-600 hover:text-purple-700 rounded-lg text-xs font-medium transition-all shadow-sm"
                                                    title="Remediate"
                                                >
                                                    <Shield className="w-3.5 h-3.5" />
                                                    Remediate
                                                </button>
                                            </div>
                                        </td>
                                    </motion.tr>

                                    {isExpanded && (
                                        <motion.tr
                                            initial={{ opacity: 0, height: 0 }}
                                            animate={{ opacity: 1, height: 'auto' }}
                                            exit={{ opacity: 0, height: 0 }}
                                        >
                                            <td colSpan={6} className="px-6 py-4 bg-slate-50/50 border-b border-slate-100">
                                                <div className="pl-4 border-l-2 border-blue-500 space-y-3">
                                                    <div className="grid grid-cols-2 gap-8 text-sm">
                                                        <div>
                                                            <span className="text-slate-500 font-medium text-xs uppercase tracking-wide">Matched Field</span>
                                                            <div className="text-slate-900 font-mono text-sm mt-1 bg-white border border-slate-200 rounded px-2 py-1 inline-block">
                                                                {finding.field}
                                                            </div>
                                                        </div>
                                                        <div>
                                                            <span className="text-slate-500 font-medium text-xs uppercase tracking-wide">Full Path</span>
                                                            <div className="text-slate-700 font-mono text-xs mt-1 break-all">
                                                                {finding.assetPath}
                                                            </div>
                                                        </div>
                                                    </div>

                                                    <div className="pt-3">
                                                        <div className="text-slate-500 font-medium text-xs uppercase tracking-wide mb-1">Context Preview</div>
                                                        <div className="bg-slate-900 rounded-lg p-3 font-mono text-xs text-slate-300 shadow-inner">
                                                            <span className="text-slate-500">{'// Only authorized personnel can view raw data'}</span>
                                                            <br />
                                                            <span className="text-blue-400">SELECT</span> <span className="text-purple-400">*</span> <span className="text-blue-400">FROM</span> <span className="text-yellow-300">users</span> <span className="text-blue-400">WHERE</span> <span className="text-green-400">id</span> = ...
                                                        </div>
                                                    </div>
                                                </div>
                                            </td>
                                        </motion.tr>
                                    )}
                                </React.Fragment>
                            );
                        })}
                    </tbody>
                </table>
            </div>

            {findings.length === 0 && (
                <div className="p-12 text-center bg-slate-50/50">
                    <div className="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mx-auto mb-4 border border-slate-200">
                        <Database className="w-8 h-8 text-slate-400" />
                    </div>
                    <h3 className="text-lg font-semibold text-slate-900 mb-1">No Findings Yet</h3>
                    <p className="text-slate-500 max-w-sm mx-auto">Run a scan on your connected data sources to discover PII and view findings here.</p>
                </div>
            )}
        </motion.div>
    );
}