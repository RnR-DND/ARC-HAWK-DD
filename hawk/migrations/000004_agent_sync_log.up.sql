-- 000004_agent_sync_log.up.sql
-- EDR agent offline-sync idempotency table.
-- PK (agent_id, scan_job_id, batch_seq) ensures exactly-once ingestion.

BEGIN;

CREATE TABLE agent_sync_log (
  agent_id TEXT NOT NULL,
  scan_job_id TEXT NOT NULL,
  batch_seq INTEGER NOT NULL,
  payload_hash TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'received',  -- received, arrived_late
  received_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (agent_id, scan_job_id, batch_seq)
);
CREATE INDEX idx_agent_sync_agent ON agent_sync_log(agent_id);
CREATE INDEX idx_agent_sync_job ON agent_sync_log(scan_job_id);

INSERT INTO schema_migrations (version) VALUES ('000004') ON CONFLICT DO NOTHING;

COMMIT;
