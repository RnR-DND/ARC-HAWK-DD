'use client';

import React from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';

interface RiskDistributionProps {
    byPiiType: Record<string, number>;
}

export default function RiskDistribution({
    byPiiType,
    byAsset = {},
    byConfidence = {}
}: {
    byPiiType: Record<string, number>;
    byAsset?: Record<string, number>;
    byConfidence?: Record<string, number>;
}) {
    const [activeTab, setActiveTab] = React.useState<'type' | 'asset' | 'confidence'>('type');

    const getData = () => {
        switch (activeTab) {
            case 'asset': return byAsset;
            case 'confidence': return byConfidence;
            case 'type':
            default: return byPiiType;
        }
    };

    const data = getData();
    const total = Object.values(data).reduce((a, b) => a + b, 0);
    const sortedData = Object.entries(data)
        .sort(([, a], [, b]) => b - a)
        .slice(0, 5);

    return (
        <div style={{
            background: colors.background.surface,
            border: `1px solid ${colors.border.default}`,
            borderRadius: theme.borderRadius.xl,
            padding: '24px',
            boxShadow: theme.shadows.sm,
            height: '100%',
            display: 'flex',
            flexDirection: 'column'
        }}>
            <div className="flex items-center justify-between mb-6">
                <h3 className="text-lg font-bold text-white">Risk Distribution</h3>
            </div>

            {/* Tabs */}
            <div className="flex p-1 bg-card rounded-lg mb-6 border border-border">
                <button
                    onClick={() => setActiveTab('type')}
                    className={`flex-1 py-1.5 text-xs font-medium rounded-md transition-all ${activeTab === 'type'
                            ? 'bg-accent text-white shadow-sm'
                            : 'text-muted-foreground hover:text-white hover:bg-muted'
                        }`}
                >
                    By Type
                </button>
                <button
                    onClick={() => setActiveTab('asset')}
                    className={`flex-1 py-1.5 text-xs font-medium rounded-md transition-all ${activeTab === 'asset'
                            ? 'bg-accent text-white shadow-sm'
                            : 'text-muted-foreground hover:text-white hover:bg-muted'
                        }`}
                >
                    By Asset
                </button>
                <button
                    onClick={() => setActiveTab('confidence')}
                    className={`flex-1 py-1.5 text-xs font-medium rounded-md transition-all ${activeTab === 'confidence'
                            ? 'bg-accent text-white shadow-sm'
                            : 'text-muted-foreground hover:text-white hover:bg-muted'
                        }`}
                >
                    By Confidence
                </button>
            </div>

            <div className="space-y-4 flex-1 overflow-auto">
                {sortedData.map(([label, count]) => {
                    const percentage = total > 0 ? (count / total) * 100 : 0;
                    return (
                        <div key={label} className="space-y-1">
                            <div className="flex justify-between text-sm">
                                <span className="text-slate-200 truncate pr-4" title={label}>{label}</span>
                                <span className="text-muted-foreground whitespace-nowrap">{count} ({percentage.toFixed(0)}%)</span>
                            </div>
                            <div className="h-2 bg-muted rounded-full overflow-hidden">
                                <div
                                    className={`h-full rounded-full transition-all duration-500 ease-out ${activeTab === 'confidence'
                                            ? label.includes('High') || label === '> 90%' ? 'bg-green-500' : 'bg-blue-500'
                                            : 'bg-blue-500'
                                        }`}
                                    style={{ width: `${percentage}%` }}
                                />
                            </div>
                        </div>
                    );
                })}
                {sortedData.length === 0 && (
                    <div className="text-center text-slate-500 py-8 text-sm">
                        No data available
                    </div>
                )}
            </div>
        </div>
    );
}
