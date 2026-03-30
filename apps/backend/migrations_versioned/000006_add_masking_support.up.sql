-- ARC Platform Database Schema - Add Masking Support
-- Migration: 000002_add_masking_support

-- ============================================================================
-- Add Masking Columns to Assets Table
-- ============================================================================

ALTER TABLE assets
ADD COLUMN IF NOT EXISTS is_masked BOOLEAN DEFAULT false NOT NULL;

ALTER TABLE assets
ADD COLUMN IF NOT EXISTS masked_at TIMESTAMP;

ALTER TABLE assets
ADD COLUMN IF NOT EXISTS masking_strategy VARCHAR(50);

-- Add index for filtering masked assets
CREATE INDEX IF NOT EXISTS idx_assets_is_masked ON assets(is_masked);

-- ============================================================================
-- Add Masked Value Column to Findings Table
-- ============================================================================

ALTER TABLE findings
ADD COLUMN IF NOT EXISTS masked_value TEXT;

-- ============================================================================
-- Create Masking Audit Log Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS masking_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    masked_by VARCHAR(255),
    masking_strategy VARCHAR(50) NOT NULL,
    findings_count INTEGER DEFAULT 0,
    masked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for audit log queries
CREATE INDEX IF NOT EXISTS idx_masking_audit_asset ON masking_audit_log(asset_id);
CREATE INDEX IF NOT EXISTS idx_masking_audit_masked_at ON masking_audit_log(masked_at DESC);

-- ============================================================================
-- Comments for Documentation
-- ============================================================================

COMMENT ON COLUMN assets.is_masked IS 'Indicates if the asset has been masked';
COMMENT ON COLUMN assets.masked_at IS 'Timestamp when the asset was masked';
COMMENT ON COLUMN assets.masking_strategy IS 'Strategy used for masking: REDACT, PARTIAL, or TOKENIZE';
COMMENT ON COLUMN findings.masked_value IS 'Masked representation of the PII value';
COMMENT ON TABLE masking_audit_log IS 'Audit trail for all masking operations';
