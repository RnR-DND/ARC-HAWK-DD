'use client';

import React, { useEffect, useState } from 'react';
import { assetsApi } from '@/services/assets.api';
import { Asset } from '@/types';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';
import LoadingState from '@/components/LoadingState';
import EmptyState from '@/components/EmptyState';
import Topbar from '@/components/Topbar';

export default function AssetInventoryPage() {
    const [assets, setAssets] = useState<Asset[]>([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [currentPage, setCurrentPage] = useState(1);
    const [searchQuery, setSearchQuery] = useState('');
    const pageSize = 20;

    useEffect(() => {
        fetchAssets();
    }, [currentPage]);

    const fetchAssets = async () => {
        try {
            setLoading(true);
            setError(null);
            const response = await assetsApi.getAssets({
                page: currentPage,
                page_size: pageSize,
            });
            setAssets(response.assets);
            setTotal(response.total);
        } catch (err: any) {
            console.error('Error fetching assets:', err);
            setError(err.message || 'Failed to fetch assets');
        } finally {
            setLoading(false);
        }
    };

    const filteredAssets = assets.filter(asset => {
        if (!searchQuery) return true;
        const query = searchQuery.toLowerCase();
        return (
            asset.name?.toLowerCase().includes(query) ||
            asset.path?.toLowerCase().includes(query) ||
            asset.asset_type?.toLowerCase().includes(query)
        );
    });

    const totalPages = Math.ceil(total / pageSize);

    const getRiskColor = (score: number) => {
        if (score >= 90) return colors.state.risk;
        if (score >= 70) return colors.state.warning;
        if (score >= 50) return '#f59e0b';
        return colors.status.success;
    };

    const getRiskLabel = (score: number) => {
        if (score >= 90) return 'Critical';
        if (score >= 70) return 'High';
        if (score >= 50) return 'Medium';
        return 'Low';
    };

    if (loading && assets.length === 0) {
        return <LoadingState fullScreen message="Loading asset inventory..." />;
    }

    return (
        <div style={{ padding: '24px', minHeight: '100vh', backgroundColor: colors.background.primary }}>
            <Topbar
                environment="Production"
                riskScore={0}
                onSearch={(q) => setSearchQuery(q)}
            />

            <div style={{ padding: '32px', maxWidth: '1600px', margin: '0 auto' }}>
                <div style={{ marginBottom: '32px' }}>
                    <h1 style={{
                        fontSize: '32px',
                        fontWeight: 800,
                        color: colors.text.primary,
                        marginBottom: '8px',
                        letterSpacing: '-0.02em',
                    }}>
                        Asset Inventory
                    </h1>
                    <p style={{ color: colors.text.secondary, fontSize: '16px', margin: 0 }}>
                        Complete catalog of scanned data assets across all environments
                    </p>
                </div>

                {error && (
                    <div style={{
                        padding: '16px 24px',
                        backgroundColor: '#FEF2F2',
                        border: '1px solid #FECACA',
                        borderRadius: '12px',
                        color: '#B91C1C',
                        marginBottom: '24px',
                    }}>
                        ⚠️ {error}
                    </div>
                )}

                <div style={{
                    background: colors.background.surface,
                    border: `1px solid ${colors.border.default}`,
                    borderRadius: theme.borderRadius.xl,
                    boxShadow: theme.shadows.sm,
                    overflow: 'hidden',
                }}>
                    <div style={{ padding: '24px', borderBottom: `1px solid ${colors.border.default}` }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <div>
                                <h2 style={{ fontSize: '18px', fontWeight: 700, margin: 0, color: colors.text.primary }}>
                                    All Assets ({total})
                                </h2>
                            </div>
                            <input
                                type="text"
                                placeholder="Search assets..."
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                style={{
                                    padding: '8px 16px',
                                    borderRadius: '8px',
                                    border: `1px solid ${colors.border.default}`,
                                    fontSize: '14px',
                                    width: '300px',
                                }}
                            />
                        </div>
                    </div>

                    {filteredAssets.length === 0 ? (
                        <EmptyState
                            icon="📦"
                            title="No Assets Found"
                            description={searchQuery ? "No assets match your search criteria" : "No assets have been scanned yet"}
                        />
                    ) : (
                        <div style={{ overflowX: 'auto' }}>
                            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                                <thead>
                                    <tr style={{ backgroundColor: colors.background.muted }}>
                                        <th style={headerStyle}>Asset Name</th>
                                        <th style={headerStyle}>Type</th>
                                        <th style={headerStyle}>Path</th>
                                        <th style={headerStyle}>Environment</th>
                                        <th style={headerStyle}>Risk Score</th>
                                        <th style={headerStyle}>Findings</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {filteredAssets.map((asset) => (
                                        <tr
                                            key={asset.id}
                                            style={{
                                                borderBottom: `1px solid ${colors.border.subtle}`,
                                                cursor: 'pointer',
                                                transition: 'background 0.2s',
                                            }}
                                            onMouseEnter={(e) => e.currentTarget.style.backgroundColor = colors.background.muted}
                                            onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
                                        >
                                            <td style={cellStyle}>
                                                <div style={{ fontWeight: 600, color: colors.text.primary }}>
                                                    {asset.name}
                                                </div>
                                            </td>
                                            <td style={cellStyle}>
                                                <span style={{
                                                    padding: '4px 8px',
                                                    borderRadius: '4px',
                                                    backgroundColor: colors.background.elevated,
                                                    fontSize: '12px',
                                                    fontWeight: 600,
                                                    color: colors.text.secondary,
                                                }}>
                                                    {asset.asset_type.toUpperCase()}
                                                </span>
                                            </td>
                                            <td style={cellStyle}>
                                                <div style={{ fontSize: '13px', color: colors.text.secondary, fontFamily: 'monospace' }}>
                                                    {asset.path}
                                                </div>
                                            </td>
                                            <td style={cellStyle}>
                                                <span style={{
                                                    padding: '4px 8px',
                                                    borderRadius: '4px',
                                                    backgroundColor: asset.environment === 'Production' ? '#FEF9C3' : '#DBEAFE',
                                                    fontSize: '12px',
                                                    fontWeight: 600,
                                                    color: asset.environment === 'Production' ? '#854D0E' : '#1E40AF',
                                                }}>
                                                    {asset.environment}
                                                </span>
                                            </td>
                                            <td style={cellStyle}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                    <div style={{
                                                        width: '60px',
                                                        height: '6px',
                                                        backgroundColor: colors.background.elevated,
                                                        borderRadius: '3px',
                                                        overflow: 'hidden',
                                                    }}>
                                                        <div style={{
                                                            width: `${asset.risk_score}%`,
                                                            height: '100%',
                                                            backgroundColor: getRiskColor(asset.risk_score),
                                                        }} />
                                                    </div>
                                                    <span style={{ fontSize: '14px', fontWeight: 600, color: getRiskColor(asset.risk_score) }}>
                                                        {asset.risk_score}
                                                    </span>
                                                    <span style={{ fontSize: '12px', color: colors.text.muted }}>
                                                        {getRiskLabel(asset.risk_score)}
                                                    </span>
                                                </div>
                                            </td>
                                            <td style={cellStyle}>
                                                <span style={{
                                                    padding: '4px 12px',
                                                    borderRadius: '12px',
                                                    backgroundColor: colors.background.elevated,
                                                    fontSize: '14px',
                                                    fontWeight: 700,
                                                    color: colors.text.primary,
                                                }}>
                                                    {asset.total_findings || 0}
                                                </span>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}

                    {totalPages > 1 && (
                        <div style={{
                            padding: '16px 24px',
                            borderTop: `1px solid ${colors.border.default}`,
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                        }}>
                            <div style={{ fontSize: '14px', color: colors.text.secondary }}>
                                Showing {((currentPage - 1) * pageSize) + 1} to {Math.min(currentPage * pageSize, total)} of {total} assets
                            </div>
                            <div style={{ display: 'flex', gap: '8px' }}>
                                <button
                                    onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                                    disabled={currentPage === 1}
                                    style={{
                                        padding: '8px 16px',
                                        borderRadius: '6px',
                                        border: `1px solid ${colors.border.default}`,
                                        backgroundColor: currentPage === 1 ? colors.background.muted : colors.background.surface,
                                        color: currentPage === 1 ? colors.text.muted : colors.text.primary,
                                        cursor: currentPage === 1 ? 'not-allowed' : 'pointer',
                                        fontWeight: 600,
                                    }}
                                >
                                    Previous
                                </button>
                                <span style={{ padding: '8px 16px', fontSize: '14px', color: colors.text.secondary }}>
                                    Page {currentPage} of {totalPages}
                                </span>
                                <button
                                    onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                                    disabled={currentPage === totalPages}
                                    style={{
                                        padding: '8px 16px',
                                        borderRadius: '6px',
                                        border: `1px solid ${colors.border.default}`,
                                        backgroundColor: currentPage === totalPages ? colors.background.muted : colors.background.surface,
                                        color: currentPage === totalPages ? colors.text.muted : colors.text.primary,
                                        cursor: currentPage === totalPages ? 'not-allowed' : 'pointer',
                                        fontWeight: 600,
                                    }}
                                >
                                    Next
                                </button>
                            </div>
                        </div>
                    )}
                </div>
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
};
