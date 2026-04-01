-- Rollback: 000016_add_masked_value_to_findings

ALTER TABLE findings DROP COLUMN IF EXISTS masked_value;
