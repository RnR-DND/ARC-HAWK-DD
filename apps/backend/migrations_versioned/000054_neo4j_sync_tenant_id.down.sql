DROP INDEX IF EXISTS idx_neo4j_sync_queue_tenant_status;
ALTER TABLE neo4j_sync_queue DROP COLUMN IF EXISTS tenant_id;
