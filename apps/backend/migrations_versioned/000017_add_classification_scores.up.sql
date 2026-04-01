-- Migration: 000017_add_classification_scores
-- Description: Add per-signal score columns to classifications table
-- These correspond to Classification entity fields: SignalBreakdown, RuleScore,
-- PresidioScore, ContextScore, EntropyScore (all missing from initial schema)

ALTER TABLE classifications
ADD COLUMN IF NOT EXISTS signal_breakdown JSONB,
ADD COLUMN IF NOT EXISTS rule_score FLOAT,
ADD COLUMN IF NOT EXISTS presidio_score FLOAT,
ADD COLUMN IF NOT EXISTS context_score FLOAT,
ADD COLUMN IF NOT EXISTS entropy_score FLOAT;
