-- Migration: 000016_add_masked_value_to_findings
-- Description: Ensure masked_value column exists on findings table
-- (000006 adds this conditionally; this migration makes it explicit and idempotent)

ALTER TABLE findings ADD COLUMN IF NOT EXISTS masked_value TEXT;
