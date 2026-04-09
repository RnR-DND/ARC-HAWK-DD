import { get, post } from '@/utils/api-client';

// DPDPA obligation gap types (from dpdpa_obligation_service.go)
export interface ObligationGap {
    asset_id: string;
    asset_name: string;
    section: string;       // e.g. "Sec4", "Sec8"
    section_title: string; // e.g. "Lawful Processing"
    status: 'pass' | 'gap';
    evidence: string;
}

export interface SectionSummary {
    section: string;
    section_title: string;
    total_assets: number;
    gaps: number;
    pass: number;
}

export interface DPDPAGapReport {
    generated_at: string;
    total_assets: number;
    total_gaps: number;
    sections: SectionSummary[];
    gaps: ObligationGap[];
}

export interface RetentionViolation {
    finding_id: string;
    asset_id: string;
    asset_name: string;
    pii_type: string;
    first_detected_at: string;
    retention_policy_days: number;
    deletion_due_at: string;
    days_overdue: number;
}

export interface RetentionPolicy {
    asset_id: string;
    retention_days: number;
    policy_name: string;
    policy_basis: string;
}

export const complianceApi = {
    getOverview: async (): Promise<any> => {
        const res = await get<any>('/compliance/overview');
        return res;
    },

    getRetentionViolations: async (): Promise<RetentionViolation[]> => {
        try {
            const res = await get<any>('/retention/violations');
            return Array.isArray(res) ? res : (res?.data ?? []);
        } catch {
            return [];
        }
    },

    setRetentionPolicy: async (assetId: string, policy: { policy_days: number; policy_name: string; policy_basis: string }): Promise<void> => {
        await post<any>('/retention/policies', {
            asset_id: assetId,
            policy_days: policy.policy_days,
            policy_name: policy.policy_name,
            policy_basis: policy.policy_basis,
        });
    },

    getRetentionPolicy: async (assetId: string): Promise<RetentionPolicy | null> => {
        try {
            const res = await get<any>(`/retention/policies/${assetId}`);
            return res?.data ?? res;
        } catch {
            return null;
        }
    },

    getDPDPAGaps: async (): Promise<DPDPAGapReport | null> => {
        try {
            const res = await get<any>('/compliance/dpdpa/gaps');
            return res?.data ?? res;
        } catch {
            return null;
        }
    },

    getDPDPAReportUrl: (): string => {
        return '/api/v1/compliance/dpdpa/report';
    },
};
