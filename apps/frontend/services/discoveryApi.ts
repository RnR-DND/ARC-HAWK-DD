// Discovery module API client.
//
// Wraps the /api/v1/discovery/* endpoints. All methods return parsed JSON or throw.
// Auth is handled by the shared api-client (axios with withCredentials: true).

import { get, post } from '@/utils/api-client';

export interface InventoryRow {
    id: string;
    tenant_id: string;
    asset_id: string;
    asset_name: string;
    source_id?: string;
    source_name?: string;
    classification: string;
    sensitivity: number;
    finding_count: number;
    last_scanned_at?: string;
    refreshed_at: string;
}

export interface RiskHotspot {
    asset_id: string;
    asset_name: string;
    score: number;
    classification: string;
    finding_count: number;
}

export interface TrendPoint {
    label: string;
    taken_at: string;
    asset_count: number;
    finding_count: number;
    composite_risk_score: number;
}

export interface OverviewSummary {
    source_count: number;
    asset_count: number;
    finding_count: number;
    high_risk_count: number;
    composite_risk_score: number;
    top_hotspots: RiskHotspot[];
    trend_quarters: TrendPoint[];
    last_snapshot_at?: string;
}

export interface Snapshot {
    id: string;
    tenant_id: string;
    taken_at: string;
    source_count: number;
    asset_count: number;
    finding_count: number;
    high_risk_count: number;
    composite_risk_score: number;
    trigger: 'manual' | 'cron';
    triggered_by?: string;
    status: 'pending' | 'running' | 'completed' | 'failed';
    error?: string;
    duration_ms?: number;
    completed_at?: string;
}

export interface DriftEvent {
    id: string;
    tenant_id: string;
    snapshot_id: string;
    event_type:
        | 'asset_added'
        | 'asset_removed'
        | 'classification_changed'
        | 'risk_increased'
        | 'risk_decreased'
        | 'finding_count_spike';
    asset_id: string;
    before_state?: Record<string, unknown>;
    after_state?: Record<string, unknown>;
    severity: 'low' | 'medium' | 'high' | 'critical';
    detected_at: string;
}

export interface GlossaryTerm {
    id: string;
    name: string;
    description: string;
    regulation_refs: string[];
    risk_level: string;
    examples?: string[];
    dpdpa_section?: string;
}

export interface Report {
    id: string;
    tenant_id: string;
    snapshot_id?: string;
    requested_by?: string;
    format: 'pdf' | 'csv' | 'json' | 'html';
    status: 'pending' | 'running' | 'completed' | 'failed';
    content_type?: string;
    error?: string;
    requested_at: string;
    completed_at?: string;
    size_bytes?: number;
}

export const discoveryApi = {
    getOverview: () =>
        get<OverviewSummary>('/discovery/overview'),

    listInventory: (params?: {
        classification?: string;
        source_id?: string;
        search?: string;
        limit?: number;
        offset?: number;
    }) => {
        const q = new URLSearchParams();
        if (params?.classification) q.set('classification', params.classification);
        if (params?.source_id) q.set('source_id', params.source_id);
        if (params?.search) q.set('search', params.search);
        if (params?.limit) q.set('limit', String(params.limit));
        if (params?.offset) q.set('offset', String(params.offset));
        const qs = q.toString();
        return get<{ items: InventoryRow[]; count: number; limit: number; offset: number }>(
            `/discovery/inventory${qs ? `?${qs}` : ''}`
        );
    },

    listSnapshots: (limit = 50, offset = 0) =>
        get<{ items: Snapshot[]; count: number }>(
            `/discovery/snapshots?limit=${limit}&offset=${offset}`
        ),

    getSnapshot: (id: string) =>
        get<{ snapshot: Snapshot; facts: unknown[] }>(`/discovery/snapshots/${id}`),

    triggerSnapshot: () =>
        post<{ snapshot_id: string; status: string; asset_count: number; duration_ms: number }>(
            '/discovery/snapshots/trigger', {}
        ),

    getRiskOverview: () =>
        get<{ hotspots: RiskHotspot[]; weights: Record<string, number> }>(
            '/discovery/risk/overview'
        ),

    getRiskHotspots: (limit = 10) =>
        get<{ hotspots: RiskHotspot[]; count: number }>(
            `/discovery/risk/hotspots?limit=${limit}`
        ),

    getDriftTimeline: (limit = 100) =>
        get<{ snapshot_id: string; events: DriftEvent[]; count: number }>(
            `/discovery/drift/timeline?limit=${limit}`
        ),

    generateReport: (format: 'html' | 'pdf' | 'csv' | 'json' = 'html', snapshotId?: string) =>
        post<{ report_id: string; status: string; format: string }>(
            '/discovery/reports/generate',
            { format, snapshot_id: snapshotId ?? null }
        ),

    getReport: (id: string) => get<Report>(`/discovery/reports/${id}`),

    downloadReportUrl: (id: string) => `/api/v1/discovery/reports/${id}/download`,

    listReports: (limit = 50, offset = 0) =>
        get<{ items: Report[]; count: number }>(
            `/discovery/reports?limit=${limit}&offset=${offset}`
        ),

    getGlossary: () =>
        jsonFetch<{ terms: GlossaryTerm[]; count: number }>(
            `${API_BASE}/glossary`
        ),
};
