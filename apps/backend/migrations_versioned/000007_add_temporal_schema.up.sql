-- Add temporal schema support: soft-deletes, audit logs, state transitions
-- Migration: 000007_add_temporal_schema

-- ============================================================================
-- Soft-Delete Support
-- ============================================================================

-- Add soft-delete columns
ALTER TABLE source_profiles ADD COLUMN IF NOT EXISTS disabled_at TIMESTAMP;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE findings ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Add environment field to assets (for environment-based risk scoring)
ALTER TABLE assets ADD COLUMN IF NOT EXISTS environment VARCHAR(100);

-- Add scan cancellation support
ALTER TABLE scan_runs ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMP;
ALTER TABLE scan_runs ADD COLUMN IF NOT EXISTS cancelled_by VARCHAR(255);

-- ============================================================================
-- State Transition Tracking
-- ============================================================================

CREATE TABLE IF NOT EXISTS scan_state_transitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_run_id UUID NOT NULL REFERENCES scan_runs(id),
    from_state VARCHAR(50),
    to_state VARCHAR(50) NOT NULL,
    transitioned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    transitioned_by VARCHAR(255),
    reason TEXT,
    metadata JSONB
);

CREATE INDEX idx_scan_transitions_scan ON scan_state_transitions(scan_run_id);
CREATE INDEX idx_scan_transitions_time ON scan_state_transitions(transitioned_at DESC);

-- ============================================================================
-- Audit Logs
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    event_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    action VARCHAR(100) NOT NULL,
    before_state JSONB,
    after_state JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_user ON audit_logs(user_id);
CREATE INDEX idx_audit_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_time ON audit_logs(event_time DESC);

-- ============================================================================
-- Remediation Actions
-- ============================================================================

CREATE TABLE IF NOT EXISTS remediation_actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    finding_id UUID NOT NULL REFERENCES findings(id),
    action_type VARCHAR(100) NOT NULL,  -- 'MASK', 'DELETE', 'ENCRYPT', 'QUARANTINE'
    executed_by VARCHAR(255) NOT NULL,
    executed_at TIMESTAMP NOT NULL,
    effective_from TIMESTAMP NOT NULL,
    effective_until TIMESTAMP,          -- NULL if still active
    rollback_reference UUID REFERENCES remediation_actions(id),
    status VARCHAR(50) NOT NULL,        -- 'PENDING', 'IN_PROGRESS', 'COMPLETED', 'FAILED', 'ROLLED_BACK'
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_remediation_finding ON remediation_actions(finding_id);
CREATE INDEX idx_remediation_status ON remediation_actions(status);
CREATE INDEX idx_remediation_executed_by ON remediation_actions(executed_by);
CREATE INDEX idx_remediation_executed_at ON remediation_actions(executed_at DESC);

-- ============================================================================
-- Views for Active Records (excluding soft-deleted)
-- ============================================================================

CREATE OR REPLACE VIEW active_assets AS
SELECT * FROM assets WHERE deleted_at IS NULL;

CREATE OR REPLACE VIEW active_findings AS
SELECT * FROM findings WHERE deleted_at IS NULL;

-- ============================================================================
-- Update Foreign Key Constraints (RESTRICT instead of CASCADE)
-- ============================================================================

-- Drop existing CASCADE constraints
ALTER TABLE findings DROP CONSTRAINT IF EXISTS findings_scan_run_id_fkey;
ALTER TABLE findings DROP CONSTRAINT IF EXISTS findings_asset_id_fkey;

-- Add RESTRICT constraints (prevent accidental deletes)
ALTER TABLE findings ADD CONSTRAINT findings_scan_run_id_fkey 
    FOREIGN KEY (scan_run_id) REFERENCES scan_runs(id) ON DELETE RESTRICT;

ALTER TABLE findings ADD CONSTRAINT findings_asset_id_fkey 
    FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT;

-- ============================================================================
-- Triggers for Audit Logging
-- ============================================================================

-- Function to log asset changes
CREATE OR REPLACE FUNCTION log_asset_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_logs (event_type, event_time, user_id, resource_type, resource_id, action, before_state, after_state)
        VALUES (
            'ASSET_UPDATED',
            NOW(),
            COALESCE(current_setting('app.current_user', true), 'system'),
            'asset',
            NEW.id,
            'UPDATE',
            row_to_json(OLD),
            row_to_json(NEW)
        );
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO audit_logs (event_type, event_time, user_id, resource_type, resource_id, action, before_state)
        VALUES (
            'ASSET_DELETED',
            NOW(),
            COALESCE(current_setting('app.current_user', true), 'system'),
            'asset',
            OLD.id,
            'DELETE',
            row_to_json(OLD)
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for asset changes
DROP TRIGGER IF EXISTS asset_audit_trigger ON assets;
CREATE TRIGGER asset_audit_trigger
AFTER UPDATE OR DELETE ON assets
FOR EACH ROW EXECUTE FUNCTION log_asset_changes();
