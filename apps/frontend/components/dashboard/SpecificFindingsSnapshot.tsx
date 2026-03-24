'use client';

import React from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';
import { AlertTriangle, Database, FileText, HardDrive, CheckCircle } from 'lucide-react';

interface FindingSnapshot {
    id: string;
    assetName: string;
    assetPath: string;
    field: string;
    piiType: string;
    confidence: number;
    risk: 'High' | 'Medium' | 'Low';
    sourceType: 'Database' | 'Filesystem' | 'S3';
}

interface SpecificFindingsSnapshotProps {
    findings: FindingSnapshot[];
}

export default function SpecificFindingsSnapshot({ findings }: SpecificFindingsSnapshotProps) {
    const getSourceIcon = (type: string) => {
        switch (type) {
            case 'Database': return <Database size={16} />;
            case 'Filesystem': return <HardDrive size={16} />;
            case 'S3': return <FileText size={16} />;
            default: return <Database size={16} />;
        }
    };

    return (
        <div style={{
            background: colors.background.surface,
            border: `1px solid ${colors.border.default}`,
            borderRadius: theme.borderRadius.xl,
            padding: '24px',
            boxShadow: theme.shadows.sm,
            height: '100%',
        }}>
            <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: '20px',
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <h3 style={{
                        fontSize: theme.fontSize.lg,
                        fontWeight: theme.fontWeight.bold,
                        color: colors.text.primary,
                        margin: 0,
                    }}>
                        Recent High-Risk Findings
                    </h3>
                </div>
                <span style={{
                    fontSize: theme.fontSize.sm,
                    color: colors.text.muted,
                }}>
                    High signal, non-sensitive examples
                </span>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {findings.map((finding) => (
                    <div
                        key={finding.id}
                        style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            padding: '12px 16px',
                            border: `1px solid ${colors.border.subtle}`,
                            borderRadius: theme.borderRadius.lg,
                            backgroundColor: colors.background.primary,
                            transition: 'all 0.2s ease',
                        }}
                    >
                        <div style={{ display: 'flex', alignItems: 'center', gap: '16px', flex: 1 }}>
                            {/* Icon */}
                            <div style={{
                                width: '36px',
                                height: '36px',
                                borderRadius: '8px',
                                backgroundColor: finding.risk === 'High' ? 'rgba(239, 68, 68, 0.1)' : 'rgba(59, 130, 246, 0.1)',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                color: finding.risk === 'High' ? colors.state.risk : colors.state.info,
                            }}>
                                <AlertTriangle size={18} />
                            </div>

                            {/* Details */}
                            <div style={{ flex: 1 }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                                    <span style={{
                                        fontWeight: theme.fontWeight.bold,
                                        color: colors.text.primary,
                                        fontSize: theme.fontSize.sm,
                                    }}>
                                        {finding.piiType}
                                    </span>
                                    <span style={{ color: colors.text.muted, fontSize: theme.fontSize.xs }}>detected in</span>
                                    <span style={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        gap: '4px',
                                        color: colors.text.secondary,
                                        fontSize: theme.fontSize.sm,
                                        fontWeight: 500
                                    }}>
                                        {getSourceIcon(finding.sourceType)}
                                        {finding.assetName} ▸ <span className="font-mono text-xs">{finding.assetPath || finding.field}</span>
                                    </span>
                                </div>
                            </div>
                        </div>

                        {/* Confidence */}
                        <div style={{ textAlign: 'right', minWidth: '100px' }}>
                            <div style={{
                                fontSize: theme.fontSize.xs,
                                color: colors.text.muted,
                                marginBottom: '2px',
                            }}>
                                Confidence
                            </div>
                            <div style={{
                                fontSize: theme.fontSize.sm,
                                fontWeight: theme.fontWeight.bold,
                                color: finding.confidence > 0.9 ? colors.state.success : colors.state.warning,
                            }}>
                                {(finding.confidence * 100).toFixed(0)}%
                            </div>
                        </div>
                    </div>
                ))}

                {findings.length === 0 && (
                    <div style={{
                        padding: '32px',
                        textAlign: 'center',
                        color: colors.text.muted,
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        gap: '12px'
                    }}>
                        <CheckCircle size={32} color={colors.state.success} />
                        <div>No high-risk findings detected in recent scans.</div>
                    </div>
                )}
            </div>
        </div>
    );
}
