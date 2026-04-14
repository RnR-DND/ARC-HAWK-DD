import { get, post } from '@/utils/api-client';
import { unwrapResponse, unwrapArray } from '@/lib/api-utils';

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
            return unwrapArray<RetentionViolation>(res);
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
            return unwrapResponse<RetentionPolicy | null>(res, null);
        } catch {
            return null;
        }
    },

    getDPDPAGaps: async (): Promise<DPDPAGapReport | null> => {
        try {
            const raw = await get<any>('/compliance/dpdpa/gaps');
            const body: any = unwrapResponse<any>(raw, null);
            if (!body) return null;

            // Backend returns { gaps_by_section: { "Sec4_LawfulProcessing": [...], ... }, summary: {...} }
            // Frontend expects { sections: SectionSummary[], gaps: ObligationGap[], total_gaps: number }
            const gapsBySection: Record<string, any[]> = body.gaps_by_section ?? {};
            const gaps: ObligationGap[] = [];
            const sections: SectionSummary[] = [];

            for (const [sectionKey, items] of Object.entries(gapsBySection)) {
                // Strip suffix after underscore: "Sec4_LawfulProcessing" → "Sec4"
                const section = sectionKey.replace(/_.*$/, '');
                const sectionItems = Array.isArray(items) ? items : [];

                const gapCount = sectionItems.filter(i => i.status === 'fail').length;
                const passCount = sectionItems.filter(i => i.status === 'pass').length;
                sections.push({
                    section,
                    section_title: section,
                    total_assets: sectionItems.length,
                    gaps: gapCount,
                    pass: passCount,
                });

                for (const item of sectionItems) {
                    gaps.push({
                        asset_id: item.asset_id,
                        asset_name: item.asset_name,
                        section,
                        section_title: section,
                        status: item.status === 'fail' ? 'gap' : 'pass',
                        evidence: item.detail ?? '',
                    });
                }
            }

            return {
                generated_at: body.generated_at ?? new Date().toISOString(),
                total_assets: body.total_assets ?? 0,
                total_gaps: body.summary?.total_gaps ?? gaps.filter(g => g.status === 'gap').length,
                sections,
                gaps,
            };
        } catch {
            return null;
        }
    },

    getDPDPAReportUrl: (): string => {
        return '/api/v1/compliance/dpdpa/report';
    },
};
