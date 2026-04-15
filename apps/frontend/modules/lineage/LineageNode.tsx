'use client';

import React from 'react';
import { Handle, Position } from 'reactflow';
import { Server, Database, Shield, FileText, Table2 } from 'lucide-react';

// ─── Risk helpers ─────────────────────────────────────────────────────────────

const RISK_CONFIG: Record<string, { header: string; badge: string; badgeText: string; accent: string }> = {
    Critical: { header: '#fef2f2', badge: '#ef4444', badgeText: '#fff',  accent: '#ef4444' },
    High:     { header: '#fff7ed', badge: '#f97316', badgeText: '#fff',  accent: '#f97316' },
    Medium:   { header: '#fefce8', badge: '#eab308', badgeText: '#713f12', accent: '#eab308' },
    Low:      { header: '#f0fdf4', badge: '#22c55e', badgeText: '#14532d', accent: '#22c55e' },
};

const DEFAULT_RISK = { header: '#f8fafc', badge: '#64748b', badgeText: '#fff', accent: '#64748b' };

// ─── Node type config ─────────────────────────────────────────────────────────

const TYPE_CONFIG: Record<string, { icon: React.ReactNode; typeLabel: string; accentColor: string; headerBg: string }> = {
    system:       { icon: <Server   size={14} strokeWidth={2.5} />, typeLabel: 'System',   accentColor: '#3b82f6', headerBg: '#eff6ff' },
    asset:        { icon: <Database size={14} strokeWidth={2.5} />, typeLabel: 'Asset',    accentColor: '#10b981', headerBg: '#f0fdf4' },
    table:        { icon: <Table2   size={14} strokeWidth={2.5} />, typeLabel: 'Table',    accentColor: '#06b6d4', headerBg: '#ecfeff' },
    file:         { icon: <FileText size={14} strokeWidth={2.5} />, typeLabel: 'File',     accentColor: '#a855f7', headerBg: '#faf5ff' },
    pii_category: { icon: <Shield   size={14} strokeWidth={2.5} />, typeLabel: 'PII Type', accentColor: '#ef4444', headerBg: '#fef2f2' },
};
const DEFAULT_TYPE = { icon: <Shield size={14} strokeWidth={2.5} />, typeLabel: 'Node', accentColor: '#64748b', headerBg: '#f8fafc' };

// ─── Mini confidence bar ──────────────────────────────────────────────────────

function ConfBar({ score }: { score: number }) {
    const pct = Math.round(score * 100);
    const color = pct >= 85 ? '#22c55e' : pct >= 65 ? '#f59e0b' : '#ef4444';
    return (
        <div style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
            <div style={{ flex: 1, height: '4px', background: '#e2e8f0', borderRadius: '2px', overflow: 'hidden' }}>
                <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: '2px' }} />
            </div>
            <span style={{ fontSize: '10px', fontWeight: 600, color: '#64748b', width: '28px', textAlign: 'right' }}>{pct}%</span>
        </div>
    );
}

// ─── Main node ────────────────────────────────────────────────────────────────

interface LineageNodeProps {
    data: any;
    selected?: boolean;
}

