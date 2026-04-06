/**
 * Assets API Service
 * 
 * Service for asset-specific API calls
 */

import { get, del, apiClient } from '@/utils/api-client';
import { Asset } from '@/types';

// ============================================
// ASSETS API
// ============================================

/**
 * Get asset by ID
 */
export async function getAsset(id: string): Promise<Asset> {
    try {
        const res = await get<{ data: Asset }>(`/assets/${id}`);
        return res.data;
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
        // Backend returns: { total: number, data: Asset[] }
        const res = await get<{ data: Asset[]; total: number }>('/assets', params);
        return {
            assets: res.data || [],
            total: res.total || 0
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
