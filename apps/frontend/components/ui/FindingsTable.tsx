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
    Critical: { color: 'text-red-400 bg-red-500/10 border-red-500/30', icon: AlertTriangle },
    High: { color: 'text-orange-400 bg-orange-500/10 border-orange-500/30', icon: AlertTriangle },
    Medium: { color: 'text-yellow-400 bg-yellow-500/10 border-yellow-500/30', icon: AlertTriangle },
    Low: { color: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/30', icon: AlertTriangle },
    Info: { color: 'text-blue-400 bg-blue-500/10 border-blue-500/30', icon: Eye },
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
            <div className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6">
                <div className="animate-pulse space-y-4">
                    <div className="h-6 w-48 bg-accent rounded" />
                    <div className="space-y-3">
                        {[1, 2, 3, 4, 5].map(i => (
                            <div key={i} className="h-16 bg-accent rounded-lg" />
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
            className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl overflow-hidden"
        >
            <RemediationConfirmationModal
                isOpen={showRemediationModal}
                onClose={() => setShowRemediationModal(false)}
                onConfirm={executeRemediation}
                findingId={selectedFindingId}
                actionType={remediationAction}
            />

            <div className="p-6 border-b border-border/30 space-y-4">
                {/* Header with stats */}
                <div className="flex items-center justify-between">
                    <div>
                        <h2 className="text-xl font-semibold text-white">PII Findings</h2>
                        <p className="text-muted-foreground text-sm mt-1">
                            {filteredFindings.length} of {findings.length} findings
                            {(searchQuery || riskFilter !== 'all' || sourceFilter !== 'all') && (
                                <span className="ml-2 text-blue-400">(filtered)</span>
                            )}
                        </p>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                        <div className="px-3 py-1 bg-accent rounded-lg text-slate-300">
                            {findings.filter(f => f.risk === 'Critical').length} Critical
                        </div>
                        <div className="px-3 py-1 bg-accent rounded-lg text-slate-300">
                            {findings.filter(f => f.risk === 'High').length} High
                        </div>
                    </div>
                </div>

                {/* Search and Filters */}
                <div className="flex flex-col sm:flex-row gap-4">
                    {/* Search */}
                    <div className="flex-1 relative">
                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                        <input
                            type="text"
                            placeholder="Search findings by asset, PII type, or field..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="w-full pl-10 pr-4 py-2 bg-accent border border-border/50 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50"
                        />
                    </div>

                    {/* Filters */}
                    <div className="flex gap-2">
                        <select
                            value={riskFilter}
                            onChange={(e) => setRiskFilter(e.target.value)}
                            className="px-3 py-2 bg-accent border border-border/50 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500/50"
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
                            className="px-3 py-2 bg-accent border border-border/50 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500/50"
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
                                className="px-3 py-2 bg-slate-600 hover:bg-slate-500 text-slate-300 hover:text-white rounded-lg transition-colors flex items-center gap-2"
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
                    <thead className="bg-accent/30">
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
                                    className="px-6 py-4 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider cursor-pointer hover:text-white transition-colors"
                                    onClick={() => handleSort(key)}
                                >
                                    <div className="flex items-center gap-2">
                                        {label}
                                        {sortField === key && (
                                            sortDirection === 'asc' ?
                                                <ChevronUp className="w-4 h-4" /> :
                                                <ChevronDown className="w-4 h-4" />
                                        )}
                                    </div>
                                </th>
                            ))}
                            <th className="px-6 py-4 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                                Actions
                            </th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-600/30">
                        {sortedFindings.map((finding) => {
                            const RiskIcon = riskConfig[finding.risk].icon;
                            const SourceIcon = sourceIcons[finding.sourceType];
                            const isExpanded = expandedRows.has(finding.id);

                            return (
                                <React.Fragment key={finding.id}>
                                    <motion.tr
                                        initial={{ opacity: 0 }}
                                        animate={{ opacity: 1 }}
                                        className="hover:bg-accent/20 transition-colors cursor-pointer"
                                        onClick={() => toggleRowExpansion(finding.id)}
                                    >
                                        <td className="px-6 py-4">
                                            <div>
                                                <div className="text-white font-medium">{finding.assetName}</div>
                                                <div className="text-muted-foreground text-sm truncate max-w-xs">
                                                    {finding.assetPath}
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-2">
                                                <div className={`px-2 py-1 rounded text-xs font-medium border ${riskConfig[finding.risk].color}`}>
                                                    {finding.piiType}
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="text-white font-medium">
                                                {Math.round(finding.confidence * 100)}%
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className={`inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium border ${riskConfig[finding.risk].color}`}>
                                                <RiskIcon className="w-3 h-3" />
                                                {finding.risk}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-2">
                                                <SourceIcon className="w-4 h-4 text-muted-foreground" />
                                                <span className="text-slate-300">{finding.sourceType}</span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-2">
                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        // Handle view details
                                                        console.log('View details for:', finding.id);
                                                    }}
                                                    className="px-3 py-1.5 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 hover:text-blue-300 rounded-lg text-sm font-medium transition-all duration-200 border border-blue-500/30 hover:border-blue-500/50"
                                                    title="View detailed information about this finding"
                                                >
                                                    <Eye className="w-3 h-3 inline mr-1" />
                                                    View
                                                </button>

                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        handleRemediateClick(finding.id, 'MASK');
                                                    }}
                                                    className="px-3 py-1.5 bg-purple-500/20 hover:bg-purple-500/30 text-purple-400 hover:text-purple-300 rounded-lg text-sm font-medium transition-all duration-200 border border-purple-500/30 hover:border-purple-500/50 flex items-center gap-1"
                                                    title="Mask PII in source"
                                                >
                                                    <Shield className="w-3 h-3" />
                                                    Mask
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
                                            <td colSpan={6} className="px-6 py-4 bg-accent/20">
                                                <div className="space-y-3">
                                                    <div className="grid grid-cols-2 gap-4 text-sm">
                                                        <div>
                                                            <span className="text-muted-foreground">Field:</span>
                                                            <span className="text-white ml-2">{finding.field}</span>
                                                        </div>
                                                        <div>
                                                            <span className="text-muted-foreground">Asset Path:</span>
                                                            <span className="text-white ml-2 font-mono text-xs">{finding.assetPath}</span>
                                                        </div>
                                                    </div>
                                                    <div className="pt-2 border-t border-border/30">
                                                        <div className="text-muted-foreground text-xs mb-1">Sample Data:</div>
                                                        <div className="bg-card/50 rounded p-2 font-mono text-xs text-slate-300">
                                                            Sample PII data would appear here...
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
                <div className="p-12 text-center">
                    <Database className="w-12 h-12 text-slate-600 mx-auto mb-4" />
                    <h3 className="text-lg font-medium text-muted-foreground mb-2">No Findings Yet</h3>
                    <p className="text-slate-500">Run a scan to discover PII in your data sources.</p>
                </div>
            )}
        </motion.div>
    );
}