-- Migration 000044: Add risk_score to findings for per-finding prioritization.
-- Previously only assets had a risk_score; findings kept RiskScore=0 (P1-9).
-- The canonical formula is in modules/shared/scoring/risk_scorer.go.

ALTER TABLE findings ADD COLUMN IF NOT EXISTS risk_score INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_findings_risk_score ON findings(risk_score DESC);
