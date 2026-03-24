/*
 * Deduplication Script for Findings (Cascaded)
 * Retains only the most recent finding for each (asset_id, pattern_name) pair.
 * Handles Foreign Key constraints by deleting from child tables first.
 */

BEGIN;

-- 1. Identify IDs to delete
CREATE TEMP TABLE findings_to_delete AS
WITH duplicates AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY asset_id, pattern_name 
               ORDER BY created_at DESC
           ) as rn
    FROM findings
)
SELECT id FROM duplicates WHERE rn > 1;

-- 2. Delete from dependent tables
DELETE FROM classifications WHERE finding_id IN (SELECT id FROM findings_to_delete);
DELETE FROM finding_feedback WHERE finding_id IN (SELECT id FROM findings_to_delete);
DELETE FROM review_states WHERE finding_id IN (SELECT id FROM findings_to_delete);

-- 3. Delete from findings
DELETE FROM findings WHERE id IN (SELECT id FROM findings_to_delete);

-- 4. Clean up temp table
DROP TABLE findings_to_delete;

COMMIT;
