-- Revert backfilled confidence scores to the previous flat defaults.
-- NOTE: this is a best-effort rollback; original per-row values are not preserved.

UPDATE findings
SET confidence_score = 0.5
WHERE confidence_score IN (0.99, 0.95, 0.90, 0.88, 0.93, 0.75)
  AND confidence_score != 0.5;

UPDATE classifications c
SET confidence_score = 0.66
FROM findings f
WHERE c.finding_id = f.id;
