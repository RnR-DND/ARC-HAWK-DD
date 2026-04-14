'use client';

import React, { useState, useEffect } from 'react';
import { X, Play, Zap, Clock, Trash2, Plus, Brain, Search, Regex } from 'lucide-react';
import { scansApi } from '@/services/scans.api';
import { connectionsApi, type Connection } from '@/services/connections.api';
import { patternsApi, type CustomPattern } from '@/services/patterns.api';

interface ScanConfigModalProps {
    isOpen: boolean;
    onClose: () => void;
    onRunScan?: (config: ScanConfig) => void;
}

interface ScanConfig {
    name: string;
    sources: string[];
    piiTypes: string[];
    executionMode: 'sequential' | 'parallel';
    classificationMode: 'regex' | 'ner' | 'contextual';
    piiTypesPerSource?: Record<string, string[]>;
}

const PII_TYPES = [
    { id: 'PAN', label: 'PAN', category: 'Financial' },
    { id: 'AADHAAR', label: 'Aadhaar', category: 'Identity' },
    { id: 'EMAIL', label: 'Email', category: 'Contact' },
    { id: 'PHONE', label: 'Phone', category: 'Contact' },
    { id: 'PASSPORT', label: 'Passport', category: 'Identity' },
    { id: 'VOTER_ID', label: 'Voter ID', category: 'Identity' },
    { id: 'DRIVING_LICENSE', label: 'Driving License', category: 'Identity' },
    { id: 'CREDIT_CARD', label: 'Credit Card', category: 'Financial' },
    { id: 'UPI_ID', label: 'UPI ID', category: 'Financial' },
    { id: 'BANK_ACCOUNT', label: 'Bank Account', category: 'Financial' },
    { id: 'GST', label: 'GSTIN', category: 'Business' },
    { id: 'IFSC', label: 'IFSC', category: 'Financial' },
];

const CLASSIFICATION_MODES = [
    {
        id: 'regex' as const,
        label: 'Regex Only',
        description: 'Fast, deterministic. Uses regex + checksum validators only.',
        icon: Regex,
        color: 'blue',
    },
    {
        id: 'ner' as const,
        label: 'Regex + NER',
        description: 'Adds spaCy Named Entity Recognition for natural language context.',
        icon: Search,
        color: 'purple',
    },
    {
        id: 'contextual' as const,
        label: 'Full (Regex + NER + Contextual)',
        description: 'All engines: regex, spaCy NER, and contextual ML scoring. Most accurate.',
        icon: Brain,
        color: 'green',
    },
];

