'use client';

import React, { useEffect, useState } from 'react';
import { assetsApi } from '@/services/assets.api';
import MaskingButton from './MaskingButton';
import { Asset } from '@/types';

interface AssetDetailsPanelProps {
    assetId: string;
    onClose: () => void;
}

export default function AssetDetailsPanel({ assetId, onClose }: AssetDetailsPanelProps) {
    const [asset, setAsset] = useState<Asset | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchAsset = async () => {
            setLoading(true);
            try {
                const data = await assetsApi.getAsset(assetId);
                setAsset(data);
            } catch (err: any) {
                console.error(err);
            } finally {
                setLoading(false);
            }
        };
        if (assetId) fetchAsset();
    }, [assetId]);

    const handleMaskingComplete = () => {
        // Refresh asset data after masking
        const fetchAsset = async () => {
            try {
                const data = await assetsApi.getAsset(assetId);
                setAsset(data);
            } catch (err: any) {
                console.error(err);
            }
        };
        fetchAsset();
    };

    if (!assetId) return null;

    return (
        <div className="fixed right-0 top-0 bottom-0 w-[400px] max-w-full bg-white border-l border-slate-200 shadow-2xl z-50 p-6 flex flex-col overflow-y-auto">
            <div className="flex justify-between items-center mb-6 border-b border-slate-200 pb-4">
                <h2 className="m-0 text-xl font-semibold text-slate-900">Asset Details</h2>
                <button 
                    onClick={onClose} 
                    className="bg-transparent border-none text-2xl text-slate-500 cursor-pointer hover:text-slate-700 transition-colors"
                    aria-label="Close"
                >
                    &times;
                </button>
            </div>

            {loading ? (
                <div className="text-slate-500 py-4">Loading details...</div>
            ) : asset ? (
                <div className="flex-1">
                    {/* Masking Status Badge */}
                    {asset.is_masked && (
                        <div className="px-4 py-3 rounded-lg mb-5 text-sm font-medium bg-green-50 border border-green-200 text-green-800">
                            🔒 Masked with {asset.masking_strategy}
                            {asset.masked_at && (
                                <div className="text-xs mt-1 opacity-80">
                                    Masked on {new Date(asset.masked_at).toLocaleString()}
                                </div>
                            )}
                        </div>
                    )}

                    {(!asset.is_masked) && asset.total_findings > 0 && (
                        <div className="px-4 py-3 rounded-lg mb-5 text-sm font-medium bg-amber-50 border border-amber-200 text-amber-800">
                            🔓 Unmasked - {asset.total_findings} PII findings
                        </div>
                    )}

                    <div className="mb-5">
                        <label className="block text-xs uppercase tracking-wide text-slate-500 mb-2">Name</label>
                        <div className="text-base font-medium text-slate-900">{asset.name}</div>
                    </div>

                    <div className="mb-5">
                        <label className="block text-xs uppercase tracking-wide text-slate-500 mb-2">Type</label>
                        <div className="text-sm text-slate-700">{asset.asset_type}</div>
                    </div>

                    <div className="mb-5">
                        <label className="block text-xs uppercase tracking-wide text-slate-500 mb-2">Environment</label>
                        <div>
                            <span className={`px-2 py-0.5 rounded font-medium text-xs ${asset.environment === 'Production' ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'}`}>
                                {asset.environment || 'Unknown'}
                            </span>
                        </div>
                    </div>

                    <div className="mb-5">
                        <label className="block text-xs uppercase tracking-wide text-slate-500 mb-2">Owner</label>
                        <div className="text-sm text-slate-700">{asset.owner || 'Unassigned'}</div>
                    </div>

                    <div className="mb-5">
                        <label className="block text-xs uppercase tracking-wide text-slate-500 mb-2">Source System</label>
                        <div className="font-mono bg-slate-100 px-2 py-1 rounded text-slate-800 text-sm inline-block">
                            {asset.source_system || asset.host}
                        </div>
                    </div>

                    <div className="mb-5">
                        <label className="block text-xs uppercase tracking-wide text-slate-500 mb-2">Full Path</label>
                        <div className="whitespace-pre-wrap break-all font-mono text-xs text-slate-700 bg-slate-50 p-2 rounded border border-slate-100" title={asset.path}>
                            {asset.path}
                        </div>
                    </div>

                    <div className="mb-5">
                        <label className="block text-xs uppercase tracking-wide text-slate-500 mb-2">Risk Score</label>
                        <div>
                            <span className={`font-semibold ${asset.risk_score > 80 ? 'text-red-600' : 'text-green-600'}`}>
                                {asset.risk_score}/100
                            </span>
                        </div>
                    </div>

                    {/* Masking Action Button */}
                    {asset.total_findings > 0 && (
                        <div className="my-5 p-4 bg-slate-50 rounded-lg border border-slate-200">
                            <MaskingButton
                                assetId={assetId}
                                assetName={asset.name}
                                findingsCount={asset.total_findings}
                                onMaskingComplete={handleMaskingComplete}
                            />
                        </div>
                    )}

                    {asset.file_metadata && (
                        <div className="mb-5">
                            <label className="block text-xs uppercase tracking-wide text-slate-500 mb-2">Metadata</label>
                            <pre className="bg-slate-100 p-3 rounded-md text-xs text-slate-700 overflow-x-auto border border-slate-200">
                                {JSON.stringify(asset.file_metadata, null, 2)}
                            </pre>
                        </div>
                    )}
                </div>
            ) : (
                <div className="text-red-500 p-4 bg-red-50 border border-red-100 rounded-lg">
                    Failed to load asset details.
                </div>
            )}
        </div>
    );
}
