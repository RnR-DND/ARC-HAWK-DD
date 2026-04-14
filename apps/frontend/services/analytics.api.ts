import { get } from '@/utils/api-client';
import { unwrapResponse } from '@/lib/api-utils';

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
    getRiskDistribution: async (): Promise<any> => {
        const response = await get<any>('/analytics/risk-distribution');
        return unwrapResponse(response, null);
    },
};

export default analyticsApi;
