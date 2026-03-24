// Lineage Graph Types - Frozen Semantic Contract: 3-Level Hierarchy
// System → Asset → PII_Category (no intermediate DataCategory)

export type NodeType = 'system' | 'asset' | 'pii_category' | 'file' | 'table';

export interface BaseNode {
    id: string;
    label: string;
    type: NodeType;
    metadata: Record<string, any>;
}

export interface SystemNode extends BaseNode {
    type: 'system';
    metadata: {
        host?: string;
        source_system?: string;
        environment?: string;
    };
}

export interface AssetNode extends BaseNode {
    type: 'asset';
    metadata: {
        path?: string;
        environment?: string;
        risk_score?: number;
    };
}

export interface PIICategoryNode extends BaseNode {
    type: 'pii_category';
    metadata: {
        pii_type?: string; // IN_AADHAAR, CREDIT_CARD, etc.
        finding_count?: number;
        risk_level?: 'Critical' | 'High' | 'Medium' | 'Low';
        avg_confidence?: number;
        dpdpa_category?: string;
    };
}

export interface FileNode extends BaseNode {
    type: 'file';
    metadata: {
        path?: string;
        size?: number;
    };
}

export interface TableNode extends BaseNode {
    type: 'table';
    metadata: {
        schema?: string;
        rows?: number;
    };
}

export type LineageNode = SystemNode | AssetNode | PIICategoryNode | FileNode | TableNode;

// Frozen Semantic Contract: Only these edge types allowed
export interface LineageEdge {
    id: string;
    source: string;
    target: string;
    type: 'SYSTEM_OWNS_ASSET' | 'EXPOSES';
    label?: string;
    metadata?: Record<string, any>;
}

export interface LineageGraphData {
    nodes: LineageNode[];
    edges: LineageEdge[];
}

// Visual layout configuration
export const NODE_COLORS: Record<NodeType, string> = {
    system: '#3b82f6',      // Blue - Infrastructure
    asset: '#10b981',       // Green - Data containers
    pii_category: '#ef4444', // Red - PII risk
    file: '#a855f7',        // Purple - Files
    table: '#10b981',       // Green - Tables (same as asset)
};

export const NODE_SIZES: Record<NodeType, number> = {
    system: 60,
    asset: 50,
    pii_category: 45,
    file: 48,
    table: 50,
};

// Risk level colors (frozen contract)
export const RISK_COLORS = {
    Critical: '#dc2626',
    High: '#f97316',
    Medium: '#fbbf24',
    Low: '#4ade80',
};

// Graph Layout Configuration
export const DEFAULT_LAYOUT_CONFIG = {
    rankdir: 'LR', // Left to right: System → Asset → PII
    nodesep: 80,
    ranksep: 150,
    edgesep: 50,
    marginx: 50,
    marginy: 50,
};
