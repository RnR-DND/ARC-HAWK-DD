'use client';

import React, { useEffect, useState } from 'react';
import { scansApi } from '@/services/scans.api';
import { ScanConfigModal } from '@/components/scans/ScanConfigModal';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';
import LoadingState from '@/components/LoadingState';
import ErrorBanner from '@/components/ErrorBanner';

interface ScanRun {
    id: string;
    scan_started_at: string;
    scan_completed_at: string;
    status: string;
    total_assets: number;        // Backend returns total_assets
    total_findings: number;       // Backend returns total_findings
    critical_findings?: number;   // Optional, may not be in response
}

export default function PosturePage() {
    const [scans, setScans] = useState<ScanRun[]>([]);
    const [latestScan, setLatestScan] = useState<ScanRun | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [scanning, setScanning] = useState(false);
    const [scanProgress, setScanProgress] = useState(0);
    const [isScanModalOpen, setIsScanModalOpen] = useState(false);

    useEffect(() => {
        fetchScans();
    }, []);

    const fetchScans = async () => {
        try {
            setLoading(true);
            setError(null);

            const latest = await scansApi.getLastScanRun();
            setLatestScan(latest);

            // For now, show latest scan only
            setScans(latest ? [latest] : []);
        } catch (err: any) {
            console.error('Error fetching scans:', err);
            setError(err.message || 'Failed to fetch scan data');
        } finally {
            setLoading(false);
        }
    };

    const handleRunScan = () => {
        setIsScanModalOpen(true);
    };

    const formatDate = (dateStr: string) => {
        if (!dateStr) return 'N/A';
        try {
            return new Date(dateStr).toLocaleString();
        } catch {
            return dateStr;
        }
    };

    const getStatusColor = (status: string) => {
        switch (status?.toLowerCase()) {
            case 'completed':
                return colors.status.success;
            case 'running':
                return colors.status.info;
            case 'failed':
                return colors.state.risk;
            default:
                return colors.text.secondary;
        }
    };

    const getStatusBadge = (status: string) => {
        const statusColor = getStatusColor(status);
        return (
            <span style={{
                padding: '4px 12px',
                borderRadius: '12px',
                fontSize: '12px',
                fontWeight: 700,
                backgroundColor: `${statusColor}20`,
                color: statusColor,
            }}>
                {status?.toUpperCase() || 'UNKNOWN'}
            </span>
        );
    };

    if (loading && !latestScan) {
        return <LoadingState fullScreen message="Loading posture data..." />;
    }

    return (
        <div style={{ minHeight: '100vh', backgroundColor: colors.background.primary, padding: '24px' }}>


            <div style={{ padding: '20px', maxWidth: '1800px', margin: '0 auto' }}>
                <div style={{ marginBottom: '24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                        <h1 style={{
                            fontSize: '32px',
                            fontWeight: 800,
                            color: colors.text.primary,
                            marginBottom: '8px',
                            letterSpacing: '-0.02em',
                        }}>
                            Security Posture
                        </h1>
                        <p style={{ color: colors.text.secondary, fontSize: '16px', margin: 0 }}>
                            Scan history and security posture tracking
                        </p>
                    </div>

                    {/* Run Scan Button */}
                    <button
                        onClick={handleRunScan}
                        disabled={scanning}
                        style={{
                            padding: '14px 28px',
                            borderRadius: '10px',
                            border: 'none',
                            backgroundColor: scanning ? colors.background.muted : colors.nodeColors.system,
                            color: '#FFFFFF',
                            fontWeight: 700,
                            fontSize: '15px',
                            cursor: scanning ? 'not-allowed' : 'pointer',
                            boxShadow: scanning ? 'none' : '0 2px 8px rgba(0, 0, 0, 0.15)',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px',
                            transition: 'all 0.2s',
                        }}
                        onMouseEnter={(e) => {
                            if (!scanning) {
                                e.currentTarget.style.transform = 'scale(1.05)';
                                e.currentTarget.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.2)';
                            }
                        }}
                        onMouseLeave={(e) => {
                            e.currentTarget.style.transform = 'scale(1)';
                            e.currentTarget.style.boxShadow = scanning ? 'none' : '0 2px 8px rgba(0, 0, 0, 0.15)';
                        }}
                    >
                        {scanning ? (
                            <>
                                <div style={{
                                    width: '16px',
                                    height: '16px',
                                    border: '2px solid #ffffff',
                                    borderTopColor: 'transparent',
                                    borderRadius: '50%',
                                    animation: 'spin 1s linear infinite',
                                }} />
                                Scanning... {scanProgress}%
                            </>
                        ) : (
                            <>
                                🔍 Run New Scan
                            </>
                        )}
                    </button>
                </div>

                {error && (
                    <ErrorBanner
                        message={error}
                        severity="error"
                        onRetry={fetchScans}
                        onDismiss={() => setError(null)}
                    />
                )}

                {/* Progress Bar */}
                {scanning && (
                    <div style={{
                        marginBottom: '24px',
                        background: colors.background.surface,
                        border: `1px solid ${colors.border.default}`,
                        borderRadius: theme.borderRadius.xl,
                        padding: '20px',
                    }}>
                        <div style={{ marginBottom: '12px', display: 'flex', justifyContent: 'space-between' }}>
                            <span style={{ fontSize: '14px', fontWeight: 600, color: colors.text.primary }}>
                                Scan in Progress
                            </span>
                            <span style={{ fontSize: '14px', fontWeight: 700, color: colors.nodeColors.system }}>
                                {scanProgress}%
                            </span>
                        </div>
                        <div style={{
                            width: '100%',
                            height: '8px',
                            background: colors.background.elevated,
                            borderRadius: '4px',
                            overflow: 'hidden',
                        }}>
                            <div style={{
                                width: `${scanProgress}%`,
                                height: '100%',
                                background: `linear-gradient(90deg, ${colors.nodeColors.system}, ${colors.nodeColors.asset})`,
                                transition: 'width 0.3s ease',
                            }} />
                        </div>
                    </div>
                )}

                {/* Latest Scan Summary */}
                {latestScan && (
                    <div style={{
                        background: colors.background.surface,
                        border: `1px solid ${colors.border.default}`,
                        borderRadius: theme.borderRadius.xl,
                        padding: '28px',
                        marginBottom: '28px',
                        boxShadow: theme.shadows.lg,
                    }}>
                        <div style={{ marginBottom: '24px' }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                                <h2 style={{ fontSize: '20px', fontWeight: 700, margin: 0, color: colors.text.primary }}>
                                    Latest Scan
                                </h2>
                                {getStatusBadge(latestScan.status)}
                            </div>
                            <div style={{ fontSize: '14px', color: colors.text.secondary }}>
                                Completed {formatDate(latestScan.scan_completed_at)}
                            </div>
                        </div>

                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: '20px' }}>
                            <MetricCard
                                label="Assets Scanned"
                                value={latestScan.total_assets || 0}
                                color={colors.nodeColors.asset}
                                icon="📦"
                            />
                            <MetricCard
                                label="Total Findings"
                                value={latestScan.total_findings || 0}
                                color={colors.status.info}
                                icon="🔍"
                            />
                            <MetricCard
                                label="Critical Issues"
                                value={latestScan.critical_findings || 0}
                                color={colors.state.risk}
                                icon="⚠️"
                            />
                        </div>
                    </div>
                )}

                {/* Scan History Table */}
                <div style={{
                    background: colors.background.surface,
                    border: `1px solid ${colors.border.default}`,
                    borderRadius: theme.borderRadius.xl,
                    overflow: 'hidden',
                    boxShadow: theme.shadows.sm,
                }}>
                    <div style={{ padding: '24px', borderBottom: `1px solid ${colors.border.default}` }}>
                        <h2 style={{ fontSize: '18px', fontWeight: 700, margin: 0, color: colors.text.primary }}>
                            Scan History
                        </h2>
                    </div>

                    {scans.length === 0 ? (
                        <div style={{ padding: '48px', textAlign: 'center' }}>
                            <div style={{ fontSize: '48px', marginBottom: '16px' }}>🔍</div>
                            <div style={{ fontSize: '18px', fontWeight: 600, color: colors.text.primary, marginBottom: '8px' }}>
                                No Scans Yet
                            </div>
                            <div style={{ fontSize: '14px', color: colors.text.secondary, marginBottom: '24px' }}>
                                Click Run New Scan to start tracking your security posture
                            </div>
                        </div>
                    ) : (
                        <div style={{ overflowX: 'auto' }}>
                            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                                <thead>
                                    <tr style={{ backgroundColor: colors.background.muted }}>
                                        <th style={headerStyle}>Started</th>
                                        <th style={headerStyle}>Completed</th>
                                        <th style={headerStyle}>Status</th>
                                        <th style={headerStyle}>Assets</th>
                                        <th style={headerStyle}>Findings</th>
                                        <th style={headerStyle}>Critical</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {scans.map((scan) => (
                                        <tr
                                            key={scan.id}
                                            style={{
                                                borderBottom: `1px solid ${colors.border.subtle}`,
                                            }}
                                        >
                                            <td style={cellStyle}>{formatDate(scan.scan_started_at)}</td>
                                            <td style={cellStyle}>{formatDate(scan.scan_completed_at)}</td>
                                            <td style={cellStyle}>{getStatusBadge(scan.status)}</td>
                                            <td style={cellStyle}>
                                                <span style={{ fontWeight: 600 }}>{scan.total_assets || 0}</span>
                                            </td>
                                            <td style={cellStyle}>
                                                <span style={{ fontWeight: 600 }}>{scan.total_findings || 0}</span>
                                            </td>
                                            <td style={cellStyle}>
                                                <span style={{ fontWeight: 700, color: colors.state.risk }}>
                                                    {scan.critical_findings || 0}
                                                </span>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>

                {/* CLI Instructions */}
                <div style={{
                    marginTop: '24px',
                    padding: '20px 24px',
                    backgroundColor: '#DBEAFE',
                    border: '1px solid #93C5FD',
                    borderRadius: '12px',
                }}>
                    <div style={{ fontSize: '14px', color: '#1E40AF', fontWeight: 600, marginBottom: '8px' }}>
                        💡 Manual Scan Instructions
                    </div>
                    <div style={{ fontSize: '14px', color: '#1E3A8A', lineHeight: 1.6 }}>
                        To run a scan manually via CLI:
                        <code style={{
                            display: 'block',
                            marginTop: '8px',
                            padding: '8px 12px',
                            backgroundColor: '#1E3A8A',
                            color: '#DBEAFE',
                            borderRadius: '6px',
                            fontFamily: 'monospace',
                            fontSize: '13px',
                        }}>
                            cd scripts/automation && python unified-scan.py
                        </code>
                    </div>
                </div>
            </div>

            <ScanConfigModal isOpen={isScanModalOpen} onClose={() => { setIsScanModalOpen(false); fetchScans(); }} />
        </div>
    );
}

function MetricCard({ label, value, color, icon }: { label: string; value: number; color: string; icon: string }) {
    return (
        <div style={{
            padding: '20px',
            backgroundColor: colors.background.elevated,
            borderRadius: '12px',
            border: `1px solid ${colors.border.subtle}`,
        }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '12px' }}>
                <span style={{ fontSize: '24px' }}>{icon}</span>
                <div style={{ fontSize: '12px', fontWeight: 600, color: colors.text.secondary, textTransform: 'uppercase' }}>
                    {label}
                </div>
            </div>
            <div style={{ fontSize: '36px', fontWeight: 900, color, lineHeight: 1 }}>
                {value.toLocaleString()}
            </div>
        </div>
    );
}

const headerStyle: React.CSSProperties = {
    padding: '16px',
    textAlign: 'left',
    fontSize: '12px',
    fontWeight: 700,
    color: colors.text.secondary,
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
};

const cellStyle: React.CSSProperties = {
    padding: '16px',
    fontSize: '14px',
    color: colors.text.primary,
};
