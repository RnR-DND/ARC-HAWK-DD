ALTER TABLE assets
    DROP COLUMN IF EXISTS declared_purpose,
    DROP COLUMN IF EXISTS dpo_assigned,
    DROP COLUMN IF EXISTS is_orphan,
    DROP COLUMN IF EXISTS orphan_since,
    DROP COLUMN IF EXISTS orphan_risk_bonus;
