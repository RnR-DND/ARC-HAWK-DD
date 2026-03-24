// Neo4j Lineage API - Unified Neo4j-Only Endpoint
// Uses /api/v1/lineage with 3-level hierarchy (System → Asset → PII_Category)

// 3-Level Hierarchy Types (System → Asset → PII_Category)
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

// API Functions
// Use relative URLs in browser to leverage Next.js rewrites
// Use absolute URLs in server-side rendering
const API_BASE = typeof window !== 'undefined'
    ? '' // Browser: use relative URLs (proxied by Next.js)
    : (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'); // SSR: use absolute URLs

export async function fetchLineage(
    systemFilter?: string,
    riskFilter?: string
): Promise<LineageResponse> {
    const params = new URLSearchParams();
    if (systemFilter) params.append('system', systemFilter);
    if (riskFilter) params.append('risk', riskFilter);

    const url = `${API_BASE}/api/v1/lineage${params.toString() ? `?${params}` : ''}`;

    const response = await fetch(url);
    if (!response.ok) {
        throw new Error(`Failed to fetch lineage: ${response.statusText}`);
    }

    const data = await response.json();
    return data.data; // Backend wraps in { status: "success", data: {...} }
}

export async function getLineage(assetId?: string, depth?: number): Promise<LineageHierarchy> {
    try {
        const params = new URLSearchParams();
        if (assetId) params.append('assetId', assetId);
        if (depth) params.append('depth', depth.toString());

        const query = params.toString() ? `?${params.toString()}` : '';
        const response = await fetch(`${API_BASE}/api/v1/lineage${query}`);

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const result = await response.json();

        // Backend returns: { data: { nodes, edges }, status }
        // Extract the nodes and edges directly from data
        // Backend returns: { data: { hierarchy: { nodes, edges }, ... }, status }
        // Extract from hierarchy object if present, otherwise try direct data
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

        // Fallback for empty/malformed response
        return { nodes: [], edges: [] };
    } catch (error) {
        console.error('Failed to fetch lineage:', error);
        throw error;
    }
}

export async function fetchLineageStats(): Promise<LineageResponse['aggregations']> {
    const url = `${API_BASE}/api/v1/lineage/stats`;

    const response = await fetch(url);
    if (!response.ok) {
        throw new Error(`Failed to fetch stats: ${response.statusText}`);
    }

    const data = await response.json();
    return data.stats;
}


// Export as lineageApi for backward compatibility
export const lineageApi = {
    fetchLineage,
    fetchLineageStats,
    getLineage,
};
