DROP VIEW IF EXISTS retention_expiring_soon;
DROP VIEW IF EXISTS retention_violations;
DROP FUNCTION IF EXISTS calculate_deletion_due(TIMESTAMPTZ, INT);
ALTER TABLE assets
    DROP COLUMN IF EXISTS retention_policy_days,
    DROP COLUMN IF EXISTS retention_policy_name,
    DROP COLUMN IF EXISTS retention_policy_basis;
