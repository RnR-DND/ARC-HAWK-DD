import { post, get } from '@/utils/api-client';
import { unwrapResponse } from '@/lib/api-utils';

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
    action: 'MASK' | 'DELETE' | 'ENCRYPT';
    target: string;
    executed_by: string;
    executed_at: string;
    scan_id?: string;
    status: 'COMPLETED' | 'FAILED' | 'ROLLED_BACK' | 'PENDING';
    finding_id?: string;
    asset_id?: string;
    // Enriched fields from backend JOIN with assets + findings
    asset_name?: string;
    asset_path?: string;
    pii_type?: string;
    risk_level?: string;
    pattern_name?: string;
    severity?: string;
}

export interface RemediationHistoryResponse {
    history: RemediationEvent[];
    total: number;
}

export interface SOP {
    issue_type: string;
    title: string;
    steps: string[];
    severity?: string;
    references?: string[];
}

export interface SOPListResponse {
    sops: SOP[];
}

export interface EscalationPreview {
    findings: Array<{
        id: string;
        asset_name?: string;
        pii_type?: string;
        risk_level?: string;
        days_open?: number;
    }>;
    total: number;
}

export interface EscalationRunResponse {
    escalated: number;
    message: string;
}

export const remediationApi = {
    executeRemediation: async (data: ExecuteRemediationRequest): Promise<ExecuteRemediationResponse> => {
        return await post<ExecuteRemediationResponse>('/remediation/execute', data);
    },

    rollback: async (id: string): Promise<void> => {
        await post<void>(`/remediation/rollback/${id}`, {});
    },

    getRemediationHistory: async (params?: {
        limit?: number;
        offset?: number;
        action?: string;
        assetId?: string;
    }): Promise<RemediationHistoryResponse> => {
        const queryParams = new URLSearchParams();
        if (params?.limit) queryParams.append('limit', params.limit.toString());
        if (params?.offset) queryParams.append('offset', params.offset.toString());
        if (params?.action) queryParams.append('action', params.action);
        if (params?.assetId) queryParams.append('asset_id', params.assetId);

        const query = queryParams.toString();
        const response = await get<any>(
            `/remediation/history${query ? `?${query}` : ''}`
        );
        return unwrapResponse<RemediationHistoryResponse>(response, { history: [], total: 0 });
    },

    getSOPs: async (): Promise<SOPListResponse> => {
        const response = await get<any>('/remediation/sops');
        return unwrapResponse<SOPListResponse>(response, { sops: [] });
    },

    getSOPByType: async (issueType: string): Promise<SOP | null> => {
        const response = await get<any>(`/remediation/sops/${encodeURIComponent(issueType)}`);
        return response;
    },

    previewEscalation: async (): Promise<EscalationPreview> => {
        const response = await get<any>('/remediation/escalation/preview');
        return unwrapResponse<EscalationPreview>(response, { findings: [], total: 0 });
    },

    runEscalation: async (): Promise<EscalationRunResponse> => {
        return await post<EscalationRunResponse>('/remediation/escalation/run', {});
    },
};

export default remediationApi;
