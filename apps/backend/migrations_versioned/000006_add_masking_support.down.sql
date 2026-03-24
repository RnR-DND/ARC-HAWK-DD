-- ARC Platform Database Schema - Rollback Masking Support
-- Migration: 000002_add_masking_support (DOWN)

-- Drop masking audit log table
DROP TABLE IF EXISTS masking_audit_log;

-- Remove masked value column from findings
ALTER TABLE findings
DROP COLUMN IF EXISTS masked_value;

-- Remove masking columns from assets
DROP INDEX IF EXISTS idx_assets_is_masked;

ALTER TABLE assets
DROP COLUMN IF EXISTS masking_strategy,
DROP COLUMN IF EXISTS masked_at,
DROP COLUMN IF EXISTS is_masked;
