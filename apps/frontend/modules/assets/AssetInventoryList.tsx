'use client';

import React, { useMemo, useState } from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';
import { Node } from '@/types';

interface AssetListProps {
    assets: Node[];
    onAssetClick: (assetId: string) => void;
}

export default function AssetList({ assets, onAssetClick }: AssetListProps) {
    const [filter, setFilter] = useState('');

    const filteredAssets = useMemo(() => {
        if (!filter) return assets;
        const lower = filter.toLowerCase();
        return assets.filter(a =>
            (a.label || '').toLowerCase().includes(lower) ||
            (a.type || '').toLowerCase().includes(lower)
        );
    }, [assets, filter]);

    return (
        <div>
            {/* Toolbar */}
            <div style={{ marginBottom: '24px', display: 'flex', gap: '16px' }}>
                <input
                    type="text"
                    placeholder="Search assets..."
                    value={filter}
                    onChange={e => setFilter(e.target.value)}
                    style={{
                        padding: '10px 16px',
                        borderRadius: theme.borderRadius.lg,
                        border: `1px solid ${colors.border.default}`,
                        backgroundColor: colors.background.surface,
                        fontSize: theme.fontSize.base,
                        flex: 1,
                        maxWidth: '400px',
                    }}
                />
            </div>

            {/* List Header */}
            <div style={{
                display: 'grid',
                gridTemplateColumns: 'minmax(200px, 2fr) 1fr 1fr 1fr 100px',
                padding: '12px 16px',
                borderBottom: `1px solid ${colors.border.subtle}`,
                color: colors.text.muted,
                fontWeight: theme.fontWeight.bold,
                fontSize: theme.fontSize.xs,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
            }}>
                <div>Asset Name</div>
                <div>Type</div>
                <div>Location</div>
                <div>Risk Score</div>
                <div style={{ textAlign: 'right' }}>Action</div>
            </div>

            {/* Items */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '8px' }}>
                {filteredAssets.length === 0 ? (
                    <div style={{ padding: '32px', textAlign: 'center', color: colors.text.muted }}>
                        No assets found matching "{filter}"
                    </div>
                ) : (
                    filteredAssets.map(asset => (
                        <div
                            key={asset.id}
                            onClick={() => onAssetClick(asset.id)}
                            style={{
                                display: 'grid',
                                gridTemplateColumns: 'minmax(200px, 2fr) 1fr 1fr 1fr 100px',
                                alignItems: 'center',
                                padding: '16px',
                                backgroundColor: colors.background.surface,
                                border: `1px solid ${colors.border.subtle}`,
                                borderRadius: theme.borderRadius.lg,
                                cursor: 'pointer',
                                transition: 'all 0.2s ease',
                            }}
                            onMouseEnter={(e) => {
                                e.currentTarget.style.borderColor = colors.border.strong;
                                e.currentTarget.style.transform = 'translateY(-1px)';
                                e.currentTarget.style.boxShadow = theme.shadows.sm;
                            }}
                            onMouseLeave={(e) => {
                                e.currentTarget.style.borderColor = colors.border.subtle;
                                e.currentTarget.style.transform = 'translateY(0)';
                                e.currentTarget.style.boxShadow = 'none';
                            }}
                        >
                            <div style={{ fontWeight: theme.fontWeight.bold, color: colors.text.primary }}>
                                {asset.label}
                            </div>
                            <div>
                                <span style={{
                                    padding: '4px 8px',
                                    borderRadius: theme.borderRadius.sm,
                                    backgroundColor: colors.background.muted,
                                    color: colors.nodeColors.asset,
                                    fontSize: theme.fontSize.xs,
                                    fontWeight: theme.fontWeight.bold,
                                    textTransform: 'uppercase',
                                }}>
                                    {asset.type}
                                </span>
                            </div>
                            <div style={{ color: colors.text.secondary, fontSize: theme.fontSize.sm, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                {asset.metadata?.path || '-'}
                            </div>
                            <div>
                                <span style={{
                                    color: asset.risk_score >= 70 ? colors.state.risk : (asset.risk_score >= 40 ? colors.state.warning : colors.state.success),
                                    fontWeight: theme.fontWeight.bold,
                                }}>
                                    {asset.risk_score}
                                </span>
                            </div>
                            <div style={{ textAlign: 'right', color: colors.text.muted }}>
                                →
                            </div>
                        </div>
                    ))
                )}
            </div>
        </div>
    );
}
