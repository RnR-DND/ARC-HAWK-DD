import { post, get } from '@/utils/api-client';

export interface MaskAssetRequest {
    asset_id: string;
    strategy: 'REDACT' | 'PARTIAL' | 'TOKENIZE';
    masked_by?: string;
}

export interface MaskingStatusResponse {
    asset_id: string;
    is_masked: boolean;
    masking_strategy?: string;
    masked_at?: string;
    findings_count: number;
}

export interface MaskingAuditEntry {
    id: string;
    asset_id: string;
    masked_by: string;
    masking_strategy: string;
    findings_count: number;
    masked_at: string;
    metadata?: Record<string, unknown>;
    created_at: string;
}

export interface MaskingAuditLogResponse {
    asset_id: string;
    audit_log: MaskingAuditEntry[];
}

export const maskingApi = {
    async maskAsset(request: MaskAssetRequest): Promise<{ message: string; asset_id: string; strategy: string }> {
        return post('/masking/mask-asset', request);
    },

    async getMaskingStatus(assetId: string): Promise<MaskingStatusResponse> {
        return get(`/masking/status/${assetId}`);
    },

    async getMaskingAuditLog(assetId: string): Promise<MaskingAuditLogResponse> {
        return get(`/masking/audit/${assetId}`);
    },
};
