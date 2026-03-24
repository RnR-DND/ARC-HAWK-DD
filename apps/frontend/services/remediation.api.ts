import { post, get } from '@/utils/api-client';

export interface ExecuteRemediationRequest {
    finding_ids: string[];
    action_type: 'MASK' | 'DELETE' | 'ENCRYPT';
    user_id: string;
}

export interface ExecuteRemediationResponse {
    action_ids: string[];
    success: number;
    failed: number;
    errors?: string[];
}

export interface RemediationEvent {
    id: string;
    action: 'MASK' | 'DELETE' | 'ANONYMIZE';
    target: string;
    executed_by: string;
    executed_at: string;
    scan_id?: string;
    status: 'COMPLETED' | 'FAILED' | 'ROLLED_BACK' | 'PENDING';
    finding_id?: string;
    asset_id?: string;
}

export interface RemediationHistoryResponse {
    history: RemediationEvent[];
    total: number;
}

export const remediationApi = {
    executeRemediation: async (data: ExecuteRemediationRequest): Promise<ExecuteRemediationResponse> => {
        return await post<ExecuteRemediationResponse>('/remediation/execute', data);
    },

    getRemediationHistory: async (params?: {
        limit?: number;
        offset?: number;
        action?: string;
    }): Promise<RemediationHistoryResponse> => {
        const queryParams = new URLSearchParams();
        if (params?.limit) queryParams.append('limit', params.limit.toString());
        if (params?.offset) queryParams.append('offset', params.offset.toString());
        if (params?.action) queryParams.append('action', params.action);

        const query = queryParams.toString();
        return await get<RemediationHistoryResponse>(
            `/remediation/history${query ? `?${query}` : ''}`
        );
    }
};

export default remediationApi;
