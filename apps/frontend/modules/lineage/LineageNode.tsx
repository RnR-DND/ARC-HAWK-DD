'use client';

import React from 'react';
import { Handle, Position } from 'reactflow';
import { Server, Database, Shield, FileText } from 'lucide-react';
import type { LineageNode as LineageNodeType } from './lineage.types';

interface LineageNodeProps {
    data: LineageNodeType;
    id: string;
}

export default function LineageNode({ data, id }: LineageNodeProps) {
    const { label, type, metadata } = data;
    const risk_score = (metadata as any)?.risk_score || 0;
    const review_status = (metadata as any)?.review_status;
    const expanded = (data as any).expanded;
    const onExpand = (data as any).onExpand;
    const childCount = (data as any).childCount;

    // Simple color scheme
    const getNodeColors = () => {
        switch (type) {
            case 'system':
                return { bg: '#1e293b', border: '#3b82f6', text: '#f8fafc' };
            case 'asset':
            case 'file':
            case 'table':
                return { bg: '#1e293b', border: '#a855f7', text: '#f8fafc' };
            case 'pii_category':
                if (risk_score >= 70) return { bg: '#1e293b', border: '#ef4444', text: '#f8fafc' };
                if (risk_score >= 40) return { bg: '#1e293b', border: '#f97316', text: '#f8fafc' };
                return { bg: '#1e293b', border: '#22c55e', text: '#f8fafc' };
            default:
                return { bg: '#1e293b', border: '#64748b', text: '#f8fafc' };
        }
    };

    const colors = getNodeColors();

    const getIcon = () => {
        switch (type) {
            case 'system': return <Server size={16} strokeWidth={2} />;
            case 'asset':
            case 'table': return <Database size={16} strokeWidth={2} />;
            case 'file': return <FileText size={16} strokeWidth={2} />;
            case 'pii_category': return <Shield size={16} strokeWidth={2} />;
            default: return <Shield size={16} strokeWidth={2} />;
        }
    };

    const getNodeSize = () => {
        switch (type) {
            case 'system': return { width: 260, minHeight: 90 };
            case 'asset':
            case 'file':
            case 'table': return { width: 240, minHeight: 85 };
            default: return { width: 220, minHeight: 80 };
        }
    };

    const size = getNodeSize();

    return (
        <div
            style={{
                background: colors.bg,
                border: `2px solid ${colors.border}`,
                borderRadius: '8px',
                minWidth: size.width,
                maxWidth: size.width,
                minHeight: size.minHeight,
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.2)',
                fontFamily: 'Inter, sans-serif',
                overflow: 'hidden',
                transition: 'box-shadow 0.15s',
                cursor: 'pointer',
                opacity: review_status === 'false_positive' ? 0.5 : 1,
            }}
        >
            <Handle
                type="target"
                position={Position.Left}
                style={{
                    background: colors.border,
                    width: 8,
                    height: 8,
                    border: '2px solid #1e293b',
                    left: -5,
                }}
            />

            {/* Header */}
            <div
                style={{
                    padding: '12px 16px',
                    borderBottom: `1px solid ${colors.border}40`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                }}
            >
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <div style={{ color: colors.border }}>
                        {getIcon()}
                    </div>
                    <span
                        style={{
                            fontSize: '11px',
                            fontWeight: 600,
                            color: '#94a3b8',
                            textTransform: 'uppercase',
                            letterSpacing: '0.05em',
                        }}
                    >
                        {type.replace('_', ' ')}
                    </span>
                </div>

                {risk_score >= 1 && (
                    <div
                        style={{
                            fontSize: '11px',
                            fontWeight: 600,
                            padding: '2px 8px',
                            borderRadius: '4px',
                            background: risk_score >= 70
                                ? 'rgba(239, 68, 68, 0.1)'
                                : risk_score >= 40
                                    ? 'rgba(249, 115, 22, 0.1)'
                                    : 'rgba(34, 197, 94, 0.1)',
                            color: risk_score >= 70
                                ? '#ef4444'
                                : risk_score >= 40
                                    ? '#f97316'
                                    : '#22c55e',
                            border: `1px solid ${risk_score >= 70
                                ? 'rgba(239, 68, 68, 0.2)'
                                : risk_score >= 40
                                    ? 'rgba(249, 115, 22, 0.2)'
                                    : 'rgba(34, 197, 94, 0.2)'}`,
                        }}
                    >
                        {risk_score}
                    </div>
                )}
            </div>

            {/* Body */}
            <div style={{ padding: '16px' }}>
                <div
                    style={{
                        fontWeight: 500,
                        fontSize: '14px',
                        color: colors.text,
                        marginBottom: '8px',
                        wordBreak: 'break-word',
                        lineHeight: '1.4',
                        display: '-webkit-box',
                        WebkitLineClamp: 2,
                        WebkitBoxOrient: 'vertical',
                        overflow: 'hidden',
                    }}
                    title={label}
                >
                    {label}
                </div>

                {(metadata as any)?.environment && (
                    <div
                        style={{
                            fontSize: '12px',
                            color: '#94a3b8',
                            marginTop: '8px',
                        }}
                    >
                        {(metadata as any).environment}
                    </div>
                )}

                {childCount && childCount > 0 && (
                    <div
                        style={{
                            fontSize: '11px',
                            color: '#64748b',
                            marginTop: '8px',
                        }}
                    >
                        {childCount} {childCount === 1 ? 'child' : 'children'}
                    </div>
                )}
            </div>

            <Handle
                type="source"
                position={Position.Right}
                style={{
                    background: colors.border,
                    width: 8,
                    height: 8,
                    border: '2px solid #1e293b',
                    right: -5,
                }}
            />
        </div>
    );
}
