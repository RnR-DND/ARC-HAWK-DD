import { get } from '@/utils/api-client';
import { unwrapResponse } from '@/lib/api-utils';
import { RiskDistribution } from '@/types/api';

const SEVERITY_ORDER = ['Critical', 'High', 'Medium', 'Low', 'Info'];

function normalizeRiskDistribution(raw: any): RiskDistribution {
    const total = Number(raw?.total) || 0;
    const rawDist = raw?.distribution;
    let entries: Array<{ severity: string; count: number }>;

    if (Array.isArray(rawDist)) {
        entries = rawDist.map((d: any) => ({ severity: String(d.severity), count: Number(d.count) || 0 }));
    } else if (rawDist && typeof rawDist === 'object') {
        entries = Object.entries(rawDist).map(([severity, count]) => ({ severity, count: Number(count) || 0 }));
    } else {
        entries = [];
    }

    entries.sort((a, b) => {
        const ai = SEVERITY_ORDER.indexOf(a.severity);
        const bi = SEVERITY_ORDER.indexOf(b.severity);
        return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi);
    });

    const distribution = entries.map(({ severity, count }) => ({
        severity,
        count,
        percentage: total > 0 ? Math.round((count / total) * 1000) / 10 : 0,
    }));

    return { distribution, total, last_updated: String(raw?.last_updated ?? '') };
}

/**
 * Analytics API service — wraps raw fetch() calls in the service layer
 * so that all analytics requests go through the shared api-client (auth
 * headers, base-URL, error normalisation) instead of hitting the backend
 * directly via window.fetch().
 */
export const analyticsApi = {
    /**
     * Fetch the PII heatmap — asset types × PII categories with density/intensity.
     */
    getHeatmap: async (): Promise<any> => {
        const response = await get<any>('/analytics/heatmap');
        return unwrapResponse(response, null);
    },

    /**
     * Fetch 30-day (or custom) risk trend timeline.
     */
    getTrends: async (days: number = 30): Promise<any> => {
        const response = await get<any>(`/analytics/trends?days=${days}`);
        return unwrapResponse(response, null);
    },

    /**
     * Fetch risk distribution buckets (Critical / High / Medium / Low counts).
     */
    getRiskDistribution: async (): Promise<RiskDistribution | null> => {
        const response = await get<any>('/analytics/risk-distribution');
        const raw = unwrapResponse(response, null);
        return raw ? normalizeRiskDistribution(raw) : null;
    },
};

export default analyticsApi;
