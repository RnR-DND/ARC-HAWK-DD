-- ARC Platform Database Schema - Initial Schema
-- Migration: 000001_initial_schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- Core Tables
-- ============================================================================

-- Scan Runs: Track each scan execution
CREATE TABLE scan_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_name VARCHAR(255) NOT NULL,
    scan_started_at TIMESTAMP NOT NULL,
    scan_completed_at TIMESTAMP NOT NULL,
    host VARCHAR(255),
    total_findings INTEGER DEFAULT 0,
    total_assets INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'completed',
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Source Profiles: Scanner configuration profiles
CREATE TABLE source_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    data_source_type VARCHAR(100) NOT NULL,
    configuration JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Assets: Normalized files/resources with stable identifiers
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stable_id VARCHAR(255) UNIQUE NOT NULL,
    asset_type VARCHAR(100) NOT NULL,
    name VARCHAR(500) NOT NULL,
    path TEXT NOT NULL,
    data_source VARCHAR(100) NOT NULL,
    host VARCHAR(255),
    environment VARCHAR(100),
    owner VARCHAR(255),
    source_system VARCHAR(255),
    file_metadata JSONB,
    risk_score INTEGER DEFAULT 0,
    total_findings INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Patterns: Detection pattern definitions
CREATE TABLE patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    pattern_type VARCHAR(100) NOT NULL,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    pattern_definition TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Findings: Individual PII/secret detections
CREATE TABLE findings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_run_id UUID NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    pattern_id UUID REFERENCES patterns(id),
    pattern_name VARCHAR(255) NOT NULL,
    matches TEXT[],
    sample_text TEXT,
    severity VARCHAR(50) NOT NULL,
    severity_description TEXT,
    confidence_score DECIMAL(5,2),
    enrichment_score DECIMAL(5,2),
    enrichment_signals JSONB,
    enrichment_failed BOOLEAN DEFAULT false,
    context JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Classifications: PII classification with confidence scores
CREATE TABLE classifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    finding_id UUID NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    classification_type VARCHAR(100) NOT NULL,
    sub_category VARCHAR(100),
    confidence_score DECIMAL(5,2) NOT NULL,
    justification TEXT,
    dpdpa_category VARCHAR(100),
    requires_consent BOOLEAN DEFAULT false,
    retention_period VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Asset Relationships: Graph edges between assets
CREATE TABLE asset_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    target_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    relationship_type VARCHAR(100) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_asset_id, target_asset_id, relationship_type)
);

-- Review States: Audit trail for finding reviews
CREATE TABLE review_states (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    finding_id UUID NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP,
    comments TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- Indexes for Query Performance
-- ============================================================================

CREATE INDEX idx_scan_runs_profile ON scan_runs(profile_name);
CREATE INDEX idx_scan_runs_started ON scan_runs(scan_started_at DESC);
CREATE INDEX idx_scan_runs_status ON scan_runs(status);

CREATE INDEX idx_assets_stable_id ON assets(stable_id);
CREATE INDEX idx_assets_type ON assets(asset_type);
CREATE INDEX idx_assets_source ON assets(data_source);
CREATE INDEX idx_assets_risk ON assets(risk_score DESC);

CREATE INDEX idx_findings_scan_run ON findings(scan_run_id);
CREATE INDEX idx_findings_asset ON findings(asset_id);
CREATE INDEX idx_findings_pattern ON findings(pattern_id);
CREATE INDEX idx_findings_severity ON findings(severity);
CREATE INDEX idx_findings_created ON findings(created_at DESC);

CREATE INDEX idx_classifications_finding ON classifications(finding_id);
CREATE INDEX idx_classifications_type ON classifications(classification_type);
CREATE INDEX idx_classifications_confidence ON classifications(confidence_score DESC);

CREATE INDEX idx_relationships_source ON asset_relationships(source_asset_id);
CREATE INDEX idx_relationships_target ON asset_relationships(target_asset_id);
CREATE INDEX idx_relationships_type ON asset_relationships(relationship_type);

CREATE INDEX idx_review_states_finding ON review_states(finding_id);
CREATE INDEX idx_review_states_status ON review_states(status);

-- ============================================================================
-- Triggers for updated_at timestamps
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_scan_runs_updated_at BEFORE UPDATE ON scan_runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_source_profiles_updated_at BEFORE UPDATE ON source_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_assets_updated_at BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_findings_updated_at BEFORE UPDATE ON findings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_patterns_updated_at BEFORE UPDATE ON patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_classifications_updated_at BEFORE UPDATE ON classifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_review_states_updated_at BEFORE UPDATE ON review_states
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
