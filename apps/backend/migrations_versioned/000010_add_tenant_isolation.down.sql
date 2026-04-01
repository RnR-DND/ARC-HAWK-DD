-- Rollback: 000010_add_tenant_isolation
-- Description: Remove tenant_id columns and indexes added for multi-tenancy

DROP INDEX IF EXISTS idx_patterns_tenant;
ALTER TABLE patterns DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_relationships_tenant;
ALTER TABLE asset_relationships DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_findings_tenant;
ALTER TABLE findings DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_assets_tenant;
ALTER TABLE assets DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_source_profiles_tenant;
ALTER TABLE source_profiles DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_scan_runs_tenant;
ALTER TABLE scan_runs DROP COLUMN IF EXISTS tenant_id;
