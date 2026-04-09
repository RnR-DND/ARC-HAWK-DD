-- Migration 000031 rollback: drop risk_score_history table.
DROP INDEX IF EXISTS idx_rsh_asset;
DROP TABLE IF EXISTS risk_score_history;
