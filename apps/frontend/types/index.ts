// TypeScript models matching backend API contracts

export interface ScanRun {
    id: string;
    profile_name: string;
    scan_started_at: string;
    scan_completed_at: string;
    host: string;
    total_findings: number;
    total_assets: number;
    status: string;
    created_at: string;
    updated_at: string;
}

export interface Asset {
    id: string;
    stable_id: string;
    asset_type: string;
    name: string;
    path: string;
    data_source: string;
    host: string;
    environment: string;
    owner: string;
    source_system: string;
    file_metadata?: Record<string, any>;
    risk_score: number;
    total_findings: number;
    is_masked?: boolean;
    masking_strategy?: string;
    masked_at?: string;
    created_at: string;
    updated_at: string;
}

export interface Finding {
    id: string;
    scan_run_id: string;
    asset_id: string;
    pattern_id?: string;
    pattern_name: string;
    matches: string[];
    sample_text: string;
    severity: string;
    severity_description: string;
    confidence_score?: number;
    created_at: string;
    updated_at: string;
}

// Multi-Signal Classification Types
export interface SignalScore {
    raw_score: number;
    weighted_score: number;
    weight: number;
    confidence: number;
    explanation: string;
}

export interface EnrichmentSignals {
    environment: string;
    asset_semantics: number;
    entropy: number;
    charset_diversity: number;
}

export interface Classification {
    id: string;
    finding_id: string;
    classification_type: string;
    sub_category?: string;
    confidence_score: number;
    justification: string;
    dpdpa_category?: string;
    requires_consent: boolean;
    retention_period?: string;

    // Multi-Signal Fields
    confidence_level?: string; // "Confirmed" | "High Confidence" | "Needs Review" | "Discard"
    engine_version?: string;
    presidio_score?: number;
    rule_score?: number;
    context_score?: number;
    entropy_score?: number;
    signal_breakdown?: {
        rule_signal?: SignalScore;
        presidio_signal?: SignalScore;
        context_signal?: SignalScore;
        entropy_signal?: SignalScore;
    };
}

export interface FindingWithDetails extends Finding {
    asset_name: string;
    asset_path: string;
    environment: string;
    owner: string;
    source_system: string;
    classifications: Classification[];
    review_status: string;
}

export interface Node {
    id: string;
    label: string;
    type: string | 'system' | 'asset' | 'pii_category' | 'file' | 'table';
    risk_score: number;
    metadata: Record<string, any>;
    parent_id?: string;
}

export interface Edge {
    id: string;
    source: string;
    target: string;
    type: string;
    label: string;
}

export interface LineageGraph {
    nodes: Node[];
    edges: Edge[];
}

export interface TypeBreakdown {
    count: number;
    avg_confidence: number;
    percentage: number;
    requires_consent?: number;
}

export interface ClassificationSummary {
    total: number;
    by_type: Record<string, TypeBreakdown>;
    by_severity?: Record<string, number>;
    high_confidence_count: number;
    requiring_consent_count: number;
    verified_count: number;
    false_positive_count: number;
    dpdpa_categories: Record<string, number>;
}

export interface FindingsResponse {
    findings: FindingWithDetails[];
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
}

export interface IngestResult {
    scan_run_id: string;
    total_findings: number;
    total_assets: number;
    assets_created: number;
    patterns_found: number;
}

// Semantic Graph Types (Neo4j)
export interface SemanticGraphFilters {
    system_id?: string;
    risk_level?: string;
    category?: string;
}

export interface SemanticGraph {
    nodes: Node[];
    edges: Edge[];
}


export interface RemediationAction {
    id: string;
    finding_id: string;
    action_type: 'MASK' | 'DELETE' | 'ENCRYPT';
    executed_by: string;
    executed_at: string;
    status: 'PENDING' | 'IN_PROGRESS' | 'COMPLETED' | 'FAILED' | 'ROLLED_BACK';
    original_value?: string;
    error?: string;
}

export interface RemediationPreview {
    request_id: string;
    finding_ids: string[];
    action_type: string;
    impact: RemediationImpact;
    findings: FindingPreview[];
    requires_confirmation: boolean;
}

export interface RemediationImpact {
    total_findings: number;
    affected_assets: number;
    affected_systems: number;
    pii_types: string[];
    estimated_records: number;
}

export interface FindingPreview {
    finding_id: string;
    asset_name: string;
    asset_path: string;
    pii_type: string;
    field_name: string;
    sample_before: string;
    sample_after: string;
}

export interface UserSettings {
    // Security
    enableJWT?: boolean;
    sessionTimeout?: string;
    passwordPolicy?: string;
    twoFactorEnabled?: boolean;

    // Scanner
    scanFrequency?: string;
    maxFileSize?: string;
    supportedFormats?: string[];
    enableDeepScan?: boolean;

    // Notifications
    emailNotifications?: boolean;
    slackNotifications?: boolean;
    criticalAlertsOnly?: boolean;
    weeklyReports?: boolean;

    // Data Retention
    logRetention?: string;
    scanHistoryRetention?: string;
    backupFrequency?: string;

    // Appearance
    theme?: 'light' | 'dark' | 'system';
    density?: 'compact' | 'comfortable';
}
