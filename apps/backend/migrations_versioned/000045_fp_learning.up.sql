CREATE TABLE IF NOT EXISTS fp_learning (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    asset_id UUID NOT NULL,
    pattern_name VARCHAR(100) NOT NULL,
    pii_type VARCHAR(50) NOT NULL,
    field_name VARCHAR(255),
    field_path VARCHAR(500),
    matched_value VARCHAR(500) NOT NULL,
    learning_type VARCHAR(50) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    previous_value VARCHAR(500),
    justification TEXT,
    source_finding_id UUID,
    scan_run_id UUID,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fp_learning_tenant_id ON fp_learning(tenant_id);
CREATE INDEX IF NOT EXISTS idx_fp_learning_user_id ON fp_learning(user_id);
CREATE INDEX IF NOT EXISTS idx_fp_learning_asset_id ON fp_learning(asset_id);
CREATE INDEX IF NOT EXISTS idx_fp_learning_pattern_name ON fp_learning(pattern_name);
CREATE INDEX IF NOT EXISTS idx_fp_learning_pii_type ON fp_learning(pii_type);
CREATE INDEX IF NOT EXISTS idx_fp_learning_is_active ON fp_learning(is_active);
CREATE INDEX IF NOT EXISTS idx_fp_learning_created_at ON fp_learning(created_at);
CREATE INDEX IF NOT EXISTS idx_fp_learning_scan_run_id ON fp_learning(scan_run_id);
CREATE INDEX IF NOT EXISTS idx_fp_learning_expires_at ON fp_learning(expires_at);
