-- Add tenant_id to neo4j_sync_queue for multi-tenant isolation
ALTER TABLE neo4j_sync_queue
    ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Create composite index for efficient per-tenant polling
CREATE INDEX IF NOT EXISTS idx_neo4j_sync_queue_tenant_status
    ON neo4j_sync_queue(tenant_id, status, created_at ASC)
    WHERE status IN ('pending', 'failed', 'processing');

-- Note: tenant_id will be NOT NULL enforced at application layer going forward.
-- Existing rows will have tenant_id = NULL; backfill is a separate operational step.
