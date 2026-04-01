-- Rollback: 000011_fix_source_constraints
-- Description: Restore global unique constraint on source_profiles.name

ALTER TABLE source_profiles
DROP CONSTRAINT IF EXISTS unique_source_profile_name_per_tenant;

ALTER TABLE source_profiles
ADD CONSTRAINT source_profiles_name_key UNIQUE (name);
