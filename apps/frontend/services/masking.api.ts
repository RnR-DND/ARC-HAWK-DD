const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

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
    metadata?: Record<string, any>;
    created_at: string;
}

export interface MaskingAuditLogResponse {
    asset_id: string;
    audit_log: MaskingAuditEntry[];
}

export const maskingApi = {
    /**
     * Mask an asset with the specified strategy
     */
    async maskAsset(request: MaskAssetRequest): Promise<{ message: string; asset_id: string; strategy: string }> {
        const response = await fetch(`${API_BASE_URL}/masking/mask-asset`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(request),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.details || error.error || 'Failed to mask asset');
        }

        return response.json();
    },

    /**
     * Get masking status for an asset
     */
    async getMaskingStatus(assetId: string): Promise<MaskingStatusResponse> {
        const response = await fetch(`${API_BASE_URL}/masking/status/${assetId}`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.details || error.error || 'Failed to get masking status');
        }

        return response.json();
    },

    /**
     * Get masking audit log for an asset
     */
    async getMaskingAuditLog(assetId: string): Promise<MaskingAuditLogResponse> {
        const response = await fetch(`${API_BASE_URL}/masking/audit/${assetId}`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.details || error.error || 'Failed to get audit log');
        }

        return response.json();
    },
};
