// Neo4j Lineage API - Unified Neo4j-Only Endpoint
// Uses /api/v1/lineage with 3-level hierarchy (System -> Asset -> PII_Category)

import { apiClient } from '@/utils/api-client';
import {
    LineageNode,
    LineageEdge,
} from '../modules/lineage/lineage.types';

export interface LineageHierarchy {
    nodes: LineageNode[];
    edges: LineageEdge[];
}

export interface PIIAggregation {
    pii_type: string;
    total_findings: number;
    risk_level: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
    confidence: number;
    affected_assets: number;
    affected_systems: number;
    categories: string[];
}

export interface LineageResponse {
    hierarchy: LineageHierarchy;
    aggregations: {
        by_pii_type: PIIAggregation[];
        total_assets: number;
        total_pii_types: number;
    };
}

export async function fetchLineage(
    systemFilter?: string,
    riskFilter?: string,
    assetId?: string
): Promise<LineageResponse> {
    const params: Record<string, string> = {};
    if (systemFilter) params.system = systemFilter;
    if (riskFilter) params.risk = riskFilter;
    if (assetId) params.asset_id = assetId;

    const response = await apiClient.get('/lineage', { params });
    return response.data.data; // Backend wraps in { status: "success", data: {...} }
}

export async function getLineage(assetId?: string, depth?: number): Promise<LineageHierarchy> {
    try {
        const params: Record<string, string> = {};
        if (assetId) params.assetId = assetId;
        if (depth) params.depth = depth.toString();

        const response = await apiClient.get('/lineage', { params });
        const result = response.data;

        // Backend returns: { data: { hierarchy: { nodes, edges }, ... }, status }
        if (result.data) {
            if (result.data.hierarchy) {
                return {
                    nodes: result.data.hierarchy.nodes || [],
                    edges: result.data.hierarchy.edges || []
                };
            }
            return {
                nodes: result.data.nodes || [],
                edges: result.data.edges || []
            };
        }

        return { nodes: [], edges: [] };
    } catch (error) {
        console.error('Failed to fetch lineage:', error);
        throw error;
    }
}

export async function fetchLineageStats(): Promise<LineageResponse['aggregations']> {
    const response = await apiClient.get('/lineage/stats');
    return response.data.stats;
}

export async function syncLineage(): Promise<{ status: string; message?: string }> {
    const response = await apiClient.post('/lineage/sync');
    return response.data;
}

// Export as lineageApi for backward compatibility
export const lineageApi = {
    fetchLineage,
    fetchLineageStats,
    getLineage,
    syncLineage,
};
