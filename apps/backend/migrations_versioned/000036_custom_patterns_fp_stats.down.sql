ALTER TABLE custom_patterns
    DROP COLUMN IF EXISTS false_positive_count,
    DROP COLUMN IF EXISTS false_positive_rate,
    DROP COLUMN IF EXISTS auto_deactivated;
