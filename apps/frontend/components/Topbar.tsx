'use client';

import React, { useState } from 'react';
import { theme, getRiskColor } from '@/design-system/theme';
import ScanAllButton from './ScanAllButton';
import AddDataSourceModal from './AddDataSourceModal';

interface TopbarProps {
    environment?: string;
    scanTime?: string;
    riskScore?: number;
    onSearch?: (query: string) => void;
}

export default function Topbar({
    environment = 'DPDPA Controls',
    scanTime,
    riskScore = 0,
    onSearch,
}: TopbarProps) {
    const [isAddSourceOpen, setIsAddSourceOpen] = useState(false);
    const riskLevel = riskScore >= 80 ? 'Critical' : riskScore >= 50 ? 'High' : riskScore >= 20 ? 'Medium' : 'Low';
    const riskColor = getRiskColor(riskLevel);

    return (
        <header
            style={{
                height: '72px',
                backgroundColor: theme.colors.background.card,
                borderBottom: `1px solid ${theme.colors.border.default}`,
                position: 'sticky',
                top: 0,
                zIndex: 50,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '0 32px',
                boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.4)',
            }}
        >
            {/* Left section - Title */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
                <h1
                    style={{
                        fontSize: '20px',
                        fontWeight: 800,
                        color: theme.colors.text.primary,
                        letterSpacing: '-0.02em',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '12px',
                    }}
                >
                    <span style={{ color: theme.colors.risk.critical }}>🛡️</span>
                    DPDPA Compliance Control Plane
                </h1>
            </div>

            {/* Right section - Actions & Metadata */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
                <ScanAllButton />

                <button
                    onClick={() => setIsAddSourceOpen(true)}
                    style={{
                        padding: '8px 16px',
                        backgroundColor: theme.colors.background.tertiary,
                        color: theme.colors.text.primary,
                        border: `1px solid ${theme.colors.border.default}`,
                        borderRadius: '8px',
                        fontWeight: 600,
                        fontSize: '13px',
                        cursor: 'pointer',
                        display: 'flex', alignItems: 'center', gap: '6px'
                    }}
                >
                    <span style={{ fontSize: '16px' }}>+</span> Add Source
                </button>
                <AddDataSourceModal isOpen={isAddSourceOpen} onClose={() => setIsAddSourceOpen(false)} />

                <div
                    style={{
                        width: '1px',
                        height: '32px',
                        backgroundColor: theme.colors.border.default,
                    }}
                />

                {/* Risk Score */}
                {riskScore >= 0 && (
                    <div
                        style={{
                            display: 'flex',
                            flexDirection: 'column',
                            alignItems: 'center',
                            padding: '6px 16px',
                            backgroundColor: `${riskColor}10`, // 10% opacity
                            borderRadius: '8px',
                            border: `1px solid ${riskColor}40`, // 40% opacity
                            minWidth: '90px',
                        }}
                    >
                        <div style={{
                            fontSize: '11px',
                            color: theme.colors.text.muted,
                            fontWeight: 700,
                            textTransform: 'uppercase',
                            letterSpacing: '0.05em',
                            marginBottom: '2px',
                        }}>
                            Risk Score
                        </div>
                        <div
                            style={{
                                fontSize: '20px',
                                color: riskColor,
                                fontWeight: 800,
                                lineHeight: 1,
                            }}
                        >
                            {riskScore}
                        </div>
                    </div>
                )}

                {/* User Profile / Context */}
                <div style={{
                    width: '32px',
                    height: '32px',
                    borderRadius: '50%',
                    backgroundColor: theme.colors.background.tertiary,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: '14px',
                    fontWeight: 600,
                    color: theme.colors.text.secondary,
                    border: `1px solid ${theme.colors.border.default}`
                }}>
                    JS
                </div>
            </div>
        </header>
    );
}
