'use client';

import React, { useEffect, useState } from 'react';
import AssetTable from '@/components/AssetTable';
import LoadingState from '@/components/LoadingState';
import { assetsApi } from '@/services/assets.api';
import { Asset } from '@/types';
import { useRouter } from 'next/navigation';

export default function AssetInventoryPage() {
    const router = useRouter();
    const [assets, setAssets] = useState<Asset[]>([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetchAssets();
    }, []);

    const fetchAssets = async () => {
        try {
            setLoading(true);
            const response = await assetsApi.getAssets({ page: 1, page_size: 50 });
            setAssets(response.assets);
            setTotal(response.total);
        } catch (error) {
            console.error('Failed to fetch assets:', error);
        } finally {
            setLoading(false);
        }
    };

    const handleAssetClick = (assetId: string) => {
        router.push(`/assets/${assetId}`);
    };

    const handleDeleteAsset = async (assetId: string) => {
        if (!confirm('Are you sure you want to delete this asset and all its findings?')) return;
        try {
            await assetsApi.deleteAsset(assetId);
            setAssets(prev => prev.filter(a => a.id !== assetId));
            setTotal(prev => prev - 1);
        } catch (error) {
            console.error('Failed to delete asset:', error);
        }
    };

    return (
        <div className="flex flex-col h-full bg-slate-50 p-8">
            <div className="mb-8">
                <h1 className="text-3xl font-extrabold text-slate-900 mb-2 tracking-tight">
                    Asset Inventory
                </h1>
                <div className="flex items-center gap-2 text-sm">
                    <span className="px-2 py-0.5 rounded-full bg-slate-100 text-slate-600 font-bold text-xs">
                        Total: {total}
                    </span>
                    <span className="text-slate-500">
                        Canonical source of truth for all tracked data assets.
                    </span>
                </div>
            </div>

            {loading ? (
                <LoadingState message="Syncing Asset Inventory..." />
            ) : (
                <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm">
                    <AssetTable
                        assets={assets}
                        total={total}
                        loading={loading}
                        onAssetClick={handleAssetClick}
                        onDeleteAsset={handleDeleteAsset}
                    />
                </div>
            )}
        </div>
    );
}
