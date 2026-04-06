'use client';

import React from 'react';
import { X, Shield, EyeOff, Trash2, CheckCircle, AlertTriangle } from 'lucide-react';
import { FindingWithDetails } from '@/types';
import { theme, getRiskColor } from '@/design-system/theme';

interface FindingDetailDrawerProps {
    finding: FindingWithDetails | null;
    isOpen: boolean;
    onClose: () => void;
    onMarkFalsePositive: (id: string) => void;
    onRemediate: (id: string, action: 'MASK' | 'DELETE') => void;
}

export function FindingDetailDrawer({
    finding,
    isOpen,
    onClose,
    onMarkFalsePositive,
    onRemediate
}: FindingDetailDrawerProps) {
    if (!finding) return null;

    const classification = finding.classifications[0];
    const piiType = classification?.classification_type || 'Unknown';
    const confidence = classification?.confidence_score || 0;

    return (
        <>
            {/* Backdrop */}
            {isOpen && (
                <div
                    className="fixed inset-0 bg-black/50 z-[60] transition-opacity"
                    onClick={onClose}
                />
            )}

            {/* Drawer */}
            <div className={`
                fixed top-0 right-0 h-full w-[480px] bg-white border-l border-slate-200 shadow-2xl z-[70] transform transition-transform duration-300 ease-in-out
                ${isOpen ? 'translate-x-0' : 'translate-x-full'}
            `}>
                <div className="flex flex-col h-full">
                    {/* Header */}
                    <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between bg-white">
                        <div>
                            <h2 className="text-lg font-semibold text-slate-900">Finding Details</h2>
                            <p className="text-sm text-slate-500 font-mono mt-0.5">{finding.id}</p>
                        </div>
                        <button
                            onClick={onClose}
                            className="p-2 text-slate-400 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>

                    {/* Content */}
                    <div className="flex-1 overflow-y-auto p-6 space-y-8">
                        {/* Key Info Card */}
                        <div className="bg-slate-50 rounded-lg p-4 border border-slate-200 space-y-4">
                            <div>
                                <label className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
                                    Asset Path
                                </label>
                                <div className="font-mono text-sm text-blue-700 break-all">
                                    {finding.asset_name} ▸ {finding.asset_path.replace(/\//g, ' ▸ ').replace(/\./g, ' ▸ ')}
                                </div>
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
                                        PII Type
                                    </label>
                                    <div className="text-slate-900 font-medium flex items-center gap-2">
                                        {piiType}
                                    </div>
                                </div>
                                <div>
                                    <label className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
                                        Confidence
                                    </label>
                                    <div className="text-slate-900 font-medium">
                                        {(confidence * 100).toFixed(0)}%
                                    </div>
                                </div>
                            </div>

                            <div>
                                <label className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
                                    Risk Level
                                </label>
                                <div className={`inline-flex items-center px-2.5 py-0.5 rounded text-sm font-medium border
                                    ${finding.severity === 'Critical' ? 'bg-red-50 text-red-700 border-red-200' : ''}
                                    ${finding.severity === 'High' ? 'bg-orange-50 text-orange-700 border-orange-200' : ''}
                                    ${finding.severity === 'Medium' ? 'bg-yellow-50 text-yellow-700 border-yellow-200' : ''}
                                    ${finding.severity === 'Low' ? 'bg-emerald-50 text-emerald-700 border-emerald-200' : ''}
                                `}>
                                    {finding.severity}
                                </div>
                            </div>
                        </div>

                        {/* Detection Method */}
                        <div>
                            <h3 className="text-sm font-semibold text-slate-900 mb-3">Detection Logic</h3>
                            <div className="bg-slate-50 rounded border border-slate-200 p-3">
                                <div className="flex items-center gap-2 text-sm text-slate-700 mb-2">
                                    <Shield className="w-4 h-4 text-green-600" />
                                    <span>Presidio Analysis + Context Validation</span>
                                </div>
                                <div className="text-xs text-slate-500">
                                    Matches found using {finding.pattern_name} pattern extractor with checksum validation.
                                </div>
                            </div>
                        </div>

                        {/* Evidence */}
                        <div>
                            <h3 className="text-sm font-semibold text-slate-900 mb-3">Matching Evidence</h3>
                            <div className="bg-slate-50 rounded border border-slate-200 p-3 font-mono text-xs text-slate-700 overflow-x-auto whitespace-pre-wrap">
                                {finding.matches?.join('\n') || finding.sample_text}
                            </div>
                        </div>
                    </div>

                    {/* Footer / Actions */}
                    <div className="p-6 border-t border-slate-200 bg-white space-y-3">
                        <button
                            onClick={() => onMarkFalsePositive(finding.id)}
                            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-slate-50 hover:bg-slate-100 border border-slate-200 text-slate-700 rounded-lg font-medium transition-colors"
                        >
                            <CheckCircle className="w-4 h-4" />
                            Mark as False Positive
                        </button>

                        <div className="grid grid-cols-2 gap-3">
                            <button
                                onClick={() => onRemediate(finding.id, 'MASK')}
                                className="flex items-center justify-center gap-2 px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
                            >
                                <EyeOff className="w-4 h-4" />
                                Mask Data
                            </button>
                            <button
                                onClick={() => onRemediate(finding.id, 'DELETE')}
                                className="flex items-center justify-center gap-2 px-4 py-2.5 bg-red-600 hover:bg-red-700 text-white rounded-lg font-medium transition-colors"
                            >
                                <Trash2 className="w-4 h-4" />
                                Delete Data
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
}
