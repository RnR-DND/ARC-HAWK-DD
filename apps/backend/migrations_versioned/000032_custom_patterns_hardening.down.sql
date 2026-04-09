ALTER TABLE custom_patterns
    DROP COLUMN IF EXISTS validation_status,
    DROP COLUMN IF EXISTS backtrack_safe,
    DROP COLUMN IF EXISTS match_count_lifetime,
    DROP COLUMN IF EXISTS last_matched_at,
    DROP COLUMN IF EXISTS test_positives,
    DROP COLUMN IF EXISTS test_negatives;
