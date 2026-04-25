CREATE TABLE IF NOT EXISTS neo4j_sync_queue (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation   TEXT NOT NULL,
    payload     JSONB NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    attempts    INT NOT NULL DEFAULT 0,
    last_error  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_neo4j_sync_queue_status ON neo4j_sync_queue(status) WHERE status IN ('pending','failed');
