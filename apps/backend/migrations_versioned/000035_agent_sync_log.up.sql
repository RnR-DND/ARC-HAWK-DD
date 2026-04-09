-- Agent sync log for idempotent batch ingestion from EDR agents.
-- PK (agent_id, scan_job_id, batch_seq) ensures each batch is processed exactly once.
CREATE TABLE IF NOT EXISTS agent_sync_log (
    agent_id     TEXT        NOT NULL,
    scan_job_id  TEXT        NOT NULL,
    batch_seq    INTEGER     NOT NULL,
    payload_hash TEXT        NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'received',
    received_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (agent_id, scan_job_id, batch_seq)
);

CREATE INDEX IF NOT EXISTS idx_agent_sync_agent ON agent_sync_log(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_sync_job   ON agent_sync_log(scan_job_id);
