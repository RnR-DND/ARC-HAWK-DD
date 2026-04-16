// Shared API type definitions

export type RiskTier = 'critical' | 'high' | 'medium' | 'low';
export type Severity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'INFO';
export type FindingStatus = 'OPEN' | 'CONFIRMED' | 'FALSE_POSITIVE' | 'REMEDIATED' | 'SUPPRESSED';
export type ScanStatus = 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED' | 'CANCELLED';

export interface Finding {
    id: string;
    tenant_id: string;
    scan_run_id: string;
    asset_id: string;
    pattern_name: string;
    pii_type: string;
    severity: Severity;
    status: FindingStatus;
    confidence_score: number;
    sample_value?: string;
    context?: string;
    column_name?: string;
    table_name?: string;
    row_count?: number;
    created_at: string;
    updated_at: string;
}

export interface ScanConfig {
    id: string;
    tenant_id: string;
    connection_id: string;
    name: string;
    scan_type: string;
    schedule?: string;
    enabled: boolean;
    created_at: string;
}

export interface Connection {
    id: string;
    tenant_id: string;
    name: string;
    type: string;
    host?: string;
    port?: number;
    database?: string;
    status: 'active' | 'inactive' | 'error';
    last_tested?: string;
    created_at: string;
}

export interface User {
    id: string;
    tenant_id: string;
    email: string;
    name: string;
    role: string;
    is_active: boolean;
    created_at: string;
}

export interface AuthResponse {
    user: User;
    access_token?: string;
    expires_in?: number;
}

export interface DPDPObligation {
    section: string;
    title: string;
    status: 'COMPLIANT' | 'NON_COMPLIANT' | 'PARTIAL' | 'NOT_ASSESSED';
    score: number;
    gaps: Array<{
        id: string;
        description: string;
        severity: string;
        recommendation: string;
    }>;
    last_checked: string;
}

export interface ComplianceOverview {
    compliance_score: number;
    compliant_assets: number;
    total_assets: number;
    total_findings: number;
    critical_findings: number;
    high_findings: number;
    obligations: DPDPObligation[];
    last_updated: string;
}

export interface RiskDistribution {
    distribution: Array<{
        severity: string;
        count: number;
        percentage: number;
    }>;
    total: number;
    last_updated: string;
}

export interface PaginatedResponse<T> {
    data: T[];
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
}
