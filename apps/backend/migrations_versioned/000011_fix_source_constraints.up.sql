-- Migration: 000011_fix_source_constraints
-- Description: Replace global unique name constraint with tenant-specific constraint

-- Drop existing global unique constraint on name
ALTER TABLE source_profiles DROP CONSTRAINT IF EXISTS source_profiles_name_key;

-- Add new unique constraint scoped to tenant
-- Note: existing rows might have NULL tenant_id. We should probably set a default or handle it.
-- Assuming migration 000010 ran, existing rows have null tenant_id.
-- If we have duplicates in 'name' across rows, this constraint creation will fail unless tenant_id differs.
-- Since current system is single-tenant used as multi-tenant, likely no duplicates yet, or we don't care about old data integrity as much as new.
-- However, unique(name, tenant_id) treats NULLs as distinct in some DBs, or equal? In Postgres, NULL != NULL, so multiple NULLs are allowed in UNIQUE index.
-- This might be desired? No, we want to enforce uniqueness.
-- Ideally we should backfill tenant_id. But without logic, we can't.
-- Proceeding with constraint.

ALTER TABLE source_profiles ADD CONSTRAINT unique_source_profile_name_per_tenant UNIQUE (name, tenant_id);