export default function LineageNode({ data, selected }: LineageNodeProps) {
    const { label, type, metadata = {} } = data;

    const typeCfg = TYPE_CONFIG[type] ?? DEFAULT_TYPE;
    const riskLevel: string = metadata.risk_level ?? '';
    const riskCfg = RISK_CONFIG[riskLevel] ?? DEFAULT_RISK;

    // Shared card style
    const cardStyle: React.CSSProperties = {
        background: '#ffffff',
        border: `2px solid ${selected ? typeCfg.accentColor : '#e2e8f0'}`,
        borderRadius: '10px',
        overflow: 'hidden',
        fontFamily: 'Inter, -apple-system, sans-serif',
        boxShadow: selected
            ? `0 0 0 3px ${typeCfg.accentColor}30, 0 4px 12px rgba(0,0,0,0.12)`
            : '0 2px 8px rgba(0,0,0,0.06)',
        cursor: 'pointer',
        transition: 'box-shadow 0.15s, border-color 0.15s',
        minWidth: type === 'system' ? 240 : type === 'pii_category' ? 210 : 220,
        maxWidth: type === 'system' ? 240 : type === 'pii_category' ? 210 : 220,
    };

    // Left accent bar color
    const accentBar = type === 'pii_category' ? riskCfg.accent : typeCfg.accentColor;

    return (
        <div style={{ ...cardStyle, borderLeft: `4px solid ${accentBar}` }}>
            {/* Target handle */}
            <Handle
                type="target"
                position={Position.Left}
                style={{ background: accentBar, width: 8, height: 8, border: '2px solid #fff', left: -5 }}
            />

            {/* ── Header ── */}
            <div style={{
                padding: '8px 12px',
                background: type === 'pii_category' ? riskCfg.header : typeCfg.headerBg,
                borderBottom: '1px solid #f1f5f9',
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <span style={{ color: accentBar, display: 'flex', alignItems: 'center' }}>
                        {typeCfg.icon}
                    </span>
                    <span style={{
                        fontSize: '10px', fontWeight: 700, color: '#64748b',
                        textTransform: 'uppercase', letterSpacing: '0.07em',
                    }}>
                        {typeCfg.typeLabel}
                    </span>
                </div>

                {/* Risk badge for PII nodes */}
                {type === 'pii_category' && riskLevel && (
                    <span style={{
                        background: riskCfg.badge, color: riskCfg.badgeText,
                        borderRadius: '4px', padding: '1px 6px',
                        fontSize: '10px', fontWeight: 700, letterSpacing: '0.04em',
                    }}>
                        {riskLevel}
                    </span>
                )}

                {/* Environment badge for asset nodes */}
                {(type === 'asset' || type === 'table' || type === 'file') && metadata.environment && (
                    <span style={{
                        background: '#f0fdf4', color: '#166534',
                        border: '1px solid #bbf7d0',
                        borderRadius: '4px', padding: '1px 6px', fontSize: '10px', fontWeight: 600,
                    }}>
                        {metadata.environment}
                    </span>
                )}
            </div>

            {/* ── Body ── */}
            <div style={{ padding: '10px 12px' }}>
                {/* Main label */}
                <div style={{
                    fontSize: '13px', fontWeight: 600, color: '#0f172a',
                    lineHeight: '1.3', wordBreak: 'break-all',
                    display: '-webkit-box', WebkitLineClamp: 2,
                    WebkitBoxOrient: 'vertical', overflow: 'hidden',
                    marginBottom: '6px',
                    fontFamily: type === 'pii_category' ? 'JetBrains Mono, monospace' : 'inherit',
                }} title={label}>
                    {label}
                </div>

                {/* ── System-specific ── */}
                {type === 'system' && metadata.host && metadata.host !== label && (
                    <div style={{ fontSize: '11px', color: '#64748b', fontFamily: 'monospace', marginBottom: '4px', wordBreak: 'break-all' }}>
                        {metadata.host}
                    </div>
                )}

                {/* ── Asset-specific ── */}
                {(type === 'asset' || type === 'table' || type === 'file') && metadata.path && (
                    <div style={{
                        fontSize: '10px', color: '#94a3b8', fontFamily: 'monospace',
                        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                        marginBottom: '4px',
                    }} title={metadata.path}>
                        {metadata.path}
                    </div>
                )}

                {/* ── PII-specific ── */}
                {type === 'pii_category' && (
                    <>
                        {/* DPDPA category */}
                        {metadata.dpdpa_category && (
                            <div style={{ fontSize: '10px', color: '#64748b', marginBottom: '6px', lineHeight: '1.3' }}>
                                {metadata.dpdpa_category}
                            </div>
                        )}

                        {/* Finding count pill */}
                        {metadata.finding_count > 0 && (
                            <div style={{
                                display: 'inline-flex', alignItems: 'center', gap: '4px',
                                background: '#fef2f2', border: '1px solid #fecaca',
                                borderRadius: '4px', padding: '2px 7px',
                                fontSize: '11px', fontWeight: 700, color: '#b91c1c',
                                marginBottom: '6px',
                            }}>
                                <span style={{ fontSize: '9px' }}>▲</span>
                                {metadata.finding_count} finding{metadata.finding_count !== 1 ? 's' : ''}
                            </div>
                        )}

                        {/* Confidence bar */}
                        {metadata.avg_confidence != null && (
                            <div style={{ marginTop: '2px' }}>
                                <div style={{ fontSize: '9px', color: '#94a3b8', marginBottom: '3px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                                    Avg Confidence
                                </div>
                                <ConfBar score={metadata.avg_confidence} />
                            </div>
                        )}
                    </>
                )}
            </div>

            {/* Source handle */}
            <Handle
                type="source"
                position={Position.Right}
                style={{ background: accentBar, width: 8, height: 8, border: '2px solid #fff', right: -5 }}
            />
        </div>
    );
}
