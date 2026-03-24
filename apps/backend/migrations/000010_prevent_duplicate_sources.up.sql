-- Migration: Add unique constraint for duplicate source prevention
-- Prevents same source from being added multiple times per tenant

-- Add unique constraint on sources table
ALTER TABLE sources 
ADD CONSTRAINT unique_source_per_tenant 
UNIQUE (tenant_id, profile_name);

-- Note: This will fail if duplicates already exist
-- To handle existing duplicates, run cleanup first:
-- DELETE FROM sources 
-- WHERE id NOT IN (
--   SELECT MIN(id) FROM sources GROUP BY tenant_id, profile_name
-- );
