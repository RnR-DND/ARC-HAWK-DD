/**
 * Assets API Service
 * 
 * Service for asset-specific API calls
 */

import { get, del, apiClient } from '@/utils/api-client';
import { Asset } from '@/types';
import { unwrapResponse } from '@/lib/api-utils';

// ============================================
// ASSETS API
// ============================================

/**
 * Get asset by ID
 */
export async function getAsset(id: string): Promise<Asset> {
    try {
        const res = await get<any>(`/assets/${id}`);
        const asset = unwrapResponse<Asset | null>(res, null);
        if (!asset) throw new Error('Asset not found');
        return asset;
    } catch (error) {
        console.error(`Error fetching asset ${id}:`, error);
        throw new Error('Failed to fetch asset details');
    }
}

/**
 * Get all assets
 */
export async function getAssets(params?: {
    page?: number;
    page_size?: number;
    sort_by?: string;
}): Promise<{ assets: Asset[]; total: number }> {
    try {
        // Backend returns either { data: Asset[], total: number } or { assets: Asset[], total: number }
        const res = await get<any>('/assets', params);
        const data = unwrapResponse<any>(res, {});
        // Handle both {data:[]} and direct array
        const assets: Asset[] = Array.isArray(data) ? data : (data.assets || data.data || []);
        return {
            assets,
            total: res?.total ?? data?.total ?? assets.length,
        };
    } catch (error) {
        console.error('Error fetching assets:', error);
        throw new Error('Failed to fetch assets');
    }
}

/**
 * Delete asset by ID
 */
export async function deleteAsset(id: string): Promise<void> {
    await del<void>(`/assets/${id}`);
}

export const assetsApi = {
    getAsset,
    getAssets,
    deleteAsset,
};

export default assetsApi;
