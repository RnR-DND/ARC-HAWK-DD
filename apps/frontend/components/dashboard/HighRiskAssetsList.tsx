'use client';

import React from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';
import { Node } from '@/types';

interface HighRiskAssetsListProps {
    assets: Node[];
    onAssetClick?: (assetId: string) => void;
}

export default function HighRiskAssetsList({ assets, onAssetClick }: HighRiskAssetsListProps) {
    const highRiskAssets = assets
        .filter(asset => asset.risk_score >= 70)
        .sort((a, b) => b.risk_score - a.risk_score);

    return (
        <div style={{
            background: colors.background.surface,
            border: `1px solid ${colors.border.default}`,
            borderRadius: theme.borderRadius.xl,
            padding: '24px',
            boxShadow: theme.shadows.sm,
        }}>
            <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: '16px',
            }}>
                <h3 style={{
                    fontSize: theme.fontSize.lg,
                    fontWeight: theme.fontWeight.bold,
                    color: colors.text.primary,
                    margin: 0,
                }}>
                    Priority Actions Needed
                </h3>
                <span style={{
                    fontSize: theme.fontSize.sm,
                    color: colors.text.secondary,
                    backgroundColor: colors.background.elevated,
                    padding: '4px 12px',
                    borderRadius: theme.borderRadius.full,
                }}>
                    {highRiskAssets.length} Critical Assets
                </span>
            </div>

            {highRiskAssets.length === 0 ? (
                <div style={{ padding: '24px', textAlign: 'center', color: colors.text.muted }}>
                    No high-risk assets found. Great job!
                </div>
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                    {highRiskAssets.map(asset => (
                        <div
                            key={asset.id}
                            onClick={() => onAssetClick?.(asset.id)}
                            style={{
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                                padding: '16px',
                                border: `1px solid ${colors.border.subtle}`,
                                borderRadius: theme.borderRadius.lg,
                                cursor: 'pointer',
                                transition: 'all 0.2s ease',
                                backgroundColor: colors.background.primary,
                            }}
                            onMouseEnter={(e) => {
                                e.currentTarget.style.borderColor = colors.border.strong;
                                e.currentTarget.style.transform = 'translateX(4px)';
                            }}
                            onMouseLeave={(e) => {
                                e.currentTarget.style.borderColor = colors.border.subtle;
                                e.currentTarget.style.transform = 'translateX(0)';
                            }}
                        >
                            <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                                <div style={{
                                    width: '40px',
                                    height: '40px',
                                    borderRadius: '8px',
                                    backgroundColor: colors.background.muted,
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    fontSize: '20px',
                                    border: `1px solid ${colors.nodeColors.asset}`,
                                    color: colors.nodeColors.asset,
                                }}>
                                    📦
                                </div>
                                <div>
                                    <div style={{
                                        fontWeight: theme.fontWeight.bold,
                                        color: colors.text.primary,
                                        fontSize: theme.fontSize.base,
                                    }}>
                                        {asset.label}
                                    </div>
                                    <div style={{
                                        fontSize: theme.fontSize.sm,
                                        color: colors.text.secondary,
                                        marginTop: '2px',
                                    }}>
                                        {asset.type.toUpperCase()} • {asset.metadata?.path || 'Unknown Path'}
                                    </div>
                                </div>
                            </div>

                            <div style={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
                                <div style={{ textAlign: 'right' }}>
                                    <div style={{
                                        fontSize: theme.fontSize.xs,
                                        color: colors.text.muted,
                                        fontWeight: theme.fontWeight.bold,
                                        textTransform: 'uppercase',
                                        marginBottom: '4px',
                                    }}>
                                        Risk Score
                                    </div>
                                    <div style={{
                                        fontSize: theme.fontSize.xl,
                                        fontWeight: theme.fontWeight.extrabold,
                                        color: asset.risk_score >= 90 ? colors.state.risk : colors.state.warning,
                                    }}>
                                        {asset.risk_score}
                                    </div>
                                </div>
                                <div style={{
                                    color: colors.text.muted,
                                    fontSize: '20px',
                                }}>
                                    →
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
