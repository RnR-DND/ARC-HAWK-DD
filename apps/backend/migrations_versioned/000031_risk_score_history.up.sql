-- Migration 000031: Per-asset risk score history for DPDPA trend analysis.
-- Complements discovery_risk_scores (which is tenant+snapshot scoped) by persisting
-- every individual ComputeDPDPAScore() call in a compact, time-series-friendly table.
CREATE TABLE IF NOT EXISTS risk_score_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    score FLOAT NOT NULL,
    tier TEXT NOT NULL,
    pii_density FLOAT,
    sensitivity_weight FLOAT,
    access_exposure FLOAT,
    retention_violation FLOAT,
    scan_run_id UUID,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rsh_asset ON risk_score_history(asset_id, recorded_at DESC);
