'use client';

import React from 'react';
import { useScanContext } from '@/contexts/ScanContext';
import { X, Eye, EyeOff } from 'lucide-react';

export function ScanContextBar() {
    const { currentScanId, currentScanName, environment, zeroValueMode, clearScan, toggleZeroValueMode } = useScanContext();

    if (!currentScanId) {
        return null;
    }

    const envColors = {
        PROD: 'bg-red-500/10 text-red-400 border-red-500/20',
        DEV: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
        QA: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
    };

    const envColor = environment ? envColors[environment] : envColors.DEV;

    return (
        <div className="flex items-center gap-3 px-4 py-2 bg-white/50 backdrop-blur-sm border-b border-slate-200/60 shadow-sm">
            {/* Scan Context */}
            <div className="flex items-center gap-2">
                <div className="text-xs text-slate-500 font-medium">Active Scan:</div>
                <div className="text-sm font-mono text-slate-700 bg-slate-100/50 px-2 py-0.5 rounded border border-slate-200/50">{currentScanName || currentScanId}</div>
            </div>

            {/* Environment Badge */}
            {environment && (
                <div className={`px-2 py-0.5 rounded text-xs font-semibold border ${envColor}`}>
                    {environment}
                </div>
            )}

            {/* Zero-Value Mode Indicator */}
            <button
                onClick={toggleZeroValueMode}
                className="flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 hover:bg-slate-200 hover:text-slate-900 transition-colors border border-slate-200"
                title={zeroValueMode ? "PII values hidden (click to show)" : "PII values visible (click to hide)"}
            >
                {zeroValueMode ? (
                    <>
                        <EyeOff className="w-3 h-3" />
                        <span>Zero-Value Mode</span>
                    </>
                ) : (
                    <>
                        <Eye className="w-3 h-3" />
                        <span>Values Visible</span>
                    </>
                )}
            </button>

            {/* Clear Scan */}
            <button
                onClick={clearScan}
                className="ml-auto p-1 rounded hover:bg-slate-100 text-slate-400 hover:text-slate-600 transition-colors"
                title="Clear scan context"
            >
                <X className="w-4 h-4" />
            </button>
        </div>
    );
}
