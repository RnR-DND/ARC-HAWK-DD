-- Rollback temporal schema changes
-- Migration: 000007_add_temporal_schema

-- Drop triggers
DROP TRIGGER IF EXISTS asset_audit_trigger ON assets;
DROP FUNCTION IF EXISTS log_asset_changes();

-- Drop views
DROP VIEW IF EXISTS active_findings;
DROP VIEW IF EXISTS active_assets;

-- Restore CASCADE constraints
ALTER TABLE findings DROP CONSTRAINT IF EXISTS findings_scan_run_id_fkey;
ALTER TABLE findings DROP CONSTRAINT IF EXISTS findings_asset_id_fkey;

ALTER TABLE findings ADD CONSTRAINT findings_scan_run_id_fkey 
    FOREIGN KEY (scan_run_id) REFERENCES scan_runs(id) ON DELETE CASCADE;

ALTER TABLE findings ADD CONSTRAINT findings_asset_id_fkey 
    FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE;

-- Drop tables
DROP TABLE IF EXISTS remediation_actions;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS scan_state_transitions;

-- Remove columns
ALTER TABLE scan_runs DROP COLUMN IF EXISTS cancelled_by;
ALTER TABLE scan_runs DROP COLUMN IF EXISTS cancelled_at;
ALTER TABLE assets DROP COLUMN IF EXISTS environment;
ALTER TABLE findings DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE assets DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE source_profiles DROP COLUMN IF EXISTS disabled_at;
