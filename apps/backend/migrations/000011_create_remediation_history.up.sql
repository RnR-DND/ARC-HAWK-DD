-- Create remediation history table for audit trail
CREATE TABLE IF NOT EXISTS remediation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    finding_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL CHECK (action IN ('MASK', 'DELETE', 'ENCRYPT', 'ANONYMIZE')),
    target TEXT NOT NULL,
    executed_by VARCHAR(255) NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    scan_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('COMPLETED', 'FAILED', 'ROLLED_BACK', 'PENDING')),
    error_message TEXT,
    original_value TEXT,
    new_value TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_remediation_history_finding ON remediation_history(finding_id);
CREATE INDEX IF NOT EXISTS idx_remediation_history_scan ON remediation_history(scan_id);
CREATE INDEX IF NOT EXISTS idx_remediation_history_executed_at ON remediation_history(executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_remediation_history_executed_by ON remediation_history(executed_by);
CREATE INDEX IF NOT EXISTS idx_remediation_history_status ON remediation_history(status);

-- Add comment
COMMENT ON TABLE remediation_history IS 'Audit trail of all remediation actions performed on PII findings';
