'use client';

import React, { useEffect, useState } from 'react';
import { theme, getRiskColor } from '@/design-system/theme';
import Tooltip, { InfoIcon } from '@/components/Tooltip';
import ErrorBanner from '@/components/ErrorBanner';
import { analyticsApi } from '@/services/analytics.api';
import { RiskDistribution } from '@/types/api';

interface PIIHeatmap {
    rows: HeatmapRow[];
    columns: string[];
}

interface HeatmapRow {
    asset_type: string;
    cells: HeatmapCell[];
    total: number;
}

interface HeatmapCell {
    pii_type: string;
    finding_count: number;
    risk_level: string;
    intensity: number;
}

interface RiskTrend {
    timeline: {
        date: string;
        total_pii: number;
        critical_pii: number;
    }[];
    newly_exposed: number;
    resolved: number;
}

export default function AnalyticsPage() {
    const [heatmap, setHeatmap] = useState<PIIHeatmap | null>(null);
    const [trend, setTrend] = useState<RiskTrend | null>(null);
    const [riskDist, setRiskDist] = useState<RiskDistribution | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        fetchData();
    }, []);

    const fetchData = async () => {
        setLoading(true);
        setError(null);

        try {
            const [heatmapData, trendData, riskDistData] = await Promise.all([
                analyticsApi.getHeatmap(),
                analyticsApi.getTrends(30),
                analyticsApi.getRiskDistribution(),
            ]);

            setHeatmap(heatmapData);
            setTrend(trendData);
            setRiskDist(riskDistData);
        } catch (err: any) {
            console.error('Failed to load analytics', err);
            setError(err.message || 'Failed to load analytics data. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    if (loading) return <div style={{ padding: '32px', color: theme.colors.text.primary }}>Loading analytics...</div>;

    return (
        <div style={{ minHeight: '100vh', backgroundColor: theme.colors.background.primary }}>

            <div className="container" style={{ padding: '32px', maxWidth: '1600px', margin: '0 auto' }}>
                {error && (
                    <ErrorBanner
                        message={error}
                        severity="error"
                        onRetry={fetchData}
                        onDismiss={() => setError(null)}
                    />
                )}
                <div style={{ marginBottom: '32px', display: 'flex', justifyContent: 'space-between', alignItems: 'end' }}>
                    <div>
                        <h2 style={{ fontSize: '24px', fontWeight: 700, color: theme.colors.text.primary, marginBottom: '8px' }}>
                            Risk Analytics & Heatmap
                        </h2>
                        <p style={{ color: theme.colors.text.secondary }}>
                            Visualizing PII distribution and exposure trends
                        </p>
                    </div>
                    <div style={{ display: 'flex', gap: '16px' }}>
                        <StatBadge label="Newly Exposed (30d)" value={trend?.newly_exposed || 0} color={theme.colors.risk.critical} />
                        <StatBadge label="Resolved (30d)" value={trend?.resolved || 0} color={theme.colors.risk.low} />
                    </div>
                </div>

                {/* Heatmap Section */}
                <div style={{ marginBottom: '40px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '20px' }}>
                        <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, margin: 0 }}>
                            Data Distribution Heatmap
                        </h3>
                        <Tooltip content="Visual representation of PII density. Darker cells indicate higher concentration of sensitive data.">
                            <InfoIcon size={16} />
                        </Tooltip>
                    </div>

                    <div style={{
                        backgroundColor: theme.colors.background.card,
                        borderRadius: '12px',
                        border: `1px solid ${theme.colors.border.default}`,
                        padding: '24px',
                        overflowX: 'auto'
                    }}>
                        {heatmap && (
                            <table style={{ width: '100%', borderCollapse: 'separate', borderSpacing: '4px' }}>
                                <thead>
                                    <tr>
                                        <th style={{ textAlign: 'left', color: theme.colors.text.secondary, fontSize: '12px', padding: '8px' }}>Asset Type</th>
                                        {heatmap.columns.map(col => (
                                            <th key={col} style={{
                                                fontSize: '11px',
                                                color: theme.colors.text.secondary,
                                                fontWeight: 600,
                                                transform: 'rotate(-45deg)',
                                                height: '100px',
                                                verticalAlign: 'bottom',
                                                textAlign: 'left',
                                                padding: '8px'
                                            }}>
                                                {col.replace('IN_', '').replace('_', ' ')}
                                            </th>
                                        ))}
                                        <th style={{ textAlign: 'right', color: theme.colors.text.secondary, fontSize: '12px', padding: '8px' }}>Total</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {heatmap.rows.map(row => (
                                        <tr key={row.asset_type}>
                                            <td style={{
                                                fontWeight: 600,
                                                color: theme.colors.text.primary,
                                                textTransform: 'capitalize',
                                                padding: '8px'
                                            }}>
                                                {row.asset_type}s
                                            </td>
                                            {row.cells.map(cell => (
                                                <td key={cell.pii_type} title={`${cell.finding_count} findings (${cell.risk_level})`}>
                                                    <div style={{
                                                        height: '40px',
                                                        backgroundColor: getCellColor(cell.intensity, cell.risk_level),
                                                        borderRadius: '4px',
                                                        display: 'flex',
                                                        alignItems: 'center',
                                                        justifyContent: 'center',
                                                        fontSize: '12px',
                                                        fontWeight: 600,
                                                        color: cell.intensity > 40 ? '#fff' : theme.colors.text.primary,
                                                        cursor: 'pointer',
                                                        transition: 'transform 0.1s',
                                                    }}
                                                        onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.1)'}
                                                        onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
                                                    >
                                                        {cell.finding_count > 0 ? cell.finding_count : ''}
                                                    </div>
                                                </td>
                                            ))}
                                            <td style={{ textAlign: 'right', fontWeight: 700, color: theme.colors.text.primary, padding: '8px' }}>
                                                {row.total}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        )}
                    </div>
                    {/* Legend */}
                    <div style={{ marginTop: '16px', display: 'flex', alignItems: 'center', gap: '16px', fontSize: '12px', color: theme.colors.text.secondary }}>
                        <span>Pixel Intensity:</span>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <div style={{ width: '16px', height: '16px', borderRadius: '4px', backgroundColor: `${theme.colors.risk.high}33` }} />
                            <span>Low Density</span>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <div style={{ width: '16px', height: '16px', borderRadius: '4px', backgroundColor: `${theme.colors.risk.high}80` }} />
                            <span>Medium Density</span>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <div style={{ width: '16px', height: '16px', borderRadius: '4px', backgroundColor: `${theme.colors.risk.high}` }} />
                            <span>High Density</span>
                        </div>
                    </div>
                </div>

                {/* Risk Distribution Section */}
                {riskDist && (
                    <div style={{ marginBottom: '40px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '20px' }}>
                            <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, margin: 0 }}>
                                Risk Distribution
                            </h3>
                            <Tooltip content="Breakdown of findings by severity tier across all scanned assets.">
                                <InfoIcon size={16} />
                            </Tooltip>
                        </div>
                        <div style={{
                            backgroundColor: theme.colors.background.card,
                            borderRadius: '12px',
                            border: `1px solid ${theme.colors.border.default}`,
                            padding: '24px',
                        }}>
                            <div style={{ display: 'flex', gap: '12px', alignItems: 'flex-end', height: '160px', marginBottom: '16px' }}>
                                {riskDist.distribution.map((tier) => {
                                    const barColor = getRiskColor(tier.severity.toLowerCase());
                                    const maxCount = Math.max(...riskDist.distribution.map(t => t.count), 1);
                                    const barHeight = Math.max(8, Math.round((tier.count / maxCount) * 120));
                                    return (
                                        <div key={tier.severity} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
                                            <div style={{ fontSize: '12px', fontWeight: 700, color: barColor }}>
                                                {tier.count.toLocaleString()}
                                            </div>
                                            <div style={{
                                                width: '100%',
                                                height: `${barHeight}px`,
                                                backgroundColor: barColor,
                                                borderRadius: '4px 4px 0 0',
                                                opacity: 0.85,
                                                transition: 'height 0.3s ease',
                                            }} title={`${tier.severity}: ${tier.count} findings (${tier.percentage}%)`} />
                                            <div style={{ fontSize: '11px', fontWeight: 600, color: theme.colors.text.secondary, textTransform: 'uppercase' }}>
                                                {tier.severity}
                                            </div>
                                            <div style={{ fontSize: '11px', color: theme.colors.text.muted }}>
                                                {tier.percentage}%
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                            <div style={{ borderTop: `1px solid ${theme.colors.border.default}`, paddingTop: '12px', fontSize: '12px', color: theme.colors.text.muted, textAlign: 'right' }}>
                                Total: {riskDist.total.toLocaleString()} findings
                            </div>
                        </div>
                    </div>
                )}

                {/* Trends Section */}
                <div>
                    <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, marginBottom: '20px' }}>
                        30-Day Risk Exposure Trend
                    </h3>
                    <div style={{ fontSize: '13px', color: theme.colors.text.secondary, marginBottom: '20px' }}>
                        Historical view of critical PII exposure and newly discovered risks.
                    </div>
                    <div style={{
                        backgroundColor: theme.colors.background.card,
                        borderRadius: '12px',
                        border: `1px solid ${theme.colors.border.default}`,
                        padding: '32px',
                        height: '300px',
                        display: 'flex',
                        alignItems: 'end',
                        gap: '8px'
                    }}>
                        {trend && trend.timeline.map((point, i) => {
                            const maxPii = Math.max(...trend.timeline.map((p: any) => p.total_pii), 1);
                            const height = Math.max(10, Math.round((point.total_pii / maxPii) * 200));
                            return (
                                <div key={point.date} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
                                    <div style={{
                                        width: '100%',
                                        height: `${height}px`,
                                        backgroundColor: point.critical_pii > 0 ? theme.colors.risk.critical : theme.colors.risk.medium,
                                        borderRadius: '4px 4px 0 0',
                                        opacity: 0.8
                                    }} />
                                    {i % 5 === 0 && (
                                        <div style={{ fontSize: '10px', color: theme.colors.text.muted, transform: 'rotate(-45deg)', marginTop: '8px' }}>
                                            {new Date(point.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                                        </div>
                                    )}
                                </div>
                            );
                        })}
                    </div>
                </div>
            </div>
        </div>
    );
}

function StatBadge({ label, value, color }: any) {
    return (
        <div style={{
            backgroundColor: theme.colors.background.card,
            border: `1px solid ${theme.colors.border.default}`,
            padding: '8px 16px',
            borderRadius: '8px',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center'
        }}>
            <div style={{ fontSize: '11px', color: theme.colors.text.secondary, textTransform: 'uppercase', marginBottom: '4px' }}>{label}</div>
            <div style={{ fontSize: '18px', fontWeight: 700, color: color }}>
                {value > 0 ? '+' : ''}{value}
            </div>
        </div>
    );
}

function getCellColor(intensity: number, riskLevel: string) {
    if (intensity === 0) return theme.colors.background.tertiary; // Empty

    // Base color on risk level
    const baseColor = getRiskColor(riskLevel);

    // Simple opacity simulation (hex alpha)
    const alpha = Math.max(20, Math.min(100, intensity)); // 20% to 100%
    const hexAlpha = Math.round((alpha / 100) * 255).toString(16).padStart(2, '0');

    return `${baseColor}${hexAlpha}`;
}