export function ScanConfigModal({ isOpen, onClose, onRunScan }: ScanConfigModalProps) {
    const [scanName, setScanName] = useState('');
    const [selectedSources, setSelectedSources] = useState<string[]>([]);
    const [selectedPiiTypes, setSelectedPiiTypes] = useState<string[]>(['PAN', 'AADHAAR', 'EMAIL']);
    const [executionMode, setExecutionMode] = useState<'sequential' | 'parallel'>('parallel');
    const [classificationMode, setClassificationMode] = useState<'regex' | 'ner' | 'contextual'>('contextual');
    const [perSourcePiiEnabled, setPerSourcePiiEnabled] = useState(false);
    const [piiTypesPerSource, setPiiTypesPerSource] = useState<Record<string, string[]>>({});

    // Real data state
    const [sources, setSources] = useState<Connection[]>([]);
    const [loadingSources, setLoadingSources] = useState(false);
    const [sourcesError, setSourcesError] = useState<string | null>(null);

    // Custom patterns
    const [customPatterns, setCustomPatterns] = useState<CustomPattern[]>([]);
    const [selectedCustomPatterns, setSelectedCustomPatterns] = useState<string[]>([]);
    const [showAddPattern, setShowAddPattern] = useState(false);
    const [newPattern, setNewPattern] = useState({ name: '', display_name: '', regex: '', category: 'Custom', description: '' });
    const [patternError, setPatternError] = useState('');
    const [savingPattern, setSavingPattern] = useState(false);

    useEffect(() => {
        if (isOpen) {
            loadSources();
            loadPatterns();
        }
    }, [isOpen]);

    const loadSources = async () => {
        try {
            setLoadingSources(true);
            setSourcesError(null);
            const data = await connectionsApi.getConnections();
            setSources(data.connections || []);
        } catch (error) {
            console.error('Failed to load sources:', error);
            setSourcesError('Unable to load data sources. Please check your connection and try again.');
        } finally {
            setLoadingSources(false);
        }
    };

    const loadPatterns = async () => {
        try {
            const patterns = await patternsApi.getPatterns();
            setCustomPatterns(patterns.filter(p => p.is_active !== false));
        } catch {
            // non-critical
        }
    };

    if (!isOpen) return null;

    const handleDeleteSource = async (e: React.MouseEvent, source: Connection) => {
        e.stopPropagation();
        if (!confirm(`Remove "${source.profile_name}" data source? This cannot be undone.`)) return;
        try {
            await connectionsApi.deleteConnection(source.id);
            setSources(prev => prev.filter(s => s.id !== source.id));
            setSelectedSources(prev => prev.filter(id => id !== source.profile_name));
        } catch (err) {
            console.error('Failed to delete source:', err);
        }
    };

    const toggleSource = (sourceId: string) => {
        setSelectedSources(prev => {
            const removing = prev.includes(sourceId);
            const next = removing ? prev.filter(id => id !== sourceId) : [...prev, sourceId];
            if (removing) {
                setPiiTypesPerSource(p => {
                    const { [sourceId]: _, ...rest } = p;
                    return rest;
                });
            } else if (perSourcePiiEnabled && !piiTypesPerSource[sourceId]) {
                setPiiTypesPerSource(p => ({ ...p, [sourceId]: [...selectedPiiTypes] }));
            }
            return next;
        });
    };

    const togglePiiType = (piiId: string) => {
        setSelectedPiiTypes(prev =>
            prev.includes(piiId) ? prev.filter(id => id !== piiId) : [...prev, piiId]
        );
    };

    const selectAllPii = () => setSelectedPiiTypes(PII_TYPES.map(p => p.id));
    const deselectAllPii = () => setSelectedPiiTypes([]);

    const togglePerSourcePii = (piiId: string, profileName: string) => {
        setPiiTypesPerSource(prev => {
            const current = prev[profileName] || [...selectedPiiTypes];
            const updated = current.includes(piiId)
                ? current.filter(id => id !== piiId)
                : [...current, piiId];
            return { ...prev, [profileName]: updated };
        });
    };

    const selectAllPiiForSource = (profileName: string) => {
        setPiiTypesPerSource(prev => ({ ...prev, [profileName]: PII_TYPES.map(p => p.id) }));
    };

    const deselectAllPiiForSource = (profileName: string) => {
        setPiiTypesPerSource(prev => ({ ...prev, [profileName]: [] }));
    };

    const toggleCustomPattern = (id: string) => {
        setSelectedCustomPatterns(prev =>
            prev.includes(id) ? prev.filter(p => p !== id) : [...prev, id]
        );
    };

    const handleSavePattern = async () => {
        setPatternError('');
        if (!newPattern.name.trim() || !newPattern.regex.trim()) {
            setPatternError('Pattern name and regex are required.');
            return;
        }
        // Validate regex
        try {
            new RegExp(newPattern.regex);
        } catch {
            setPatternError('Invalid regular expression.');
            return;
        }
        try {
            setSavingPattern(true);
            const created = await patternsApi.createPattern({
                ...newPattern,
                display_name: newPattern.display_name || newPattern.name,
                is_active: true,
            });
            setCustomPatterns(prev => [...prev, created]);
            setSelectedCustomPatterns(prev => [...prev, created.id!]);
            setNewPattern({ name: '', display_name: '', regex: '', category: 'Custom', description: '' });
            setShowAddPattern(false);
        } catch (err: any) {
            setPatternError(err?.response?.data?.error || 'Failed to save pattern.');
        } finally {
            setSavingPattern(false);
        }
    };

    const handleDeletePattern = async (id: string) => {
        try {
            await patternsApi.deletePattern(id);
            setCustomPatterns(prev => prev.filter(p => p.id !== id));
            setSelectedCustomPatterns(prev => prev.filter(p => p !== id));
        } catch {
            console.error('Failed to delete pattern');
        }
    };

    const estimatePerformance = () => {
        const sourceCount = selectedSources.length;
        const piiCount = selectedPiiTypes.length;
        const isParallel = executionMode === 'parallel';
        const modeMultiplier = classificationMode === 'regex' ? 1 : classificationMode === 'ner' ? 1.5 : 2.2;

        const cpuUsage = Math.min(100, Math.round((sourceCount * 15 + piiCount * 5) * modeMultiplier));
        const ioUsage = Math.min(100, sourceCount * 20 + piiCount * 3);
        const estimatedTime = Math.round((isParallel
            ? Math.max(5, sourceCount * 8 + piiCount * 2)
            : sourceCount * 15 + piiCount * 3) * modeMultiplier);

        return { cpuUsage, ioUsage, estimatedTime };
    };

    const { cpuUsage, ioUsage, estimatedTime } = estimatePerformance();

    const handleRunScan = async () => {
        try {
            const activeCustomPatterns = customPatterns
                .filter(p => selectedCustomPatterns.includes(p.id!))
                .map(p => ({ name: p.name, display_name: p.display_name, regex: p.regex, category: p.category }));

            const config: any = {
                name: scanName || `Scan_${new Date().toISOString().split('T')[0]}`,
                sources: selectedSources,
                pii_types: selectedPiiTypes,
                execution_mode: executionMode,
                classification_mode: classificationMode,
                custom_patterns: activeCustomPatterns,
            };

            if (perSourcePiiEnabled && Object.keys(piiTypesPerSource).length > 0) {
                config.pii_types_per_source = piiTypesPerSource;
            }

            const response = await scansApi.triggerScan(config);
            console.log(`Scan triggered: ${response.scan_id}`);
            onRunScan?.(config);
            onClose();
        } catch (error) {
            console.error('Failed to trigger scan:', error);
            alert('Unable to start PII discovery scan. Please verify your data sources are configured correctly and try again.');
        }
    };

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] overflow-hidden border border-slate-200">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-green-50 rounded-lg">
                            <Play className="w-5 h-5 text-green-600" />
                        </div>
                        <div>
                            <h2 className="text-xl font-semibold text-slate-900">Run Scan</h2>
                            <p className="text-sm text-slate-500 mt-0.5">Configure and execute PII detection scan</p>
                        </div>
                    </div>
                    <button onClick={onClose} className="p-2 hover:bg-slate-100 rounded-lg transition-colors">
                        <X className="w-5 h-5 text-slate-400" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 overflow-y-auto max-h-[calc(90vh-180px)] space-y-6">
                    {/* Scan Name */}
                    <div>
                        <label className="block text-sm font-medium text-slate-600 mb-2">Scan Name</label>
                        <input
                            type="text"
                            value={scanName}
                            onChange={(e) => setScanName(e.target.value)}
                            placeholder={`Scan_${new Date().toISOString().split('T')[0]}`}
                            className="w-full px-4 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-green-500"
                        />
                    </div>

                    {/* Target Sources */}
                    <div>
                        <label className="block text-sm font-medium text-slate-600 mb-3">
                            Target Sources ({selectedSources.length} selected)
                        </label>
                        <div className="grid grid-cols-3 gap-3">
                            {loadingSources ? (
                                <div className="col-span-3 text-center py-8 text-slate-500">Loading data sources...</div>
                            ) : sourcesError ? (
                                <div className="col-span-3 text-center py-8">
                                    <p className="text-red-600 mb-2">{sourcesError}</p>
                                    <button onClick={loadSources} className="px-4 py-2 bg-slate-100 hover:bg-slate-200 rounded text-sm text-slate-900">Retry</button>
                                </div>
                            ) : sources.length === 0 ? (
                                <div className="col-span-3 text-center py-8 text-slate-500">No data sources configured. Please add a source first.</div>
                            ) : (
                                sources.map((source) => (
                                    <div
                                        key={source.id}
                                        className={`relative p-3 rounded-lg border-2 transition-all text-left cursor-pointer ${selectedSources.includes(source.profile_name) ? 'border-green-500 bg-green-50' : 'border-slate-200 bg-slate-50 hover:border-slate-300'}`}
                                        onClick={() => toggleSource(source.profile_name)}
                                    >
                                        <button
                                            onClick={(e) => handleDeleteSource(e, source)}
                                            className="absolute top-2 right-2 p-1 rounded hover:bg-red-100 text-slate-400 hover:text-red-500 transition-colors"
                                            title="Remove data source"
                                        >
                                            <Trash2 className="w-3.5 h-3.5" />
                                        </button>
                                        <div className="font-medium text-slate-900 text-sm pr-6">{source.profile_name}</div>
                                        <div className="text-xs text-slate-500 mt-1">{source.source_type}</div>
                                        {source.validation_status && (
                                            <div className={`text-xs mt-1 ${source.validation_status === 'valid' ? 'text-green-600' : 'text-yellow-600'}`}>
                                                {source.validation_status}
                                            </div>
                                        )}
                                    </div>
                                ))
                            )}
                        </div>
                    </div>

                    {/* PII Scope */}
                    <div>
                        <div className="flex items-center justify-between mb-3">
                            <label className="text-sm font-medium text-slate-600">
                                PII Scope ({selectedPiiTypes.length}/{PII_TYPES.length} types)
                            </label>
                            <div className="flex gap-2">
                                <button onClick={selectAllPii} className="text-xs text-blue-600 hover:text-blue-700">Select All</button>
                                <span className="text-slate-300">|</span>
                                <button onClick={deselectAllPii} className="text-xs text-blue-600 hover:text-blue-700">Deselect All</button>
                            </div>
                        </div>
                        <div className="grid grid-cols-4 gap-2">
                            {PII_TYPES.map((pii) => (
                                <button
                                    key={pii.id}
                                    onClick={() => togglePiiType(pii.id)}
                                    className={`px-3 py-2 rounded-lg text-sm font-medium transition-all ${selectedPiiTypes.includes(pii.id) ? 'bg-green-500 text-white' : 'bg-slate-100 text-slate-700 hover:bg-slate-200'}`}
                                >
                                    {pii.label}
                                </button>
                            ))}
                        </div>
                    </div>

                    {/* Per-Source PII Config Toggle */}
                    {selectedSources.length > 1 && (
                        <div>
                            <div className="flex items-center justify-between">
                                <div>
                                    <label className="text-sm font-medium text-slate-600">Per-Source PII Configuration</label>
                                    <p className="text-xs text-slate-400 mt-0.5">Assign different PII types to each data source</p>
                                </div>
                                <button
                                    onClick={() => {
                                        const next = !perSourcePiiEnabled;
                                        setPerSourcePiiEnabled(next);
                                        if (next) {
                                            const init: Record<string, string[]> = {};
                                            selectedSources.forEach(s => { init[s] = [...selectedPiiTypes]; });
                                            setPiiTypesPerSource(init);
                                        } else {
                                            setPiiTypesPerSource({});
                                        }
                                    }}
                                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${perSourcePiiEnabled ? 'bg-green-500' : 'bg-slate-300'}`}
                                >
                                    <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${perSourcePiiEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
                                </button>
                            </div>

                            {perSourcePiiEnabled && (
                                <div className="mt-3 space-y-3">
                                    {selectedSources.map(sourceName => {
                                        const source = sources.find(s => s.profile_name === sourceName);
                                        const sourcePii = piiTypesPerSource[sourceName] || [];
                                        return (
                                            <div key={sourceName} className="p-3 bg-slate-50 border border-slate-200 rounded-lg">
                                                <div className="flex items-center justify-between mb-2">
                                                    <div>
                                                        <span className="text-sm font-medium text-slate-900">{sourceName}</span>
                                                        {source && <span className="text-xs text-slate-400 ml-2">{source.source_type}</span>}
                                                    </div>
                                                    <div className="flex gap-2">
                                                        <button onClick={() => selectAllPiiForSource(sourceName)} className="text-xs text-blue-600 hover:text-blue-700">All</button>
                                                        <span className="text-slate-300">|</span>
                                                        <button onClick={() => deselectAllPiiForSource(sourceName)} className="text-xs text-blue-600 hover:text-blue-700">None</button>
                                                    </div>
                                                </div>
                                                <div className="flex flex-wrap gap-1.5">
                                                    {PII_TYPES.map(pii => (
                                                        <button
                                                            key={pii.id}
                                                            onClick={() => togglePerSourcePii(pii.id, sourceName)}
                                                            className={`px-2 py-1 rounded text-xs font-medium transition-all ${sourcePii.includes(pii.id) ? 'bg-green-500 text-white' : 'bg-slate-200 text-slate-600 hover:bg-slate-300'}`}
                                                        >
                                                            {pii.label}
                                                        </button>
                                                    ))}
                                                </div>
                                                <div className="text-xs text-slate-400 mt-1.5">{sourcePii.length} / {PII_TYPES.length} types selected</div>
                                            </div>
                                        );
                                    })}
                                </div>
                            )}
                        </div>
                    )}

                    {/* Classification Engine */}
                    <div>
                        <label className="block text-sm font-medium text-slate-600 mb-3">
                            Classification Engine
                        </label>
                        <div className="grid grid-cols-3 gap-3">
                            {CLASSIFICATION_MODES.map((mode) => {
                                const Icon = mode.icon;
                                const isSelected = classificationMode === mode.id;
                                return (
                                    <button
                                        key={mode.id}
                                        onClick={() => setClassificationMode(mode.id)}
                                        className={`p-3 rounded-lg border-2 text-left transition-all ${isSelected ? 'border-green-500 bg-green-50' : 'border-slate-200 bg-slate-50 hover:border-slate-300'}`}
                                    >
                                        <div className="flex items-center gap-2 mb-1">
                                            <Icon className={`w-4 h-4 ${isSelected ? 'text-green-600' : 'text-slate-400'}`} />
                                            <span className="text-sm font-semibold text-slate-900">{mode.label}</span>
                                        </div>
                                        <p className="text-xs text-slate-500">{mode.description}</p>
                                    </button>
                                );
                            })}
                        </div>
                    </div>

                    {/* Custom Patterns */}
                    <div>
                        <div className="flex items-center justify-between mb-3">
                            <label className="text-sm font-medium text-slate-600">
                                Custom Patterns
                                {selectedCustomPatterns.length > 0 && (
                                    <span className="ml-2 text-xs text-green-600">{selectedCustomPatterns.length} active</span>
                                )}
                            </label>
                            <button
                                onClick={() => setShowAddPattern(v => !v)}
                                className="flex items-center gap-1 text-xs text-blue-600 hover:text-blue-700"
                            >
                                <Plus className="w-3.5 h-3.5" />
                                Add Pattern
                            </button>
                        </div>

                        {/* Add Pattern Form */}
                        {showAddPattern && (
                            <div className="mb-3 p-4 bg-blue-50 border border-blue-200 rounded-lg space-y-3">
                                <div className="grid grid-cols-2 gap-3">
                                    <div>
                                        <label className="block text-xs font-medium text-slate-600 mb-1">Internal Name *</label>
                                        <input
                                            type="text"
                                            placeholder="CUSTOM_SSN"
                                            value={newPattern.name}
                                            onChange={e => setNewPattern(p => ({ ...p, name: e.target.value.toUpperCase().replace(/\s/g, '_') }))}
                                            className="w-full px-3 py-1.5 text-sm bg-white border border-slate-200 rounded text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-xs font-medium text-slate-600 mb-1">Display Name</label>
                                        <input
                                            type="text"
                                            placeholder="US Social Security Number"
                                            value={newPattern.display_name}
                                            onChange={e => setNewPattern(p => ({ ...p, display_name: e.target.value }))}
                                            className="w-full px-3 py-1.5 text-sm bg-white border border-slate-200 rounded text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                        />
                                    </div>
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-slate-600 mb-1">Regex Pattern *</label>
                                    <input
                                        type="text"
                                        placeholder="\b\d{3}-\d{2}-\d{4}\b"
                                        value={newPattern.regex}
                                        onChange={e => setNewPattern(p => ({ ...p, regex: e.target.value }))}
                                        className="w-full px-3 py-1.5 text-sm bg-white border border-slate-200 rounded font-mono text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                    />
                                </div>
                                <div className="grid grid-cols-2 gap-3">
                                    <div>
                                        <label className="block text-xs font-medium text-slate-600 mb-1">Category</label>
                                        <select
                                            value={newPattern.category}
                                            onChange={e => setNewPattern(p => ({ ...p, category: e.target.value }))}
                                            className="w-full px-3 py-1.5 text-sm bg-white border border-slate-200 rounded text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                        >
                                            <option>Custom</option>
                                            <option>Financial</option>
                                            <option>Identity</option>
                                            <option>Contact</option>
                                            <option>Health</option>
                                            <option>Business</option>
                                        </select>
                                    </div>
                                    <div>
                                        <label className="block text-xs font-medium text-slate-600 mb-1">Description</label>
                                        <input
                                            type="text"
                                            placeholder="Optional description"
                                            value={newPattern.description}
                                            onChange={e => setNewPattern(p => ({ ...p, description: e.target.value }))}
                                            className="w-full px-3 py-1.5 text-sm bg-white border border-slate-200 rounded text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                        />
                                    </div>
                                </div>
                                {patternError && (
                                    <div className="flex items-start gap-2 bg-red-50 border border-red-200 rounded p-2 text-xs text-red-700" role="alert">
                                        <span aria-hidden="true">⚠</span>
                                        <span>{patternError}</span>
                                    </div>
                                )}
                                <div className="flex gap-2 justify-end">
                                    <button
                                        onClick={() => { setShowAddPattern(false); setPatternError(''); setNewPattern({ name: '', display_name: '', regex: '', category: 'Custom', description: '' }); }}
                                        className="px-3 py-1.5 text-sm text-slate-600 hover:text-slate-900"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        onClick={handleSavePattern}
                                        disabled={savingPattern}
                                        className="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded disabled:opacity-50"
                                    >
                                        {savingPattern ? 'Saving…' : 'Save Pattern'}
                                    </button>
                                </div>
                            </div>
                        )}

                        {customPatterns.length === 0 ? (
                            <div className="py-4 text-center text-sm text-slate-400 border border-dashed border-slate-200 rounded-lg">
                                No custom patterns yet. Click "Add Pattern" to define your own PII types.
                            </div>
                        ) : (
                            <div className="grid grid-cols-3 gap-2">
                                {customPatterns.map(p => (
                                    <div
                                        key={p.id}
                                        onClick={() => toggleCustomPattern(p.id!)}
                                        className={`relative p-3 rounded-lg border-2 cursor-pointer transition-all ${selectedCustomPatterns.includes(p.id!) ? 'border-blue-500 bg-blue-50' : 'border-slate-200 bg-slate-50 hover:border-slate-300'}`}
                                    >
                                        <button
                                            onClick={e => { e.stopPropagation(); handleDeletePattern(p.id!); }}
                                            className="absolute top-2 right-2 p-1 rounded hover:bg-red-100 text-slate-400 hover:text-red-500"
                                        >
                                            <Trash2 className="w-3 h-3" />
                                        </button>
                                        <div className="font-medium text-slate-900 text-xs pr-5">{p.display_name || p.name}</div>
                                        <div className="text-xs text-slate-400 mt-0.5 font-mono truncate">{p.regex}</div>
                                        <div className="text-xs text-slate-400 mt-0.5">{p.category}</div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Execution Mode */}
                    <div>
                        <label className="block text-sm font-medium text-slate-600 mb-3">Execution Mode</label>
                        <div className="grid grid-cols-2 gap-3">
                            <button
                                onClick={() => setExecutionMode('sequential')}
                                className={`p-4 rounded-lg border-2 transition-all text-left ${executionMode === 'sequential' ? 'border-green-500 bg-green-50' : 'border-slate-200 bg-slate-50 hover:border-slate-300'}`}
                            >
                                <div className="flex items-center gap-2 mb-2">
                                    <Clock className="w-4 h-4 text-green-600" />
                                    <span className="font-semibold text-slate-900">Sequential</span>
                                </div>
                                <p className="text-xs text-slate-500">Lower resource usage, longer duration</p>
                            </button>

                            <button
                                onClick={() => setExecutionMode('parallel')}
                                className={`p-4 rounded-lg border-2 transition-all text-left ${executionMode === 'parallel' ? 'border-green-500 bg-green-50' : 'border-slate-200 bg-slate-50 hover:border-slate-300'}`}
                            >
                                <div className="flex items-center gap-2 mb-2">
                                    <Zap className="w-4 h-4 text-green-600" />
                                    <span className="font-semibold text-slate-900">Parallel</span>
                                </div>
                                <p className="text-xs text-slate-500">Faster execution, higher resource usage</p>
                            </button>
                        </div>
                    </div>

                    {/* Performance Impact */}
                    <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
                        <div className="text-sm font-medium text-slate-600 mb-3">Performance Impact Estimate</div>
                        <div className="space-y-3">
                            <div>
                                <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
                                    <span>CPU Usage</span>
                                    <span>{cpuUsage}%</span>
                                </div>
                                <div className="h-2 bg-slate-200 rounded-full overflow-hidden">
                                    <div className="h-full bg-gradient-to-r from-green-500 to-yellow-500 transition-all" style={{ width: `${cpuUsage}%` }} />
                                </div>
                            </div>
                            <div>
                                <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
                                    <span>I/O Usage</span>
                                    <span>{ioUsage}%</span>
                                </div>
                                <div className="h-2 bg-slate-200 rounded-full overflow-hidden">
                                    <div className="h-full bg-gradient-to-r from-blue-500 to-purple-500 transition-all" style={{ width: `${ioUsage}%` }} />
                                </div>
                            </div>
                            <div className="flex items-center justify-between pt-2 border-t border-slate-200">
                                <span className="text-xs text-slate-500">Estimated Time</span>
                                <span className="text-sm font-semibold text-slate-900">{estimatedTime}m</span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Footer */}
                <div className="flex items-center justify-between px-6 py-4 border-t border-slate-200 bg-slate-50">
                    <button onClick={onClose} className="px-4 py-2 text-slate-500 hover:text-slate-900 transition-colors">
                        Cancel
                    </button>
                    <button
                        onClick={handleRunScan}
                        disabled={selectedSources.length === 0 || selectedPiiTypes.length === 0 || (perSourcePiiEnabled && selectedSources.some(s => !(piiTypesPerSource[s]?.length)))}
                        className="flex items-center gap-2 px-6 py-2 bg-green-600 hover:bg-green-700 disabled:bg-slate-200 disabled:text-slate-400 text-white rounded-lg font-medium transition-colors"
                    >
                        <Play className="w-4 h-4" />
                        <span>Run Scan</span>
                    </button>
                </div>
            </div>
        </div>
    );
}
