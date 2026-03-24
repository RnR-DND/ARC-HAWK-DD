'use client';

import React from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';
import { Node } from '@/types';

interface AssetDetailPanelProps {
    asset: Node | null;
    isOpen: boolean;
    onClose: () => void;
    onViewLineage: (assetId: string) => void;
}

export default function AssetDetailPanel({
    asset,
    isOpen,
    onClose,
    onViewLineage,
}: AssetDetailPanelProps) {
    if (!asset) return null;

    return (
        <>
            {/* Backdrop */}
            {isOpen && (
                <div
                    onClick={onClose}
                    style={{
                        position: 'fixed',
                        top: 0,
                        left: 0,
                        right: 0,
                        bottom: 0,
                        backgroundColor: 'rgba(0,0,0,0.4)',
                        zIndex: theme.zIndex.overlay - 1,
                        backdropFilter: 'blur(2px)',
                    }}
                />
            )}

            {/* Panel */}
            <div
                style={{
                    position: 'fixed',
                    top: 0,
                    right: 0,
                    bottom: 0,
                    width: '480px',
                    backgroundColor: colors.background.surface,
                    borderLeft: `1px solid ${colors.border.default}`,
                    boxShadow: theme.shadows.xl,
                    zIndex: theme.zIndex.overlay,
                    transform: isOpen ? 'translateX(0)' : 'translateX(100%)',
                    transition: 'transform 0.3s cubic-bezier(0.16, 1, 0.3, 1)',
                    display: 'flex',
                    flexDirection: 'column',
                }}
            >
                {/* Header */}
                <div style={{
                    padding: '24px',
                    borderBottom: `1px solid ${colors.border.subtle}`,
                    display: 'flex',
                    alignItems: 'flex-start',
                    justifyContent: 'space-between',
                }}>
                    <div>
                        <div style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px',
                            marginBottom: '8px',
                        }}>
                            <span style={{
                                backgroundColor: colors.background.muted,
                                border: `1px solid ${colors.nodeColors.asset}`,
                                color: colors.nodeColors.asset,
                                padding: '4px 8px',
                                borderRadius: theme.borderRadius.sm,
                                fontSize: theme.fontSize.xs,
                                fontWeight: theme.fontWeight.bold,
                                textTransform: 'uppercase',
                            }}>
                                {asset.type}
                            </span>
                            <span style={{
                                backgroundColor: asset.risk_score >= 70 ? colors.state.risk : colors.state.info,
                                color: '#fff',
                                padding: '4px 8px',
                                borderRadius: theme.borderRadius.sm,
                                fontSize: theme.fontSize.xs,
                                fontWeight: theme.fontWeight.bold,
                            }}>
                                Risk: {asset.risk_score}
                            </span>
                        </div>
                        <h2 style={{
                            fontSize: '24px',
                            fontWeight: theme.fontWeight.bold,
                            color: colors.text.primary,
                            margin: 0,
                            lineHeight: 1.2,
                        }}>
                            {asset.label}
                        </h2>
                    </div>
                    <button
                        onClick={onClose}
                        style={{
                            background: 'transparent',
                            border: 'none',
                            fontSize: '24px',
                            cursor: 'pointer',
                            color: colors.text.muted,
                        }}
                    >
                        &times;
                    </button>
                </div>

                {/* Content */}
                <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>

                    {/* Why is it risky? */}
                    <div style={{ marginBottom: '32px' }}>
                        <h3 style={sectionTitleStyle}>Why is this risky?</h3>
                        <div style={{
                            backgroundColor: colors.background.primary,
                            borderRadius: theme.borderRadius.lg,
                            padding: '16px',
                            border: `1px solid ${colors.border.subtle}`,
                        }}>
                            {asset.risk_score > 50 ? (
                                <p style={{ margin: 0, color: colors.text.secondary }}>
                                    This asset contains <strong>sensitive data</strong> and is exposed to potential vulnerabilities.
                                    Immediate review is recommended.
                                </p>
                            ) : (
                                <p style={{ margin: 0, color: colors.text.secondary }}>
                                    This asset has a low risk profile but should still be monitored for changes.
                                </p>
                            )}
                        </div>
                    </div>

                    {/* Metadata */}
                    <div style={{ marginBottom: '32px' }}>
                        <h3 style={sectionTitleStyle}>Asset Details</h3>
                        <div style={{ display: 'grid', gap: '12px' }}>
                            <DetailRow label="Location" value={asset.metadata?.path || 'N/A'} />
                            <DetailRow label="Type" value={asset.metadata?.type || asset.type} />
                            <DetailRow label="Owner" value={asset.metadata?.owner || 'Unassigned'} />
                            <DetailRow label="Last Scanned" value={new Date().toLocaleDateString()} />
                        </div>
                    </div>

                    {/* PII Found */}
                    <div style={{ marginBottom: '32px' }}>
                        <h3 style={sectionTitleStyle}>Data Elements Found</h3>
                        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
                            {['EMAIL_ADDRESS', 'PHONE_NUMBER'].map(tag => (
                                <span key={tag} style={{
                                    padding: '6px 12px',
                                    borderRadius: theme.borderRadius.full,
                                    backgroundColor: colors.background.muted,
                                    color: colors.nodeColors.pii,
                                    border: `1px solid ${colors.nodeColors.pii}`,
                                    fontSize: theme.fontSize.sm,
                                    fontWeight: theme.fontWeight.medium,
                                }}>
                                    {tag}
                                </span>
                            ))}
                            {/* Mock data for now, ideally populated from asset.metadata or edges */}
                        </div>
                    </div>

                </div>

                {/* Footer Actions */}
                <div style={{
                    padding: '24px',
                    borderTop: `1px solid ${colors.border.default}`,
                    display: 'flex',
                    gap: '12px',
                }}>
                    <button
                        onClick={() => onViewLineage(asset.id)}
                        style={{
                            flex: 1,
                            padding: '12px',
                            backgroundColor: colors.nodeColors.system,
                            color: '#fff',
                            border: 'none',
                            borderRadius: theme.borderRadius.lg,
                            fontWeight: theme.fontWeight.bold,
                            cursor: 'pointer',
                            fontSize: theme.fontSize.base,
                        }}
                    >
                        Explore Lineage Graph
                    </button>
                </div>
            </div>
        </>
    );
}

const sectionTitleStyle = {
    fontSize: theme.fontSize.sm,
    fontWeight: theme.fontWeight.bold,
    color: colors.text.muted,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    marginBottom: '12px',
};

function DetailRow({ label, value }: { label: string, value: string }) {
    return (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ color: colors.text.secondary }}>{label}</span>
            <span style={{ fontWeight: theme.fontWeight.medium, color: colors.text.primary }}>{value}</span>
        </div>
    );
}
