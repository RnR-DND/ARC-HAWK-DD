-- Rollback: Remove unique constraint for duplicate source prevention

ALTER TABLE sources 
DROP CONSTRAINT IF EXISTS unique_source_per_tenant;
