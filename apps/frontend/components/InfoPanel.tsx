'use client';

import React, { useEffect, useState } from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';
import { assetsApi } from '@/services/assets.api';
import { Asset } from '@/types';
import LoadingState from './LoadingState';

import { BaseNode } from '@/modules/lineage/lineage.types';

interface InfoPanelProps {
    nodeId: string;
    nodeData?: BaseNode;
    onClose: () => void;
}

export default function InfoPanel({ nodeId, nodeData, onClose }: InfoPanelProps) {
    const [asset, setAsset] = useState<Asset | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        // If it's a System, Classification, or Finding node, we rely on nodeData, NOT the API
        // Findings MIGHT have an API but let's stick to metadata for now to avoid 404s
        const isAsset = nodeData?.type === 'asset';

        if (!isAsset) {
            setLoading(false);
            return;
        }

        async function fetchAssetDetails() {
            setLoading(true);
            setError(null);
            try {
                const data = await assetsApi.getAsset(nodeId);
                setAsset(data);
            } catch (err: any) {
                console.error('Error fetching asset:', err);
                // Fallback to nodeData if API fails
                if (nodeData) {
                    console.warn('Falling back to node metadata');
                    setError(null);
                } else {
                    setError(err.message || 'Failed to load asset details');
                }
            } finally {
                setLoading(false);
            }
        }

        fetchAssetDetails();
    }, [nodeId, nodeData]);

    return (
        <>
            {/* Backdrop */}
            <div
                onClick={onClose}
                style={{
                    position: 'fixed',
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    backgroundColor: 'rgba(0, 0, 0, 0.3)',
                    zIndex: theme.zIndex.overlay,
                    backdropFilter: 'blur(2px)',
                }}
            />

            {/* Panel */}
            <div
                style={{
                    position: 'fixed',
                    top: 0,
                    right: 0,
                    bottom: 0,
                    width: '480px',
                    maxWidth: '90vw',
                    backgroundColor: colors.background.surface,
                    boxShadow: theme.shadows['2xl'],
                    zIndex: theme.zIndex.modal,
                    display: 'flex',
                    flexDirection: 'column',
                    animation: 'slideIn 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                }}
            >
                {/* Header */}
                <div
                    style={{
                        padding: '24px',
                        borderBottom: `1px solid ${colors.border.subtle}`,
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        background: `linear-gradient(to bottom, #ffffff, ${colors.background.elevated})`,
                    }}
                >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <div style={{
                            width: 40, height: 40,
                            borderRadius: '8px',
                            background: colors.background.surface,
                            border: `1px solid ${colors.border.default}`,
                            display: 'flex', alignItems: 'center', justifyContent: 'center',
                            fontSize: '20px'
                        }}>
                            {nodeData?.type === 'system' ? '🏢' :
                                nodeData?.type === 'asset' ? '📦' :
                                    nodeData?.type === 'pii_category' ? '🔑' : '📁'}
                        </div>
                        <div>
                            <h2
                                style={{
                                    fontSize: theme.fontSize.lg,
                                    fontWeight: 800,
                                    color: colors.text.primary,
                                    margin: 0,
                                    lineHeight: 1.2
                                }}
                            >
                                {nodeData?.label || 'Details'}
                            </h2>
                            <span style={{
                                fontSize: '11px',
                                color: colors.text.secondary,
                                textTransform: 'uppercase',
                                fontWeight: 700,
                                letterSpacing: '0.05em'
                            }}>
                                {nodeData?.type?.toUpperCase() || 'INFO'}
                            </span>
                        </div>
                    </div>

                    <button
                        onClick={onClose}
                        style={{
                            background: 'transparent',
                            border: 'none',
                            fontSize: '20px',
                            cursor: 'pointer',
                            padding: '8px',
                            borderRadius: '6px',
                            color: colors.text.muted,
                            transition: 'all 0.2s',
                        }}
                        onMouseEnter={(e) => {
                            e.currentTarget.style.backgroundColor = colors.background.surface;
                            e.currentTarget.style.color = colors.text.primary;
                        }}
                        onMouseLeave={(e) => {
                            e.currentTarget.style.backgroundColor = 'transparent';
                            e.currentTarget.style.color = colors.text.muted;
                        }}
                    >
                        ✕
                    </button>
                </div>

                {/* Content */}
                <div
                    style={{
                        flex: 1,
                        overflowY: 'auto',
                        padding: '24px',
                    }}
                >
                    {loading && <LoadingState message="Loading asset details..." />}

                    {error && (
                        <div
                            style={{
                                padding: '16px',
                                backgroundColor: colors.red[50],
                                border: `1px solid ${colors.red[200]}`,
                                borderRadius: theme.borderRadius.md,
                                color: colors.red[700],
                            }}
                        >
                            {error}
                        </div>
                    )}

                    {/* Generic Node View (for System, Classification) */}
                    {!asset && nodeData && !loading && (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                            <Section title={nodeData.type || 'Node Details'}>
                                <PropertyRow label="Name" value={nodeData.label} />
                                <PropertyRow label="Type" value={nodeData.type} />
                                <PropertyRow label="ID" value={nodeData.id} copyable />
                            </Section>

                            {nodeData.metadata && (
                                <Section title="Metadata">
                                    {Object.entries(nodeData.metadata).map(([key, value]) => (
                                        <PropertyRow
                                            key={key}
                                            label={key.replace(/_/g, ' ')}
                                            value={
                                                typeof value === 'object' && value !== null ? (
                                                    <pre style={{ margin: 0, fontSize: '11px', fontFamily: 'monospace' }}>
                                                        {JSON.stringify(value, null, 2)}
                                                    </pre>
                                                ) : (
                                                    String(value)
                                                )
                                            }
                                        />
                                    ))}
                                </Section>
                            )}

                            {nodeData.type === 'asset' && nodeData.metadata?.risk_score !== undefined && (
                                <Section title="Risk Assessment">
                                    <PropertyRow
                                        label="Risk Score"
                                        value={
                                            <div
                                                style={{
                                                    display: 'inline-block',
                                                    padding: '4px 12px',
                                                    borderRadius: theme.borderRadius.full,
                                                    backgroundColor:
                                                        (nodeData.metadata.risk_score as number) >= 80 ? colors.red[100] :
                                                            (nodeData.metadata.risk_score as number) >= 60 ? colors.amber[100] : colors.emerald[100],
                                                    color:
                                                        (nodeData.metadata.risk_score as number) >= 80 ? colors.red[700] :
                                                            (nodeData.metadata.risk_score as number) >= 60 ? colors.amber[700] : colors.emerald[700],
                                                    fontWeight: theme.fontWeight.bold,
                                                }}
                                            >
                                                {nodeData.metadata.risk_score}/100
                                            </div>
                                        }
                                    />
                                </Section>
                            )}
                        </div>
                    )}

                    {asset && (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                            {/* Asset Name */}
                            <Section title="Asset">
                                <PropertyRow label="Name" value={asset.name} />
                                <PropertyRow label="Type" value={asset.asset_type} />
                                <PropertyRow
                                    label="Path"
                                    value={asset.path}
                                    highlight
                                    copyable
                                />
                            </Section>

                            {/* Location */}
                            <Section title="Location">
                                <PropertyRow label="Source System" value={asset.source_system} />
                                <PropertyRow label="Data Source" value={asset.data_source} />
                                <PropertyRow label="Host" value={asset.host} />
                                <PropertyRow label="Environment" value={asset.environment} />
                            </Section>

                            {/* Risk & Findings */}
                            <Section title="Risk Assessment">
                                <PropertyRow
                                    label="Risk Score"
                                    value={
                                        <div
                                            style={{
                                                display: 'inline-block',
                                                padding: '4px 12px',
                                                borderRadius: theme.borderRadius.full,
                                                backgroundColor:
                                                    asset.risk_score >= 80
                                                        ? colors.red[100]
                                                        : asset.risk_score >= 60
                                                            ? colors.amber[100]
                                                            : colors.emerald[100],
                                                color:
                                                    asset.risk_score >= 80
                                                        ? colors.red[700]
                                                        : asset.risk_score >= 60
                                                            ? colors.amber[700]
                                                            : colors.emerald[700],
                                                fontWeight: theme.fontWeight.bold,
                                            }}
                                        >
                                            {asset.risk_score}/100
                                        </div>
                                    }
                                />
                                <PropertyRow label="Total Findings" value={asset.total_findings} />
                            </Section>

                            {/* Metadata */}
                            {asset.owner && (
                                <Section title="Ownership">
                                    <PropertyRow label="Owner" value={asset.owner} />
                                </Section>
                            )}

                            {/* Timestamps */}
                            <Section title="Metadata">
                                <PropertyRow
                                    label="Created"
                                    value={new Date(asset.created_at).toLocaleString()}
                                />
                                <PropertyRow
                                    label="Last Updated"
                                    value={new Date(asset.updated_at).toLocaleString()}
                                />
                            </Section>
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div
                    style={{
                        padding: '16px 24px',
                        borderTop: `1px solid ${colors.border.default}`,
                        backgroundColor: colors.neutral[50],
                    }}
                >
                    <button
                        onClick={onClose}
                        style={{
                            width: '100%',
                            padding: '10px',
                            backgroundColor: '#ffffff',
                            color: colors.text.primary,
                            border: `1px solid ${colors.border.strong}`,
                            borderRadius: '8px',
                            fontSize: '13px',
                            fontWeight: 600,
                            cursor: 'pointer',
                            transition: 'all 0.2s ease',
                            boxShadow: '0 1px 2px rgba(0,0,0,0.05)'
                        }}
                        onMouseEnter={(e) => {
                            e.currentTarget.style.backgroundColor = colors.background.surface;
                            e.currentTarget.style.transform = 'translateY(-1px)';
                            e.currentTarget.style.boxShadow = '0 2px 4px rgba(0,0,0,0.1)';
                        }}
                        onMouseLeave={(e) => {
                            e.currentTarget.style.backgroundColor = '#ffffff';
                            e.currentTarget.style.transform = 'translateY(0)';
                            e.currentTarget.style.boxShadow = '0 1px 2px rgba(0,0,0,0.05)';
                        }}
                    >
                        Close Panel
                    </button>
                </div>
            </div >

            {/* CSS for slide-in animation */}
            < style jsx > {`
        @keyframes slideIn {
          from {
            transform: translateX(100%);
          }
          to {
            transform: translateX(0);
          }
        }
      `}</style >
        </>
    );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
    return (
        <div>
            <h3
                style={{
                    fontSize: '11px',
                    fontWeight: 700,
                    color: colors.text.muted,
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                    marginBottom: '16px',
                    borderBottom: `1px solid ${colors.border.subtle}`,
                    paddingBottom: '8px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px'
                }}
            >
                {title}
            </h3>
            <div
                style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '12px',
                }}
            >
                {children}
            </div>
        </div>
    );
}

function PropertyRow({
    label,
    value,
    highlight,
    copyable,
}: {
    label: string;
    value: React.ReactNode;
    highlight?: boolean;
    copyable?: boolean;
}) {
    const [copied, setCopied] = useState(false);

    const handleCopy = () => {
        if (typeof value === 'string') {
            navigator.clipboard.writeText(value);
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        }
    };

    return (
        <div
            style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'flex-start',
                padding: highlight ? '12px' : '8px 0',
                backgroundColor: highlight ? colors.background.surface : 'transparent',
                borderRadius: highlight ? '8px' : 0,
                border: highlight ? `1px solid ${colors.border.subtle}` : 'none',
            }}
        >
            <div
                style={{
                    fontSize: '13px',
                    color: colors.text.secondary,
                    fontWeight: 500,
                    minWidth: '120px',
                    marginTop: '2px' // optical alignment
                }}
            >
                {label}
            </div>
            <div
                style={{
                    fontSize: '13px',
                    color: colors.text.primary,
                    fontWeight: 600,
                    textAlign: 'right',
                    flex: 1,
                    display: 'flex',
                    justifyContent: 'flex-end',
                    alignItems: 'center',
                    gap: '8px',
                    wordBreak: 'break-word',
                    lineHeight: 1.5
                }}
            >
                {value}
                {copyable && (
                    <button
                        onClick={handleCopy}
                        style={{
                            background: 'transparent',
                            border: 'none',
                            cursor: 'pointer',
                            fontSize: '14px',
                            padding: '4px',
                            opacity: 0.6,
                            transition: 'opacity 0.2s'
                        }}
                        onMouseEnter={(e) => e.currentTarget.style.opacity = '1'}
                        onMouseLeave={(e) => e.currentTarget.style.opacity = '0.6'}
                        title="Copy to clipboard"
                    >
                        {copied ? '✓' : '❐'}
                    </button>
                )}
            </div>
        </div>
    );
}
