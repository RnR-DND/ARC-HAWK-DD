// Discovery module API client.
//
// Wraps the /api/v1/discovery/* endpoints. All methods return parsed JSON or throw.
// Auth headers are handled by the global fetch interceptor (same as other modules).

const API_BASE = '/api/v1/discovery';

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

async function jsonFetch<T>(url: string, init?: RequestInit): Promise<T> {
    const res = await fetch(url, {
        ...init,
        headers: {
            'Content-Type': 'application/json',
            ...(init?.headers || {}),
        },
        credentials: 'include',
    });
    if (!res.ok) {
        let body = '';
        try {
            body = await res.text();
        } catch {
            // ignore
        }
        throw new Error(`Discovery API ${res.status}: ${body || res.statusText}`);
    }
    return res.json() as Promise<T>;
}

export const discoveryApi = {
    getOverview: () => jsonFetch<OverviewSummary>(`${API_BASE}/overview`),

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
        return jsonFetch<{ items: InventoryRow[]; count: number; limit: number; offset: number }>(
            `${API_BASE}/inventory${qs ? `?${qs}` : ''}`
        );
    },

    listSnapshots: (limit = 50, offset = 0) =>
        jsonFetch<{ items: Snapshot[]; count: number }>(
            `${API_BASE}/snapshots?limit=${limit}&offset=${offset}`
        ),

    getSnapshot: (id: string) =>
        jsonFetch<{ snapshot: Snapshot; facts: unknown[] }>(`${API_BASE}/snapshots/${id}`),

    triggerSnapshot: () =>
        jsonFetch<{ snapshot_id: string; status: string; asset_count: number; duration_ms: number }>(
            `${API_BASE}/snapshots/trigger`,
            { method: 'POST' }
        ),

    getRiskOverview: () =>
        jsonFetch<{ hotspots: RiskHotspot[]; weights: Record<string, number> }>(
            `${API_BASE}/risk/overview`
        ),

    getRiskHotspots: (limit = 10) =>
        jsonFetch<{ hotspots: RiskHotspot[]; count: number }>(
            `${API_BASE}/risk/hotspots?limit=${limit}`
        ),

    getDriftTimeline: (limit = 100) =>
        jsonFetch<{ snapshot_id: string; events: DriftEvent[]; count: number }>(
            `${API_BASE}/drift/timeline?limit=${limit}`
        ),

    generateReport: (format: 'html' | 'pdf' | 'csv' | 'json' = 'html', snapshotId?: string) =>
        jsonFetch<{ report_id: string; status: string; format: string }>(
            `${API_BASE}/reports/generate`,
            {
                method: 'POST',
                body: JSON.stringify({ format, snapshot_id: snapshotId ?? null }),
            }
        ),

    getReport: (id: string) => jsonFetch<Report>(`${API_BASE}/reports/${id}`),

    downloadReportUrl: (id: string) => `${API_BASE}/reports/${id}/download`,

    listReports: (limit = 50, offset = 0) =>
        jsonFetch<{ items: Report[]; count: number }>(
            `${API_BASE}/reports?limit=${limit}&offset=${offset}`
        ),

    getGlossary: () =>
        jsonFetch<{ terms: GlossaryTerm[]; count: number }>(
            `${API_BASE}/glossary`
        ),
};
