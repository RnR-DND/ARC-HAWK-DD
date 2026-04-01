-- Rollback: 000017_add_classification_scores

ALTER TABLE classifications
DROP COLUMN IF EXISTS signal_breakdown,
DROP COLUMN IF EXISTS rule_score,
DROP COLUMN IF EXISTS presidio_score,
DROP COLUMN IF EXISTS context_score,
DROP COLUMN IF EXISTS entropy_score;
