-- Migration: 000010_add_tenant_isolation
-- Description: Add tenant_id to core tables for multi-tenancy isolation

-- Scan Runs
ALTER TABLE scan_runs ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_scan_runs_tenant ON scan_runs(tenant_id);

-- Source Profiles
ALTER TABLE source_profiles ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_source_profiles_tenant ON source_profiles(tenant_id);

-- Assets
ALTER TABLE assets ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_assets_tenant ON assets(tenant_id);

-- Findings
ALTER TABLE findings ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_findings_tenant ON findings(tenant_id);

-- Asset Relationships (link edges to tenant too, or rely on asset isolation)
ALTER TABLE asset_relationships ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_relationships_tenant ON asset_relationships(tenant_id);

-- Patterns (System vs Tenant patterns?)
-- For now, assume patterns are system-wide or we add tenant_id later if needed.
-- Adding it to be safe for custom patterns.
ALTER TABLE patterns ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_patterns_tenant ON patterns(tenant_id);
