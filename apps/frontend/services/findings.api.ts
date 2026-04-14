import { get, post } from '@/utils/api-client';
import apiClient from '@/utils/api-client';
import { FindingsResponse } from '@/types';

export interface FeedbackRequest {
    feedback_type: 'FALSE_POSITIVE' | 'FALSE_NEGATIVE' | 'CONFIRMED';
    original_classification?: string;
    proposed_classification?: string;
    comments?: string;
}

export const findingsApi = {
    submitFeedback: async (findingId: string, feedback: FeedbackRequest): Promise<void> => {
        try {
            await post<void>(`/findings/${findingId}/feedback`, feedback);
        } catch (error) {
            console.error(`Error submitting feedback for finding ${findingId}:`, error);
            throw new Error('Failed to submit feedback');
        }
    },

    getFindings: async (params?: {
        page?: number;
        page_size?: number;
        severity?: string;
        asset_id?: string;
        status?: string;
        asset?: string;
        pii_type?: string;
        search?: string;
    }): Promise<FindingsResponse> => {
        // Backend returns wrapped response: { data: { findings: [], total: ... } }
        const res = await get<any>('/findings', params);

        let findingsList: any[] = [];
        let totalCount = 0;
        let totalPages = 0;

        if (res.data && Array.isArray(res.data.findings)) {
            findingsList = res.data.findings;
            totalCount = res.data.total || 0;
            totalPages = res.data.total_pages || Math.ceil(totalCount / (params?.page_size || 20));
        } else if (Array.isArray(res.data)) {
            // Fallback for legacy unwrapped array
            findingsList = res.data;
            totalCount = findingsList.length;
            totalPages = 1;
        } else if (res.findings && Array.isArray(res.findings)) {
            // Fallback if 'data' wrapper is missing but 'findings' key exists
            findingsList = res.findings;
            totalCount = res.total || 0;
            totalPages = res.total_pages || Math.ceil(totalCount / (params?.page_size || 20));
        }

        return {
            findings: findingsList,
            total: totalCount,
            page: params?.page || 1,
            page_size: params?.page_size || 20,
            total_pages: totalPages
        };
    },

    async getFacets(): Promise<{ pii_types: string[]; assets: string[]; severities: string[] }> {
        try {
            const res = await apiClient.get<{ pii_types: string[]; assets: string[]; severities: string[] }>('/findings/facets')
            return res.data ?? { pii_types: [], assets: [], severities: [] }
        } catch {
            return { pii_types: [], assets: [], severities: [] }
        }
    },
};

export default findingsApi;
