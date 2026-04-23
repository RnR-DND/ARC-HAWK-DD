import { get } from '@/utils/api-client';
import { ClassificationSummary, Finding } from '@/types';

export interface DashboardMetrics {
    totalPII: number;
    highRiskFindings: number;
    assetsHit: number;
    actionsRequired: number;
}

export interface DashboardFinding {
    id: string;
    assetName: string;
    assetPath: string;
    field: string;
    piiType: string;
    confidence: number;
    risk: 'High' | 'Medium' | 'Low';
    sourceType: 'Database' | 'File' | 'Cloud' | 'API';
}

export interface DashboardData {
    metrics: DashboardMetrics;
    recentFindings: DashboardFinding[];
    riskDistribution: Record<string, number>;
    riskByAsset: Record<string, number>;
    riskByConfidence: Record<string, number>;
    latestScanId: string | null;
}

async function getMetricsFromSummary(summary: ClassificationSummary | null): Promise<DashboardMetrics> {
    if (!summary) {
        return { totalPII: 0, highRiskFindings: 0, assetsHit: 0, actionsRequired: 0 };
    }

    const highRiskCount = (summary.by_severity?.['Critical'] ?? 0) + (summary.by_severity?.['High'] ?? 0);

    return {
        totalPII: summary.total ?? 0,
        highRiskFindings: highRiskCount,
        assetsHit: (summary as any).assets_hit ?? (summary as any).total_assets ?? 0,
        actionsRequired: (summary.total ?? 0) - (summary.verified_count ?? 0) - (summary.false_positive_count ?? 0)
    };
}

async function getRecentFindings(): Promise<DashboardFinding[]> {
    try {
        const response = await get<any>('/findings', {
            page: 1,
            page_size: 10,
            severity: 'High,Medium'
        });

        // Backend returns { data: { findings: [...] } }
        const findings = response?.data?.findings || response?.findings || [];

        return findings.slice(0, 5).map((f: any) => ({
            id: f.id || f.finding_id,
            assetName: f.asset_name || f.asset?.name || 'Unknown Asset',
            assetPath: f.asset_path || f.asset?.path || f.field_name || '',
            field: f.field_name || f.matches?.[0] || '',
            piiType: f.pattern_name || f.classifications?.[0]?.classification_type || 'Unknown',
            confidence: f.confidence_score || f.confidence || 0,
            risk: mapSeverityToRisk(f.severity || f.risk),
            sourceType: mapSourceType(f.source_type || f.asset?.asset_type || f.data_source)
        }));
    } catch (error) {
        console.error('Failed to fetch recent findings:', error);
        return [];
    }
}

function mapSeverityToRisk(severity: string): 'High' | 'Medium' | 'Low' {
    const s = severity?.toLowerCase();
    if (s === 'high' || s === 'critical') return 'High';
    if (s === 'medium') return 'Medium';
    return 'Low';
}

function mapSourceType(sourceType: string): 'Database' | 'File' | 'Cloud' | 'API' {
    if (!sourceType) return 'Database';
    const s = sourceType.toLowerCase();
    if (s.includes('s3') || s.includes('bucket') || s.includes('cloud') || s.includes('gcs')) return 'Cloud';
    if (s.includes('fs') || s.includes('file') || s.includes('filesystem')) return 'File';
    if (s.includes('api') || s.includes('endpoint')) return 'API';
    return 'Database';
}

async function getRiskDistribution(): Promise<{
    byPiiType: Record<string, number>;
    byAsset: Record<string, number>;
    byConfidence: Record<string, number>;
}> {
    try {
        const summaryRes = await get<any>('/classification/summary');
        // Backend returns { data: {...} } wrapper
        const summary: ClassificationSummary = summaryRes?.data ?? summaryRes;

        const byPiiType: Record<string, number> = {};
        const byAsset: Record<string, number> = {};
        const byConfidence: Record<string, number> = {
            '> 90% (High)': 0,
            '70-90% (Med)': 0,
            '< 70% (Low)': 0
        };

        if (summary?.by_type) {
            for (const [piiType, data] of Object.entries(summary.by_type as Record<string, any>)) {
                byPiiType[piiType] = data.count || 0;
            }
        }

        const findingsRes = await get<any>('/findings', {
            page: 1,
            page_size: 50
        });

        // Backend returns { data: { findings: [...] } }
        const findings = findingsRes?.data?.findings || findingsRes?.findings || [];

        for (const f of findings) {
            const assetName = f.asset_name || f.asset?.name || 'Unknown';
            byAsset[assetName] = (byAsset[assetName] || 0) + 1;

            const conf = f.confidence_score || f.confidence || 0;
            if (conf > 0.9) byConfidence['> 90% (High)']++;
            else if (conf >= 0.7) byConfidence['70-90% (Med)']++;
            else byConfidence['< 70% (Low)']++;
        }

        return { byPiiType: byPiiType || {}, byAsset, byConfidence };
    } catch (error) {
        console.error('Failed to fetch risk distribution:', error);
        return {
            byPiiType: {},
            byAsset: {},
            byConfidence: {}
        };
    }
}

export const dashboardApi = {
    async getDashboardData(): Promise<DashboardData> {
        try {
            // Try new dashboard metrics endpoint first (more accurate)
            let metrics: DashboardMetrics;
            try {
                const metricsRes = await get<any>('/dashboard/metrics');
                if (metricsRes) {
                    metrics = {
                        totalPII: metricsRes.total_pii ?? 0,
                        highRiskFindings: metricsRes.high_risk_findings ?? 0,
                        assetsHit: metricsRes.assets_hit ?? 0,
                        actionsRequired: metricsRes.actions_required ?? 0
                    };
                } else {
                    throw new Error('No metrics data');
                }
            } catch {
                // Fallback: derive metrics from classification summary
                const summaryRaw = await get<any>('/classification/summary').catch(() => null);
                const summary: ClassificationSummary | null = summaryRaw?.data ?? summaryRaw;
                metrics = await getMetricsFromSummary(summary);
            }

            const latestScan = await get<any>('/scans/latest').catch(() => null);

            const [recentFindings, riskDist] = await Promise.all([
                getRecentFindings(),
                getRiskDistribution()
            ]);

            // Backend wraps scan in { data: {...} }
            const latestScanData = latestScan?.data ?? latestScan;
            return {
                metrics,
                recentFindings,
                riskDistribution: riskDist.byPiiType,
                riskByAsset: riskDist.byAsset,
                riskByConfidence: riskDist.byConfidence,
                latestScanId: latestScanData?.id || latestScanData?.scan_id || null
            };
        } catch (error) {
            console.error('Failed to fetch dashboard data:', error);
            throw error;
        }
    }
};

export interface RiskTrendPoint {
    date: string;
    score: number;
    scan_count: number;
}

export const getRiskTrend = async (days = 30): Promise<RiskTrendPoint[]> => {
    try {
        const res = await get<any>(`/dashboard/risk-trend?days=${days}`);
        return Array.isArray(res?.trend) ? res.trend : [];
    } catch {
        return [];
    }
};

export default dashboardApi;
