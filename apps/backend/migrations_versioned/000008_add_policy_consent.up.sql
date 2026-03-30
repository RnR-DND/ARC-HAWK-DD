-- Add policy and consent management schema
-- Migration: 000008_add_policy_consent

-- ============================================================================
-- Policies Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    policy_type VARCHAR(100) NOT NULL,  -- 'REMEDIATION', 'RETENTION', 'CONSENT'
    conditions JSONB NOT NULL,          -- Policy conditions (e.g., {"pii_type": "IN_AADHAAR", "environment": "development"})
    actions JSONB NOT NULL,             -- Policy actions (e.g., [{"type": "MASK", "auto_execute": true}])
    is_active BOOLEAN DEFAULT true,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_policies_type ON policies(policy_type);
CREATE INDEX IF NOT EXISTS idx_policies_active ON policies(is_active);
CREATE INDEX IF NOT EXISTS idx_policies_created_by ON policies(created_by);

-- ============================================================================
-- Policy Executions Table (audit trail)
-- ============================================================================

CREATE TABLE IF NOT EXISTS policy_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID NOT NULL REFERENCES policies(id),
    finding_id UUID NOT NULL REFERENCES findings(id),
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    result VARCHAR(50) NOT NULL,  -- 'SUCCESS', 'FAILED', 'SKIPPED'
    error_message TEXT,
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_policy_exec_policy ON policy_executions(policy_id);
CREATE INDEX IF NOT EXISTS idx_policy_exec_finding ON policy_executions(finding_id);
CREATE INDEX IF NOT EXISTS idx_policy_exec_time ON policy_executions(executed_at DESC);

-- ============================================================================
-- Consent Records Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS consent_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_subject_id VARCHAR(255) NOT NULL,  -- PII hash or pseudonym
    pii_type VARCHAR(100) NOT NULL,
    consent_given_at TIMESTAMP NOT NULL,
    consent_expires_at TIMESTAMP,
    consent_withdrawn_at TIMESTAMP,
    consent_purpose TEXT NOT NULL,
    consent_source VARCHAR(255),            -- 'web_form', 'mobile_app', 'email'
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_consent_subject ON consent_records(data_subject_id);
CREATE INDEX IF NOT EXISTS idx_consent_type ON consent_records(pii_type);
CREATE INDEX IF NOT EXISTS idx_consent_expiry ON consent_records(consent_expires_at);
CREATE INDEX IF NOT EXISTS idx_consent_withdrawn ON consent_records(consent_withdrawn_at);

-- ============================================================================
-- Trigger for policy updated_at
-- ============================================================================

DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;
CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

