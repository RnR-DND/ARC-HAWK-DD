'use client';

import React, { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import FindingsTable from '@/components/FindingsTable';
import LoadingState from '@/components/LoadingState';
import { assetsApi } from '@/services/assets.api';
import { findingsApi } from '@/services/findings.api';
import { Asset, FindingsResponse } from '@/types';
import { ArrowLeft, Database, User, FolderOpen, Shield, Activity, Share2, FileJson, Server } from 'lucide-react';

export default function AssetDetailPage() {
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;

    const [asset, setAsset] = useState<Asset | null>(null);
    const [findingsData, setFindingsData] = useState<FindingsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Tab state
    const [activeTab, setActiveTab] = useState<'findings' | 'lineage' | 'metadata'>('findings');

    useEffect(() => {
        if (id) fetchData();
    }, [id]);

    const fetchData = async () => {
        try {
            setLoading(true);
            setError(null);

            const [assetData, findingsRes] = await Promise.all([
                assetsApi.getAsset(id),
                findingsApi.getFindings({ asset_id: id, page: 1, page_size: 50 })
            ]);

            setAsset(assetData);
            setFindingsData(findingsRes as FindingsResponse);

        } catch (err: any) {
            console.error('Error fetching asset details:', err);
            setError('Failed to load asset details.');
        } finally {
            setLoading(false);
        }
    };

    if (loading) return <LoadingState fullScreen message="Loading Asset Details..." />;

    if (error || !asset) return (
        <div className="flex flex-col items-center justify-center min-h-screen bg-background p-8">
            <div className="text-center max-w-md">
                <div className="w-16 h-16 bg-red-500/10 rounded-full flex items-center justify-center mx-auto mb-6">
                    <Shield className="w-8 h-8 text-red-500" />
                </div>
                <h2 className="text-2xl font-bold text-white mb-2">Error Loading Asset</h2>
                <p className="text-muted-foreground mb-8">{error || 'Asset not found or access denied.'}</p>
                <button
                    onClick={() => router.push('/')}
                    className="px-6 py-2 bg-muted hover:bg-accent text-white rounded-lg transition-colors font-medium border border-border"
                >
                    Back to Dashboard
                </button>
            </div>
        </div>
    );

    const getRiskColor = (score: number) => {
        if (score >= 90) return 'text-red-500';
        if (score >= 70) return 'text-orange-500';
        if (score >= 40) return 'text-yellow-500';
        return 'text-blue-500';
    };

    return (
        <div className="min-h-screen bg-background text-slate-200 font-sans">
            <div className="max-w-7xl mx-auto p-6 md:p-8">
                {/* Header / Breadcrumb */}
                <button
                    onClick={() => router.push('/assets')}
                    className="flex items-center gap-2 text-muted-foreground hover:text-white mb-6 text-sm font-medium transition-colors group"
                >
                    <ArrowLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
                    Back to Inventory
                </button>

                {/* Asset Header Card */}
                <div className="bg-card rounded-xl border border-border p-8 mb-8 shadow-xl">
                    <div className="flex flex-col md:flex-row md:items-start justify-between gap-6">
                        <div className="flex-1">
                            <div className="flex items-center gap-3 mb-4">
                                <span className="px-2.5 py-1 rounded bg-muted text-slate-300 text-xs font-bold uppercase tracking-wider border border-border">
                                    {asset.asset_type}
                                </span>
                                <span className="font-mono text-xs text-slate-500">ID: {asset.id.substring(0, 8)}...</span>
                            </div>

                            <h1 className="text-3xl md:text-4xl font-extrabold text-white mb-4 tracking-tight">
                                {asset.name}
                            </h1>

                            <div className="flex flex-wrap items-center gap-6 text-sm text-muted-foreground font-medium">
                                <div className="flex items-center gap-2">
                                    <Server className="w-4 h-4 text-slate-500" />
                                    <span className="text-slate-300">{asset.source_system}</span>
                                </div>
                                <div className="flex items-center gap-2">
                                    <User className="w-4 h-4 text-slate-500" />
                                    <span className="text-slate-300">{asset.owner || 'Unassigned'}</span>
                                </div>
                                <div className="flex items-center gap-2">
                                    <FolderOpen className="w-4 h-4 text-slate-500" />
                                    <span className="font-mono bg-background px-2 py-0.5 rounded border border-border text-xs text-blue-300 truncate max-w-[300px]" title={asset.path}>
                                        {asset.path}
                                    </span>
                                </div>
                            </div>
                        </div>

                        <div className="flex flex-col items-end">
                            <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">Risk Score</div>
                            <div className={`text-5xl font-black ${getRiskColor(asset.risk_score)}`}>
                                {asset.risk_score}
                            </div>
                        </div>
                    </div>
                </div>

                {/* Tabs */}
                <div className="flex items-center gap-1 mb-6 border-b border-border">
                    <TabButton
                        active={activeTab === 'findings'}
                        onClick={() => setActiveTab('findings')}
                        label={`Findings (${asset.total_findings})`}
                        icon={<Shield className="w-4 h-4" />}
                    />
                    <TabButton
                        active={activeTab === 'lineage'}
                        onClick={() => setActiveTab('lineage')}
                        label="Lineage Graph"
                        icon={<Share2 className="w-4 h-4" />}
                    />
                    <TabButton
                        active={activeTab === 'metadata'}
                        onClick={() => setActiveTab('metadata')}
                        label="Metadata"
                        icon={<FileJson className="w-4 h-4" />}
                    />
                </div>

                {/* Content Area */}
                <div className="bg-card rounded-xl border border-border min-h-[400px] shadow-sm">
                    {activeTab === 'findings' && (
                        findingsData ? (
                            <div className="p-4">
                                <FindingsTable
                                    findings={findingsData.findings}
                                    total={findingsData.total}
                                    page={1}
                                    pageSize={50}
                                    totalPages={1}
                                    onPageChange={() => { }}
                                    onFilterChange={() => { }}
                                />
                            </div>
                        ) : (
                            <div className="flex flex-col items-center justify-center py-24 text-slate-500">
                                <Shield className="w-12 h-12 mb-4 opacity-20" />
                                <p>No findings loaded.</p>
                            </div>
                        )
                    )}

                    {activeTab === 'lineage' && (
                        <div className="flex flex-col items-center justify-center py-24 px-6 text-center">
                            <div className="w-16 h-16 bg-blue-500/10 rounded-full flex items-center justify-center mb-6">
                                <Share2 className="w-8 h-8 text-blue-500" />
                            </div>
                            <h3 className="text-xl font-semibold text-white mb-2">Visual Lineage</h3>
                            <p className="text-muted-foreground max-w-md mb-8">
                                Trace data flow, understand dependencies, and visualize impact analysis for {asset.name}.
                            </p>
                            <button
                                onClick={() => router.push(`/lineage?assetId=${asset.id}`)}
                                className="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors shadow-lg shadow-blue-900/20"
                            >
                                <Activity className="w-4 h-4" />
                                Open Lineage Graph
                            </button>
                        </div>
                    )}

                    {activeTab === 'metadata' && (
                        <div className="p-8 grid grid-cols-1 md:grid-cols-2 gap-12">
                            <div>
                                <h3 className="text-lg font-bold text-white mb-4 flex items-center gap-2">
                                    <Database className="w-5 h-5 text-purple-400" />
                                    Technical Metadata
                                </h3>
                                <div className="bg-background rounded-lg border border-border p-4 font-mono text-xs text-slate-300 overflow-auto max-h-[500px]">
                                    <pre>{JSON.stringify(asset.file_metadata || {}, null, 2)}</pre>
                                </div>
                            </div>
                            <div>
                                <h3 className="text-lg font-bold text-white mb-4 flex items-center gap-2">
                                    <Server className="w-5 h-5 text-green-400" />
                                    System Information
                                </h3>
                                <div className="space-y-4">
                                    <InfoRow label="Host" value={asset.host} />
                                    <InfoRow label="Last Scanned" value={new Date(asset.updated_at).toLocaleString()} />
                                    <InfoRow
                                        label="Environment"
                                        value={
                                            <span className={`px-2 py-0.5 rounded text-xs font-bold ${asset.environment === 'Production' ? 'bg-red-500/10 text-red-400' : 'bg-green-500/10 text-green-400'
                                                }`}>
                                                {asset.environment}
                                            </span>
                                        }
                                    />
                                    {/* Removing zone which caused error, fallback if needed or use optional chaining safely if type updated */}
                                    {/* <InfoRow label="Zone" value={asset.zone || 'Default'} /> */}
                                </div>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

function TabButton({ active, onClick, label, icon }: { active: boolean, onClick: () => void, label: string, icon: React.ReactNode }) {
    return (
        <button
            onClick={onClick}
            className={`
                flex items-center gap-2 px-6 py-3 text-sm font-medium transition-all relative
                ${active
                    ? 'text-white border-b-2 border-blue-500 bg-secondary rounded-t-lg'
                    : 'text-muted-foreground hover:text-white hover:bg-muted'
                }
            `}
        >
            {icon}
            {label}
        </button>
    );
}

function InfoRow({ label, value }: { label: string, value: React.ReactNode }) {
    return (
        <div className="flex items-center justify-between py-3 border-b border-border last:border-0">
            <span className="text-slate-500 font-medium text-sm">{label}</span>
            <span className="text-slate-200 font-mono text-sm">{value}</span>
        </div>
    );
}
