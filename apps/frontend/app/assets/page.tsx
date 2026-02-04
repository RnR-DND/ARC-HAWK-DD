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

    return (
        <div className="flex flex-col h-full bg-background p-8">
            <div className="mb-8">
                <h1 className="text-3xl font-extrabold text-white mb-2 tracking-tight">
                    Asset Inventory
                </h1>
                <div className="flex items-center gap-2 text-sm">
                    <span className="px-2 py-0.5 rounded-full bg-muted text-muted-foreground font-bold text-xs">
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
                <div className="bg-card border border-border rounded-xl overflow-hidden shadow-sm">
                    <AssetTable
                        assets={assets}
                        total={total}
                        loading={loading}
                        onAssetClick={handleAssetClick}
                    />
                </div>
            )}
        </div>
    );
}
